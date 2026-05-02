package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// RulesPath returns the resolved path to rules.toml.
// Priority: $CASHEW_RULES env var → ~/.cashew/rules.toml
// When falling back to the home dir, the parent directory is created if missing.
func RulesPath() (string, error) {
	if p := os.Getenv("CASHEW_RULES"); p != "" {
		return p, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve rules path: %w", err)
	}

	dir := filepath.Join(home, ".cashew")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create config dir %s: %w", dir, err)
	}

	return filepath.Join(dir, "rules.toml"), nil
}
