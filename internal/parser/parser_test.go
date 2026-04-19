package parser_test

import (
	"github.com/spolla-l/cashew/internal/domain"
	"github.com/spolla-l/cashew/internal/parser"
	"testing"
	"time"
)

func makeTx(date, desc string, amount float64, bank string) domain.Transaction {
	t, _ := time.Parse("2006-01-02", date)
	return domain.Transaction{Date: t, Description: desc, Amount: amount, Currency: "EUR", Bank: bank}
}

func TestDeduplicate_RemovesDuplicates(t *testing.T) {
	txs := []domain.Transaction{
		makeTx("2026-01-01", "Mercadona", 50.00, "BBVA"),
		makeTx("2026-01-01", "Mercadona", 50.00, "BBVA"), // duplicate
		makeTx("2026-01-02", "Mercadona", 50.00, "BBVA"), // different date
	}

	got := parser.Deduplicate(txs)

	if len(got) != 2 {
		t.Errorf("got %d transactions, want 2", len(got))
	}
}

func TestDeduplicate_KeepsDifferentAmounts(t *testing.T) {
	txs := []domain.Transaction{
		makeTx("2026-01-01", "Mercadona", 50.00, "BBVA"),
		makeTx("2026-01-01", "Mercadona", 51.00, "BBVA"),
	}

	got := parser.Deduplicate(txs)

	if len(got) != 2 {
		t.Errorf("got %d transactions, want 2 (different amounts)", len(got))
	}
}

func TestDeduplicate_KeepsDifferentBanks(t *testing.T) {
	txs := []domain.Transaction{
		makeTx("2026-01-01", "Transfer", 100.00, "BBVA"),
		makeTx("2026-01-01", "Transfer", 100.00, "Revolut"),
	}

	got := parser.Deduplicate(txs)

	if len(got) != 2 {
		t.Errorf("got %d transactions, want 2 (different banks)", len(got))
	}
}

func TestDeduplicate_PreservesOrder(t *testing.T) {
	txs := []domain.Transaction{
		makeTx("2026-01-03", "C", 10, "BBVA"),
		makeTx("2026-01-01", "A", 10, "BBVA"),
		makeTx("2026-01-02", "B", 10, "BBVA"),
	}

	got := parser.Deduplicate(txs)

	if len(got) != 3 || got[0].Description != "C" {
		t.Errorf("Deduplicate changed order: got %v", got)
	}
}
