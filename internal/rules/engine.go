package rules

import (
	"github.com/SpollaL/cashew/internal/domain"
	"strings"
)

// Apply runs all rules against each transaction in a single pass.
// A rule can set Type, Category, or both. First match per field wins.
func Apply(txs []domain.Transaction, rules []domain.Rule) []domain.Transaction {
	result := make([]domain.Transaction, len(txs))
	copy(result, txs)

	for i := range result {
		tx := &result[i]
		typeSet := false
		categorySet := false

		for _, r := range rules {
			if !containsFold(tx.Description, r.Pattern) {
				continue
			}
			if !typeSet && r.Type != "" {
				tx.Type = r.Type
				typeSet = true
			}
			if !categorySet && r.Category != "" && tx.Type == domain.Expense {
				tx.Category = r.Category
				categorySet = true
			}
			if typeSet && categorySet {
				break
			}
		}
		if tx.Type != domain.Expense {
			tx.Category = ""
		}
	}
	return result
}

// Uncategorised returns one representative transaction per unique description
// for expense transactions that have no category and are not already
// acknowledged by any rule. This lets the user "skip" a description in the
// review queue by saving a pattern-only rule for it.
func Uncategorised(txs []domain.Transaction, rulesList []domain.Rule) []domain.Transaction {
	seen := map[string]bool{}
	var out []domain.Transaction
	for _, tx := range txs {
		if tx.Type == domain.Expense && tx.Category == "" && !seen[tx.Description] {
			if !matchedByAnyRule(tx.Description, rulesList) {
				seen[tx.Description] = true
				out = append(out, tx)
			}
		}
	}
	return out
}

// UnreviewedTransfers returns one representative transaction per unique description
// for transfer transactions not yet acknowledged by any rule.
func UnreviewedTransfers(txs []domain.Transaction, rulesList []domain.Rule) []domain.Transaction {
	seen := map[string]bool{}
	var out []domain.Transaction
	for _, tx := range txs {
		if tx.Type == domain.Transfer && !seen[tx.Description] {
			if !matchedByAnyRule(tx.Description, rulesList) {
				seen[tx.Description] = true
				out = append(out, tx)
			}
		}
	}
	return out
}

func matchedByAnyRule(description string, rulesList []domain.Rule) bool {
	for _, r := range rulesList {
		if containsFold(description, r.Pattern) {
			return true
		}
	}
	return false
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
