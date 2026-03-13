package worktree

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	lockDir     = "/tmp"
	lockPrefix  = "dev-worktree-"
	lockSuffix  = ".lock"
	lockTimeout = 30 * time.Second
	lockPoll    = 100 * time.Millisecond
	pidFile     = "pid"
)

// Lock acquires a mutex lock for a dev-worktree operation.
// Uses mkdir-based atomic lock at /tmp/dev-worktree-<key>.lock.
// The key is sanitized by replacing "/" with "-".
// Returns an unlock function that removes the lock directory.
//
// If a stale lock is detected (the owning process no longer exists),
// the lock is automatically removed and re-acquired.
func Lock(key string) (unlock func(), err error) {
	safeKey := strings.ReplaceAll(key, "/", "-")
	lockPath := fmt.Sprintf("%s/%s%s%s", lockDir, lockPrefix, safeKey, lockSuffix)

	deadline := time.Now().Add(lockTimeout)
	for {
		// os.Mkdir is atomic on POSIX systems: only one process can succeed.
		if err := os.Mkdir(lockPath, 0o755); err == nil {
			// Write our PID into the lock directory.
			writePID(lockPath)
			return func() {
				os.Remove(filepath.Join(lockPath, pidFile))
				os.Remove(lockPath)
			}, nil
		}

		// Lock directory exists — check if the holder is still alive.
		if removedStale := removeIfStale(lockPath); removedStale {
			// Retry immediately after removing stale lock.
			continue
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("worktree: lock timeout after %s waiting for %q", lockTimeout, lockPath)
		}

		time.Sleep(lockPoll)
	}
}

// writePID writes the current process PID to a file inside the lock directory.
func writePID(lockPath string) {
	pidPath := filepath.Join(lockPath, pidFile)
	_ = os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0o644)
}

// removeIfStale checks if the lock is held by a dead process and removes it.
// Returns true if a stale lock was removed.
func removeIfStale(lockPath string) bool {
	pidPath := filepath.Join(lockPath, pidFile)
	data, err := os.ReadFile(pidPath)
	if err != nil {
		// No PID file — can't determine staleness. Leave the lock alone.
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		// Corrupt PID file — treat as stale.
		os.Remove(pidPath)
		os.Remove(lockPath)
		return true
	}

	if isProcessAlive(pid) {
		return false
	}

	// Process is dead — remove stale lock.
	os.Remove(pidPath)
	os.Remove(lockPath)
	return true
}

// isProcessAlive checks if a process with the given PID exists on Unix.
// Sending signal 0 checks for existence without actually sending a signal.
func isProcessAlive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = p.Signal(syscall.Signal(0))
	return err == nil
}
