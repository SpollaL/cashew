package server

import (
	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/ledger"
	"fmt"
	"net/http"
	"time"
)

type txRow struct {
	Date        string
	Description string
	Amount      string
	Currency    string
	Bank        string
	Type        string
	Category    string
	TypeClass   string
}

type txFilter struct {
	Type     string
	Category string
	From     string
	To       string
}

type txTotals struct {
	Income      float64
	Expenses    float64
	Investments float64
	Net         float64
}

type transactionsData struct {
	ActiveNav string
	Filter    txFilter
	Rows      []txRow
	Totals    txTotals
	Buckets   []string
	Count     int
}

func (s *Server) handleTransactions(w http.ResponseWriter, r *http.Request) {
	l, _, buckets := s.snapshot()

	f := txFilter{
		Type:     r.URL.Query().Get("type"),
		Category: r.URL.Query().Get("category"),
		From:     r.URL.Query().Get("from"),
		To:       r.URL.Query().Get("to"),
	}

	filtered := applyFilter(l, f)
	txs := filtered.All()

	rows := make([]txRow, len(txs))
	var totals txTotals
	for i, tx := range txs {
		rows[i] = txRow{
			Date:        tx.Date.Format("2006-01-02"),
			Description: tx.Description,
			Amount:      fmt.Sprintf("%.2f", tx.Amount),
			Currency:    tx.Currency,
			Bank:        tx.Bank,
			Type:        string(tx.Type),
			Category:    tx.Category,
			TypeClass:   string(tx.Type),
		}
		switch tx.Type {
		case domain.Income:
			totals.Income += tx.Amount
		case domain.Expense:
			totals.Expenses += tx.Amount
		case domain.Investment:
			totals.Investments += tx.Amount
		}
	}
	totals.Net = totals.Income - totals.Expenses - totals.Investments

	s.render(w, "transactions", transactionsData{
		ActiveNav: "transactions",
		Filter:    f,
		Rows:      rows,
		Totals:    totals,
		Buckets:   buckets,
		Count:     len(txs),
	})
}

func applyFilter(l ledger.Ledger, f txFilter) ledger.Ledger {
	switch f.Type {
	case "expense":
		l = l.OnlyExpenses()
	case "income":
		l = l.OnlyIncome()
	case "investment":
		l = l.OnlyInvestments()
	}
	if f.Category != "" {
		l = l.InCategory(f.Category)
	}
	if f.From != "" && f.To != "" {
		from, errF := time.Parse("2006-01-02", f.From)
		to, errT := time.Parse("2006-01-02", f.To)
		if errF == nil && errT == nil {
			l = l.InRange(from, to)
		}
	}
	return l
}
