package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteEnvReadEnv_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	original := &EnvConfig{
		WTName:             "my-worktree",
		ComposeProjectName: "dev-my-worktree",
		ExecCmd:            "claude",
		Ports: map[string]int{
			"web": 3000,
			"db":  5432,
		},
		Extra: map[string]string{
			"FOO": "bar",
			"BAZ": "qux",
		},
	}

	if err := WriteEnv(path, original); err != nil {
		t.Fatalf("WriteEnv returned error: %v", err)
	}

	got, err := ReadEnv(path)
	if err != nil {
		t.Fatalf("ReadEnv returned error: %v", err)
	}

	if got.WTName != original.WTName {
		t.Errorf("WTName = %q, want %q", got.WTName, original.WTName)
	}
	if got.ComposeProjectName != original.ComposeProjectName {
		t.Errorf("ComposeProjectName = %q, want %q", got.ComposeProjectName, original.ComposeProjectName)
	}
	if got.ExecCmd != original.ExecCmd {
		t.Errorf("ExecCmd = %q, want %q", got.ExecCmd, original.ExecCmd)
	}
	if got.Ports["web"] != original.Ports["web"] {
		t.Errorf("Ports[web] = %d, want %d", got.Ports["web"], original.Ports["web"])
	}
	if got.Ports["db"] != original.Ports["db"] {
		t.Errorf("Ports[db] = %d, want %d", got.Ports["db"], original.Ports["db"])
	}
	if got.Extra["FOO"] != original.Extra["FOO"] {
		t.Errorf("Extra[FOO] = %q, want %q", got.Extra["FOO"], original.Extra["FOO"])
	}
	if got.Extra["BAZ"] != original.Extra["BAZ"] {
		t.Errorf("Extra[BAZ] = %q, want %q", got.Extra["BAZ"], original.Extra["BAZ"])
	}
}

func TestWriteEnv_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	cfg := &EnvConfig{
		WTName:             "test",
		ComposeProjectName: "test",
		ExecCmd:            "claude",
	}

	if err := WriteEnv(path, cfg); err != nil {
		t.Fatalf("WriteEnv returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("file mode = %o, want 0600", mode)
	}
}

func TestWriteEnvReadEnv_EmptyPortsAndExtra(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")

	original := &EnvConfig{
		WTName:             "minimal",
		ComposeProjectName: "dev-minimal",
		ExecCmd:            "claude",
	}

	if err := WriteEnv(path, original); err != nil {
		t.Fatalf("WriteEnv returned error: %v", err)
	}

	got, err := ReadEnv(path)
	if err != nil {
		t.Fatalf("ReadEnv returned error: %v", err)
	}

	if got.WTName != original.WTName {
		t.Errorf("WTName = %q, want %q", got.WTName, original.WTName)
	}
	if len(got.Ports) != 0 {
		t.Errorf("Ports should be empty, got %v", got.Ports)
	}
	if len(got.Extra) != 0 {
		t.Errorf("Extra should be empty, got %v", got.Extra)
	}
}

func TestReadEnv_MissingFile(t *testing.T) {
	_, err := ReadEnv("/nonexistent/path/.env")
	if err == nil {
		t.Error("ReadEnv should return error for missing file")
	}
}
