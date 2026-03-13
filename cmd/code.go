package cmd

import (
	"context"
	"fmt"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/selector"
	"github.com/spf13/cobra"
)

var codeSafe bool

var codeCmd = &cobra.Command{
	Use:   "code [name]",
	Short: "Start a Claude coding session in a container",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCode,
}

func init() {
	codeCmd.Flags().BoolVar(&codeSafe, "safe", false, "Run Claude without --dangerously-skip-permissions")
	rootCmd.AddCommand(codeCmd)
}

func runCode(cmd *cobra.Command, args []string) error {
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

	execCmd := []string{"claude"}
	if !codeSafe {
		execCmd = append(execCmd, "--dangerously-skip-permissions")
	}

	fmt.Printf("Starting Claude session in '%s'...\n", devKey)
	return dc.ExecInteractive(ctx, containerID, execCmd)
}

// resolveRunningEnv resolves a running environment from args or interactive selection.
// Returns devKey and containerID.
func resolveRunningEnv(ctx context.Context, dc *container.Client, args []string) (string, string, error) {
	envs, err := dc.List(ctx)
	if err != nil {
		return "", "", fmt.Errorf("listing environments: %w", err)
	}

	var devKey string
	if len(args) > 0 {
		devKey, err = resolveDevKeyFromName(args[0])
		if err != nil {
			return "", "", err
		}

		// Find running container for this key
		for _, e := range envs {
			if e.Key == devKey && e.State == "running" {
				return devKey, e.Container, nil
			}
		}
		return "", "", fmt.Errorf("environment '%s' is not running. Run 'dev up %s' first", devKey, args[0])
	}

	// Interactive selection
	devKey, err = selector.Select(envs, "running")
	if err != nil {
		return "", "", err
	}

	for _, e := range envs {
		if e.Key == devKey && e.State == "running" {
			return devKey, e.Container, nil
		}
	}
	return "", "", fmt.Errorf("environment '%s' not found", devKey)
}
