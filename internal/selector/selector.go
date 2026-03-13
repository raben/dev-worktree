package selector

import (
	"errors"
	"fmt"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/charmbracelet/huh"
)

// Select presents an interactive selection UI for choosing an environment.
// Filters environments by the given state (e.g., "running"). If stateFilter
// is empty, shows all. Returns the selected environment's Key.
// If only one environment matches, returns it without prompting.
// If no environments match, returns an error.
func Select(envs []container.Environment, stateFilter string) (string, error) {
	var filtered []container.Environment
	for _, e := range envs {
		if stateFilter == "" || e.State == stateFilter {
			filtered = append(filtered, e)
		}
	}

	if len(filtered) == 0 {
		return "", errors.New("no environments found")
	}

	if len(filtered) == 1 {
		return filtered[0].Key, nil
	}

	opts := make([]huh.Option[string], len(filtered))
	for i, e := range filtered {
		label := fmt.Sprintf("%s (%s)", e.Key, e.State)
		opts[i] = huh.NewOption(label, e.Key)
	}

	var selected string
	err := huh.NewSelect[string]().
		Title("Select environment").
		Options(opts...).
		Value(&selected).
		Run()
	if err != nil {
		return "", fmt.Errorf("selection failed: %w", err)
	}

	return selected, nil
}
