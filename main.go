package main

import "fmt"

func main() {
	transactions, err := ParseRevolutTransactions(
		"account-statement_2026-01-01_2026-03-30_es-es_49993c.csv",
	)
	if err != nil {
		fmt.Printf("Error parsing transactions: %v\n", err)
		return
	}
	summaries := SummarizeTransactions(transactions)
	for _, summary := range summaries {
		fmt.Printf(
			"%d-%02d: Income: %.2f, Expense: %.2f, Investment: %.2f, Net: %.2f\n",
			summary.Year,
			summary.Month,
			summary.TotalIncome,
			summary.TotalExpense,
			summary.TotalInvestment,
			summary.NetAmount,
		)
	}
}
