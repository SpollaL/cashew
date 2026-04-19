package parser

import (
	"testing"

	"github.com/spolla-l/cashew/internal/domain"
)

func TestBBVADescription(t *testing.T) {
	tests := []struct {
		concepto   string
		movimiento string
		want       string
	}{
		// card payment: concepto is merchant, movimiento is generic
		{"Mercadona", "Pago con tarjeta", "Mercadona"},
		// transfer: concepto is generic, movimiento is the real description
		{"Transferencia realizada", "Al Pocket", "Al Pocket"},
		// both non-generic: prefer concepto
		{"Salary Inc", "Company Payment", "Salary Inc"},
		// both generic: fall back to concepto when non-empty
		{"Pago con tarjeta", "Transferencia realizada", "Pago con tarjeta"},
		// concepto empty, movimiento is the only option
		{"", "Some merchant", "Some merchant"},
	}
	for _, tc := range tests {
		got := bbvaDescription(tc.concepto, tc.movimiento)
		if got != tc.want {
			t.Errorf("bbvaDescription(%q, %q) = %q, want %q",
				tc.concepto, tc.movimiento, got, tc.want)
		}
	}
}

func TestBBVAType_KeywordIncome(t *testing.T) {
	tests := []struct {
		concepto   string
		movimiento string
		amount     float64
		want       domain.TransactionType
	}{
		{"Abono nómina", "", -1000, domain.Income},
		{"", "Transferencia recibida", -500, domain.Income},
		{"Nomina empresa", "", 2000, domain.Income},
		{"abono", "", -100, domain.Income},
	}
	for _, tc := range tests {
		got := bbvaType(tc.concepto, tc.movimiento, tc.amount)
		if got != tc.want {
			t.Errorf("bbvaType(%q, %q, %v) = %q, want %q",
				tc.concepto, tc.movimiento, tc.amount, got, tc.want)
		}
	}
}

func TestBBVAType_AmountFallback(t *testing.T) {
	if got := bbvaType("Mercadona", "Pago con tarjeta", -62.40); got != domain.Expense {
		t.Errorf("negative amount expense: got %q, want Expense", got)
	}
	if got := bbvaType("Reembolso", "", 10.00); got != domain.Income {
		t.Errorf("positive amount income: got %q, want Income", got)
	}
}
