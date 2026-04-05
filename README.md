# cashew

[![CI](https://github.com/SpollaL/cashew/actions/workflows/ci.yml/badge.svg)](https://github.com/SpollaL/cashew/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Personal finance tracker for the terminal. Reads CSV/XLSX exports from your bank, categorises transactions with a rule engine, and presents income, expenses and investments across configurable time granularities.

## Supported banks

| Bank | Format | Notes |
|------|--------|-------|
| Revolut | CSV | es-ES and en-GB exports |
| BBVA | XLSX | Standard account statement export |

> Other banks are not supported yet. Adding a new one requires implementing the `Parser` interface in `internal/parser/`.

## Features

- **Rule-based categorisation**: pattern matching with a TOML config file
- **Review queue**: triage uncategorised expenses without leaving the terminal
- **Category pivot table**: drill down from any cell into filtered transactions
- **Granularities**: daily / weekly / monthly / yearly / all-time with a single keypress

## Installation

```bash
git clone https://github.com/SpollaL/cashew
cd cashew
go build -o cashew ./cmd/cashew          # terminal UI
go build -o cashew-server ./cmd/server   # web UI (optional)
```

Requires Go 1.24+.

## Usage

**Terminal UI**

```bash
./cashew data/*.csv data/*.xlsx
```

**Web UI** (handy on mobile)

```bash
./cashew-server -addr :8080 data/*.csv data/*.xlsx
# then open http://localhost:8080 in any browser
```

Place your bank exports in a `data/` directory (gitignored). The app detects each bank automatically.

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| `s` | Summary view |
| `c` | Categories pivot |
| `t` | Transactions view |
| `r` | Review queue |
| `q` | Quit |

**Summary / Categories**

| Key | Granularity |
|-----|------------|
| `a` | All time |
| `y` | Yearly |
| `M` | Monthly |
| `w` | Weekly |
| `d` | Daily |

**Transactions**

| Key | Action |
|-----|--------|
| `↑`/`↓` | Navigate |
| `e` | Edit type and category |
| `f` | Filter panel |
| `esc` | Clear filter / go back |

**Review queue**

| Key | Action |
|-----|--------|
| `enter` | Pick category (expense) |
| `i` | Mark as income |
| `T` | Mark as transfer |
| `I` | Mark as investment |
| `n` | No category (dismiss) |

## Rules

If `rules.toml` doesn't exist, cashew creates one with a default set of category buckets and no rules — ready for you to populate via the review workflow. Rules match transaction descriptions by substring (case-insensitive) and can set a type, a category, or both.

```toml
[categories]
buckets = ["Groceries", "Dining", "Transport", "Housing"]

[[rules]]
pattern = "Mercadona"
category = "Groceries"

[[rules]]
pattern = "Al Pocket"
type = "transfer"

[[rules]]
pattern = "Nómina"
type = "income"
```

Rules saved from the TUI are prepended so they take precedence over handwritten substring rules. `rules.toml` is gitignored — use `rules.example.toml` as a template.

## Architecture

```
cmd/cashew/        terminal UI entry point
cmd/server/        web UI entry point
internal/
  domain/          Transaction, TransactionType, Period, Granularity, Rule
  parser/          bank-specific parsers (Revolut, BBVA) + deduplication
  rules/           rule engine + TOML persistence
  ledger/          immutable, chainable filters + aggregation
  tui/             Bubble Tea app + views (summary, categories, transactions, review)
  server/          HTTP handlers + HTML templates (summary, categories, transactions, review)
```
