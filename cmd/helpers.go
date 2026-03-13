package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/autor-dev/dev-worktree/internal/worktree"
)

// findProjectDir walks up from cwd to find a git repository root.
func findProjectDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository found")
		}
		dir = parent
	}
}

// newWorktreeManager creates a worktree manager for the current project.
func newWorktreeManager(projectDir string) (*worktree.Manager, error) {
	return worktree.NewManager(projectDir)
}

// resolveDevKeyFromName resolves a full dev key from a name argument.
func resolveDevKeyFromName(name string) (string, error) {
	if strings.Contains(name, "/") {
		return name, nil
	}
	projectDir, err := findProjectDir()
	if err != nil {
		return "", fmt.Errorf("not in a git repository and no full key provided: %w", err)
	}
	wm, err := newWorktreeManager(projectDir)
	if err != nil {
		return "", err
	}
	return wm.DevKey(name), nil
}
