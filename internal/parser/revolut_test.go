package parser_test

import (
	"strings"
	"testing"

	"github.com/spolla-l/cashew/internal/domain"
	"github.com/spolla-l/cashew/internal/parser"
)

func TestRevolutDetect(t *testing.T) {
	r := parser.Revolut{}
	tests := []struct {
		name   string
		header []string
		want   bool
	}{
		{"spanish export", []string{"Tipo", "Producto", "Fecha de inicio"}, true},
		{"english export", []string{"Type", "Product", "Started Date"}, true},
		{"bbva xlsx", nil, false},
		{"unknown csv", []string{"Date", "Amount", "Balance"}, false},
	}
	for _, tc := range tests {
		got := r.Detect("file.csv", tc.header)
		if got != tc.want {
			t.Errorf("Detect(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

var revolutCSV = `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2025-03-01 09:00:00,2025-03-01 09:00:01,Mercadona,-62.40,0.00,EUR,COMPLETED,3137.60
TRANSFER,Current,2025-03-02 10:00:00,2025-03-02 10:00:01,Al Pocket,100.00,0.00,EUR,COMPLETED,3237.60
CARD_PAYMENT,Current,2025-03-03 11:00:00,2025-03-03 11:00:01,Salary March,3200.00,0.00,EUR,COMPLETED,6437.60
`

func TestRevolutParse_Basic(t *testing.T) {
	txs, err := parser.Revolut{}.Parse(strings.NewReader(revolutCSV))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 3 {
		t.Fatalf("got %d transactions, want 3", len(txs))
	}
}

func TestRevolutParse_NegativeAmountIsExpense(t *testing.T) {
	txs, err := parser.Revolut{}.Parse(strings.NewReader(revolutCSV))
	if err != nil {
		t.Fatal(err)
	}
	tx := txs[0]
	if tx.Description != "Mercadona" {
		t.Errorf("description %q, want Mercadona", tx.Description)
	}
	if tx.Amount != 62.40 {
		t.Errorf("amount %v, want 62.40 (always positive)", tx.Amount)
	}
	if tx.Type != domain.Expense {
		t.Errorf("type %q, want Expense", tx.Type)
	}
	if tx.Currency != "EUR" {
		t.Errorf("currency %q, want EUR", tx.Currency)
	}
	if tx.Bank != "Revolut" {
		t.Errorf("bank %q, want Revolut", tx.Bank)
	}
}

func TestRevolutParse_TransferType(t *testing.T) {
	txs, err := parser.Revolut{}.Parse(strings.NewReader(revolutCSV))
	if err != nil {
		t.Fatal(err)
	}
	if txs[1].Type != domain.Transfer {
		t.Errorf("TRANSFER row: got type %q, want Transfer", txs[1].Type)
	}
}

func TestRevolutParse_PositiveAmountIsIncome(t *testing.T) {
	txs, err := parser.Revolut{}.Parse(strings.NewReader(revolutCSV))
	if err != nil {
		t.Fatal(err)
	}
	if txs[2].Type != domain.Income {
		t.Errorf("positive CARD_PAYMENT: got type %q, want Income", txs[2].Type)
	}
}

func TestRevolutParse_EmptyCSV(t *testing.T) {
	csv := "Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n"
	txs, err := parser.Revolut{}.Parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error on empty csv: %v", err)
	}
	if len(txs) != 0 {
		t.Errorf("got %d transactions, want 0", len(txs))
	}
}

func TestRevolutParse_SkipsMalformedRows(t *testing.T) {
	csv := `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,bad-date,2025-03-01,Mercadona,-10.00,0.00,EUR,COMPLETED,100.00
CARD_PAYMENT,Current,2025-03-01 09:00:00,2025-03-01 09:00:01,Netflix,-15.99,0.00,EUR,COMPLETED,84.01
`
	txs, err := parser.Revolut{}.Parse(strings.NewReader(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(txs) != 1 {
		t.Errorf("got %d transactions, want 1 (bad row skipped)", len(txs))
	}
}
