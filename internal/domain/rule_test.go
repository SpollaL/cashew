package domain_test

import (
	"testing"

	"github.com/SpollaL/cashew/internal/domain"
)

func TestRuleAllPatterns_SinglePattern(t *testing.T) {
	r := domain.Rule{Pattern: "Mercadona"}
	got := r.AllPatterns()
	if len(got) != 1 || got[0] != "Mercadona" {
		t.Fatalf("AllPatterns() = %v, want [Mercadona]", got)
	}
}

func TestRuleAllPatterns_MultiPatterns(t *testing.T) {
	r := domain.Rule{Patterns: []string{"Lidl", "Aldi"}}
	got := r.AllPatterns()
	if len(got) != 2 || got[0] != "Lidl" || got[1] != "Aldi" {
		t.Fatalf("AllPatterns() = %v, want [Lidl Aldi]", got)
	}
}

func TestRuleAllPatterns_Both(t *testing.T) {
	r := domain.Rule{Pattern: "Mercadona", Patterns: []string{"Lidl"}}
	got := r.AllPatterns()
	if len(got) != 2 {
		t.Fatalf("AllPatterns() = %v, want 2 patterns", got)
	}
}

func TestRuleAllPatterns_Empty(t *testing.T) {
	r := domain.Rule{}
	got := r.AllPatterns()
	if len(got) != 0 {
		t.Fatalf("AllPatterns() = %v, want empty", got)
	}
}
