package cmd

import (
	"fmt"

	"github.com/autor-dev/dev-worktree/internal/browser"
	"github.com/autor-dev/dev-worktree/internal/session"
	"github.com/spf13/cobra"
)

var openPort int

var openCmd = &cobra.Command{
	Use:   "open",
	Short: "Open a container port in the browser",
	Args:  cobra.NoArgs,
	RunE:  runOpen,
}

func init() {
	openCmd.Flags().IntVarP(&openPort, "port", "p", 3000, "Port to open")
}

func runOpen(cmd *cobra.Command, args []string) error {
	if !session.Exists() {
		return fmt.Errorf("no active session. Run 'dev' first")
	}

	url := fmt.Sprintf("http://localhost:%d", openPort)
	fmt.Printf("Opening %s ...\n", url)
	return browser.Open(url)
}
