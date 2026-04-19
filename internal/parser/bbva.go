package parser

import (
	"github.com/SpollaL/cashew/internal/domain"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type BBVA struct{}

func (BBVA) Name() string { return "BBVA" }

// Detect claims .xlsx files (BBVA exports as Excel, not CSV).
func (BBVA) Detect(path string, _ []string) bool {
	return strings.ToLower(filepath.Ext(path)) == ".xlsx"
}

func (BBVA) Parse(r io.Reader) ([]domain.Transaction, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}

	// BBVA xlsx layout:
	// Rows 0–3: metadata (title, generation date, empty lines)
	// Row 4:    headers (F.Valor, Fecha, Concepto, Movimiento, Importe, Divisa, ...)
	// Row 5+:   data
	const dataStartRow = 5

	var txs []domain.Transaction
	skipped := 0
	for i, row := range rows {
		if i < dataStartRow {
			continue
		}
		if len(row) < 6 {
			skipped++
			continue
		}
		tx, err := parseBBVARow(row)
		if err != nil {
			skipped++
			continue
		}
		txs = append(txs, tx)
	}
	if skipped > 0 {
		fmt.Fprintf(os.Stderr, "bbva: skipped %d malformed rows\n", skipped)
	}
	return txs, nil
}

func parseBBVARow(row []string) (domain.Transaction, error) {
	// Column A (index 0) is blank in BBVA's xlsx export. Real data:
	// 1: F.Valor (value date)
	// 2: Fecha (booking date)
	// 3: Concepto — merchant name for card payments, bank category for transfers
	// 4: Movimiento — bank category for card payments, description for transfers
	// 5: Importe (amount, signed)
	// 6: Divisa (currency)
	if len(row) < 7 {
		return domain.Transaction{}, fmt.Errorf("too few columns: %d", len(row))
	}

	dateStr := strings.TrimSpace(row[1])
	if dateStr == "" {
		return domain.Transaction{}, fmt.Errorf("empty date")
	}
	date, err := time.Parse("02/01/2006", dateStr)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse date %q: %w", dateStr, err)
	}

	amountStr := strings.ReplaceAll(strings.TrimSpace(row[5]), ",", ".")
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return domain.Transaction{}, fmt.Errorf("parse amount %q: %w", row[5], err)
	}

	concepto := strings.TrimSpace(row[3])
	movimiento := strings.TrimSpace(row[4])
	description := bbvaDescription(concepto, movimiento)
	if description == "" {
		return domain.Transaction{}, fmt.Errorf("empty description")
	}

	return domain.Transaction{
		Date:        date,
		Description: description,
		Amount:      abs(amount),
		Currency:    strings.TrimSpace(row[6]),
		Type:        bbvaType(concepto, movimiento, amount),
		Bank:        "BBVA",
	}, nil
}

// genericBBVALabels are bank-generated labels that carry no useful merchant info.
// When a field matches one of these, the other field is preferred as the description.
var genericBBVALabels = []string{
	"pago con tarjeta",
	"transferencia realizada",
	"transferencia recibida",
	"domiciliación",
	"abono de nómina",
	"abono nómina",
	"recibo",
}

func isGenericLabel(s string) bool {
	lower := strings.ToLower(s)
	for _, g := range genericBBVALabels {
		if strings.Contains(lower, g) {
			return true
		}
	}
	return false
}

// bbvaDescription picks the most informative description between the two BBVA fields.
// BBVA is inconsistent: for card payments Concepto has the merchant and Movimiento
// has "Pago con tarjeta"; for transfers it's the opposite.
func bbvaDescription(concepto, movimiento string) string {
	concepGeneric := isGenericLabel(concepto)
	moviGeneric := isGenericLabel(movimiento)

	switch {
	case concepGeneric && !moviGeneric:
		return movimiento
	case moviGeneric && !concepGeneric:
		return concepto
	case concepto != "":
		return concepto // both generic or both descriptive: prefer concepto
	default:
		return movimiento
	}
}

func bbvaType(concepto, movimiento string, amount float64) domain.TransactionType {
	combined := strings.ToLower(concepto + " " + movimiento)
	if strings.Contains(combined, "nómina") ||
		strings.Contains(combined, "nomina") ||
		strings.Contains(combined, "abono") ||
		strings.Contains(combined, "transferencia recibida") {
		return domain.Income
	}
	if amount >= 0 {
		return domain.Income
	}
	return domain.Expense
}
