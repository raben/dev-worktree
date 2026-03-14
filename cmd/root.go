package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/session"
	"github.com/spf13/cobra"
)

var version = "dev"

const defaultImage = "node:22-bookworm"

var safeMode bool

var rootCmd = &cobra.Command{
	Use:   "dev",
	Short: "Sandboxed AI development environment",
	Long:  "Launch a Docker-sandboxed Claude Code session for parallel AI-assisted development.",
	RunE:  runDev,

	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	rootCmd.SetContext(ctx)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}

func init() {
	rootCmd.Version = version
	rootCmd.Flags().BoolVar(&safeMode, "safe", false, "Run Claude without --dangerously-skip-permissions")
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(stopCmd)
}

func runDev(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	dc, err := container.NewClient()
	if err != nil {
		return fmt.Errorf("connecting to Docker: %w", err)
	}
	defer dc.Close()

	sess, err := session.Load()
	if errors.Is(err, session.ErrNoSession) {
		sess = &session.Session{Image: defaultImage}
		return createAndStart(ctx, dc, sess)
	}
	if err != nil {
		return fmt.Errorf("loading session: %w", err)
	}

	// Check if container still exists
	state, err := dc.ContainerState(ctx, sess.ContainerID)
	if err != nil {
		fmt.Println("Previous container not found. Creating new one...")
		session.Clear()
		return createAndStart(ctx, dc, sess)
	}

	if state != "running" {
		fmt.Println("Restarting container...")
		dc.Remove(ctx, sess.ContainerID)
		return createAndStart(ctx, dc, sess)
	}

	// Container is running, connect
	fmt.Println("Connecting to existing session...")
	return dc.StartClaude(ctx, sess.ContainerID, safeMode)
}

func createAndStart(ctx context.Context, dc *container.Client, sess *session.Session) error {
	containerID, err := dc.Create(ctx, container.CreateParams{
		Image:  sess.Image,
		Binds:  sess.Binds(),
		Labels: container.Labels(),
		Env:    envVars(),
	})
	if err != nil {
		return fmt.Errorf("creating container: %w", err)
	}
	sess.ContainerID = containerID

	if err := dc.InstallClaude(ctx, containerID); err != nil {
		dc.Remove(ctx, containerID)
		return err
	}

	if err := session.Save(sess); err != nil {
		dc.Remove(ctx, containerID)
		return fmt.Errorf("saving session: %w", err)
	}

	fmt.Println("Session ready.")
	return dc.StartClaude(ctx, containerID, safeMode)
}

func envVars() []string {
	var env []string
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		env = append(env, "ANTHROPIC_API_KEY="+key)
	}
	return env
}
