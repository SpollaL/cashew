package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	transactions, err := ParseRevolutTransactions(
		"account-statement_2026-01-01_2026-03-30_es-es_49993c.csv",
	)
	if err != nil {
		fmt.Printf("Error parsing transactions: %v\n", err)
		return
	}
	rules, err := LoadRules("rules.toml")
	if err != nil {
		fmt.Printf("Error loading rules: %v\n", err)
		return
	}
	transactions = ApplyRules(transactions, rules)
	summaries := SummarizeTransactions(transactions)
	p := tea.NewProgram(model{monthlySummaries: summaries})
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
