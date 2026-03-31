package main

import (
	"fmt"
	"sort"
	"time"
)

type MonthlySummary struct {
	Year            int
	Month           time.Month
	TotalIncome     float64
	TotalExpense    float64
	TotalInvestment float64
	NetAmount       float64
}

func SummarizeTransactions(transactions []Transaction) []MonthlySummary {
	groups := make(map[string][]Transaction)
	for _, transaction := range transactions {
		groupsKey := fmt.Sprintf("%d-%02d", transaction.Date.Year(), transaction.Date.Month())
		groups[groupsKey] = append(groups[groupsKey], transaction)
	}
	monthlySummaries := make([]MonthlySummary, 0, len(groups))
	for _, group := range groups {
		totalIncome := 0.0
		totalExpense := 0.0
		totalInvestment := 0.0
		for _, transaction := range group {
			switch transaction.Type {
			case TypeTransfer:
				continue
			case TypeIncome:
				totalIncome += transaction.Amount
			case TypeInvestment:
				totalInvestment += -transaction.Amount
			case TypeExpense:
				totalExpense += -transaction.Amount
				// We assume that expenses are negative amounts, so we negate them to get the total expense.
			}
			if transaction.Fee > 0 {
				totalExpense += transaction.Fee
			}
		}
		monthlySummary := MonthlySummary{
			Year:            group[0].Date.Year(),
			Month:           group[0].Date.Month(),
			TotalIncome:     totalIncome,
			TotalExpense:    totalExpense,
			TotalInvestment: totalInvestment,
			NetAmount:       totalIncome - totalExpense - totalInvestment,
		}
		monthlySummaries = append(monthlySummaries, monthlySummary)
	}
	sort.Slice(monthlySummaries, func(i, j int) bool {
		return monthlySummaries[i].Year < monthlySummaries[j].Year ||
			(monthlySummaries[i].Year == monthlySummaries[j].Year && monthlySummaries[i].Month < monthlySummaries[j].Month)
	})
	return monthlySummaries
}
