package worktree

import "testing"

func TestIsProtected(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"main", true},
		{"master", true},
		{"develop", true},
		{"feature-x", false},
		{"fix/bug-123", false},
		{"", false},
		{"Main", false},    // case-sensitive
		{"MASTER", false},  // case-sensitive
	}

	for _, tt := range tests {
		got := IsProtected(tt.name)
		if got != tt.want {
			t.Errorf("IsProtected(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
