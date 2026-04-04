package tui

import (
	"cashew/internal/domain"
	"cashew/internal/ledger"
	"cashew/internal/rules"
	"cashew/internal/tui/views"

	tea "github.com/charmbracelet/bubbletea"
)

type viewKey int

const (
	viewSummary viewKey = iota
	viewCategories
	viewTransactions
	viewReview
)

type App struct {
	fullLedger  ledger.Ledger
	rulesList   []domain.Rule
	rulesPath   string
	buckets     []string
	granularity domain.Granularity

	active       viewKey
	prevView     viewKey
	summary      views.SummaryModel
	categories   views.CategoriesModel
	transactions views.TransactionsModel
	review       views.ReviewModel

	width  int
	height int
}

func New(l ledger.Ledger, rulesList []domain.Rule, buckets []string, rulesPath string) App {
	txs := l.All()
	uncategorised := rules.Uncategorised(txs, rulesList)

	g := domain.Monthly
	summaries := l.Aggregate(g)

	active := viewSummary
	if len(uncategorised) > 0 {
		active = viewReview
	}

	return App{
		fullLedger:   l,
		rulesList:    rulesList,
		rulesPath:    rulesPath,
		buckets:      buckets,
		granularity:  g,
		active:       active,
		summary:      views.NewSummary(summaries, g),
		categories:   views.NewCategories(summaries, g),
		transactions: views.NewTransactions(txs, buckets),
		review:       views.NewReview(uncategorised, buckets),
	}
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
		return a, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
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

func (a App) applyRefresh(msg refreshMsg) App {
	txs := rules.Apply(a.fullLedger.All(), msg.rulesList)
	a.fullLedger = ledger.New(txs)
	a.rulesList = msg.rulesList
	a.buckets = msg.buckets

	summaries := a.fullLedger.Aggregate(a.granularity)
	a.summary = a.summary.SetData(summaries, a.granularity)
	a.categories = a.categories.SetData(summaries, a.granularity)
	a.transactions = views.NewTransactions(txs, msg.buckets).SetSize(a.transactions.Height())
	a.review = a.review.SetDescriptions(rules.Uncategorised(txs, msg.rulesList))
	return a
}
