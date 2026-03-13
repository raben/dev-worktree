package container

import (
	"github.com/docker/docker/api/types/filters"
)

const (
	// LabelKey identifies a dev-worktree container. Value is "project/name".
	LabelKey = "dev-worktree"
	// LabelPath stores the worktree filesystem path.
	LabelPath = "dev-worktree.path"
	// LabelVersion stores the CLI version that created the container.
	LabelVersion = "dev-worktree.version"
)

// Labels creates a label map for a dev-worktree container.
func Labels(key, wtPath, version string) map[string]string {
	return map[string]string{
		LabelKey:     key,
		LabelPath:    wtPath,
		LabelVersion: version,
	}
}

// FilterByKey creates a Docker filter that matches containers with the given dev key.
func FilterByKey(key string) filters.Args {
	f := filters.NewArgs()
	f.Add("label", LabelKey+"="+key)
	return f
}

// FilterAll creates a Docker filter that matches all dev-worktree containers.
func FilterAll() filters.Args {
	f := filters.NewArgs()
	f.Add("label", LabelKey)
	return f
}
