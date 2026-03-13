package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const leftPanelWidth = 30

var (
	statusBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")) // green

	stoppedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")) // red

	otherStateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")) // yellow

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	portStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")) // blue
)

// renderStatus renders the left panel with the environment list.
func (m Model) renderStatus() string {
	innerWidth := leftPanelWidth - 2 // account for border
	// Reserve lines: 1 title + 1 blank + envs + 1 blank + ports + 1 divider + 2 keybinds
	contentHeight := m.height - 2 // border top/bottom

	var lines []string

	// Environment list.
	for i, env := range m.envs {
		indicator := "  "
		nameStyle := lipgloss.NewStyle()
		if i == m.selected {
			indicator = lipgloss.NewStyle().Bold(true).Render("▶ ")
			nameStyle = selectedStyle
		}

		state := stateStyled(env.State)
		name := truncate(env.Key, innerWidth-len(indicator)-len(env.State)-2)
		padding := innerWidth - lipgloss.Width(indicator) - lipgloss.Width(name) - lipgloss.Width(state)
		if padding < 1 {
			padding = 1
		}

		line := indicator + nameStyle.Render(name) + strings.Repeat(" ", padding) + state
		lines = append(lines, line)
	}

	if len(m.envs) == 0 {
		lines = append(lines, dimStyle.Render(" No environments found"))
	}

	// Port info for selected environment.
	lines = append(lines, "")
	if m.selected >= 0 && m.selected < len(m.envs) {
		env := m.envs[m.selected]
		if len(env.Ports) > 0 {
			lines = append(lines, dimStyle.Render("Ports:"))
			for _, p := range env.Ports {
				lines = append(lines, portStyle.Render(fmt.Sprintf("  %d→%d", p.HostPort, p.ContainerPort)))
			}
		}
	}

	// Pad to fill available space before keybindings.
	keybindLines := []string{
		dimStyle.Render("↑↓:select q:quit"),
		dimStyle.Render("o:open in browser"),
	}
	// +1 for the separator line
	neededPadding := contentHeight - len(lines) - len(keybindLines) - 1
	for i := 0; i < neededPadding; i++ {
		lines = append(lines, "")
	}

	// Separator and keybindings.
	lines = append(lines, dimStyle.Render(strings.Repeat("─", innerWidth)))
	lines = append(lines, keybindLines...)

	content := strings.Join(lines, "\n")

	return statusBorderStyle.
		Width(innerWidth).
		Height(contentHeight).
		Render(titleStyle.Render("Environments") + "\n" + content)
}

// stateStyled returns a styled state string.
func stateStyled(state string) string {
	switch state {
	case "running":
		return runningStyle.Render(state)
	case "exited", "dead", "stopped":
		return stoppedStyle.Render("stopped")
	default:
		return otherStateStyle.Render(state)
	}
}

// truncate shortens a string to maxLen runes, appending "…" if truncated.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
