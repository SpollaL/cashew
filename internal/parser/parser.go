package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"cashew/internal/domain"
)

// Parser knows how to read one bank's CSV format.
type Parser interface {
	Name() string
	Detect(header []string) bool
	Parse(r io.Reader) ([]domain.Transaction, error)
}

// Load opens a CSV file, finds the right parser, and returns transactions.
func Load(path string, parsers []Parser) ([]domain.Transaction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	// Read just the header row to detect the bank.
	cr := csv.NewReader(f)
	cr.LazyQuotes = true
	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header from %s: %w", path, err)
	}

	var matched Parser
	for _, p := range parsers {
		if p.Detect(header) {
			matched = p
			break
		}
	}
	if matched == nil {
		return nil, fmt.Errorf("no parser recognised the file: %s", path)
	}

	// Reopen so the parser sees the full file including the header.
	f.Seek(0, io.SeekStart)
	txs, err := matched.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("parse %s (%s): %w", path, matched.Name(), err)
	}
	return txs, nil
}
