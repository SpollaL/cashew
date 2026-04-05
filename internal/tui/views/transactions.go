package views

import (
	"cashew/internal/domain"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var editTypes = []domain.TransactionType{
	domain.Expense,
	domain.Income,
	domain.Investment,
	domain.Transfer,
}

type TransactionsModel struct {
	all           []domain.Transaction
	buckets       []string
	filter        txFilter
	fromDrillDown bool // esc goes straight back instead of clearing filter first
	cursor        int  // selected row (absolute index into filtered())
	offset        int  // first visible row
	height        int
	editing       bool
	searching     bool // free-text search input active
	editSection   int  // 0 = type, 1 = category
	editCursor    int  // index into editTypes
	editCatCursor int  // 0 = (none), 1..N = buckets index
}

type txFilter struct {
	open        bool
	txType      string
	category    string
	periodLabel string
	dateFrom    time.Time
	dateTo      time.Time
	section     int
	typeCursor  int
	catCursor   int
	text        string // free-text search on description
}

var allTypes = []string{"all", "expense", "income", "investment", "transfer"}

func NewTransactions(txs []domain.Transaction, buckets []string) TransactionsModel {
	sorted := make([]domain.Transaction, len(txs))
	copy(sorted, txs)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Date.After(sorted[j].Date)
	})
	return TransactionsModel{all: sorted, buckets: buckets}
}

func (m TransactionsModel) SetSize(height int) TransactionsModel {
	m.height = height
	return m
}

func (m TransactionsModel) Height() int { return m.height }

func (m TransactionsModel) SetFilter(category string) TransactionsModel {
	m.filter = txFilter{category: category}
	m.fromDrillDown = true
	m.cursor = 0
	m.offset = 0
	return m
}

func (m TransactionsModel) SetCellFilter(category string, p domain.Period) TransactionsModel {
	m.filter = txFilter{
		category:    category,
		periodLabel: p.Label,
		dateFrom:    p.Start,
		dateTo:      p.End,
	}
	m.fromDrillDown = true
	m.cursor = 0
	m.offset = 0
	return m
}

func (m TransactionsModel) ClearFilter() TransactionsModel {
	m.filter = txFilter{}
	m.searching = false
	m.cursor = 0
	m.offset = 0
	return m
}

func (m TransactionsModel) Update(msg tea.Msg) (TransactionsModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.editing {
		return m.updateEditing(keyMsg)
	}
	if m.searching {
		return m.updateSearching(keyMsg)
	}
	if m.filter.open {
		return m.updateFilter(keyMsg)
	}
	return m.updateBrowsing(keyMsg)
}

func (m TransactionsModel) updateBrowsing(msg tea.KeyMsg) (TransactionsModel, tea.Cmd) {
	filtered := m.filtered()
	pageSize := m.pageSize()

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(filtered)-1 {
			m.cursor++
			if m.cursor >= m.offset+pageSize {
				m.offset = m.cursor - pageSize + 1
			}
		}
	case "pgup":
		m.cursor -= pageSize
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.offset = m.cursor
	case "pgdown":
		last := len(filtered) - 1
		m.cursor += pageSize
		if m.cursor > last {
			m.cursor = last
		}
		if m.cursor >= m.offset+pageSize {
			m.offset = m.cursor - pageSize + 1
		}
	case "g":
		m.cursor = 0
		m.offset = 0
	case "G":
		m.cursor = len(filtered) - 1
		m.offset = m.cursor - pageSize + 1
		if m.offset < 0 {
			m.offset = 0
		}
	case "e":
		if len(filtered) > 0 {
			tx := filtered[m.cursor]
			m.editing = true
			m.editSection = 0
			m.editCursor = indexOfTxType(tx.Type)
			m.editCatCursor = indexOfCat(tx.Category, m.buckets)
		}
	case "/":
		m.searching = true
	case "f":
		m.filter.open = true
		m.filter.section = 0
		m.filter.typeCursor = indexOfType(m.filter.txType)
		m.filter.catCursor = indexOfCat(m.filter.category, m.buckets)
	case "esc":
		if m.fromDrillDown {
			return m, func() tea.Msg { return GoBackMsg{} }
		}
		if m.filter.txType != "" || m.filter.category != "" || !m.filter.dateFrom.IsZero() || m.filter.periodLabel != "" {
			m.filter = txFilter{}
			m.cursor = 0
			m.offset = 0
		} else {
			return m, func() tea.Msg { return GoBackMsg{} }
		}
	}
	return m, nil
}

