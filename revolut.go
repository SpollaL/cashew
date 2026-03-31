package main

import (
	"encoding/csv"
	"strconv"
	"os"
	"time"
)

func ParseRevolutTransactions(filePath string) ([]Transaction, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	_, err = reader.Read() // Skip header
	if err != nil {
		return nil, err
	}
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	transactions := make([]Transaction, 0, len(rows))
	for _, row := range rows {
		amount, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			return nil, err
		}
		fee, err := strconv.ParseFloat(row[6], 64)
		if err != nil {
			return nil, err
		}
		date, err := time.Parse("2006-01-02 15:04:05", row[2])
		if err != nil {
			return nil, err
		}
		transaction := Transaction{
			Type:        TransactionType(row[0]),
			Date:        date,
			Description: row[4],
			Amount:      amount,
			Fee:         fee,
			Currency:    row[7],
			Bank:        "Revolut",
		}
		transactions = append(transactions, transaction)
	}
	return transactions, nil
}
