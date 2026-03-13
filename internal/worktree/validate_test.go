package worktree

import "testing"

func TestValidateName_Valid(t *testing.T) {
	valid := []string{
		"feature-x",
		"fix/bug-123",
		"release_1.0",
		"a",
		"abc123",
		"my-feature/sub",
	}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) returned error: %v", name, err)
		}
	}
}

func TestValidateName_Empty(t *testing.T) {
	if err := ValidateName(""); err == nil {
		t.Error("ValidateName('') should return error")
	}
}

func TestValidateName_StartsWithDot(t *testing.T) {
	if err := ValidateName(".hidden"); err == nil {
		t.Error("ValidateName('.hidden') should return error (starts with dot)")
	}
}

func TestValidateName_ContainsDoubleDot(t *testing.T) {
	if err := ValidateName("path..traversal"); err == nil {
		t.Error("ValidateName('path..traversal') should return error (contains '..')")
	}
}

func TestValidateName_ShellMetachars(t *testing.T) {
	metachars := []string{
		"name;rm",
		"name|pipe",
		"name&bg",
		"name$var",
		"name`cmd`",
		"name(paren)",
		"name{brace}",
		"name!bang",
		"name<redir",
		"name>redir",
		"name~tilde",
		"name*glob",
		"name?question",
	}
	for _, name := range metachars {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) should return error (shell metacharacter)", name)
		}
	}
}

func TestValidateName_StartsWithHyphen(t *testing.T) {
	if err := ValidateName("-feature"); err == nil {
		t.Error("ValidateName('-feature') should return error (starts with non-alphanumeric)")
	}
}

func TestSanitizeName_SlashToHyphen(t *testing.T) {
	got := SanitizeName("feature/my-branch")
	want := "feature-my-branch"
	if got != want {
		t.Errorf("SanitizeName('feature/my-branch') = %q, want %q", got, want)
	}
}

func TestSanitizeName_BackslashToHyphen(t *testing.T) {
	got := SanitizeName("feature\\my-branch")
	want := "feature-my-branch"
	if got != want {
		t.Errorf("SanitizeName('feature\\\\my-branch') = %q, want %q", got, want)
	}
}

func TestSanitizeName_DotToHyphen(t *testing.T) {
	got := SanitizeName("release.1.0")
	want := "release-1-0"
	if got != want {
		t.Errorf("SanitizeName('release.1.0') = %q, want %q", got, want)
	}
}

func TestSanitizeName_CollapseMultipleHyphens(t *testing.T) {
	got := SanitizeName("a//b..c")
	want := "a-b-c"
	if got != want {
		t.Errorf("SanitizeName('a//b..c') = %q, want %q", got, want)
	}
}

func TestSanitizeName_TrimLeadingTrailingHyphens(t *testing.T) {
	got := SanitizeName("/leading")
	want := "leading"
	if got != want {
		t.Errorf("SanitizeName('/leading') = %q, want %q", got, want)
	}

	got = SanitizeName("trailing/")
	want = "trailing"
	if got != want {
		t.Errorf("SanitizeName('trailing/') = %q, want %q", got, want)
	}
}
