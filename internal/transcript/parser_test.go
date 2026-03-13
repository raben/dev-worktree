package transcript

import (
	"testing"
)

func TestParseLine_UserMessage(t *testing.T) {
	data := `{"type":"user","message":{"content":"hello world"}}`
	lines := ParseLine([]byte(data))
	if len(lines) != 1 {
		t.Fatalf("ParseLine returned %d lines, want 1", len(lines))
	}
	if lines[0].Type != "user" {
		t.Errorf("Type = %q, want %q", lines[0].Type, "user")
	}
	if lines[0].Text != "hello world" {
		t.Errorf("Text = %q, want %q", lines[0].Text, "hello world")
	}
}

func TestParseLine_AssistantText(t *testing.T) {
	data := `{"type":"assistant","message":{"content":[{"type":"text","text":"response here"}]}}`
	lines := ParseLine([]byte(data))
	if len(lines) != 1 {
		t.Fatalf("ParseLine returned %d lines, want 1", len(lines))
	}
	if lines[0].Type != "asst" {
		t.Errorf("Type = %q, want %q", lines[0].Type, "asst")
	}
	if lines[0].Text != "response here" {
		t.Errorf("Text = %q, want %q", lines[0].Text, "response here")
	}
}

func TestParseLine_ToolUse(t *testing.T) {
	data := `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Bash","input":{"command":"ls"}}]}}`
	lines := ParseLine([]byte(data))
	if len(lines) != 1 {
		t.Fatalf("ParseLine returned %d lines, want 1", len(lines))
	}
	if lines[0].Type != "tool" {
		t.Errorf("Type = %q, want %q", lines[0].Type, "tool")
	}
	if lines[0].Text != `Bash({"command":"ls"})` {
		t.Errorf("Text = %q, want %q", lines[0].Text, `Bash({"command":"ls"})`)
	}
}

func TestParseLine_TurnDuration(t *testing.T) {
	data := `{"type":"system","subtype":"turn_duration","durationMs":5432.1}`
	lines := ParseLine([]byte(data))
	if len(lines) != 1 {
		t.Fatalf("ParseLine returned %d lines, want 1", len(lines))
	}
	if lines[0].Type != "done" {
		t.Errorf("Type = %q, want %q", lines[0].Type, "done")
	}
	if lines[0].Text != "5.4s" {
		t.Errorf("Text = %q, want %q", lines[0].Text, "5.4s")
	}
}

func TestParseLine_InvalidJSON(t *testing.T) {
	lines := ParseLine([]byte(`not valid json`))
	if lines != nil {
		t.Errorf("ParseLine should return nil for invalid JSON, got %v", lines)
	}
}

func TestParseLine_UnknownType(t *testing.T) {
	data := `{"type":"unknown","message":{}}`
	lines := ParseLine([]byte(data))
	if lines != nil {
		t.Errorf("ParseLine should return nil for unknown type, got %v", lines)
	}
}

func TestParseLine_AssistantMultipleItems(t *testing.T) {
	data := `{"type":"assistant","message":{"content":[{"type":"text","text":"first"},{"type":"text","text":"second"}]}}`
	lines := ParseLine([]byte(data))
	if len(lines) != 2 {
		t.Fatalf("ParseLine returned %d lines, want 2", len(lines))
	}
	if lines[0].Text != "first" {
		t.Errorf("lines[0].Text = %q, want %q", lines[0].Text, "first")
	}
	if lines[1].Text != "second" {
		t.Errorf("lines[1].Text = %q, want %q", lines[1].Text, "second")
	}
}

func TestParseAll(t *testing.T) {
	data := `{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"content":[{"type":"text","text":"world"}]}}
{"type":"system","subtype":"turn_duration","durationMs":1000}`

	lines := ParseAll(data)
	if len(lines) != 3 {
		t.Fatalf("ParseAll returned %d lines, want 3", len(lines))
	}
	if lines[0].Type != "user" {
		t.Errorf("lines[0].Type = %q, want %q", lines[0].Type, "user")
	}
	if lines[1].Type != "asst" {
		t.Errorf("lines[1].Type = %q, want %q", lines[1].Type, "asst")
	}
	if lines[2].Type != "done" {
		t.Errorf("lines[2].Type = %q, want %q", lines[2].Type, "done")
	}
}

func TestParseAll_EmptyLines(t *testing.T) {
	data := `
{"type":"user","message":{"content":"hello"}}

`
	lines := ParseAll(data)
	if len(lines) != 1 {
		t.Fatalf("ParseAll returned %d lines, want 1", len(lines))
	}
}

func TestParseLine_SystemNonDuration(t *testing.T) {
	data := `{"type":"system","subtype":"other","message":{}}`
	lines := ParseLine([]byte(data))
	if lines != nil {
		t.Errorf("ParseLine should return nil for non-duration system event, got %v", lines)
	}
}
