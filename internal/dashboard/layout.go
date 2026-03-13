package dashboard

import "strings"

// joinHorizontal places two rendered panels side by side, padding as needed.
func joinHorizontal(left, right string, totalWidth, totalHeight int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	// Determine max height.
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	// Determine left panel display width (from the rendered content).
	leftWidth := 0
	for _, l := range leftLines {
		w := printWidth(l)
		if w > leftWidth {
			leftWidth = w
		}
	}

	var sb strings.Builder
	for i := 0; i < maxLines; i++ {
		var l, r string
		if i < len(leftLines) {
			l = leftLines[i]
		}
		if i < len(rightLines) {
			r = rightLines[i]
		}
		// Pad left line to consistent width.
		padding := leftWidth - printWidth(l)
		if padding < 0 {
			padding = 0
		}
		sb.WriteString(l)
		sb.WriteString(strings.Repeat(" ", padding))
		sb.WriteString(r)
		if i < maxLines-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// printWidth returns the visible width of a string, accounting for ANSI escape sequences.
// This is a simple approximation; for full accuracy, use lipgloss.Width.
func printWidth(s string) int {
	// Use lipgloss's width calculation which handles ANSI.
	return len(stripAnsi(s))
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
