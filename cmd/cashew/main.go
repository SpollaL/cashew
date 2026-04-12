package main

import (
	"cashew/internal/domain"
	"cashew/internal/ledger"
	"cashew/internal/parser"
	"cashew/internal/rules"
	"cashew/internal/tui"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	model := flag.String("model", "gemma3", "Ollama model to use for chat (e.g. gemma3, llama3.2)")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: cashew [-model <name>] <file> [file2 ...]")
		os.Exit(1)
	}

	parsers := []parser.Parser{
		parser.Revolut{},
		parser.BBVA{},
	}
	rulesPath := "rules.toml"

	var allTxs []domain.Transaction
	for _, path := range flag.Args() {
		txs, err := parser.Load(path, parsers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error loading %s: %v\n", path, err)
			os.Exit(1)
		}
		allTxs = append(allTxs, txs...)
	}

	allTxs = parser.Deduplicate(allTxs)

	rulesList, buckets, err := rules.Load(rulesPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading rules: %v\n", err)
		os.Exit(1)
	}

	allTxs = rules.Apply(allTxs, rulesList)
	l := ledger.New(allTxs)

	app := tui.New(l, rulesList, buckets, rulesPath, *model)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
