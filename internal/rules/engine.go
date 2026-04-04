package rules

import (
	"cashew/internal/domain"
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
	}
	return result
}

// Uncategorised returns unique descriptions of expense transactions
// that have no category assigned.
func Uncategorised(txs []domain.Transaction) []string {
	seen := map[string]bool{}
	var out []string
	for _, tx := range txs {
		if tx.Type == domain.Expense && tx.Category == "" && !seen[tx.Description] {
			seen[tx.Description] = true
			out = append(out, tx.Description)
		}
	}
	return out
}

func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}
