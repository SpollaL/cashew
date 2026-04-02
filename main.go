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
	typeRules, categoryRules, categories, err := LoadRules("rules.toml")
	if err != nil {
		fmt.Printf("Error loading rules: %v\n", err)
		return
	}
	transactions = ApplyTypeRules(transactions, typeRules)
	transactions = ApplyCategoryRules(transactions, categoryRules)
	reviewItems := BuildReviewQueue(transactions)
	activeView := ViewSummary
	if len(reviewItems) > 0 {
		activeView = ViewReview
	}
	summaries := SummarizeTransactions(transactions)
	p := tea.NewProgram(
		model{
			monthlySummaries: summaries,
			transactions:     transactions,
			activeView:       activeView,
			categories:       categories,
			reviewQueue:      reviewItems,
			height:           20,
		},
	)
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
	}
}
