package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/autor-dev/dev-worktree/internal/config"
	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/spf13/cobra"
)

const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

func green(s string) string  { return colorGreen + s + colorReset }
func red(s string) string    { return colorRed + s + colorReset }
func yellow(s string) string { return colorYellow + s + colorReset }

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment health",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	passed := 0
	failed := 0

	// 1. Docker daemon
	dc, dockerErr := container.NewClient()
	if dockerErr != nil {
		fmt.Printf("Docker daemon         %s\n", red("✗"))
		failed++
	} else {
		defer dc.Close()
		fmt.Printf("Docker daemon         %s\n", green("✓"))
		passed++
	}

	// 2. Git repository
	projectDir, gitErr := findProjectDir()
	if gitErr != nil {
		fmt.Printf("Git repository        %s\n", red("✗"))
		failed++
	} else {
		projectName := filepath.Base(projectDir)
		fmt.Printf("Git repository        %s (%s)\n", green("✓"), projectName)
		passed++
	}

	// 3. .dev.yml
	if gitErr != nil {
		fmt.Printf(".dev.yml              %s (no git repository)\n", red("✗"))
		failed++
	} else {
		cfg, cfgErr := config.LoadDevYml(projectDir)
		if cfgErr != nil {
			fmt.Printf(".dev.yml              %s (%s)\n", red("✗"), cfgErr)
			failed++
		} else {
			details := fmt.Sprintf("image: %s", cfg.Image)
			if len(cfg.Ports) > 0 {
				portStrs := make([]string, len(cfg.Ports))
				for i, p := range cfg.Ports {
					portStrs[i] = fmt.Sprintf("%d", p)
				}
				details += fmt.Sprintf(", ports: [%s]", strings.Join(portStrs, " "))
			}
			fmt.Printf(".dev.yml              %s (%s)\n", green("✓"), details)
			passed++
		}
	}

	// 4. Active environments
	if dockerErr != nil {
		fmt.Printf("Active environments   %s (Docker not available)\n", red("✗"))
		failed++
	} else {
		envs, listErr := dc.List(ctx)
		if listErr != nil {
			fmt.Printf("Active environments   %s (%s)\n", red("✗"), listErr)
			failed++
		} else if len(envs) == 0 {
			fmt.Printf("Active environments   %s\n", yellow("none"))
			passed++
		} else {
			running := 0
			stopped := 0
			for _, e := range envs {
				if e.State == "running" {
					running++
				} else {
					stopped++
				}
			}
			parts := []string{}
			if running > 0 {
				parts = append(parts, fmt.Sprintf("%d running", running))
			}
			if stopped > 0 {
				parts = append(parts, fmt.Sprintf("%d stopped", stopped))
			}
			fmt.Printf("Active environments   %s\n", strings.Join(parts, ", "))
			passed++

			// 5. Claude in containers
			for _, e := range envs {
				if e.State == "running" {
					_, execErr := dc.Exec(ctx, e.Container, []string{"which", "claude"})
					if execErr != nil {
						fmt.Printf("  %-20s %s running, %s\n", e.Key, green("✓"), yellow("claude not found"))
					} else {
						fmt.Printf("  %-20s %s running, claude installed\n", e.Key, green("✓"))
					}
				} else {
					fmt.Printf("  %-20s %s %s\n", e.Key, red("✗"), e.State)
				}
			}
		}
	}

	// Summary
	fmt.Println()
	fmt.Printf("%d checks passed, %d failed\n", passed, failed)

	if failed > 0 {
		return fmt.Errorf("%d checks failed", failed)
	}
	return nil
}
