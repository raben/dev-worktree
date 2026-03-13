package worktree

import (
	"fmt"
	"regexp"
	"strings"
)

// validNamePattern matches names that start with an alphanumeric character
// and contain only alphanumeric, dots, hyphens, slashes, or underscores.
var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/\-]*$`)

// shellMetachars contains characters that are dangerous in shell contexts.
var shellMetachars = regexp.MustCompile(`[;&|$` + "`" + `(){}!<>~*?\[\]#"'\\` + `]`)

// ValidateName validates a worktree name.
// Rules:
//   - Must not be empty
//   - Must start with alphanumeric
//   - May contain alphanumeric, dots, hyphens, slashes, underscores
//   - Must not contain ".."
//   - Must not contain shell metacharacters
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("worktree name must not be empty")
	}

	if strings.Contains(name, "..") {
		return fmt.Errorf("worktree name %q must not contain %q", name, "..")
	}

	if shellMetachars.MatchString(name) {
		return fmt.Errorf("worktree name %q contains shell metacharacters", name)
	}

	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("worktree name %q must start with alphanumeric and contain only alphanumeric, dots, hyphens, slashes, or underscores", name)
	}

	return nil
}

// SanitizeName converts a raw name to a safe worktree name.
// Replaces path separators and dots with hyphens, then trims leading/trailing hyphens.
func SanitizeName(name string) string {
	r := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		".", "-",
	)
	sanitized := r.Replace(name)

	// Collapse multiple hyphens into one.
	multi := regexp.MustCompile(`-{2,}`)
	sanitized = multi.ReplaceAllString(sanitized, "-")

	// Trim leading/trailing hyphens.
	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}
