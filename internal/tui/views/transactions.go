package views

import (
	"cashew/internal/domain"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TransactionsModel struct {
	all     []domain.Transaction // full unfiltered list
	buckets []string
	filter  txFilter
	offset  int
	height  int
}

type txFilter struct {
	open        bool
	txType      string // "" = all
	category    string // "" = all
	periodLabel string // display only
	dateFrom    time.Time
	dateTo      time.Time
	section     int // 0 = type section, 1 = category section
	typeCursor  int
	catCursor   int
}

var allTypes = []string{"all", "expense", "income", "investment", "transfer"}

func NewTransactions(txs []domain.Transaction, buckets []string) TransactionsModel {
	return TransactionsModel{all: txs, buckets: buckets}
}

func (m TransactionsModel) SetSize(height int) TransactionsModel {
	m.height = height
	return m
}

// SetFilter pre-sets the category filter, used when drilling from Categories row.
func (m TransactionsModel) SetFilter(category string) TransactionsModel {
	m.filter = txFilter{category: category}
	m.offset = 0
	return m
}

// SetCellFilter pre-sets both category and period, used when drilling from a cell.
func (m TransactionsModel) SetCellFilter(category string, p domain.Period) TransactionsModel {
	m.filter = txFilter{
		category:    category,
		periodLabel: p.Label,
		dateFrom:    p.Start,
		dateTo:      p.End,
	}
	m.offset = 0
	return m
}

func (m TransactionsModel) ClearFilter() TransactionsModel {
	m.filter = txFilter{}
	m.offset = 0
	return m
}

func (m TransactionsModel) Update(msg tea.Msg) (TransactionsModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.filter.open {
		return m.updateFilter(keyMsg)
	}

	switch keyMsg.String() {
	case "up", "k":
		if m.offset > 0 {
			m.offset--
		}
	case "down", "j":
		visible := m.filtered()
		if m.offset < len(visible)-m.pageSize() {
			m.offset++
		}
	case "f":
		m.filter.open = true
		// Position cursors to match current filter values.
		m.filter.section = 0
		m.filter.typeCursor = indexOfType(m.filter.txType)
		m.filter.catCursor = indexOfCat(m.filter.category, m.buckets)
	case "esc":
		m.filter = txFilter{}
		m.offset = 0
	}
	return m, nil
}

func (m TransactionsModel) updateFilter(msg tea.KeyMsg) (TransactionsModel, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.filter.section = 1 - m.filter.section // toggle 0/1
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
			maxCat := len(m.buckets) // +1 for "all"
			if m.filter.catCursor < maxCat {
				m.filter.catCursor++
			}
		}
	case "enter":
		// Apply selections.
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
		m.offset = 0
	case "esc":
		m.filter.open = false
	}
	return m, nil
}

func (m TransactionsModel) View() string {
	var sb strings.Builder

	filtered := m.filtered()
	bold := lipgloss.NewStyle().Bold(true)

	// Header
	filterDesc := m.filterDescription()
	fmt.Fprintf(&sb, "\n  %s  %s  (%d shown)\n\n",
		bold.Render("Transactions"), filterDesc, len(filtered))

	if m.filter.open {
		sb.WriteString(m.renderFilterPanel())
		return sb.String()
	}

	// Column headers
	fmt.Fprintf(&sb, "  %-12s  %-36s  %10s  %-6s  %-12s  %s\n",
		"Date", "Description", "Amount", "Curr", "Type", "Category")
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", 98))

	start := m.offset
	end := start + m.pageSize()
	if end > len(filtered) {
		end = len(filtered)
	}

	for _, tx := range filtered[start:end] {
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

		fmt.Fprintf(&sb, "  %s  %-36s  %s  %-6s  %-12s  %s\n",
			tx.Date.Format("2006-01-02"), desc, amount, tx.Currency, tx.Type, tx.Category)
	}

	if len(filtered) == 0 {
		sb.WriteString("  No transactions match the current filter.\n")
	}

	// Totals line — always computed over all filtered rows, not just the visible page.
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
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", 98))
	fmt.Fprintf(&sb,
		"  %-50s  %s  %s  %s  net %s\n",
		bold.Render("Totals"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render(fmt.Sprintf("in %8.2f", income)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render(fmt.Sprintf("out %8.2f", expenses)),
		lipgloss.NewStyle().Foreground(lipgloss.Color("4")).Render(fmt.Sprintf("inv %8.2f", investments)),
		lipgloss.NewStyle().Foreground(netColor).Bold(true).Render(fmt.Sprintf("%8.2f", net)),
	)

	fmt.Fprintf(&sb, "\n  %d–%d of %d   ↑/↓ scroll   f filter   esc clear filter   s summary   c categories   r review   q quit\n",
		start+1, end, len(filtered))

	return sb.String()
}

func (m TransactionsModel) renderFilterPanel() string {
	var sb strings.Builder

	activeStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cursorStyle := lipgloss.NewStyle().Bold(true)

	sb.WriteString("  ┌─ Filter ───────────────────────────────┐\n")

	// Type section
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
		if t == "" || t == "all" {
			label = "all"
		}
		if i == m.filter.typeCursor {
			label = cursorStyle.Render(label)
		}
		fmt.Fprintf(&sb, "  │  %s%s\n", marker, label)
	}

	sb.WriteString("  │\n")

	// Category section
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
	noFilter := f.txType == "" && f.category == "" && f.dateFrom.IsZero()
	if noFilter {
		return m.all
	}
	var out []domain.Transaction
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
	if len(parts) == 0 {
		return lipgloss.NewStyle().Faint(true).Render("[all]")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Render(strings.Join(parts, " "))
}

func (m TransactionsModel) pageSize() int {
	size := m.height - 8
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
			return i + 1 // +1 because index 0 is "all"
		}
	}
	return 0
}
