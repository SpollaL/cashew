package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	transactions, err := ParseRevolutTransactions(
		"account-statement.csv",
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
	uniqueDescriptions := GetUniqueDescriptions(transactions)
	clusteredDescriptions := ClusterTransactions(uniqueDescriptions)
	transactions = ApplyClusterInheritance(transactions, uniqueDescriptions, clusteredDescriptions)
	reviewItems := BuildReviewQueue(uniqueDescriptions, clusteredDescriptions)
	activeView := ViewSummary
	if len(reviewItems) > 0 {
		activeView = ViewReview
	}
	monthlySummary := SummarizeTransactions(transactions)
	expensesSummary := SummarizeExpenses(transactions, categories)
	barData := createExpensesBarChartData(expensesSummary)
	barChart := createExpensesBarChart(barData)
	barChart.Draw()
	p := tea.NewProgram(
		model{
			monthlySummary:   monthlySummary,
			expensesBarChart: barChart,
			transactions:     transactions,
			clusters:         clusteredDescriptions,
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
