package server

import (
	"github.com/spolla-l/cashew/internal/domain"
	"github.com/spolla-l/cashew/internal/ledger"
	"github.com/spolla-l/cashew/internal/rules"
	"embed"
	"html/template"
	"net/http"
	"sync"
)

//go:embed templates/*.html
var templateFS embed.FS

// Server holds the shared application state and serves HTTP requests.
type Server struct {
	mu        sync.RWMutex
	rawTxs    []domain.Transaction // pre-rule; never mutated after construction
	ledger    ledger.Ledger
	rulesList []domain.Rule
	buckets   []string
	rulesPath string

	tmpls map[string]*template.Template
	mux   *http.ServeMux
}

func New(rawTxs []domain.Transaction, rulesList []domain.Rule, buckets []string, rulesPath string) *Server {
	applied := rules.Apply(rawTxs, rulesList)

	s := &Server{
		rawTxs:    rawTxs,
		ledger:    ledger.New(applied),
		rulesList: rulesList,
		buckets:   buckets,
		rulesPath: rulesPath,
	}

	s.tmpls = mustParseTemplates()
	s.mux = http.NewServeMux()

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/summary", http.StatusFound)
	})
	s.mux.HandleFunc("/summary", s.handleSummary)
	s.mux.HandleFunc("/categories", s.handleCategories)
	s.mux.HandleFunc("/transactions", s.handleTransactions)
	s.mux.HandleFunc("/review", s.handleReview)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// snapshot returns a consistent read of all shared state.
func (s *Server) snapshot() (ledger.Ledger, []domain.Rule, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ledger, s.rulesList, s.buckets
}

// applyRefresh re-loads rules from disk and rebuilds the ledger.
// Must be called with the write lock held.
func (s *Server) applyRefresh() error {
	rulesList, buckets, err := rules.Load(s.rulesPath)
	if err != nil {
		return err
	}
	s.ledger = ledger.New(rules.Apply(s.rawTxs, rulesList))
	s.rulesList = rulesList
	s.buckets = buckets
	return nil
}

func (s *Server) render(w http.ResponseWriter, page string, data any) {
	tmpl, ok := s.tmpls[page]
	if !ok {
		http.Error(w, "unknown page: "+page, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var funcMap = template.FuncMap{
	// pct returns val/total*100, safe against division by zero.
	"pct": func(val, total float64) float64 {
		if total == 0 {
			return 0
		}
		return val / total * 100
	},
}

func mustParseTemplates() map[string]*template.Template {
	pages := []string{"summary", "categories", "transactions", "review"}
	tmpls := make(map[string]*template.Template, len(pages))
	for _, page := range pages {
		t := template.Must(
			template.New("base").Funcs(funcMap).ParseFS(templateFS,
				"templates/base.html",
				"templates/"+page+".html",
			),
		)
		tmpls[page] = t
	}
	return tmpls
}

// granularityFromString maps a URL param value to a domain.Granularity.
// Unknown values default to Monthly.
func granularityFromString(s string) domain.Granularity {
	switch s {
	case "daily":
		return domain.Daily
	case "weekly":
		return domain.Weekly
	case "yearly":
		return domain.Yearly
	case "alltime":
		return domain.AllTime
	default:
		return domain.Monthly
	}
}

// granularityToString returns the URL-safe param value for a granularity.
func granularityToString(g domain.Granularity) string {
	switch g {
	case domain.Daily:
		return "daily"
	case domain.Weekly:
		return "weekly"
	case domain.Yearly:
		return "yearly"
	case domain.AllTime:
		return "alltime"
	default:
		return "monthly"
	}
}
