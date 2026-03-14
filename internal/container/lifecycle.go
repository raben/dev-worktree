package container

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/image"
	containertypes "github.com/docker/docker/api/types/container"
)

type CreateParams struct {
	Image  string
	Binds  []string
	Env    []string
	Labels map[string]string
}

func (c *Client) Create(ctx context.Context, params CreateParams) (string, error) {
	if err := c.ensureImage(ctx, params.Image); err != nil {
		return "", fmt.Errorf("pulling image: %w", err)
	}

	labels := params.Labels
	if labels == nil {
		labels = Labels()
	}

	resp, err := c.docker.ContainerCreate(ctx,
		&containertypes.Config{
			Image:      params.Image,
			Cmd:        []string{"sleep", "infinity"},
			Labels:     labels,
			WorkingDir: "/workspace",
			Env:        params.Env,
		},
		&containertypes.HostConfig{
			Binds: params.Binds,
		},
		nil, nil, "",
	)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if err := c.docker.ContainerStart(ctx, resp.ID, containertypes.StartOptions{}); err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

func (c *Client) ensureImage(ctx context.Context, img string) error {
	_, _, err := c.docker.ImageInspectWithRaw(ctx, img)
	if err == nil {
		return nil
	}
	fmt.Printf("Pulling image %s...\n", img)
	reader, err := c.docker.ImagePull(ctx, img, image.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()
	_, err = io.Copy(io.Discard, reader)
	return err
}

func (c *Client) Stop(ctx context.Context, containerID string) error {
	return c.docker.ContainerStop(ctx, containerID, containertypes.StopOptions{})
}

func (c *Client) Remove(ctx context.Context, containerID string) error {
	return c.docker.ContainerRemove(ctx, containerID, containertypes.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})
}

func (c *Client) ContainerState(ctx context.Context, containerID string) (string, error) {
	info, err := c.docker.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", err
	}
	return info.State.Status, nil
}

func (c *Client) InstallClaude(ctx context.Context, containerID string) error {
	fmt.Println("Installing Claude Code...")
	output, err := c.Exec(ctx, containerID, []string{"npm", "install", "-g", "@anthropic-ai/claude-code"})
	if err != nil {
		return fmt.Errorf("installing Claude Code: %w\n%s", err, output)
	}
	return nil
}

func (c *Client) StartClaude(ctx context.Context, containerID string, safe bool) error {
	cmd := []string{"claude", "--dangerously-skip-permissions"}
	if safe {
		cmd = []string{"claude"}
	}
	return c.ExecInteractive(ctx, containerID, cmd)
}
