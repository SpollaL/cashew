package rules_test

import (
	"cashew/internal/domain"
	"cashew/internal/rules"
	"testing"
	"time"
)

var baseDate = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func tx(desc string, txType domain.TransactionType) domain.Transaction {
	return domain.Transaction{Date: baseDate, Description: desc, Amount: 10, Currency: "EUR", Type: txType}
}

func TestApply_SetsCategory(t *testing.T) {
	txs := []domain.Transaction{tx("Mercadona compra", domain.Expense)}
	rulesList := []domain.Rule{{Pattern: "mercadona", Category: "Groceries"}}

	got := rules.Apply(txs, rulesList)

	if got[0].Category != "Groceries" {
		t.Errorf("got category %q, want %q", got[0].Category, "Groceries")
	}
}

func TestApply_SetsType(t *testing.T) {
	txs := []domain.Transaction{tx("Al Pocket transfer", domain.Expense)}
	rulesList := []domain.Rule{{Pattern: "Al Pocket", Type: domain.Transfer}}

	got := rules.Apply(txs, rulesList)

	if got[0].Type != domain.Transfer {
		t.Errorf("got type %q, want %q", got[0].Type, domain.Transfer)
	}
}

func TestApply_FirstMatchWins(t *testing.T) {
	txs := []domain.Transaction{tx("Mercadona compra", domain.Expense)}
	rulesList := []domain.Rule{
		{Pattern: "mercadona", Category: "Groceries"},
		{Pattern: "compra", Category: "Shopping"},
	}

	got := rules.Apply(txs, rulesList)

	if got[0].Category != "Groceries" {
		t.Errorf("got category %q, want first match %q", got[0].Category, "Groceries")
	}
}

func TestApply_CategoryOnlyForExpenses(t *testing.T) {
	txs := []domain.Transaction{tx("Salary payment", domain.Income)}
	rulesList := []domain.Rule{{Pattern: "salary", Category: "Work"}}

	got := rules.Apply(txs, rulesList)

	if got[0].Category != "" {
		t.Errorf("income transaction got category %q, want empty", got[0].Category)
	}
}

func TestApply_CaseInsensitiveMatch(t *testing.T) {
	txs := []domain.Transaction{tx("MERCADONA VILLAVERDE", domain.Expense)}
	rulesList := []domain.Rule{{Pattern: "mercadona", Category: "Groceries"}}

	got := rules.Apply(txs, rulesList)

	if got[0].Category != "Groceries" {
		t.Errorf("got category %q, want %q", got[0].Category, "Groceries")
	}
}

func TestApply_DoesNotMutateInput(t *testing.T) {
	original := []domain.Transaction{tx("Mercadona", domain.Expense)}
	rulesList := []domain.Rule{{Pattern: "mercadona", Category: "Groceries"}}

	rules.Apply(original, rulesList)

	if original[0].Category != "" {
		t.Error("Apply mutated the input slice")
	}
}

func TestUncategorised_ReturnsUncategorisedExpenses(t *testing.T) {
	txs := []domain.Transaction{
		tx("Mercadona", domain.Expense),
		tx("Salary", domain.Income),
	}

	got := rules.Uncategorised(txs, nil)

	if len(got) != 1 || got[0] != "Mercadona" {
		t.Errorf("got %v, want [Mercadona]", got)
	}
}

func TestUncategorised_ExcludesAcknowledgedDescriptions(t *testing.T) {
	txs := []domain.Transaction{tx("Bizum", domain.Expense)}
	// Pattern-only rule acknowledges Bizum without assigning a category.
	rulesList := []domain.Rule{{Pattern: "Bizum"}}

	got := rules.Uncategorised(txs, rulesList)

	if len(got) != 0 {
		t.Errorf("got %v, want empty — Bizum is acknowledged", got)
	}
}

func TestUncategorised_DeduplicatesDescriptions(t *testing.T) {
	txs := []domain.Transaction{
		tx("Mercadona", domain.Expense),
		tx("Mercadona", domain.Expense),
	}

	got := rules.Uncategorised(txs, nil)

	if len(got) != 1 {
		t.Errorf("got %d entries, want 1 (deduplication)", len(got))
	}
}
