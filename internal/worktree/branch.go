package worktree

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing"
)

// ProtectedBranches that should not be directly used as worktree names.
var ProtectedBranches = []string{"main", "master", "develop"}

// IsProtected checks if a branch name is protected.
func IsProtected(name string) bool {
	for _, b := range ProtectedBranches {
		if name == b {
			return true
		}
	}
	return false
}

// DeleteBranch deletes the branch for a given name.
// Uses safe delete (equivalent to git branch -d). Returns error if not fully merged.
func (m *Manager) DeleteBranch(name string) error {
	branchRef := plumbing.NewBranchReferenceName(m.BranchName(name))

	// Verify the branch reference exists before attempting deletion.
	if _, err := m.repo.Reference(branchRef, false); err != nil {
		return fmt.Errorf("worktree: branch %q not found: %w", m.BranchName(name), err)
	}

	// Remove the reference from the store.
	if err := m.repo.Storer.RemoveReference(branchRef); err != nil {
		return fmt.Errorf("worktree: delete branch %q: %w", m.BranchName(name), err)
	}

	return nil
}
