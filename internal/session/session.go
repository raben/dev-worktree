package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var ErrNoSession = errors.New("no active session")

type RepoMount struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	Name          string `json:"name"`
}

type Session struct {
	ContainerID string      `json:"container_id"`
	Image       string      `json:"image"`
	Repos       []RepoMount `json:"repos"`
}

func sessionDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, ".dev"), nil
}

func sessionPath() (string, error) {
	dir, err := sessionDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "session.json"), nil
}

func Load() (*Session, error) {
	path, err := sessionPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSession
		}
		return nil, fmt.Errorf("reading session: %w", err)
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}
	return &s, nil
}

func Save(s *Session) error {
	dir, err := sessionDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating session dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}
	path, err := sessionPath()
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}
	return os.Rename(tmp, path)
}

func Exists() bool {
	path, err := sessionPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

func Clear() error {
	path, err := sessionPath()
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *Session) HasRepo(hostPath string) bool {
	for _, r := range s.Repos {
		if r.HostPath == hostPath {
			return true
		}
	}
	return false
}

func (s *Session) Binds() []string {
	binds := make([]string, len(s.Repos))
	for i, r := range s.Repos {
		binds[i] = r.HostPath + ":" + r.ContainerPath + ":cached"
	}
	return binds
}
