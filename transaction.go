package main

import "time"

type TransactionType string

const (
	TypeIncome		 TransactionType = "income"
	TypeExpense	 TransactionType = "expense"
	TypeInvestment TransactionType = "investment"
	TypeTransfer	 TransactionType = "transfer"
)

const CategoryUncategorized = "Other"

type Transaction struct {
	Date        time.Time
	Description string
	Amount      float64
	Fee         float64
	Currency    string
	Type        TransactionType
	Category    string
	Bank        string
}
