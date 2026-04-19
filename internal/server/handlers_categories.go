package server

import (
	"github.com/SpollaL/cashew/internal/ledger"
	"fmt"
	"net/http"
	"net/url"
	"sort"
)

type categoryRow struct {
	Name  string
	Total float64
	Link  string
}

type categoriesData struct {
	ActiveNav   string
	Gran        string
	PeriodLabel string
	PrevLink    string
	NextLink    string
	Rows        []categoryRow
	GrandTotal  float64
}

func (s *Server) handleCategories(w http.ResponseWriter, r *http.Request) {
	l, _, _ := s.snapshot()

	granParam := r.URL.Query().Get("gran")
	if granParam == "" {
		granParam = "monthly"
	}
	g := granularityFromString(granParam)

	summaries := l.Aggregate(g)
	if len(summaries) == 0 {
		s.render(w, "categories", categoriesData{
			ActiveNav:   "categories",
			Gran:        granParam,
			PeriodLabel: "—",
		})
		return
	}

	// Find the requested period or default to most recent.
	periodLabel := r.URL.Query().Get("period")
	selectedIdx := len(summaries) - 1
	for i, sum := range summaries {
		if sum.Period.Label == periodLabel {
			selectedIdx = i
			break
		}
	}
	selected := summaries[selectedIdx]

	// Build category rows sorted by total descending.
	var rows []categoryRow
	var grandTotal float64
	for cat, total := range selected.ByCategory {
		link := fmt.Sprintf("/transactions?category=%s&from=%s&to=%s",
			url.QueryEscape(cat),
			selected.Period.Start.Format("2006-01-02"),
			selected.Period.End.Format("2006-01-02"),
		)
		rows = append(rows, categoryRow{Name: cat, Total: total, Link: link})
		grandTotal += total
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Total > rows[j].Total })

	prevLink, nextLink := buildPrevNext(summaries, selectedIdx, granParam)

	s.render(w, "categories", categoriesData{
		ActiveNav:   "categories",
		Gran:        granParam,
		PeriodLabel: selected.Period.Label,
		PrevLink:    prevLink,
		NextLink:    nextLink,
		Rows:        rows,
		GrandTotal:  grandTotal,
	})
}

func buildPrevNext(summaries []ledger.Summary, idx int, granParam string) (prev, next string) {
	base := fmt.Sprintf("/categories?gran=%s&period=", granParam)
	if idx > 0 {
		prev = base + summaries[idx-1].Period.Label
	}
	if idx < len(summaries)-1 {
		next = base + summaries[idx+1].Period.Label
	}
	return
}
