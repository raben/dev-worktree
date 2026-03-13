package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/dashboard"
	"github.com/spf13/cobra"
)

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "Open a TUI dashboard monitoring all environments",
	Args:  cobra.NoArgs,
	RunE:  runDash,
}

func init() {
	rootCmd.AddCommand(dashCmd)
}

func runDash(cmd *cobra.Command, args []string) error {
	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	m := dashboard.New(dc)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("dashboard error: %w", err)
	}

	return nil
}