func (m TransactionsModel) updateEditing(msg tea.KeyMsg) (TransactionsModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.editSection = 1 - m.editSection
	case "up", "k":
		if m.editSection == 0 {
			if m.editCursor > 0 {
				m.editCursor--
			}
		} else {
			if m.editCatCursor > 0 {
				m.editCatCursor--
			}
		}
	case "down", "j":
		if m.editSection == 0 {
			if m.editCursor < len(editTypes)-1 {
				m.editCursor++
			}
		} else {
			if m.editCatCursor < len(m.buckets) {
				m.editCatCursor++
			}
		}
	case "enter":
		filtered := m.filtered()
		if m.cursor < len(filtered) {
			pattern := filtered[m.cursor].Description
			newType := editTypes[m.editCursor]
			var newCategory string
			if m.editCatCursor > 0 {
				newCategory = m.buckets[m.editCatCursor-1]
			}
			m.editing = false
			return m, func() tea.Msg {
				return RuleSavedMsg{Pattern: pattern, Type: newType, Category: newCategory}
			}
		}
	case "esc":
		m.editing = false
	}
	return m, nil
}

func (m TransactionsModel) updateFilter(msg tea.KeyMsg) (TransactionsModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.filter.section = 1 - m.filter.section
	case "up", "k":
		if m.filter.section == 0 {
			if m.filter.typeCursor > 0 {
				m.filter.typeCursor--
			}
		} else {
			if m.filter.catCursor > 0 {
				m.filter.catCursor--
			}
		}
	case "down", "j":
		if m.filter.section == 0 {
			if m.filter.typeCursor < len(allTypes)-1 {
				m.filter.typeCursor++
			}
		} else {
			if m.filter.catCursor < len(m.buckets) {
				m.filter.catCursor++
			}
		}
	case "enter":
		if m.filter.typeCursor == 0 {
			m.filter.txType = ""
		} else {
			m.filter.txType = allTypes[m.filter.typeCursor]
		}
		if m.filter.catCursor == 0 {
			m.filter.category = ""
		} else {
			m.filter.category = m.buckets[m.filter.catCursor-1]
		}
		m.filter.open = false
		m.cursor = 0
		m.offset = 0
	case "esc":
		m.filter.open = false
	}
	return m, nil
}

func (m TransactionsModel) updateSearching(msg tea.KeyMsg) (TransactionsModel, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.searching = false
		m.cursor = 0
		m.offset = 0
	case "backspace", "ctrl+h":
		if len(m.filter.text) > 0 {
			m.filter.text = m.filter.text[:len(m.filter.text)-1]
			m.cursor = 0
			m.offset = 0
		}
	default:
		// Accept printable single characters
		if len(msg.String()) == 1 {
			m.filter.text += msg.String()
			m.cursor = 0
			m.offset = 0
		}
	}
	return m, nil
}

