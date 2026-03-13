package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDevYml_Valid(t *testing.T) {
	dir := t.TempDir()
	content := `image: golang:1.22
ports:
  - 3000
  - 8080
services:
  db:
    image: postgres:16
    ports:
      - 5432
    env:
      POSTGRES_PASSWORD: test
setup: "go mod download"
exec_cmd: "claude"
`
	if err := os.WriteFile(filepath.Join(dir, ".dev.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadDevYml(dir)
	if err != nil {
		t.Fatalf("LoadDevYml returned error: %v", err)
	}

	if cfg.Image != "golang:1.22" {
		t.Errorf("Image = %q, want %q", cfg.Image, "golang:1.22")
	}
	if len(cfg.Ports) != 2 {
		t.Errorf("Ports count = %d, want 2", len(cfg.Ports))
	}
	if cfg.Ports[0] != 3000 || cfg.Ports[1] != 8080 {
		t.Errorf("Ports = %v, want [3000 8080]", cfg.Ports)
	}
	if len(cfg.Services) != 1 {
		t.Errorf("Services count = %d, want 1", len(cfg.Services))
	}
	db, ok := cfg.Services["db"]
	if !ok {
		t.Fatal("missing service 'db'")
	}
	if db.Image != "postgres:16" {
		t.Errorf("db.Image = %q, want %q", db.Image, "postgres:16")
	}
	if cfg.Setup != "go mod download" {
		t.Errorf("Setup = %q, want %q", cfg.Setup, "go mod download")
	}
	if cfg.ExecCmd != "claude" {
		t.Errorf("ExecCmd = %q, want %q", cfg.ExecCmd, "claude")
	}
}

func TestLoadDevYml_MissingImage(t *testing.T) {
	dir := t.TempDir()
	content := `ports:
  - 3000
`
	if err := os.WriteFile(filepath.Join(dir, ".dev.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadDevYml(dir)
	if err == nil {
		t.Error("LoadDevYml should return error when image is missing")
	}
}

func TestLoadDevYml_InvalidPort(t *testing.T) {
	dir := t.TempDir()
	content := `image: golang:1.22
ports:
  - 80
`
	if err := os.WriteFile(filepath.Join(dir, ".dev.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadDevYml(dir)
	if err == nil {
		t.Error("LoadDevYml should return error for port below 1024")
	}
}

func TestLoadDevYml_InvalidPortHigh(t *testing.T) {
	dir := t.TempDir()
	content := `image: golang:1.22
ports:
  - 70000
`
	if err := os.WriteFile(filepath.Join(dir, ".dev.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadDevYml(dir)
	if err == nil {
		t.Error("LoadDevYml should return error for port above 65535")
	}
}

func TestLoadDevYml_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadDevYml(dir)
	if err == nil {
		t.Error("LoadDevYml should return error when .dev.yml is missing")
	}
}

func TestLoadDevYml_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := `image: [invalid yaml`
	if err := os.WriteFile(filepath.Join(dir, ".dev.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadDevYml(dir)
	if err == nil {
		t.Error("LoadDevYml should return error for invalid YAML")
	}
}

func TestLoadDevYml_ServiceInvalidPort(t *testing.T) {
	dir := t.TempDir()
	content := `image: golang:1.22
services:
  db:
    image: postgres:16
    ports:
      - 80
`
	if err := os.WriteFile(filepath.Join(dir, ".dev.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadDevYml(dir)
	if err == nil {
		t.Error("LoadDevYml should return error for service with invalid port")
	}
}
