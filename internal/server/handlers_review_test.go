package server_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/rules"
	"github.com/SpollaL/cashew/internal/server"
)

func newTestServer(t *testing.T) (*server.Server, string) {
	t.Helper()
	dir := t.TempDir()
	rulesPath := filepath.Join(dir, "rules.toml")
	if err := os.WriteFile(rulesPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	txs := []domain.Transaction{
		{Description: "AMAZON PURCHASE", Amount: -10.0, Type: domain.Expense},
	}
	s := server.New(txs, nil, []string{"shopping", "food"}, rulesPath)
	return s, rulesPath
}

func TestHandleReviewPost_PlainPattern(t *testing.T) {
	s, rulesPath := newTestServer(t)

	form := url.Values{
		"pattern":  {"amazon"},
		"category": {"shopping"},
		"type":     {""},
	}
	req := httptest.NewRequest(http.MethodPost, "/review", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("want 303, got %d", w.Code)
	}

	rulesList, _, err := rules.Load(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rulesList) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rulesList))
	}
	r := rulesList[0]
	if r.Pattern != "amazon" {
		t.Errorf("want pattern 'amazon', got %q", r.Pattern)
	}
	if r.Regex {
		t.Error("want regex=false for plain pattern")
	}
}

func TestHandleReviewPost_RegexOn(t *testing.T) {
	s, rulesPath := newTestServer(t)

	form := url.Values{
		"pattern":  {"^amazon"},
		"category": {"shopping"},
		"type":     {""},
		"regex":    {"on"},
	}
	req := httptest.NewRequest(http.MethodPost, "/review", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("want 303, got %d", w.Code)
	}

	rulesList, _, err := rules.Load(rulesPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(rulesList) != 1 {
		t.Fatalf("want 1 rule, got %d", len(rulesList))
	}
	r := rulesList[0]
	if !r.Regex {
		t.Error("want regex=true when checkbox is on")
	}
	if r.Pattern != "^amazon" {
		t.Errorf("want pattern '^amazon', got %q", r.Pattern)
	}
}

func TestHandleReviewPost_MissingPattern(t *testing.T) {
	s, _ := newTestServer(t)

	form := url.Values{"category": {"shopping"}}
	req := httptest.NewRequest(http.MethodPost, "/review", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
