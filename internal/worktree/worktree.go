package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// Manager handles git worktree operations.
type Manager struct {
	repo       *git.Repository
	projectDir string // git root directory (absolute path)
}

// NewManager creates a new worktree manager for the given project directory.
// Opens the git repository at projectDir.
func NewManager(projectDir string) (*Manager, error) {
	absDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("worktree: resolve path %q: %w", projectDir, err)
	}

	repo, err := git.PlainOpen(absDir)
	if err != nil {
		return nil, fmt.Errorf("worktree: open repository %q: %w", absDir, err)
	}

	return &Manager{
		repo:       repo,
		projectDir: absDir,
	}, nil
}

// ProjectName returns the basename of the project directory.
func (m *Manager) ProjectName() string {
	return filepath.Base(m.projectDir)
}

// ProjectDir returns the project root directory.
func (m *Manager) ProjectDir() string {
	return m.projectDir
}

// WorktreePath returns the expected worktree path for a given name.
// Pattern: <parent-of-project>/dev/<project-name>/<name>
// e.g., if project is /home/user/myapp and name is "feature-auth":
//
//	/home/user/dev/myapp/feature-auth
func (m *Manager) WorktreePath(name string) string {
	parent := filepath.Dir(m.projectDir)
	return filepath.Join(parent, "dev", m.ProjectName(), name)
}

// DevKey returns the dev-worktree key for a given name.
// Pattern: <project-name>/<name>
func (m *Manager) DevKey(name string) string {
	return m.ProjectName() + "/" + name
}

// BranchName returns the branch name for a given name.
// Pattern: dev/<project-name>/<name>
func (m *Manager) BranchName(name string) string {
	return "dev/" + m.ProjectName() + "/" + name
}

// Create creates a new worktree with a new branch.
//
// name is the environment name (e.g., "feature-auth").
// baseBranch is the base branch to create from (e.g., "main"). If empty, uses HEAD.
// Returns the worktree path.
// If worktree already exists at the path, returns the path without error.
//
// NOTE: go-git has limited worktree support (no built-in worktree add).
// We shell out to "git worktree add" which is the reliable approach.
func (m *Manager) Create(name, baseBranch string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", fmt.Errorf("worktree: create: %w", err)
	}

	wtPath := m.WorktreePath(name)

	// If the worktree directory already exists, treat as success.
	if info, err := os.Stat(wtPath); err == nil && info.IsDir() {
		return wtPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(wtPath), 0o755); err != nil {
		return "", fmt.Errorf("worktree: create parent directories: %w", err)
	}

	branch := m.BranchName(name)

	// Resolve base commit for the new branch.
	var startPoint string
	if baseBranch != "" {
		startPoint = baseBranch
	}

	// git worktree add -b <branch> <path> [<start-point>]
	args := []string{"worktree", "add", "-b", branch, wtPath}
	if startPoint != "" {
		args = append(args, startPoint)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = m.projectDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("worktree: git worktree add: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return wtPath, nil
}

// Remove removes a worktree and optionally its branch.
//
// name is the environment name. force enables forced removal even with
// uncommitted changes. Returns error if worktree doesn't exist.
//
// NOTE: go-git has limited worktree support (no built-in worktree remove).
// We shell out to "git worktree remove" which is the reliable approach.
func (m *Manager) Remove(name string, force bool) error {
	wtPath := m.WorktreePath(name)

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return fmt.Errorf("worktree: %q does not exist", name)
	}

	args := []string{"worktree", "remove", wtPath}
	if force {
		args = []string{"worktree", "remove", "--force", wtPath}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = m.projectDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("worktree: git worktree remove: %s: %w", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// Exists checks if a worktree exists for the given name.
func (m *Manager) Exists(name string) bool {
	wtPath := m.WorktreePath(name)
	info, err := os.Stat(wtPath)
	return err == nil && info.IsDir()
}

// HasUncommittedChanges checks if the worktree has uncommitted changes.
func (m *Manager) HasUncommittedChanges(name string) (bool, error) {
	wtPath := m.WorktreePath(name)

	if _, err := os.Stat(wtPath); os.IsNotExist(err) {
		return false, fmt.Errorf("worktree: %q does not exist", name)
	}

	// Open the worktree as a separate repository to inspect its status.
	wtRepo, err := git.PlainOpen(wtPath)
	if err != nil {
		return false, fmt.Errorf("worktree: open worktree %q: %w", name, err)
	}

	wt, err := wtRepo.Worktree()
	if err != nil {
		return false, fmt.Errorf("worktree: get worktree %q: %w", name, err)
	}

	status, err := wt.Status()
	if err != nil {
		return false, fmt.Errorf("worktree: status %q: %w", name, err)
	}

	return !status.IsClean(), nil
}

