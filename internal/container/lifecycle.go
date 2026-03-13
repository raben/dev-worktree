package container

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
)

// Environment represents a running dev-worktree environment's container info.
type Environment struct {
	Key       string        // "project/name"
	Path      string        // worktree filesystem path
	State     string        // "running", "exited", etc.
	Container string        // container ID
	Ports     []PortBinding // exposed port mappings
}

// PortBinding represents a host-to-container port mapping.
type PortBinding struct {
	HostPort      int
	ContainerPort int
}

// Up creates and starts a container for the given worktree.
// key is the dev-worktree key (e.g., "myapp/feature-auth"), wtPath is the
// worktree directory path, image is the Docker image, portBindings maps
// hostPort to containerPort, and version is the CLI version string.
// Returns the container ID.
//
// If a stopped container already exists for the given key, it is restarted
// instead of creating a new one.
func (c *Client) Up(ctx context.Context, key, wtPath, image, version string, portBindings map[int]int) (string, error) {
	// Check for an existing stopped container with this key.
	existing, err := c.docker.ContainerList(ctx, containertypes.ListOptions{
		All:     true,
		Filters: FilterByKey(key),
	})
	if err != nil {
		return "", fmt.Errorf("list existing containers: %w", err)
	}
	for _, ctr := range existing {
		if ctr.State != "running" {
			// Restart the stopped container.
			if err := c.docker.ContainerStart(ctx, ctr.ID, containertypes.StartOptions{}); err != nil {
				return "", fmt.Errorf("restart container: %w", err)
			}
			return ctr.ID, nil
		}
	}

	containerName := sanitizeName(key)

	// Build port bindings for the host config.
	exposedPorts := nat.PortSet{}
	portMap := nat.PortMap{}
	for hostPort, containerPort := range portBindings {
		cp := nat.Port(fmt.Sprintf("%d/tcp", containerPort))
		exposedPorts[cp] = struct{}{}
		portMap[cp] = []nat.PortBinding{
			{HostIP: "0.0.0.0", HostPort: strconv.Itoa(hostPort)},
		}
	}

	resp, err := c.docker.ContainerCreate(ctx,
		&containertypes.Config{
			Image:        image,
			Cmd:          []string{"sleep", "infinity"},
			Labels:       Labels(key, wtPath, version),
			WorkingDir:   "/workspace",
			ExposedPorts: exposedPorts,
		},
		&containertypes.HostConfig{
			Binds:        []string{wtPath + ":/workspace:cached"},
			PortBindings: portMap,
		},
		nil, // network config
		nil, // platform
		containerName,
	)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if err := c.docker.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

// Down stops containers for the given dev key.
func (c *Client) Down(ctx context.Context, key string) error {
	containers, err := c.docker.ContainerList(ctx, containertypes.ListOptions{
		Filters: FilterByKey(key),
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	var errs []error
	for _, ctr := range containers {
		if ctr.State != "running" {
			continue
		}
		if err := c.docker.ContainerStop(ctx, ctr.ID, containertypes.StopOptions{}); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", ctr.ID[:12], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("down: %v", errs)
	}
	return nil
}

// Remove force-removes containers and associated volumes for the given dev key.
func (c *Client) Remove(ctx context.Context, key string) error {
	containers, err := c.docker.ContainerList(ctx, containertypes.ListOptions{
		All:     true,
		Filters: FilterByKey(key),
	})
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}

	var errs []error
	for _, ctr := range containers {
		if err := c.docker.ContainerRemove(ctx, ctr.ID, containertypes.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		}); err != nil {
			errs = append(errs, fmt.Errorf("remove %s: %w", ctr.ID[:12], err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("remove: %v", errs)
	}
	return nil
}

// List returns all dev-worktree environments.
// When multiple containers share the same key, running containers are preferred.
func (c *Client) List(ctx context.Context) ([]Environment, error) {
	containers, err := c.docker.ContainerList(ctx, containertypes.ListOptions{
		All:     true,
		Filters: FilterAll(),
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	seen := make(map[string]Environment)
	for _, ctr := range containers {
		key := ctr.Labels[LabelKey]
		env := Environment{
			Key:       key,
			Path:      ctr.Labels[LabelPath],
			State:     ctr.State,
			Container: ctr.ID,
			Ports:     extractPorts(ctr.Ports),
		}

		existing, exists := seen[key]
		if !exists || (existing.State != "running" && env.State == "running") {
			seen[key] = env
		}
	}

	envs := make([]Environment, 0, len(seen))
	for _, env := range seen {
		envs = append(envs, env)
	}
	return envs, nil
}

// IsRunning checks if a container with the given key is running.
func (c *Client) IsRunning(ctx context.Context, key string) (bool, error) {
	containers, err := c.docker.ContainerList(ctx, containertypes.ListOptions{
		Filters: FilterByKey(key),
	})
	if err != nil {
		return false, fmt.Errorf("list containers: %w", err)
	}
	for _, ctr := range containers {
		if ctr.State == "running" {
			return true, nil
		}
	}
	return false, nil
}

// extractPorts converts Docker port types to PortBinding slices.
func extractPorts(ports []types.Port) []PortBinding {
	var bindings []PortBinding
	for _, p := range ports {
		if p.PublicPort == 0 {
			continue
		}
		bindings = append(bindings, PortBinding{
			HostPort:      int(p.PublicPort),
			ContainerPort: int(p.PrivatePort),
		})
	}
	return bindings
}

// sanitizeName converts a dev key like "project/branch" to a valid container
// name like "project-branch".
func sanitizeName(key string) string {
	return strings.ReplaceAll(key, "/", "-")
}
