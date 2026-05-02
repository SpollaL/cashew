package tui

import (
	"fmt"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/SpollaL/cashew/internal/domain"
	"github.com/SpollaL/cashew/internal/ledger"
	"github.com/SpollaL/cashew/internal/llm"
	"github.com/SpollaL/cashew/internal/rules"
	"github.com/SpollaL/cashew/internal/tui/views"
)

// sharedState is heap-allocated and shared across all App copies (value receiver
// methods capture App by value, but the pointer stays live). LLM goroutine and
// TUI goroutine both touch it; mu protects rulesList and buckets.
type sharedState struct {
	mu        sync.RWMutex
	rawTxs    []domain.Transaction // pre-Apply snapshot; immutable after init
	rulesList []domain.Rule
	buckets   []string
	rulesPath string // immutable after init
	prog      *tea.Program
}

type viewKey int

const (
	viewSummary viewKey = iota
	viewCategories
	viewTransactions
	viewReview
	viewChat
)

type App struct {
	fullLedger  ledger.Ledger
	rulesList   []domain.Rule
	rulesPath   string
	buckets     []string
	granularity domain.Granularity
	state       *sharedState

	active       viewKey
	prevView     viewKey
	summary      views.SummaryModel
	categories   views.CategoriesModel
	transactions views.TransactionsModel
	review       views.ReviewModel
	chat         views.ChatModel

	width  int
	height int
}

func New(l ledger.Ledger, rawTxs []domain.Transaction, rulesList []domain.Rule, buckets []string, rulesPath, model string, debug bool) App {
	txs := l.All()
	uncategorised := rules.Uncategorised(txs, rulesList)
	unreviewed := rules.UnreviewedTransfers(txs, rulesList)

	g := domain.Monthly
	summaries := l.Aggregate(g)

	active := viewSummary
	if len(uncategorised) > 0 || len(unreviewed) > 0 {
		active = viewReview
	}

	state := &sharedState{
		rawTxs:    rawTxs,
		rulesList: rulesList,
		buckets:   buckets,
		rulesPath: rulesPath,
	}

	app := App{
		fullLedger:   l,
		rulesList:    rulesList,
		rulesPath:    rulesPath,
		buckets:      buckets,
		granularity:  g,
		active:       active,
		state:        state,
		summary:      views.NewSummary(summaries, g),
		categories:   views.NewCategories(summaries, g),
		transactions: views.NewTransactions(txs, buckets),
		review:       views.NewReview(uncategorised, unreviewed, buckets),
	}

	llmClient := llm.NewClient(model, llm.Tools{
		GetUncategorizedTransactions: app.getUncategorizedTransactions,
		GetTransactions:              app.getTransactions,
		GetMonthlySummary:            app.getMonthlySummary,
		GetCategories:                app.getCategories,
		SaveCategoryRule:             app.saveCategoryRuleSync,
		BulkSaveCategoryRules:        app.bulkSaveCategoryRules,
	}, debug)
	app.chat = views.NewChat(llmClient, debug)

	return app
}

