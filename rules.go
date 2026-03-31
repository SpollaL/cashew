package main

import (
	"strings"

	"github.com/BurntSushi/toml"
)

type ruleConfig struct {
	Rules []Rule `toml:"rules"`
}

type Rule struct {
	Pattern string
	Type    TransactionType
}

func ApplyRules(transactions []Transaction, rules []Rule) []Transaction {
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

func LoadRules(filePath string) ([]Rule, error) {
	var config ruleConfig
	_, err := toml.DecodeFile(filePath, &config)
	if err != nil {
		return nil, err
	}
	return config.Rules, nil
}
