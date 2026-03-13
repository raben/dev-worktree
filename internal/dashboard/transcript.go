package dashboard

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/autor-dev/dev-worktree/internal/container"
	"github.com/autor-dev/dev-worktree/internal/transcript"
)

// Transcript styles.
var (
	transcriptBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240"))

	transcriptTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1)

	userTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Bold(true)

	asstTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Bold(true)

	toolTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("12")).
			Bold(true)

	doneTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)

// renderTranscript renders the right panel with the Claude transcript.
func (m Model) renderTranscript() string {
	rightWidth := m.width - leftPanelWidth - 2 // subtract left panel and margin
	if rightWidth < 10 {
		rightWidth = 10
	}
	contentHeight := m.height - 2 // border

	innerWidth := rightWidth - 2 // border padding

	// Title with selected environment name.
	title := "Transcript"
	if m.selected >= 0 && m.selected < len(m.envs) {
		title = fmt.Sprintf("Transcript (%s)", m.envs[m.selected].Key)
	}

	var content string
	if len(m.transcript) == 0 {
		content = dimStyle.Render("Waiting for Claude session...")
	} else {
		// Auto-scroll: show only the last lines that fit.
		lines := formatTranscriptLines(m.transcript, innerWidth)
		visibleHeight := contentHeight - 1 // -1 for title line
		if len(lines) > visibleHeight {
			lines = lines[len(lines)-visibleHeight:]
		}
		content = strings.Join(lines, "\n")
	}

	return transcriptBorderStyle.
		Width(innerWidth).
		Height(contentHeight).
		Render(transcriptTitleStyle.Render(title) + "\n" + content)
}

// formatTranscriptLines formats transcript lines for display.
func formatTranscriptLines(lines []TranscriptLine, maxWidth int) []string {
	var result []string
	for _, line := range lines {
		var styled string
		switch line.Type {
		case "user":
			styled = userTagStyle.Render("[user]") + " " + truncate(line.Text, maxWidth-7)
		case "asst":
			styled = asstTagStyle.Render("[asst]") + " " + truncate(line.Text, maxWidth-7)
		case "tool":
			styled = toolTagStyle.Render("[tool]") + " " + truncate(line.Text, maxWidth-7)
		case "done":
			styled = doneTagStyle.Render("[done] " + line.Text)
		default:
			styled = truncate(line.Text, maxWidth)
		}
		result = append(result, styled)
	}
	return result
}

// streamTranscript finds and reads the latest Claude transcript from a container.
func streamTranscript(dc *container.Client, containerID string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Find the latest JSONL file in the Claude data directory.
		findCmd := []string{
			"sh", "-c",
			"find $HOME/.claude -path '*/subagents/*' -prune -o -name '*.jsonl' ! -name 'history.jsonl' -type f -print 2>/dev/null | xargs ls -t 2>/dev/null | head -1",
		}
		jsonlPath, err := dc.Exec(ctx, containerID, findCmd)
		if err != nil {
			return transcriptMsg{containerID: containerID, lines: nil}
		}
		jsonlPath = strings.TrimSpace(jsonlPath)
		if jsonlPath == "" {
			return transcriptMsg{containerID: containerID, lines: nil}
		}

		// Read the file content.
		output, err := dc.Exec(ctx, containerID, []string{"cat", jsonlPath})
		if err != nil {
			return transcriptMsg{containerID: containerID, lines: nil}
		}

		lines := parseTranscript(output)
		return transcriptMsg{containerID: containerID, lines: lines}
	}
}

// parseTranscript parses JSONL content into TranscriptLine entries.
// Delegates to internal/transcript.ParseAll() to avoid duplicating parsing logic.
func parseTranscript(raw string) []TranscriptLine {
	parsed := transcript.ParseAll(raw)
	result := make([]TranscriptLine, len(parsed))
	for i, l := range parsed {
		result[i] = TranscriptLine{Type: l.Type, Text: l.Text}
	}
	return result
}