func (m TransactionsModel) View() string {
	var sb strings.Builder
	filtered := m.filtered()
	bold := lipgloss.NewStyle().Bold(true)

	filterDesc := m.filterDescription()
	fmt.Fprintf(&sb, "\n  %s  %s  (%d shown)\n\n",
		bold.Render("Transactions"), filterDesc, len(filtered))

	if m.searching {
		cursor := lipgloss.NewStyle().Bold(true).Render("█")
		fmt.Fprintf(&sb, "  Search: %s%s\n\n", m.filter.text, cursor)
	}

	if m.filter.open {
		sb.WriteString(m.renderFilterPanel())
		return sb.String()
	}

	if m.editing {
		sb.WriteString(m.renderEditPanel(filtered))
		return sb.String()
	}

	// Column headers
	fmt.Fprintf(&sb, "  %-12s  %-36s  %10s  %-6s  %-10s  %-12s  %s\n",
		"Date", "Description", "Amount", "Curr", "Bank", "Type", "Category")
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", 110))

	start := m.offset
	end := start + m.pageSize()
	if end > len(filtered) {
		end = len(filtered)
	}

	selectedStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))

	for i, tx := range filtered[start:end] {
		absIdx := start + i
		isSelected := absIdx == m.cursor

		desc := tx.Description
		if len(desc) > 36 {
			desc = desc[:33] + "..."
		}

		var amountColor lipgloss.Color
		switch tx.Type {
		case domain.Income:
			amountColor = "2"
		case domain.Expense:
			amountColor = "1"
		case domain.Investment:
			amountColor = "4"
		default:
			amountColor = "7"
		}
		amount := lipgloss.NewStyle().Foreground(amountColor).Render(fmt.Sprintf("%10.2f", tx.Amount))

		prefix := "  "
		if isSelected {
			prefix = "> "
		}
		row := fmt.Sprintf("%s%s  %-36s  %s  %-6s  %-10s  %-12s  %s",
			prefix, tx.Date.Format("2006-01-02"), desc, amount, tx.Currency, tx.Bank, tx.Type, tx.Category)
		if isSelected {
			row = selectedStyle.Render(row)
		}
		sb.WriteString(row + "\n")
	}

	if len(filtered) == 0 {
		sb.WriteString("  No transactions match the current filter.\n")
	}

	// Totals
	var income, expenses, investments float64
	for _, tx := range filtered {
		switch tx.Type {
		case domain.Income:
			income += tx.Amount
		case domain.Expense:
			expenses += tx.Amount
		case domain.Investment:
			investments += tx.Amount
		}
	}
	net := income - expenses - investments
	netColor := lipgloss.Color("2")
	if net < 0 {
		netColor = "1"
	}
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", 110))
	fmt.Fprintf(&sb,
		"  %-50s  %s  %s  %s  net %s\n",
		bold.Render("Totals"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("in %8.2f", income)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("out %8.2f", expenses)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render(fmt.Sprintf("inv %8.2f", investments)),
		lipgloss.NewStyle().Foreground(netColor).Bold(true).Render(fmt.Sprintf("%8.2f", net)),
	)

	fmt.Fprintf(&sb, "\n  %d–%d of %d   ↑/↓ navigate   e edit   f filter   / search   esc clear / back\n",
		start+1, end, len(filtered))
	sb.WriteString(globalHint("transactions") + "\n")

	return sb.String()
}

func (m TransactionsModel) renderEditPanel(filtered []domain.Transaction) string {
	var sb strings.Builder

	if m.cursor < len(filtered) {
		tx := filtered[m.cursor]
		desc := tx.Description
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		bold := lipgloss.NewStyle().Bold(true)
		faint := lipgloss.NewStyle().Faint(true)
		var amountColor lipgloss.Color
		switch tx.Type {
		case domain.Income:
			amountColor = "2"
		case domain.Expense:
			amountColor = "1"
		case domain.Investment:
			amountColor = "4"
		default:
			amountColor = "7"
		}
		amount := lipgloss.NewStyle().Foreground(amountColor).Render(fmt.Sprintf("%.2f %s", tx.Amount, tx.Currency))
		fmt.Fprintf(&sb, "  %s\n", bold.Render(desc))
		fmt.Fprintf(&sb, "  %s   %s   %s   %s\n\n",
			amount,
			faint.Render(tx.Date.Format("2006-01-02")),
			faint.Render(string(tx.Bank)),
			faint.Render(string(tx.Type)),
		)
	}

	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cursorStyle := lipgloss.NewStyle().Bold(true)

	sb.WriteString("  ┌─ Edit ──────────────────────────────────┐\n")

	// Type section
	typeHeader := "  │  Type"
	if m.editSection == 0 {
		typeHeader = "  │  " + activeStyle.Render("Type")
	}
	sb.WriteString(typeHeader + "\n")
	for i, t := range editTypes {
		marker := "   "
		if i == m.editCursor && m.editSection == 0 {
			marker = " > "
		}
		label := string(t)
		if i == m.editCursor {
			label = cursorStyle.Render(label)
		}
		fmt.Fprintf(&sb, "  │  %s%s\n", marker, label)
	}

	sb.WriteString("  │\n")

	// Category section
	catHeader := "  │  Category"
	if m.editSection == 1 {
		catHeader = "  │  " + activeStyle.Render("Category")
	}
	sb.WriteString(catHeader + "\n")

	catOptions := append([]string{"(none)"}, m.buckets...)
	for i, cat := range catOptions {
		marker := "   "
		if i == m.editCatCursor && m.editSection == 1 {
			marker = " > "
		}
		label := cat
		if i == m.editCatCursor {
			label = cursorStyle.Render(label)
		}
		fmt.Fprintf(&sb, "  │  %s%s\n", marker, label)
	}

	sb.WriteString("  │\n")
	sb.WriteString("  │  tab switch   enter apply   esc cancel\n")
	sb.WriteString("  └─────────────────────────────────────────┘\n")

	return sb.String()
}

