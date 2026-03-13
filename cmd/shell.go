package cmd

import (
	"fmt"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell [name]",
	Short: "Open a shell in a running container",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runShell,
}

func init() {
	rootCmd.AddCommand(shellCmd)
}

func runShell(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	devKey, containerID, err := resolveRunningEnv(ctx, dc, args)
	if err != nil {
		return err
	}

	fmt.Printf("Opening shell in '%s'...\n", devKey)
	return dc.ExecInteractive(ctx, containerID, []string{"bash"})
}
