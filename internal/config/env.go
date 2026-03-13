package config

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// EnvConfig holds runtime configuration written to and read from .env files.
type EnvConfig struct {
	WTName             string            // container name prefix
	ComposeProjectName string            // docker-compose isolation
	ExecCmd            string            // AI CLI command (default: "claude")
	Ports              map[string]int    // service_name -> allocated_port
	Extra              map[string]string // other variables
}

// Well-known .env keys.
const (
	keyWTName             = "WT_NAME"
	keyComposeProjectName = "COMPOSE_PROJECT_NAME"
	keyExecCmd            = "EXEC_CMD"
	portPrefix            = "PORT_"
)

// WriteEnv writes cfg to path as a KEY=value .env file with mode 0600.
func WriteEnv(path string, cfg *EnvConfig) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating env file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	writeLine := func(key, value string) {
		fmt.Fprintf(w, "%s=%s\n", key, value)
	}

	// Core fields.
	writeLine(keyWTName, cfg.WTName)
	writeLine(keyComposeProjectName, cfg.ComposeProjectName)
	writeLine(keyExecCmd, cfg.ExecCmd)

	// Ports, sorted by service name for deterministic output.
	if len(cfg.Ports) > 0 {
		fmt.Fprintln(w, "# Allocated ports")
		names := make([]string, 0, len(cfg.Ports))
		for name := range cfg.Ports {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			writeLine(portPrefix+strings.ToUpper(name), strconv.Itoa(cfg.Ports[name]))
		}
	}

	// Extra variables, sorted by key.
	if len(cfg.Extra) > 0 {
		fmt.Fprintln(w, "# Extra")
		keys := make([]string, 0, len(cfg.Extra))
		for k := range cfg.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			writeLine(k, cfg.Extra[k])
		}
	}

	return w.Flush()
}

// ReadEnv reads an .env file from path and returns the parsed EnvConfig.
func ReadEnv(path string) (*EnvConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening env file: %w", err)
	}
	defer f.Close()

	cfg := &EnvConfig{
		Ports: make(map[string]int),
		Extra: make(map[string]string),
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		switch key {
		case keyWTName:
			cfg.WTName = value
		case keyComposeProjectName:
			cfg.ComposeProjectName = value
		case keyExecCmd:
			cfg.ExecCmd = value
		default:
			if strings.HasPrefix(key, portPrefix) {
				svcName := strings.ToLower(strings.TrimPrefix(key, portPrefix))
				port, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("invalid port value for %s: %w", key, err)
				}
				cfg.Ports[svcName] = port
			} else {
				cfg.Extra[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading env file: %w", err)
	}

	return cfg, nil
}
