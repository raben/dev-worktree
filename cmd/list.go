package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/spf13/cobra"
)

var listJSON bool

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all dev-worktree environments",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output as JSON")
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	envs, err := dc.List(ctx)
	if err != nil {
		return fmt.Errorf("listing environments: %w", err)
	}

	if len(envs) == 0 {
		fmt.Println("No environments. Run 'dev up <name>' to create one.")
		return nil
	}

	if listJSON {
		return printJSON(envs)
	}

	return printTable(envs)
}

func printTable(envs []container.Environment) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tPORTS")
	fmt.Fprintln(w, "----\t------\t-----")

	for _, e := range envs {
		ports := ""
		for i, p := range e.Ports {
			if i > 0 {
				ports += " "
			}
			ports += fmt.Sprintf("%d→%d", p.HostPort, p.ContainerPort)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Key, e.State, ports)
	}

	return w.Flush()
}

func printJSON(envs []container.Environment) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(envs)
}
