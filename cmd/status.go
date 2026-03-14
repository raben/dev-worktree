package cmd

import (
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/session"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show dev environment status",
	Args:  cobra.NoArgs,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	sess, err := session.Load()
	if errors.Is(err, session.ErrNoSession) {
		fmt.Println("No active session. Run 'dev' to start.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("loading session: %w", err)
	}

	state := "unknown"
	dc, dcErr := container.NewClient()
	if dcErr == nil {
		defer dc.Close()
		s, err := dc.ContainerState(ctx, sess.ContainerID)
		if err != nil {
			state = "not found"
		} else {
			state = s
		}
	}

	cid := sess.ContainerID
	if len(cid) > 12 {
		cid = cid[:12]
	}

	fmt.Printf("Image:     %s\n", sess.Image)
	fmt.Printf("Container: %s\n", cid)
	fmt.Printf("State:     %s\n", state)
	fmt.Println()

	if len(sess.Repos) == 0 {
		fmt.Println("No repositories mounted. Run 'dev add' in a git repository.")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tMOUNT")
		for _, r := range sess.Repos {
			fmt.Fprintf(w, "%s\t%s\n", r.Name, r.ContainerPath)
		}
		w.Flush()
	}

	return nil
}
