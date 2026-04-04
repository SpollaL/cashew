package domain

import "time"

type TransactionType string

const (
	Income     TransactionType = "income"
	Expense    TransactionType = "expense"
	Investment TransactionType = "investment"
	Transfer   TransactionType = "transfer"
	Unknown    TransactionType = "unknown"
)

type Transaction struct {
	Date        time.Time
	Description string
	Amount      float64 // always positive; Type encodes direction
	Currency    string
	Type        TransactionType
	Category    string
	Bank        string
}
