package main

import "time"

type Transaction struct {
	Date        time.Time
	Description string
	Amount      float64
	Fee         float64
	Currency    string
	Type        string
	Category    string
	Bank        string
}
