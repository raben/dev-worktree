package cmd

import (
	"context"
	"fmt"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/selector"
	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down [name]",
	Short: "Stop a container (worktree is kept)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDown,
}

func runDown(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	var devKey string
	if len(args) > 0 {
		devKey, err = resolveDevKeyFromName(args[0])
		if err != nil {
			return err
		}
	} else {
		envs, err := dc.List(ctx)
		if err != nil {
			return fmt.Errorf("listing environments: %w", err)
		}
		devKey, err = selector.Select(envs, "running")
		if err != nil {
			return err
		}
	}

	running, err := dc.IsRunning(ctx, devKey)
	if err != nil {
		return fmt.Errorf("checking status: %w", err)
	}
	if !running {
		fmt.Printf("Environment '%s' is not running.\n", devKey)
		return nil
	}

	fmt.Printf("Stopping '%s'...\n", devKey)
	if err := dc.Down(ctx, devKey); err != nil {
		return fmt.Errorf("stopping container: %w", err)
	}

	fmt.Printf("Stopped '%s'. Worktree is kept.\n", devKey)
	return nil
}
