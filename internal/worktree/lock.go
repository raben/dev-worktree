package worktree

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	lockDir     = "/tmp"
	lockPrefix  = "dev-worktree-"
	lockSuffix  = ".lock"
	lockTimeout = 30 * time.Second
	lockPoll    = 100 * time.Millisecond
)

// Lock acquires a mutex lock for a dev-worktree operation.
// Uses mkdir-based atomic lock at /tmp/dev-worktree-<key>.lock.
// The key is sanitized by replacing "/" with "-".
// Returns an unlock function that removes the lock directory.
func Lock(key string) (unlock func(), err error) {
	safeKey := strings.ReplaceAll(key, "/", "-")
	lockPath := fmt.Sprintf("%s/%s%s%s", lockDir, lockPrefix, safeKey, lockSuffix)

	deadline := time.Now().Add(lockTimeout)
	for {
		// os.Mkdir is atomic on POSIX systems: only one process can succeed.
		if err := os.Mkdir(lockPath, 0o755); err == nil {
			return func() { os.Remove(lockPath) }, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("worktree: lock timeout after %s waiting for %q", lockTimeout, lockPath)
		}

		time.Sleep(lockPoll)
	}
}
