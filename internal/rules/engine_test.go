package rules_test

import (
	"github.com/spolla-l/cashew/internal/domain"
	"github.com/spolla-l/cashew/internal/rules"
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

func TestApply_ClearsCategoryWhenTypeChangedToIncome(t *testing.T) {
	// Simulate an expense that already has a category (from a previous rule run),
	// and a new rule changes its type to Income — category must be cleared.
	txWithCategory := tx("Freelance payment", domain.Expense)
	txWithCategory.Category = "Work"
	txs := []domain.Transaction{txWithCategory}
	rulesList := []domain.Rule{{Pattern: "freelance", Type: domain.Income}}

	got := rules.Apply(txs, rulesList)

	if got[0].Type != domain.Income {
		t.Errorf("got type %q, want Income", got[0].Type)
	}
	if got[0].Category != "" {
		t.Errorf("income transaction kept category %q, want empty", got[0].Category)
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

	if len(got) != 1 || got[0].Description != "Mercadona" {
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

func TestUnreviewedTransfers_ReturnsUnmatchedTransfers(t *testing.T) {
	txs := []domain.Transaction{
		tx("Al Pocket", domain.Transfer),
		tx("Salary", domain.Income),      // not a transfer, excluded
		tx("Al Pocket", domain.Transfer), // duplicate description
	}

	got := rules.UnreviewedTransfers(txs, nil)

	if len(got) != 1 || got[0].Description != "Al Pocket" {
		t.Errorf("got %v, want [Al Pocket]", got)
	}
}

func TestUnreviewedTransfers_ExcludesMatchedByRule(t *testing.T) {
	txs := []domain.Transaction{tx("Al Pocket", domain.Transfer)}
	rulesList := []domain.Rule{{Pattern: "pocket"}}

	got := rules.UnreviewedTransfers(txs, rulesList)

	if len(got) != 0 {
		t.Errorf("got %v, want empty — Al Pocket is matched by rule", got)
	}
}

func TestUnreviewedTransfers_ExcludesExpensesAndIncome(t *testing.T) {
	txs := []domain.Transaction{
		tx("Mercadona", domain.Expense),
		tx("Salary", domain.Income),
		tx("Bizum", domain.Transfer),
	}

	got := rules.UnreviewedTransfers(txs, nil)

	if len(got) != 1 || got[0].Description != "Bizum" {
		t.Errorf("got %v, want only [Bizum]", got)
	}
}
