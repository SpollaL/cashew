package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SpollaL/cashew/internal/config"
)

func TestRulesPath_EnvVar(t *testing.T) {
	t.Setenv("CASHEW_RULES", "/custom/path/rules.toml")

	got, err := config.RulesPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "/custom/path/rules.toml" {
		t.Errorf("got %q, want %q", got, "/custom/path/rules.toml")
	}
}

func TestRulesPath_HomeDir(t *testing.T) {
	t.Setenv("CASHEW_RULES", "")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got, err := config.RulesPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(tmp, ".cashew", "rules.toml")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	dir := filepath.Dir(got)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("expected dir %q to exist", dir)
	}
}

func TestRulesPath_HomeDir_EndsCorrectly(t *testing.T) {
	t.Setenv("CASHEW_RULES", "")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got, err := config.RulesPath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(got, filepath.Join(".cashew", "rules.toml")) {
		t.Errorf("path %q does not end with .cashew/rules.toml", got)
	}
}
