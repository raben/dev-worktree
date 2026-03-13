package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/autor-dev/dev-worktree/internal/container"
)

// JSONL types for parsing Claude session logs.

type jsonlEntry struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
	// For system turn_duration messages.
	DurationMs float64 `json:"durationMs,omitempty"`
}

type messageContent struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

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
func parseTranscript(raw string) []TranscriptLine {
	var result []TranscriptLine
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var entry jsonlEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		switch entry.Type {
		case "user":
			result = append(result, parseUserEntry(entry)...)
		case "assistant":
			result = append(result, parseAssistantEntry(entry)...)
		case "system":
			if entry.Subtype == "turn_duration" {
				// Parse durationMs from the top-level or from message.
				dur := entry.DurationMs
				if dur == 0 && entry.Message != nil {
					var sysMsg struct {
						DurationMs float64 `json:"durationMs"`
					}
					if json.Unmarshal(entry.Message, &sysMsg) == nil {
						dur = sysMsg.DurationMs
					}
				}
				if dur > 0 {
					secs := dur / 1000.0
					result = append(result, TranscriptLine{
						Type: "done",
						Text: fmt.Sprintf("%.1fs", secs),
					})
				}
			}
		}
	}
	return result
}

// parseUserEntry extracts transcript lines from a user message.
func parseUserEntry(entry jsonlEntry) []TranscriptLine {
	if entry.Message == nil {
		return nil
	}
	var msg messageContent
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil
	}

	// content can be a string directly.
	var contentStr string
	if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
		return []TranscriptLine{{Type: "user", Text: contentStr}}
	}

	return nil
}

// parseAssistantEntry extracts transcript lines from an assistant message.
func parseAssistantEntry(entry jsonlEntry) []TranscriptLine {
	if entry.Message == nil {
		return nil
	}
	var msg messageContent
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return nil
	}

	// content is an array of blocks.
	var blocks []contentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil
	}

	var result []TranscriptLine
	for _, block := range blocks {
		switch block.Type {
		case "text":
			if block.Text != "" {
				result = append(result, TranscriptLine{Type: "asst", Text: block.Text})
			}
		case "tool_use":
			inputStr := summarizeInput(block.Input, 60)
			result = append(result, TranscriptLine{
				Type: "tool",
				Text: fmt.Sprintf("%s(%s)", block.Name, inputStr),
			})
		}
	}
	return result
}

// summarizeInput returns a compact string representation of a tool's input JSON,
// truncated to maxLen characters.
func summarizeInput(raw json.RawMessage, maxLen int) string {
	if raw == nil {
		return ""
	}
	s := string(raw)
	s = strings.ReplaceAll(s, "\n", " ")
	return truncate(s, maxLen)
}
