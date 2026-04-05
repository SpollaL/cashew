package server

import (
	"cashew/internal/domain"
	"cashew/internal/rules"
	"net/http"
)

type reviewItem struct {
	Description string
}

type reviewData struct {
	ActiveNav string
	Items     []reviewItem
	Buckets   []string
	Count     int
}

func (s *Server) handleReview(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		s.handleReviewPost(w, r)
		return
	}

	l, rulesList, buckets := s.snapshot()
	descs := rules.Uncategorised(l.All(), rulesList)

	items := make([]reviewItem, len(descs))
	for i, d := range descs {
		items[i] = reviewItem{Description: d.Description}
	}

	s.render(w, "review", reviewData{
		ActiveNav: "review",
		Items:     items,
		Buckets:   buckets,
		Count:     len(items),
	})
}

func (s *Server) handleReviewPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form data", http.StatusBadRequest)
		return
	}

	pattern := r.FormValue("pattern")
	if pattern == "" {
		http.Error(w, "pattern is required", http.StatusBadRequest)
		return
	}

	rule := domain.Rule{
		Pattern:  pattern,
		Category: r.FormValue("category"),
		Type:     domain.TransactionType(r.FormValue("type")),
	}

	s.mu.Lock()
	err := rules.SaveRule(s.rulesPath, rule)
	if err == nil {
		err = s.applyRefresh()
	}
	s.mu.Unlock()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/review", http.StatusSeeOther)
}
