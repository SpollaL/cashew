package rules

import (
	"cashew/internal/domain"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type config struct {
	Categories struct {
		Buckets []string `toml:"buckets"`
	} `toml:"categories"`
	Rules []tomlRule `toml:"rules"`
}

type tomlRule struct {
	Pattern  string `toml:"pattern"`
	Type     string `toml:"type,omitempty"`
	Category string `toml:"category,omitempty"`
}

var defaultBuckets = []string{
	"Groceries", "Dining", "Transport", "Health",
	"Subscriptions", "Entertainment", "Investments",
}

// Load reads the rules file. If the file doesn't exist it creates one
// with the default category buckets and no rules — ready for the user
// to populate via the review workflow.
func Load(path string) ([]domain.Rule, []string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := write(path, nil, defaultBuckets); err != nil {
			return nil, nil, fmt.Errorf("create %s: %w", path, err)
		}
		return nil, defaultBuckets, nil
	}

	var cfg config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, nil, fmt.Errorf("load rules %s: %w", path, err)
	}

	buckets := cfg.Categories.Buckets
	if len(buckets) == 0 {
		buckets = defaultBuckets
	}

	rules := make([]domain.Rule, len(cfg.Rules))
	for i, r := range cfg.Rules {
		rules[i] = domain.Rule{
			Pattern:  r.Pattern,
			Type:     domain.TransactionType(r.Type),
			Category: r.Category,
		}
	}

	return rules, buckets, nil
}

// SaveCategoryRule appends a new category rule, skipping duplicates.
// Rewrites the full file so there are no stale duplicate entries.
func SaveCategoryRule(path, pattern, category string) error {
	rulesList, buckets, err := Load(path)
	if err != nil {
		return err
	}

	patternLower := strings.ToLower(pattern)
	for _, r := range rulesList {
		if strings.ToLower(r.Pattern) == patternLower && r.Category == category {
			return nil // already exists
		}
	}

	rulesList = append(rulesList, domain.Rule{Pattern: pattern, Category: category})
	return write(path, rulesList, buckets)
}

func write(path string, rulesList []domain.Rule, buckets []string) error {
	var sb strings.Builder

	sb.WriteString("[categories]\nbuckets = [")
	for i, b := range buckets {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%q", b)
	}
	sb.WriteString("]\n")

	for _, r := range rulesList {
		fmt.Fprintf(&sb, "\n[[rules]]\npattern = %q\n", r.Pattern)
		if r.Type != "" {
			fmt.Fprintf(&sb, "type = %q\n", r.Type)
		}
		if r.Category != "" {
			fmt.Fprintf(&sb, "category = %q\n", r.Category)
		}
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
