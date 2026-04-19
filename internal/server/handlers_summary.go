package server

import (
	"github.com/spolla-l/cashew/internal/domain"
	"net/http"
)

type granChoice struct {
	Value  string
	Label  string
	Active bool
}

type summaryRow struct {
	Label       string
	Income      float64
	Expenses    float64
	Investments float64
	Net         float64
}

type summaryData struct {
	ActiveNav     string
	Gran          string
	Granularities []granChoice
	Rows          []summaryRow
}

var allGranularities = []struct {
	value string
	label string
	g     domain.Granularity
}{
	{"alltime", "All time", domain.AllTime},
	{"yearly", "Yearly", domain.Yearly},
	{"monthly", "Monthly", domain.Monthly},
	{"weekly", "Weekly", domain.Weekly},
	{"daily", "Daily", domain.Daily},
}

func (s *Server) handleSummary(w http.ResponseWriter, r *http.Request) {
	l, _, _ := s.snapshot()

	granParam := r.URL.Query().Get("gran")
	if granParam == "" {
		granParam = "monthly"
	}
	g := granularityFromString(granParam)

	summaries := l.Aggregate(g)

	rows := make([]summaryRow, len(summaries))
	for i, sum := range summaries {
		rows[i] = summaryRow{
			Label:       sum.Period.Label,
			Income:      sum.Income,
			Expenses:    sum.Expenses,
			Investments: sum.Investments,
			Net:         sum.Net,
		}
	}

	// Most recent period last → reverse for display (newest first).
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}

	grans := make([]granChoice, len(allGranularities))
	for i, ag := range allGranularities {
		grans[i] = granChoice{Value: ag.value, Label: ag.label, Active: ag.value == granParam}
	}

	s.render(w, "summary", summaryData{
		ActiveNav:     "summary",
		Gran:          granParam,
		Granularities: grans,
		Rows:          rows,
	})
}
