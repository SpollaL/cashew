package rules_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/rules"
)

func TestValidateRules_ValidRules(t *testing.T) {
	rulesList := []domain.Rule{
		{Pattern: "Mercadona", Category: "Groceries"},
		{Pattern: `^salary\s`, Regex: true, Type: domain.Income},
		{Patterns: []string{"Lidl", "Aldi"}, Category: "Groceries"},
	}
	if err := rules.ValidateRules(rulesList); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRules_InvalidRegex(t *testing.T) {
	rulesList := []domain.Rule{
		{Pattern: `[invalid`, Regex: true, Category: "Groceries"},
	}
	if err := rules.ValidateRules(rulesList); err == nil {
		t.Fatal("expected error for invalid regex, got nil")
	}
}

func TestValidateRules_NonRegexSkipped(t *testing.T) {
	rulesList := []domain.Rule{
		{Pattern: "[not a regex]", Category: "Groceries"},
	}
	if err := rules.ValidateRules(rulesList); err != nil {
		t.Fatalf("non-regex rule should not be validated: %v", err)
	}
}

func TestLoad_InvalidRegexFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.toml")
	content := `
[categories]
buckets = ["Groceries"]

[[rules]]
pattern = "[invalid"
regex = true
category = "Groceries"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := rules.Load(path); err == nil {
		t.Fatal("expected error for invalid regex in TOML, got nil")
	}
}

func TestSaveAndLoad_MultiplePatterns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.toml")

	r := domain.Rule{Patterns: []string{"Lidl", "Aldi"}, Category: "Groceries"}
	if err := rules.SaveRule(path, r); err != nil {
		t.Fatal(err)
	}

	loaded, _, err := rules.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) == 0 {
		t.Fatal("expected at least one rule")
	}
	got := loaded[0]
	if len(got.Patterns) != 2 || got.Patterns[0] != "Lidl" || got.Patterns[1] != "Aldi" {
		t.Errorf("Patterns = %v, want [Lidl Aldi]", got.Patterns)
	}
	if got.Category != "Groceries" {
		t.Errorf("Category = %q, want Groceries", got.Category)
	}
}

func TestSaveAndLoad_RegexPattern(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.toml")

	r := domain.Rule{Pattern: `^salary\s`, Regex: true, Type: domain.Income}
	if err := rules.SaveRule(path, r); err != nil {
		t.Fatal(err)
	}

	loaded, _, err := rules.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) == 0 {
		t.Fatal("expected at least one rule")
	}
	got := loaded[0]
	if got.Pattern != `^salary\s` {
		t.Errorf("Pattern = %q, want ^salary\\s", got.Pattern)
	}
	if !got.Regex {
		t.Error("Regex = false, want true")
	}
}

func TestSaveAndLoad_BackwardCompat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rules.toml")
	content := `
[categories]
buckets = ["Groceries"]

[[rules]]
pattern = "Mercadona"
category = "Groceries"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	loaded, _, err := rules.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].Pattern != "Mercadona" || loaded[0].Category != "Groceries" {
		t.Errorf("backward compat failed: %+v", loaded)
	}
}
