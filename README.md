# cashew

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
go build -o cashew ./cmd/cashew
```

Requires Go 1.24+.

## Usage

```bash
./cashew data/*.csv data/*.xlsx
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
| `enter` | Pick category |
| `i` | Mark as income |
| `x` | Mark as transfer |
| `v` | Mark as investment |
| `n` | No category (dismiss) |

## Rules

On first run, cashew creates `rules.toml` from the example file. Rules match transaction descriptions by substring (case-insensitive) and can set a type, a category, or both.

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
cmd/cashew/        entry point
internal/
  domain/          Transaction, TransactionType, Period, Granularity, Rule
  parser/          bank-specific parsers (Revolut, BBVA) + deduplication
  rules/           rule engine + TOML persistence
  ledger/          immutable, chainable filters + aggregation
  tui/             Bubble Tea app + views (summary, categories, transactions, review)
```
