package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/NimbleMarkets/ntcharts/barchart"
)

type ruleConfig struct {
	Categories struct {
		Buckets []string `toml:"buckets"`
	} `toml:"categories"`
	TypeRules     []TypeRule     `toml:"type_rules"`
	CategoryRules []CategoryRule `toml:"category_rules"`
}

type TypeRule struct {
	Pattern string
	Type    TransactionType
}

type CategoryRule struct {
	Pattern  string
	Category string
}

func ApplyTypeRules(transactions []Transaction, rules []TypeRule) []Transaction {
	for i := range transactions {
		rulesApplied := false
		for _, rule := range rules {
			if strings.Contains(transactions[i].Description, rule.Pattern) {
				transactions[i].Type = rule.Type
				rulesApplied = true
				break
			}
		}
		if !rulesApplied {
			if transactions[i].Amount > 0 {
				transactions[i].Type = TypeIncome
			} else if transactions[i].Amount < 0 {
				transactions[i].Type = TypeExpense
			}
		}
	}
	return transactions
}

func ApplyCategoryRules(transactions []Transaction, rules []CategoryRule) []Transaction {
	for i := range transactions {
		rulesApplied := false
		for _, rule := range rules {
			if strings.Contains(transactions[i].Description, rule.Pattern) {
				transactions[i].Category = rule.Category
				rulesApplied = true
				break
			}
		}
		if !rulesApplied {
			transactions[i].Category = CategoryUncategorized
		}
	}
	return transactions
}

func ApplyClusterInheritance(
	transactions []Transaction,
	uniqueDescription map[string]string,
	clusters [][]string,
) []Transaction {
	clusterCategory := make(map[string][]string)
	for _, cluster := range clusters {
		for _, desc := range cluster {
			if uniqueDescription[desc] != CategoryUncategorized {
				clusterCategory[uniqueDescription[desc]] = cluster
				break
			}
		}
	}
	for category, cluster := range clusterCategory {
		for _, desc := range cluster {
			for transaction := range transactions {
				if transactions[transaction].Description == desc {
					transactions[transaction].Category = category
					break
				}
			}
		}
	}
	return transactions
}

func LoadRules(filePath string) ([]TypeRule, []CategoryRule, []string, error) {
	var config ruleConfig
	_, err := toml.DecodeFile(filePath, &config)
	if err != nil {
		return nil, nil, nil, err
	}
	return config.TypeRules, config.CategoryRules, config.Categories.Buckets, nil
}

func saveCategoryRule(filePath string, pattern string, category string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	fmt.Fprintf(file, "\n[[category_rules]]\n")
	fmt.Fprintf(file, "pattern = \"%s\"\n", pattern)
	fmt.Fprintf(file, "category = \"%s\"\n", category)
	return nil
}

func Refresh(
	filePath string,
	transactions []Transaction,
	clusteredDescriptions [][]string,
	categories []string,
) ([]Transaction, []MonthlySummary, []ReviewItem, barchart.Model, error) {
	typeRules, categoryRules, _, err := LoadRules(filePath)
	if err != nil {
		return nil, nil, nil, barchart.Model{}, err
	}
	transactions = ApplyTypeRules(transactions, typeRules)
	transactions = ApplyCategoryRules(transactions, categoryRules)
	uniqueDescriptions := GetUniqueDescriptions(transactions)
	transactions = ApplyClusterInheritance(transactions, uniqueDescriptions, clusteredDescriptions)
	expensesSummary := SummarizeExpenses(transactions, categories)
	barData := createExpensesBarChartData(expensesSummary)
	barChart := createExpensesBarChart(barData)
	barChart.Draw()
	return transactions, SummarizeTransactions(
			transactions,
		), BuildReviewQueue(
			uniqueDescriptions,
			clusteredDescriptions,
		), 
		barChart,
		nil
}
