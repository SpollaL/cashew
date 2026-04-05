# Design Patterns in Cashew

This document walks through the deliberate design decisions made in the non-TUI layers of cashew (`domain`, `parser`, `rules`, `ledger`). For each pattern you'll find what it is, where it appears in the code, and why it was chosen over simpler alternatives.

---

## 1. Layered Architecture

### What it is

The codebase is split into layers where each layer only depends on layers below it, never above:

```
cmd/cashew         ← entry point, wires everything together
internal/tui       ← user interface
internal/ledger    ← aggregation and filtering
internal/rules     ← categorisation engine + persistence
internal/parser    ← file parsing per bank
internal/domain    ← shared data types, no dependencies
```

`domain` knows nothing about parsers. `parser` knows about `domain` but not `rules`. `rules` knows about `domain` but not `ledger`. And so on.

### Why it matters

If you ever want to replace the TUI with a web interface, or add a new bank parser, or change how rules are stored — you only touch one layer. The others don't need to change.

It also makes testing easier. You can test `rules.Apply()` without starting the TUI, without parsing files, without touching the filesystem. Each layer is independently testable.

### The alternative

The alternative is to put everything in one place — one big file where parsing, categorising, and displaying all happen together. This is tempting when a project is small. But it compounds over time: every change risks breaking something unrelated, and testing requires setting up the full app.

---

## 2. The Domain Model as a Neutral Language

### What it is

`internal/domain` defines the core types — `Transaction`, `Rule`, `Period`, `Granularity` — with no dependencies on any other internal package. It's the shared vocabulary everyone else speaks.

```go
// internal/domain/transaction.go
type Transaction struct {
    Date        time.Time
    Description string
    Amount      float64 // always positive; Type encodes direction
    Currency    string
    Type        TransactionType
    Category    string
    Bank        string
}
```

### Why it matters

When `parser` produces a `Transaction` and `ledger` consumes one, they don't need to know about each other — they just agree on the shape of the data. This is sometimes called a **shared kernel**: a small, stable core that all layers agree on.

Notice `Amount` is always positive. The sign information lives in `Type` (`income`, `expense`, etc.). This is a deliberate choice: it means you never have to remember "is a negative amount an expense or income for this bank?" The parser figures that out once; everyone else just reads `Type`.

---

## 3. The Strategy Pattern (Parser Interface)

### What it is

The Strategy pattern defines a family of algorithms behind a common interface, making them interchangeable. In cashew this is the `Parser` interface:

```go
// internal/parser/parser.go
type Parser interface {
    Name() string
    Detect(path string, header []string) bool
    Parse(r io.Reader) ([]domain.Transaction, error)
}
```

`Revolut` and `BBVA` are two concrete strategies. `Load()` selects the right one at runtime:

```go
for _, p := range parsers {
    if p.Detect(path, header) {
        matched = p
        break
    }
}
```

### Why it matters

Adding a new bank requires writing one new file that implements three methods. Nothing else changes — not `main.go`'s logic, not the rules engine, not the ledger. This is the **Open/Closed Principle**: open for extension, closed for modification.

