package cmd

import (
	"errors"
	"fmt"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/session"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop and remove the dev environment",
	Args:  cobra.NoArgs,
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	sess, err := session.Load()
	if errors.Is(err, session.ErrNoSession) {
		fmt.Println("No active session.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("loading session: %w", err)
	}

	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	fmt.Println("Stopping environment...")
	dc.Stop(ctx, sess.ContainerID)
	dc.Remove(ctx, sess.ContainerID)

	if err := session.Clear(); err != nil {
		return fmt.Errorf("clearing session: %w", err)
	}

	fmt.Println("Stopped.")
	return nil
}
