package parser

import (
	"cashew/internal/domain"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Parser knows how to read one bank's file format.
type Parser interface {
	Name() string
	// Detect returns true if this parser owns the file.
	// path is the file path (useful for extension checks).
	// header is the first CSV row; nil for non-CSV formats.
	Detect(path string, header []string) bool
	Parse(r io.Reader) ([]domain.Transaction, error)
}

// Load opens a file, finds the right parser, and returns transactions.
func Load(path string, parsers []Parser) ([]domain.Transaction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// For CSV files, read the header row for content-based detection.
	// For other formats (xlsx etc), pass nil — parsers detect by extension.
	var header []string
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".csv" {
		cr := csv.NewReader(f)
		cr.LazyQuotes = true
		header, err = cr.Read()
		if err != nil {
			return nil, fmt.Errorf("read header from %s: %w", path, err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek %s: %w", path, err)
		}
	}

	var matched Parser
	for _, p := range parsers {
		if p.Detect(path, header) {
			matched = p
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("no parser recognised the file: %s", path)
	}

	txs, err := matched.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse %s (%s): %w", path, matched.Name(), err)
	}
	return txs, nil
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// Deduplicate removes transactions with identical date+description+amount+currency+bank.
// This handles overlapping date ranges when loading multiple files from the same bank.
func Deduplicate(txs []domain.Transaction) []domain.Transaction {
	seen := map[string]bool{}
	out := make([]domain.Transaction, 0, len(txs))
	for _, tx := range txs {
		key := fmt.Sprintf("%s|%s|%.2f|%s|%s",
			tx.Date.Format("2006-01-02"), tx.Description, tx.Amount, tx.Currency, tx.Bank)
		if !seen[key] {
			seen[key] = true
			out = append(out, tx)
		}
	}
	return out
}