// SetProgram wires up the tea.Program so LLM saves can push refreshMsg into
// the TUI event loop. Call this after tea.NewProgram, before p.Run().
func (a App) SetProgram(p *tea.Program) {
	a.state.prog = p
}

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.summary = a.summary.SetSize(msg.Height)
		a.categories = a.categories.SetSize(msg.Width, msg.Height)
		a.transactions = a.transactions.SetSize(msg.Height)
		a.review = a.review.SetSize(msg.Height)
		a.chat = a.chat.SetSize(msg.Width, msg.Height)
		return a, nil

	case tea.KeyMsg:
		// ctrl+c always quits, regardless of which view is active.
		if msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
		// When chat is active, nav keys must reach the text input — only esc
		// escapes back to the previous view.
		if a.active == viewChat {
			if msg.String() == "esc" {
				a.active = a.prevView
				return a, nil
			}
			break
		}
		switch msg.String() {
		case "q":
			return a, tea.Quit
		case "s":
			a.active = viewSummary
			return a, nil
		case "c":
			a.active = viewCategories
			return a, nil
		case "t":
			a.active = viewTransactions
			a.transactions = a.transactions.ClearFilter()
			return a, nil
		case "r":
			a.active = viewReview
			return a, nil
		case "a":
			a.prevView = a.active
			a.active = viewChat
			return a, nil
		}

	case views.GranularityChangedMsg:
		a.granularity = msg.Granularity
		summaries := a.fullLedger.Aggregate(a.granularity)
		a.summary = a.summary.SetData(summaries, a.granularity)
		a.categories = a.categories.SetData(summaries, a.granularity)
		return a, nil

	case views.DrillDownMsg:
		if msg.Period != nil {
			a.transactions = a.transactions.SetCellFilter(msg.Category, *msg.Period)
		} else {
			a.transactions = a.transactions.SetFilter(msg.Category)
		}
		a.prevView = a.active
		a.active = viewTransactions
		return a, nil

	case views.GoBackMsg:
		a.active = a.prevView
		return a, nil

	case views.RuleSavedMsg:
		return a, a.saveAndRefresh(msg)

	case refreshMsg:
		return a.applyRefresh(msg), nil

	case error:
		return a, tea.Quit
	}

	// Delegate to active view.
	var cmd tea.Cmd
	switch a.active {
	case viewSummary:
		a.summary, cmd = a.summary.Update(msg)
	case viewCategories:
		a.categories, cmd = a.categories.Update(msg)
	case viewTransactions:
		a.transactions, cmd = a.transactions.Update(msg)
	case viewReview:
		a.review, cmd = a.review.Update(msg)
	case viewChat:
		a.chat, cmd = a.chat.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	switch a.active {
	case viewSummary:
		return a.summary.View()
	case viewCategories:
		return a.categories.View()
	case viewTransactions:
		return a.transactions.View()
	case viewReview:
		return a.review.View()
	case viewChat:
		return a.chat.View()
	}
	return ""
}

func (a App) saveAndRefresh(msg views.RuleSavedMsg) tea.Cmd {
	return func() tea.Msg {
		r := domain.Rule{
			Pattern:  msg.Pattern,
			Category: msg.Category,
			Type:     msg.Type,
		}
		if err := rules.SaveRule(a.rulesPath, r); err != nil {
			return err
		}
		rulesList, buckets, err := rules.Load(a.rulesPath)
		if err != nil {
			return err
		}
		return refreshMsg{rulesList: rulesList, buckets: buckets}
	}
}

type refreshMsg struct {
	rulesList []domain.Rule
	buckets   []string
}

// ── LLM tool callbacks ────────────────────────────────────────────────────────

const uncategorizedBatchSize = 20

func (a App) getUncategorizedTransactions(offset int) []string {
	a.state.mu.RLock()
	rawTxs := a.state.rawTxs
	rulesList := a.state.rulesList
	a.state.mu.RUnlock()

	applied := rules.Apply(rawTxs, rulesList)
	var all []string
	for _, tx := range applied {
		if tx.Category == "" {
			all = append(all, fmt.Sprintf("%s | %8.2f %s | %s",
				tx.Date.Format("2006-01-02"),
				tx.Amount, tx.Currency,
				tx.Description,
			))
		}
	}
	total := len(all)
	if offset >= total {
		return []string{"No more uncategorized transactions."}
	}
	end := offset + uncategorizedBatchSize
	if end > total {
		end = total
	}
	batch := make([]string, end-offset+1)
	copy(batch, all[offset:end])
	remaining := total - end
	if remaining > 0 {
		batch[len(batch)-1] = fmt.Sprintf("[%d-%d of %d shown — call again with offset=%d for the next batch]", offset+1, end, total, end)
	} else {
		batch[len(batch)-1] = fmt.Sprintf("[%d-%d of %d shown — this is the last batch]", offset+1, end, total)
	}
	return batch
}

