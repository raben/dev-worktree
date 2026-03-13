package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/selector"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [name]",
	Short: "Open the environment's port in a browser",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runOpen,
}

func init() {
	rootCmd.AddCommand(openCmd)
}

func runOpen(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	envs, err := dc.List(ctx)
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	var devKey string
	if len(args) > 0 {
		devKey, err = resolveDevKeyFromName(args[0])
		if err != nil {
			return err
		}
	} else {
		devKey, err = selector.Select(envs, "running")
		if err != nil {
			return err
		}
	}

	// Find the environment
	var env *container.Environment
	for i, e := range envs {
		if e.Key == devKey {
			env = &envs[i]
			break
		}
	}
	if env == nil {
		return fmt.Errorf("environment '%s' not found", devKey)
	}

	if len(env.Ports) == 0 {
		return fmt.Errorf("environment '%s' has no exposed ports", devKey)
	}

	// Open the first port
	url := fmt.Sprintf("http://localhost:%d", env.Ports[0].HostPort)
	fmt.Printf("Opening %s ...\n", url)

	return openBrowser(url)
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
