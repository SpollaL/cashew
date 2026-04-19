package ledger_test

import (
	"testing"
	"time"

	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/ledger"
)

func date(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

var testTxs = []domain.Transaction{
	{Date: date(2025, 3, 1), Amount: 3000, Type: domain.Income},
	{Date: date(2025, 3, 5), Amount: 100, Type: domain.Expense, Category: "Groceries"},
	{Date: date(2025, 3, 10), Amount: 200, Type: domain.Expense, Category: "Dining"},
	{Date: date(2025, 3, 15), Amount: 500, Type: domain.Investment},
	{Date: date(2025, 3, 20), Amount: 150, Type: domain.Transfer},
	{Date: date(2025, 4, 1), Amount: 3000, Type: domain.Income},
	{Date: date(2025, 4, 5), Amount: 400, Type: domain.Expense, Category: "Groceries"},
}

func TestAggregate_Monthly(t *testing.T) {
	l := ledger.New(testTxs)
	summaries := l.Aggregate(domain.Monthly)

	if len(summaries) != 2 {
		t.Fatalf("got %d periods, want 2", len(summaries))
	}

	mar := summaries[0]
	if mar.Period.Label != "2025-03" {
		t.Errorf("period label %q, want 2025-03", mar.Period.Label)
	}
	if mar.Income != 3000 {
		t.Errorf("income %.2f, want 3000", mar.Income)
	}
	if mar.Expenses != 300 {
		t.Errorf("expenses %.2f, want 300", mar.Expenses)
	}
	if mar.Investments != 500 {
		t.Errorf("investments %.2f, want 500", mar.Investments)
	}
	want := float64(3000 - 300 - 500)
	if mar.Net != want {
		t.Errorf("net %.2f, want %.2f", mar.Net, want)
	}
}

func TestAggregate_TransfersExcluded(t *testing.T) {
	l := ledger.New(testTxs)
	summaries := l.Aggregate(domain.Monthly)

	mar := summaries[0]
	// Transfer of 150 must not affect any bucket
	if mar.Income+mar.Expenses+mar.Investments == 0 {
		t.Fatal("unexpected zero summary")
	}
	total := mar.Income - mar.Expenses - mar.Investments
	if mar.Net != total {
		t.Errorf("net mismatch, suggests transfer was counted")
	}
}

func TestAggregate_AllTime(t *testing.T) {
	l := ledger.New(testTxs)
	summaries := l.Aggregate(domain.AllTime)

	if len(summaries) != 1 {
		t.Fatalf("got %d periods, want 1", len(summaries))
	}
	s := summaries[0]
	if s.Income != 6000 {
		t.Errorf("income %.2f, want 6000", s.Income)
	}
	if s.Expenses != 700 {
		t.Errorf("expenses %.2f, want 700", s.Expenses)
	}
}

func TestTotalExpensesByCategory(t *testing.T) {
	l := ledger.New(testTxs)
	totals := l.TotalExpensesByCategory()

	if totals["Groceries"] != 500 {
		t.Errorf("Groceries %.2f, want 500", totals["Groceries"])
	}
	if totals["Dining"] != 200 {
		t.Errorf("Dining %.2f, want 200", totals["Dining"])
	}
	if len(totals) != 2 {
		t.Errorf("got %d categories, want 2", len(totals))
	}
}

func TestOnlyExpenses(t *testing.T) {
	l := ledger.New(testTxs).OnlyExpenses()
	for _, tx := range l.All() {
		if tx.Type != domain.Expense {
			t.Errorf("unexpected type %q in OnlyExpenses result", tx.Type)
		}
	}
	if len(l.All()) != 3 {
		t.Errorf("got %d transactions, want 3", len(l.All()))
	}
}

func TestInRange(t *testing.T) {
	l := ledger.New(testTxs).InRange(date(2025, 3, 1), date(2025, 4, 1))
	if len(l.All()) != 5 {
		t.Errorf("got %d transactions, want 5", len(l.All()))
	}
}

func TestInCategory(t *testing.T) {
	l := ledger.New(testTxs).InCategory("Groceries")
	if len(l.All()) != 2 {
		t.Errorf("got %d transactions, want 2", len(l.All()))
	}
}