func (m TransactionsModel) renderFilterPanel() string {
	var sb strings.Builder

	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cursorStyle := lipgloss.NewStyle().Bold(true)

	sb.WriteString("  ┌─ Filter ───────────────────────────────┐\n")

	typeHeader := "  │  Type"
	if m.filter.section == 0 {
		typeHeader = "  │  " + activeStyle.Render("Type")
	}
	sb.WriteString(typeHeader + "\n")
	for i, t := range allTypes {
		marker := "   "
		if i == m.filter.typeCursor && m.filter.section == 0 {
			marker = " > "
		}
		label := t
		if i == m.filter.typeCursor {
			label = cursorStyle.Render(label)
		}
		fmt.Fprintf(&sb, "  │  %s%s\n", marker, label)
	}

	sb.WriteString("  │\n")

	catHeader := "  │  Category"
	if m.filter.section == 1 {
		catHeader = "  │  " + activeStyle.Render("Category")
	}
	sb.WriteString(catHeader + "\n")

	catOptions := append([]string{"all"}, m.buckets...)
	for i, cat := range catOptions {
		marker := "   "
		if i == m.filter.catCursor && m.filter.section == 1 {
			marker = " > "
		}
		label := cat
		if i == m.filter.catCursor {
			label = cursorStyle.Render(label)
		}
		fmt.Fprintf(&sb, "  │  %s%s\n", marker, label)
	}

	sb.WriteString("  │\n")
	sb.WriteString("  │  tab switch section   enter apply   esc cancel\n")
	sb.WriteString("  └────────────────────────────────────────┘\n")

	return sb.String()
}

func (m TransactionsModel) filtered() []domain.Transaction {
	f := m.filter
	if f.txType == "" && f.category == "" && f.dateFrom.IsZero() && f.text == "" {
		return m.all
	}
	var out []domain.Transaction
	textLower := strings.ToLower(f.text)
	for _, tx := range m.all {
		if f.txType != "" && string(tx.Type) != f.txType {
			continue
		}
		if f.category != "" && tx.Category != f.category {
			continue
		}
		if !f.dateFrom.IsZero() && tx.Date.Before(f.dateFrom) {
			continue
		}
		if !f.dateTo.IsZero() && !tx.Date.Before(f.dateTo) {
			continue
		}
		if textLower != "" && !strings.Contains(strings.ToLower(tx.Description), textLower) {
			continue
		}
		out = append(out, tx)
	}
	return out
}

func (m TransactionsModel) filterDescription() string {
	var parts []string
	if m.filter.txType != "" {
		parts = append(parts, fmt.Sprintf("[%s]", m.filter.txType))
	}
	if m.filter.category != "" {
		parts = append(parts, fmt.Sprintf("[%s]", m.filter.category))
	}
	if m.filter.periodLabel != "" {
		parts = append(parts, fmt.Sprintf("[%s]", m.filter.periodLabel))
	}
	if m.filter.text != "" {
		parts = append(parts, fmt.Sprintf("[/%s]", m.filter.text))
	}
	if len(parts) == 0 {
		return lipgloss.NewStyle().Faint(true).Render("[all]")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(strings.Join(parts, " "))
}

func (m TransactionsModel) pageSize() int {
	size := m.height - 10
	if size < 5 {
		return 5
	}
	return size
}

func indexOfType(t string) int {
	for i, v := range allTypes {
		if v == t {
			return i
		}
	}
	return 0
}

func indexOfCat(cat string, buckets []string) int {
	if cat == "" {
		return 0
	}
	for i, b := range buckets {
		if b == cat {
			return i + 1
		}
	}
	return 0
}

func indexOfTxType(t domain.TransactionType) int {
	for i, v := range editTypes {
		if v == t {
			return i
		}
	}
	return 0
}