The `Detect` method is particularly interesting. For CSV files it receives the header row; for non-CSV files (like BBVA's `.xlsx`) it receives `nil` and falls back to checking the file extension. This lets each parser define its own detection logic without the orchestrator needing to know anything about file formats.

### The alternative

Without the interface you might write:

```go
if filepath.Ext(path) == ".xlsx" {
    // bbva logic
} else if isBBVACSV(header) {
    // ...
} else if isRevolutCSV(header) {
    // ...
}
```

This works for two banks but becomes a wall of if-else as you add more. Every addition touches the same block of code, increasing the risk of breaking existing parsers.

---

## 4. Pure Functions in the Rules Engine

### What it is

`rules.Apply()` takes a slice of transactions and a list of rules, and returns a new slice with types and categories filled in. It does not modify the input slice, write to files, or have any side effects:

```go
func Apply(txs []domain.Transaction, rules []domain.Rule) []domain.Transaction {
    result := make([]domain.Transaction, len(txs))
    copy(result, txs)
    // ... modifies result, never txs
    return result
}
```

Functions with no side effects are called **pure functions**.

### Why it matters

Pure functions are trivial to test — you call them with inputs, check the output, done. No setup, no teardown, no mocking. This is exactly why the rules engine tests are so straightforward:

```go
func TestApply_DoesNotMutateInput(t *testing.T) {
    original := []domain.Transaction{tx("Mercadona", domain.Expense)}
    rules.Apply(original, rulesList)
    if original[0].Category != "" {
        t.Error("Apply mutated the input slice")
    }
}
```

It also means you can safely call `Apply` multiple times with different rule sets and always get a predictable result. There's no hidden state that builds up between calls.

---

## 5. The Immutable Builder Pattern (Ledger)

### What it is

The `Ledger` type wraps a slice of transactions. Every filter method returns a **new** Ledger rather than modifying the existing one:

```go
// internal/ledger/ledger.go
func (l Ledger) OnlyExpenses() Ledger {
    return l.where(func(tx domain.Transaction) bool {
        return tx.Type == domain.Expense
    })
}
```

This allows you to chain filters:

```go
l.OnlyExpenses().InCategory("Groceries").InRange(start, end).Aggregate(Monthly)
```

Each step produces a new, independent view — the original `l` is untouched.

### Why it matters

This pattern makes the data flow explicit and safe. The TUI can hold one `fullLedger` (all transactions) and derive filtered views from it whenever it needs to, without worrying that a filtered view will corrupt the original data.

It also means a drill-down in the UI (e.g. "show me Groceries in March") is just two method calls on the full ledger — no special-case code needed.

### The alternative

A mutable approach would keep a single list and add/remove filter state on it. This is error-prone: you have to remember to reset filters, it's hard to hold multiple views simultaneously, and tests require careful setup and cleanup.

---

## 6. The Single Aggregation Function

### What it is

Rather than writing separate functions for "aggregate by month", "aggregate by year", etc., there is exactly one function:

```go
func (l Ledger) Aggregate(g domain.Granularity) []Summary {
    for _, tx := range l.txs {
        p := domain.Bucket(tx.Date, g)  // ← this is the key
        // group by p.Label ...
    }
}
```

`domain.Bucket()` maps any timestamp to a period label. Change `g` from `Monthly` to `Weekly` and the same loop groups transactions by week instead.

### Why it matters

This is the **Don't Repeat Yourself** principle applied to aggregation. Without it, you'd have five near-identical functions — `aggregateByDay`, `aggregateByWeek`, etc. — each with its own subtle bugs and each needing its own tests.

The abstraction pays for itself immediately: the summary view, the categories pivot, and the transactions filter all use the same `Aggregate` call. Adding a new granularity (e.g. quarterly) means adding one case to `Bucket()` — nothing else changes.

---

## 7. Separation of Persistence from Logic (Rules)

### What it is

The rules package has two files with distinct responsibilities:

- `engine.go` — pure logic: apply rules to transactions, find uncategorised ones
- `store.go` — I/O: read from and write to `rules.toml`

They are separate functions, not mixed together. `Apply()` never touches the filesystem. `SaveRule()` never runs the matching logic.

### Why it matters

This is sometimes called **separating concerns**. The practical benefit: you can test the entire rules engine — matching logic, precedence, case-insensitivity — without creating any files. The tests just call `Apply()` with a list of rules in memory.

If the storage format ever changed (say, from TOML to a database), you'd only touch `store.go`. The engine and its tests would be completely unaffected.

---

## 8. Explicit Dependency Injection in main.go

### What it is

`main.go` is the only place in the whole codebase that knows which concrete types are used. It builds the list of parsers, loads rules, creates the ledger, and passes everything to `tui.New()`:

```go
parsers := []parser.Parser{
    parser.Revolut{},
    parser.BBVA{},
}
// ...
app := tui.New(l, rulesList, buckets, rulesPath)
```

None of the internal packages create their own dependencies. They receive what they need as arguments.

### Why it matters

This is **Dependency Injection** — pushing the assembly of components to the edges of the program. Each package does one job and accepts its inputs rather than fetching them.

The practical result: to add a new bank you add one line to `main.go`. To run tests with a fake parser, you pass in the fake. No global variables, no `init()` functions, no hidden coupling.

---

## Summary

| Pattern | Where | Benefit |
|---------|-------|---------|
| Layered architecture | whole repo | isolate changes, independent testing |
| Domain model | `internal/domain` | shared vocabulary, stable core |
| Strategy (interface) | `internal/parser` | add banks without changing existing code |
| Pure functions | `internal/rules/engine.go` | predictable, easily tested |
| Immutable builder | `internal/ledger` | safe multi-view derived data |
| Single aggregation fn | `internal/ledger/ledger.go` | one implementation for all granularities |
| Separated persistence | `internal/rules` | test logic without filesystem |
| Dependency injection | `cmd/cashew/main.go` | no hidden coupling, easy to extend |

These patterns aren't rules to follow dogmatically — they're solutions to recurring problems. The question to ask is always: *what problem does this solve, and is that problem present here?* In a small script that runs once, most of this would be overkill. In a codebase that grows, gets tested, and has multiple banks to support, each of these pays for itself.