func (a App) getTransactions(filters llm.TransactionFilters) []string {
	a.state.mu.RLock()
	rawTxs := a.state.rawTxs
	rulesList := a.state.rulesList
	a.state.mu.RUnlock()

	l := ledger.New(rules.Apply(rawTxs, rulesList))
	if filters.Type != "" {
		switch filters.Type {
		case "expense":
			l = l.OnlyExpenses()
		case "income":
			l = l.OnlyIncome()
		case "investment":
			l = l.OnlyInvestments()
		}
	}
	if filters.Category != "" {
		l = l.InCategory(filters.Category)
	}
	if filters.Month != "" {
		start, err := time.Parse("2006-01", filters.Month)
		if err == nil {
			end := start.AddDate(0, 1, 0)
			l = l.InRange(start, end)
		}
	}

	txs := l.All()
	const maxRows = 50
	if len(txs) > maxRows {
		txs = txs[len(txs)-maxRows:]
	}
	rows := make([]string, len(txs))
	for i, tx := range txs {
		rows[i] = fmt.Sprintf("%s | %8.2f %s | %-30s | %-12s | %s",
			tx.Date.Format("2006-01-02"),
			tx.Amount, tx.Currency,
			tx.Description,
			string(tx.Type),
			tx.Category,
		)
	}
	return rows
}

func (a App) getMonthlySummary(months []string) []string {
	a.state.mu.RLock()
	rawTxs := a.state.rawTxs
	rulesList := a.state.rulesList
	a.state.mu.RUnlock()

	summaries := ledger.New(rules.Apply(rawTxs, rulesList)).Aggregate(domain.Monthly)
	want := make(map[string]bool, len(months))
	for _, m := range months {
		want[m] = true
	}
	var rows []string
	for _, s := range summaries {
		if len(want) > 0 && !want[s.Period.Label] {
			continue
		}
		rows = append(rows, fmt.Sprintf("%-10s  income: %8.2f  expenses: %8.2f  investments: %8.2f  net: %8.2f",
			s.Period.Label, s.Income, s.Expenses, s.Investments, s.Net,
		))
	}
	return rows
}

func (a App) getCategories() []string {
	a.state.mu.RLock()
	defer a.state.mu.RUnlock()
	return a.state.buckets
}

func (a App) saveCategoryRuleSync(pattern, category string) error {
	r := domain.Rule{Pattern: pattern, Category: category, Type: domain.Expense}
	if err := rules.SaveRule(a.state.rulesPath, r); err != nil {
		return err
	}
	return a.reloadAndRefresh()
}

func (a App) bulkSaveCategoryRules(categoryRules []llm.CategoryRule) (int, error) {
	for i, cr := range categoryRules {
		r := domain.Rule{
			Pattern:  cr.Pattern,
			Patterns: cr.Patterns,
			Regex:    cr.Regex,
			Category: cr.Category,
			Type:     domain.Expense,
		}
		if err := rules.SaveRule(a.state.rulesPath, r); err != nil {
			return i, err
		}
	}
	return len(categoryRules), a.reloadAndRefresh()
}

func (a App) reloadAndRefresh() error {
	rulesList, buckets, err := rules.Load(a.state.rulesPath)
	if err != nil {
		return err
	}
	a.state.mu.Lock()
	a.state.rulesList = rulesList
	a.state.buckets = buckets
	a.state.mu.Unlock()
	if a.state.prog != nil {
		a.state.prog.Send(refreshMsg{rulesList: rulesList, buckets: buckets})
	}
	return nil
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func (a App) applyRefresh(msg refreshMsg) App {
	a.state.mu.Lock()
	a.state.rulesList = msg.rulesList
	a.state.buckets = msg.buckets
	a.state.mu.Unlock()

	txs := rules.Apply(a.fullLedger.All(), msg.rulesList)
	a.fullLedger = ledger.New(txs)
	a.rulesList = msg.rulesList
	a.buckets = msg.buckets

	summaries := a.fullLedger.Aggregate(a.granularity)
	a.summary = a.summary.SetData(summaries, a.granularity)
	a.categories = a.categories.SetData(summaries, a.granularity)
	a.transactions = a.transactions.WithData(txs, msg.buckets)
	a.review = a.review.SetQueues(rules.Uncategorised(txs, msg.rulesList), rules.UnreviewedTransfers(txs, msg.rulesList))
	return a
}
