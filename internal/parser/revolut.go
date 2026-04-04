package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"cashew/internal/domain"
)

type Revolut struct{}

func (Revolut) Name() string { return "Revolut" }

// Detect claims the file if the header contains "Fecha de inicio" (Spanish Revolut export).
func (Revolut) Detect(header []string) bool {
	for _, h := range header {
		if strings.TrimSpace(h) == "Fecha de inicio" {
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
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}
		if len(row) < 9 {
			continue
		}

		tx, err := parseRevolutRow(row)
		if err != nil {
			// Skip malformed rows rather than aborting.
			continue
		}
		txs = append(txs, tx)
	}
	return txs, nil
}

func parseRevolutRow(row []string) (domain.Transaction, error) {
	// Revolut CSV columns (es-ES export):
	// 0: Tipo, 1: Producto, 2: Fecha de inicio, 3: Fecha de finalización,
	// 4: Descripción, 5: Importe, 6: Comisión, 7: Divisa, 8: State, 9: Saldo

	date, err := time.Parse("2006-01-02 15:04:05", strings.TrimSpace(row[2]))
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse date %q: %w", row[2], err)
	}

	amountStr := strings.ReplaceAll(strings.TrimSpace(row[5]), ",", ".")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse amount %q: %w", row[5], err)
	}

	txType := revolutType(strings.TrimSpace(row[0]), amount)

	return domain.Transaction{
		Date:        date,
		Description: strings.TrimSpace(row[4]),
		Amount:      abs(amount),
		Currency:    strings.TrimSpace(row[7]),
		Type:        txType,
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

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
