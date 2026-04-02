package main

import (
	"testing"
	"time"
)

func makeTransaction(description string, category string) Transaction {
	return Transaction{
		Description: description,
		Category:    category,
		Amount:      -10.0,
		Currency:    "EUR",
		Date:        time.Now(),
		Type:        TypeExpense,
		Bank:        "Revolut",
	}
}

func TestBuildReviewQueue(t *testing.T) {
	transactions := []Transaction{
		makeTransaction("Grocery Store", CategoryUncategorized),
		makeTransaction("Grocery Store S.A.", CategoryUncategorized),
		makeTransaction("Coffee Shop", CategoryUncategorized),
		makeTransaction("Online Subscription", CategoryUncategorized),
		makeTransaction("Grocery Store M.C", "Food"),
	}
	reviewQueue := BuildReviewQueue(transactions)
	t.Logf("Review Queue: %+v", reviewQueue)
	if len(reviewQueue) != 2 {
		t.Errorf("Expected 2 review item, got %d", len(reviewQueue))
	}
}
