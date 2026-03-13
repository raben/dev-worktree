package main

import (
	"os"

	"github.com/autor-dev/dev-worktree/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
