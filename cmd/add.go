package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/session"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Mount the current repository into the dev environment",
	Args:  cobra.NoArgs,
	RunE:  runAdd,
}

func runAdd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	sess, err := session.Load()
	if errors.Is(err, session.ErrNoSession) {
		return fmt.Errorf("no active session. Run 'dev' first")
	}
	if err != nil {
		return fmt.Errorf("loading session: %w", err)
	}

	repoDir, err := findGitRoot()
	if err != nil {
		return err
	}

	absPath, err := filepath.Abs(repoDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	if sess.HasRepo(absPath) {
		fmt.Printf("Repository '%s' is already mounted.\n", filepath.Base(absPath))
		return nil
	}

	name := filepath.Base(absPath)
	// Check for name collision
	for _, r := range sess.Repos {
		if r.Name == name {
			return fmt.Errorf("name collision: '%s' already used by %s", name, r.HostPath)
		}
	}

	sess.Repos = append(sess.Repos, session.RepoMount{
		HostPath:      absPath,
		ContainerPath: "/workspace/" + name,
		Name:          name,
	})

	// Recreate container with new mounts
	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	fmt.Printf("Adding '%s'...\n", name)

	oldContainerID := sess.ContainerID

	// Stop old container before creating new one (port conflicts)
	dc.Stop(ctx, oldContainerID)

	// Create new container with all mounts
	containerID, err := dc.Create(ctx, container.CreateParams{
		Image:  sess.Image,
		Binds:  sess.Binds(),
		Labels: container.Labels(),
		Env:    envVars(),
	})
	if err != nil {
		// Rollback: remove new repo from session
		sess.Repos = sess.Repos[:len(sess.Repos)-1]
		return fmt.Errorf("creating container: %w", err)
	}
	sess.ContainerID = containerID

	// Remove old container now that new one is running
	dc.Remove(ctx, oldContainerID)

	if err := dc.InstallClaude(ctx, containerID); err != nil {
		dc.Remove(ctx, containerID)
		return err
	}

	if err := session.Save(sess); err != nil {
		dc.Remove(ctx, containerID)
		return fmt.Errorf("saving session: %w", err)
	}

	fmt.Printf("Added. Repository mounted at /workspace/%s\n", name)
	return nil
}

func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a git repository")
		}
		dir = parent
	}
}
