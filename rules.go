package main

import (
	"strings"

	"github.com/BurntSushi/toml"
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

func LoadRules(filePath string) ([]TypeRule, []CategoryRule, []string, error) {
	var config ruleConfig
	_, err := toml.DecodeFile(filePath, &config)
	if err != nil {
		return nil, nil, nil, err
	}
	return config.TypeRules, config.CategoryRules, config.Categories.Buckets, nil
}
