package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/selector"
	"github.com/autor-dev/dev-worktree/internal/worktree"
	"github.com/spf13/cobra"
)

var pruneCmd = &cobra.Command{
	Use:   "prune [name]",
	Short: "Remove worktree, containers, and branch",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPrune,
}

func init() {
	rootCmd.AddCommand(pruneCmd)
}

func runPrune(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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
		devKey, err = selector.Select(envs, "")
		if err != nil {
			return err
		}
	}

	// Extract the name part after "/"
	parts := strings.SplitN(devKey, "/", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid dev key format: %s", devKey)
	}
	name := parts[1]

	// Acquire lock
	unlock, err := worktree.Lock(devKey)
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer unlock()

	// First confirmation
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Remove '%s'? [y/N]: ", devKey)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" {
		fmt.Println("Aborted.")
		return nil
	}

	// Set up worktree manager
	projectDir, err := findProjectDir()
	if err != nil {
		return fmt.Errorf("finding project directory: %w", err)
	}
	wm, err := newWorktreeManager(projectDir)
	if err != nil {
		return fmt.Errorf("creating worktree manager: %w", err)
	}

	// Check for uncommitted changes
	if wm.Exists(name) {
		dirty, err := wm.HasUncommittedChanges(name)
		if err != nil {
			return fmt.Errorf("checking worktree status: %w", err)
		}
		if dirty {
			fmt.Printf("Worktree has uncommitted changes. Continue? [y/N]: ")
			answer, _ = reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" {
				fmt.Println("Aborted.")
				return nil
			}
		}
	}

	// Remove containers
	fmt.Printf("Removing containers for '%s'...\n", devKey)
	if err := dc.Remove(ctx, devKey); err != nil {
		return fmt.Errorf("removing containers: %w", err)
	}

	// Remove worktree (force mode)
	if wm.Exists(name) {
		fmt.Printf("Removing worktree '%s'...\n", name)
		if err := wm.Remove(name, true); err != nil {
			return fmt.Errorf("removing worktree: %w", err)
		}
	}

	// Delete branch (warn on failure, don't error)
	if err := wm.DeleteBranch(name); err != nil {
		fmt.Printf("Warning: could not delete branch '%s': %v\n", name, err)
	}

	fmt.Printf("Pruned '%s'.\n", devKey)
	return nil
}
