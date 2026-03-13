package transcript

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Line represents a parsed and formatted transcript line.
type Line struct {
	Type string // "user", "asst", "tool", "done"
	Text string
}

// record is the raw JSONL structure from Claude sessions.
type record struct {
	Type    string        `json:"type"`
	Subtype string        `json:"subtype"`
	Message *message      `json:"message"`
	DurMS   float64       `json:"durationMs"`
}

type message struct {
	Content json.RawMessage `json:"content"`
}

type contentItem struct {
	Type  string          `json:"type"`
	Text  string          `json:"text"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ParseLine parses a single JSONL line into transcript lines.
// A single JSONL record may produce multiple Lines (e.g., assistant with multiple content items).
func ParseLine(data []byte) []Line {
	var r record
	if err := json.Unmarshal(data, &r); err != nil {
		return nil
	}

	switch r.Type {
	case "user":
		return parseUser(&r)
	case "assistant":
		return parseAssistant(&r)
	case "system":
		return parseSystem(&r)
	default:
		return nil
	}
}

func parseUser(r *record) []Line {
	if r.Message == nil {
		return nil
	}

	// Try as string first
	var s string
	if err := json.Unmarshal(r.Message.Content, &s); err == nil {
		return []Line{{Type: "user", Text: s}}
	}

	return nil
}

func parseAssistant(r *record) []Line {
	if r.Message == nil {
		return nil
	}

	var items []contentItem
	if err := json.Unmarshal(r.Message.Content, &items); err != nil {
		return nil
	}

	var lines []Line
	for _, item := range items {
		switch item.Type {
		case "text":
			if item.Text != "" {
				lines = append(lines, Line{Type: "asst", Text: item.Text})
			}
		case "tool_use":
			input := truncate(string(item.Input), 60)
			lines = append(lines, Line{Type: "tool", Text: fmt.Sprintf("%s(%s)", item.Name, input)})
		}
	}

	return lines
}

func parseSystem(r *record) []Line {
	if r.Subtype == "turn_duration" {
		secs := r.DurMS / 1000
		return []Line{{Type: "done", Text: fmt.Sprintf("%.1fs", secs)}}
	}
	return nil
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// ParseAll parses multiple JSONL lines (newline-separated) into transcript lines.
func ParseAll(data string) []Line {
	var lines []Line
	for _, raw := range strings.Split(data, "\n") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		lines = append(lines, ParseLine([]byte(raw))...)
	}
	return lines
}
