# Home-directory rules path — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move `rules.toml` default to `~/.cashew/rules.toml`, add `$CASHEW_RULES` env-var override, and add `cashew rules` subcommand that opens the file in `$EDITOR`.

**Architecture:** New `internal/config` package exposes `RulesPath()` — reads env var or builds home-dir path + creates the parent dir. Both binaries call it; `internal/rules` is untouched.

**Tech Stack:** Go stdlib (`os`, `os/exec`, `path/filepath`), existing `github.com/SpollaL/cashew` module.

---

## Task 1: `internal/config` package with `RulesPath()`

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/config/... -v
```

Expected: compile error — package `config` does not exist yet.

- [ ] **Step 3: Implement `internal/config/config.go`**

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all three tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add RulesPath resolving CASHEW_RULES or ~/.cashew/rules.toml"
```

---

## Task 2: Wire `cmd/cashew/main.go`

**Files:**
- Modify: `cmd/cashew/main.go`

- [ ] **Step 1: Add `cashew rules` subcommand and replace hardcoded path**

Replace the current `main.go` content with the version below. Key changes:
1. Subcommand check before `flag.Parse`
2. `rulesPath` resolved via `config.RulesPath()`

```go
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/SpollaL/cashew/internal/config"
	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/ledger"
	"github.com/SpollaL/cashew/internal/parser"
	"github.com/SpollaL/cashew/internal/rules"
	"github.com/SpollaL/cashew/internal/tui"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "rules" {
		path, err := config.RulesPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	model := flag.String("model", "hf.co/Qwen/Qwen3-4B-GGUF:Q4_K_M", "Ollama model to use for chat (e.g. gemma3, llama3.2)")
	debug := flag.Bool("debug", false, "show LLM roundtrip details in the chat view")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println(version)
		return
	}

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cashew [-model <name>] <file> [file2 ...]")
		os.Exit(1)
	}

	rulesPath, err := config.RulesPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving rules path:", err)
		os.Exit(1)
	}

	parsers := []parser.Parser{
		parser.Revolut{},
		parser.BBVA{},
	}

	var allTxs []domain.Transaction
	for _, path := range flag.Args() {
		txs, err := parser.Load(path, parsers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", path, err)
			os.Exit(1)
		}
		allTxs = append(allTxs, txs...)
	}

	allTxs = parser.Deduplicate(allTxs)

	rulesList, buckets, err := rules.Load(rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading rules: %v\n", err)
		os.Exit(1)
	}

	allTxs = rules.Apply(allTxs, rulesList)
	l := ledger.New(allTxs)

	app := tui.New(l, rulesList, buckets, rulesPath, *model, *debug)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
go build ./cmd/cashew/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Smoke-test `cashew rules`**

```bash
CASHEW_RULES=/tmp/test-rules.toml EDITOR=cat ./cashew rules
```

Expected: `cat` opens (or prints) `/tmp/test-rules.toml` (may be empty or show contents if file exists).

- [ ] **Step 4: Commit**

```bash
git add cmd/cashew/main.go
git commit -m "feat(cashew): default rules path to ~/.cashew/rules.toml; add 'cashew rules' subcommand"
```

---

## Task 3: Wire `cmd/server/main.go`

**Files:**
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Replace hardcoded `-rules` default with `config.RulesPath()`**

Replace `cmd/server/main.go` with:

```go
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/SpollaL/cashew/internal/config"
	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/parser"
	"github.com/SpollaL/cashew/internal/rules"
	"github.com/SpollaL/cashew/internal/server"
)

var version = "dev"

func main() {
	defaultRules, err := config.RulesPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error resolving rules path:", err)
		os.Exit(1)
	}

	addr := flag.String("addr", ":8080", "listen address")
	rulesPath := flag.String("rules", defaultRules, "path to rules.toml")
	ver := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *ver {
		fmt.Println(version)
		return
	}

	files := flag.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "usage: cashew-server [flags] <file> [file2 ...]")
		os.Exit(1)
	}

	parsers := []parser.Parser{parser.Revolut{}, parser.BBVA{}}

	var allTxs []domain.Transaction
	for _, path := range files {
		txs, err := parser.Load(path, parsers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", path, err)
			os.Exit(1)
		}
		allTxs = append(allTxs, txs...)
	}
	allTxs = parser.Deduplicate(allTxs)

	rulesList, buckets, err := rules.Load(*rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading rules: %v\n", err)
		os.Exit(1)
	}

	srv := server.New(allTxs, rulesList, buckets, *rulesPath)

	fmt.Printf("cashew running at http://localhost%s\n", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
go build ./cmd/server/...
```

Expected: no output, exit 0.

- [ ] **Step 3: Verify `-rules` flag still works**

```bash
./cashew-server -rules /tmp/other.toml 2>&1 | head -5
```

Expected: error about no CSV files (no positional args), not a rules-path error — confirms the flag is wired.

- [ ] **Step 4: Run full test suite**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat(server): default -rules flag to ~/.cashew/rules.toml via config.RulesPath"
```
