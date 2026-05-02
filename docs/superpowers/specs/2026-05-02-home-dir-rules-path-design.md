# Design: Home-directory rules path

## Goal

Move the default `rules.toml` location from the current working directory to
`~/.cashew/rules.toml`, add an env-var override, and expose a `cashew rules`
subcommand that opens the file in `$EDITOR`.

## Architecture

New package `internal/config` with one exported function:

```go
// RulesPath returns the resolved path to rules.toml.
// Priority: $CASHEW_RULES env var → ~/.cashew/rules.toml
// MkdirAll is called on the parent dir when falling back to the home path.
func RulesPath() (string, error)
```

- If `$CASHEW_RULES` is set, return it as-is (no MkdirAll — user's responsibility).
- Otherwise: call `os.UserHomeDir()`, append `/.cashew/rules.toml`, call
  `os.MkdirAll` on `~/.cashew/`, return the path.
- Returns `(string, error)` — errors from `UserHomeDir` or `MkdirAll` propagate to callers.

`internal/rules/store.go` is unchanged — it remains fully path-agnostic.

## Binary changes

### `cmd/cashew/main.go`

1. Before `flag.Parse`, check `len(os.Args) > 1 && os.Args[1] == "rules"`:
   - Resolve path via `config.RulesPath()`.
   - Open file in `$EDITOR`; fallback to `vi` if unset.
   - `exec.Command` the editor, connect stdio, exit when done.
2. Replace hardcoded `rulesPath := "rules.toml"` with `rulesPath, err := config.RulesPath()`.

### `cmd/server/main.go`

- Change `-rules` flag default from `"rules.toml"` to the result of
  `config.RulesPath()` — env var wins, flag overrides both.
- Error from `config.RulesPath()` at startup exits with message.

## Error handling

| Scenario | Behaviour |
|---|---|
| `UserHomeDir` fails | Error returned → main prints to stderr, exits 1 |
| `MkdirAll` fails | Same |
| `$EDITOR` unset | Fallback to `vi` |
| Editor exec fails | Print error to stderr, exit 1 |
| `$CASHEW_RULES` set to bad path | `rules.Load` will create or error — existing behaviour |

## Testing

- `config.RulesPath()` tested with `t.Setenv("CASHEW_RULES", ...)` set and unset.
- Env-var branch: verify returned path matches env var exactly.
- Home-dir branch: verify path ends with `/.cashew/rules.toml` and parent dir exists.

## Out of scope

- Migrating an existing CWD `rules.toml` to the new location (user does this manually).
- The CWD `rules.toml` file is ignored — not deleted, not read.
