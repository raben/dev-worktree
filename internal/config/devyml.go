package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Service represents a supporting service (e.g. database) defined in .dev.yml.
type Service struct {
	Image string            `yaml:"image"`
	Env   map[string]string `yaml:"env"`
	Ports []int             `yaml:"ports"`
}

// DevYml represents the parsed contents of a .dev.yml file.
type DevYml struct {
	Image    string             `yaml:"image"`
	Ports    []int              `yaml:"ports"`
	Services map[string]Service `yaml:"services"`
	Setup    string             `yaml:"setup"`
	ExecCmd  string             `yaml:"exec_cmd"`
}

// LoadDevYml reads and parses .dev.yml from dir.
// It returns an error if the file is missing or the content is invalid.
func LoadDevYml(dir string) (*DevYml, error) {
	path := filepath.Join(dir, ".dev.yml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(".dev.yml not found in %s", dir)
		}
		return nil, fmt.Errorf("reading .dev.yml: %w", err)
	}

	var cfg DevYml
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing .dev.yml: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid .dev.yml: %w", err)
	}

	return &cfg, nil
}

func (d *DevYml) validate() error {
	if d.Image == "" {
		return fmt.Errorf("image is required")
	}

	if err := validatePorts(d.Ports, "top-level"); err != nil {
		return err
	}

	for name, svc := range d.Services {
		if err := validatePorts(svc.Ports, fmt.Sprintf("service %q", name)); err != nil {
			return err
		}
	}

	return nil
}

func validatePorts(ports []int, context string) error {
	for _, p := range ports {
		if p < 1024 || p > 65535 {
			return fmt.Errorf("%s: port %d out of range 1024-65535", context, p)
		}
	}
	return nil
}
