package config_test

import (
	"os"
	"path/filepath"
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
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		t.Errorf("expected dir %q to exist", dir)
	} else if info.Mode().Perm() != 0700 {
		t.Errorf("dir permissions = %o, want 0700", info.Mode().Perm())
	}
}

func TestRulesPath_HomeDir_MkdirAllFails(t *testing.T) {
	t.Setenv("CASHEW_RULES", "")
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Place a regular file where the dir should be, so MkdirAll fails.
	blocker := filepath.Join(tmp, ".cashew")
	if err := os.WriteFile(blocker, []byte("block"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := config.RulesPath()
	if err == nil {
		t.Fatal("expected error when MkdirAll fails, got nil")
	}
}

