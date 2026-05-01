package llm_test

import (
	"testing"

	"github.com/SpollaL/cashew/internal/llm"
)

func TestCategoryRule_Fields(t *testing.T) {
	r := llm.CategoryRule{
		Pattern:  "^salary",
		Patterns: []string{"lidl", "aldi"},
		Regex:    true,
		Category: "Groceries",
	}
	if r.Pattern != "^salary" {
		t.Errorf("Pattern = %q", r.Pattern)
	}
	if len(r.Patterns) != 2 {
		t.Errorf("Patterns = %v", r.Patterns)
	}
	if !r.Regex {
		t.Error("Regex = false, want true")
	}
}
