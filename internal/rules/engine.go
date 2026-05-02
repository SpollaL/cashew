package rules

import (
	"regexp"
	"strings"

	"github.com/SpollaL/cashew/internal/domain"
)

var spaceRe = regexp.MustCompile(`\s+`)

func normalizeDescription(s string) string {
	return spaceRe.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), " ")
}

// compiledRule wraps a domain.Rule with precompiled regexps for performance.
// Invalid regex patterns are silently skipped — ValidateRules catches them at load time.
type compiledRule struct {
	domain.Rule
	regexps []*regexp.Regexp
}

func compileRules(rules []domain.Rule) []compiledRule {
	out := make([]compiledRule, len(rules))
	for i, r := range rules {
		cr := compiledRule{Rule: r}
		if r.Regex {
			for _, p := range r.AllPatterns() {
				if re, err := regexp.Compile(p); err == nil {
					cr.regexps = append(cr.regexps, re)
				}
			}
		}
		out[i] = cr
	}
	return out
}

// matches reports whether the compiled rule matches the normalized description.
func (cr compiledRule) matches(norm string) bool {
	if cr.Regex {
		for _, re := range cr.regexps {
			if re.MatchString(norm) {
				return true
			}
		}
		return false
	}
	for _, p := range cr.AllPatterns() {
		if strings.Contains(norm, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// Apply runs all rules against each transaction in a single pass.
// A rule can set Type, Category, or both. First match per field wins.
// Descriptions are normalized before matching (lowercase, collapsed whitespace).
func Apply(txs []domain.Transaction, rules []domain.Rule) []domain.Transaction {
	compiled := compileRules(rules)
	result := make([]domain.Transaction, len(txs))
	copy(result, txs)

	for i := range result {
		tx := &result[i]
		norm := normalizeDescription(tx.Description)
		typeSet := false
		categorySet := false

		for _, cr := range compiled {
			if !cr.matches(norm) {
				continue
			}
			if !typeSet && cr.Type != "" {
				tx.Type = cr.Type
				typeSet = true
			}
			if !categorySet && cr.Category != "" && tx.Type == domain.Expense {
				tx.Category = cr.Category
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
// acknowledged by any rule.
func Uncategorised(txs []domain.Transaction, rulesList []domain.Rule) []domain.Transaction {
	compiled := compileRules(rulesList)
	seen := map[string]bool{}
	var out []domain.Transaction
	for _, tx := range txs {
		if tx.Type == domain.Expense && tx.Category == "" && !seen[tx.Description] {
			if !matchedByAnyCompiled(normalizeDescription(tx.Description), compiled) {
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
	compiled := compileRules(rulesList)
	seen := map[string]bool{}
	var out []domain.Transaction
	for _, tx := range txs {
		if tx.Type == domain.Transfer && !seen[tx.Description] {
			if !matchedByAnyCompiled(normalizeDescription(tx.Description), compiled) {
				seen[tx.Description] = true
				out = append(out, tx)
			}
		}
	}
	return out
}

func matchedByAnyCompiled(norm string, compiled []compiledRule) bool {
	for _, cr := range compiled {
		if cr.matches(norm) {
			return true
		}
	}
	return false
}
