package ledger

import (
	"github.com/SpollaL/cashew/internal/domain"
	"sort"
	"time"
)

// Summary holds aggregated figures for one period.
type Summary struct {
	Period      domain.Period
	Income      float64
	Expenses    float64
	Investments float64
	Net         float64
	ByCategory  map[string]float64 // expense totals per category
}

// Ledger is an immutable, filterable view over a set of transactions.
// All filter methods return a new Ledger — the original is never modified.
type Ledger struct {
	txs []domain.Transaction
}

func New(txs []domain.Transaction) Ledger {
	return Ledger{txs: txs}
}

func (l Ledger) All() []domain.Transaction {
	return l.txs
}

// ── Filters ───────────────────────────────────────────────────────────────────

func (l Ledger) OnlyExpenses() Ledger {
	return l.where(func(tx domain.Transaction) bool {
		return tx.Type == domain.Expense
	})
}

func (l Ledger) OnlyIncome() Ledger {
	return l.where(func(tx domain.Transaction) bool {
		return tx.Type == domain.Income
	})
}

func (l Ledger) OnlyInvestments() Ledger {
	return l.where(func(tx domain.Transaction) bool {
		return tx.Type == domain.Investment
	})
}

func (l Ledger) InRange(start, end time.Time) Ledger {
	return l.where(func(tx domain.Transaction) bool {
		return !tx.Date.Before(start) && tx.Date.Before(end)
	})
}

func (l Ledger) InCategory(cat string) Ledger {
	return l.where(func(tx domain.Transaction) bool {
		return tx.Category == cat
	})
}

func (l Ledger) where(keep func(domain.Transaction) bool) Ledger {
	var out []domain.Transaction
	for _, tx := range l.txs {
		if keep(tx) {
			out = append(out, tx)
		}
	}
	return Ledger{txs: out}
}

// ── Aggregation ───────────────────────────────────────────────────────────────

// Aggregate buckets transactions by granularity and sums each bucket.
// This single function handles all time granularities — daily, weekly,
// monthly, yearly, or all-time — without any duplicated logic.
func (l Ledger) Aggregate(g domain.Granularity) []Summary {
	type entry struct {
		summary Summary
		order   int
	}

	buckets := map[string]*entry{}
	seq := 0

	for _, tx := range l.txs {
		p := domain.Bucket(tx.Date, g)
		e, exists := buckets[p.Label]
		if !exists {
			e = &entry{
				summary: Summary{Period: p, ByCategory: map[string]float64{}},
				order:   seq,
			}
			buckets[p.Label] = e
			seq++
		}
		s := &e.summary
		switch tx.Type {
		case domain.Income:
			s.Income += tx.Amount
		case domain.Expense:
			s.Expenses += tx.Amount
			if tx.Category != "" {
				s.ByCategory[tx.Category] += tx.Amount
			}
		case domain.Investment:
			s.Investments += tx.Amount
		}
		s.Net = s.Income - s.Expenses - s.Investments
	}

	result := make([]Summary, len(buckets))
	for _, e := range buckets {
		result[e.order] = e.summary
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Period.Start.Before(result[j].Period.Start)
	})

	return result
}

// TotalExpensesByCategory sums all expenses across all time, grouped by category.
// Useful for the category breakdown chart.
func (l Ledger) TotalExpensesByCategory() map[string]float64 {
	totals := map[string]float64{}
	for _, tx := range l.txs {
		if tx.Type == domain.Expense {
			totals[tx.Category] += tx.Amount
		}
	}
	return totals
}
