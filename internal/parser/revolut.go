package parser

import (
	"github.com/spolla-l/cashew/internal/domain"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Revolut struct{}

func (Revolut) Name() string { return "Revolut" }

// Detect claims CSV files whose header contains Revolut's date column
// in either the Spanish ("Fecha de inicio") or English ("Started Date") export.
func (Revolut) Detect(path string, header []string) bool {
	for _, h := range header {
		h = strings.TrimSpace(h)
		if h == "Fecha de inicio" || h == "Started Date" {
			return true
		}
	}
	return false
}

func (Revolut) Parse(r io.Reader) ([]domain.Transaction, error) {
	cr := csv.NewReader(r)
	cr.LazyQuotes = true

	// Skip header row.
	if _, err := cr.Read(); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	var txs []domain.Transaction
	skipped := 0
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		if len(row) < 9 {
			skipped++
			continue
		}
		tx, err := parseRevolutRow(row)
		if err != nil {
			skipped++
			continue
		}
		txs = append(txs, tx)
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "revolut: skipped %d malformed rows\n", skipped)
	}
	return txs, nil
}

func parseRevolutRow(row []string) (domain.Transaction, error) {
	// Columns are identical in es-ES and en-GB exports:
	// 0: Type/Tipo, 1: Product/Producto, 2: Started Date/Fecha de inicio,
	// 3: Completed Date/Fecha de finalización, 4: Description/Descripción,
	// 5: Amount/Importe, 6: Fee/Comisión, 7: Currency/Divisa, 8: State, 9: Balance/Saldo

	date, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(row[2]))
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse date %q: %w", row[2], err)
	}

	amountStr := strings.ReplaceAll(strings.TrimSpace(row[5]), ",", ".")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse amount %q: %w", row[5], err)
	}

	return domain.Transaction{
		Date:        date,
		Description: strings.TrimSpace(row[4]),
		Amount:      abs(amount),
		Currency:    strings.TrimSpace(row[7]),
		Type:        revolutType(strings.TrimSpace(row[0]), amount),
		Bank:        "Revolut",
	}, nil
}

func revolutType(rawType string, amount float64) domain.TransactionType {
	switch strings.ToLower(rawType) {
	case "transfer", "transferir":
		return domain.Transfer
	case "exchange", "intercambio":
		return domain.Transfer
	}
	if amount >= 0 {
		return domain.Income
	}
	return domain.Expense
}

