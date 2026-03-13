package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/autor-dev/dev-worktree/internal/config"
	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/port"
	"github.com/autor-dev/dev-worktree/internal/worktree"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up [name] [base-branch]",
	Short: "Create or resume a worktree + container environment",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runUp,
}

func runUp(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Resolve project directory
	projectDir, err := findProjectDir()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load .dev.yml
	cfg, err := config.LoadDevYml(projectDir)
	if err != nil {
		return fmt.Errorf("loading .dev.yml: %w", err)
	}

	// Resolve name
	name := ""
	if len(args) > 0 {
		name = args[0]
	}
	if name == "" {
		return fmt.Errorf("environment name is required")
	}

	name = worktree.SanitizeName(name)
	if err := worktree.ValidateName(name); err != nil {
		return fmt.Errorf("invalid name: %w", err)
	}

	// Resolve base branch
	baseBranch := ""
	if len(args) > 1 {
		baseBranch = args[1]
	}

	// Handle protected branches: treat as base branch
	if worktree.IsProtected(name) {
		baseBranch = name
		name = fmt.Sprintf("dev-%s", name)
	}

	// Initialize worktree manager
	wm, err := worktree.NewManager(projectDir)
	if err != nil {
		return fmt.Errorf("initializing worktree manager: %w", err)
	}

	devKey := wm.DevKey(name)

	// Acquire lock
	unlock, err := worktree.Lock(devKey)
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer unlock()

	// Initialize Docker client
	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	// Check if already running
	running, err := dc.IsRunning(ctx, devKey)
	if err != nil {
		return fmt.Errorf("checking container status: %w", err)
	}
	if running {
		fmt.Printf("Environment '%s' is already running.\n", devKey)
		return nil
	}

	// Create worktree
	wtPath := wm.WorktreePath(name)
	if wm.Exists(name) {
		fmt.Printf("Worktree exists: %s\n", wtPath)
	} else {
		fmt.Printf("Creating worktree: %s\n", wtPath)
		wtPath, err = wm.Create(name, baseBranch)
		if err != nil {
			return fmt.Errorf("creating worktree: %w", err)
		}
	}

	// Allocate ports (held open until Docker binds them)
	portBindings := make(map[int]int)
	var portAllocs []*port.Allocation
	if len(cfg.Ports) > 0 {
		allocs, err := port.AllocateMultipleAndHold(cfg.Ports)
		if err != nil {
			return fmt.Errorf("allocating ports: %w", err)
		}
		portAllocs = allocs
		for i, containerPort := range cfg.Ports {
			portBindings[allocs[i].Port] = containerPort
		}
	}

	// Release held ports right before Docker binds them
	for _, a := range portAllocs {
		a.Release()
	}

	// Start container
	fmt.Printf("Starting container (image: %s)...\n", cfg.Image)
	containerID, err := dc.Up(ctx, devKey, wtPath, cfg.Image, version, portBindings)
	if err != nil {
		return fmt.Errorf("starting container: %w", err)
	}

	// Install Claude Code
	fmt.Println("Installing Claude Code...")
	output, err := dc.Exec(ctx, containerID, []string{"npm", "install", "-g", "@anthropic-ai/claude-code"})
	if err != nil {
		return fmt.Errorf("installing Claude Code: %w\n%s", err, output)
	}

	// Run setup command if specified
	if cfg.Setup != "" {
		fmt.Printf("Running setup: %s\n", cfg.Setup)
		output, err := dc.Exec(ctx, containerID, []string{"sh", "-c", cfg.Setup})
		if err != nil {
			return fmt.Errorf("setup command failed: %w\n%s", err, output)
		}
	}

	// Save env config
	envCfg := &config.EnvConfig{
		WTName:             filepath.Base(wtPath),
		ComposeProjectName: fmt.Sprintf("%s-%s", wm.ProjectName(), name),
		ExecCmd:            cfg.ExecCmd,
		Ports:              make(map[string]int),
	}
	for hostPort, containerPort := range portBindings {
		envCfg.Ports[fmt.Sprintf("%d", containerPort)] = hostPort
	}
	envPath := filepath.Join(wtPath, ".dev.env")
	if err := config.WriteEnv(envPath, envCfg); err != nil {
		return fmt.Errorf("writing env config: %w", err)
	}

	// Print summary
	fmt.Println()
	fmt.Printf("Environment '%s' is ready.\n", devKey)
	fmt.Printf("  Worktree: %s\n", wtPath)
	fmt.Printf("  Branch:   %s\n", wm.BranchName(name))
	for hostPort, containerPort := range portBindings {
		fmt.Printf("  Port:     http://localhost:%d → :%d\n", hostPort, containerPort)
	}
	fmt.Println()
	fmt.Printf("  dev code %s    # start Claude session\n", name)
	fmt.Printf("  dev shell %s   # open shell\n", name)

	return nil
}

