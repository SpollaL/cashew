package views

import (
	"cashew/internal/domain"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GoBackMsg is emitted when the user wants to return to the previous view.
type GoBackMsg struct{}

// RuleSavedMsg is emitted when the user assigns a category or type.
// The App saves the rule and refreshes all state.
type RuleSavedMsg struct {
	Pattern  string
	Category string
	Type     domain.TransactionType // empty = don't change type
}

var reviewTypes = []domain.TransactionType{
	domain.Income,
	domain.Expense,
	domain.Transfer,
	domain.Investment,
}

type ReviewModel struct {
	expenses  []domain.Transaction
	transfers []domain.Transaction
	tab       int // 0 = expenses, 1 = transfers

	cursor         int
	offset         int
	height         int
	buckets        []string
	picking        bool // category picker (expenses tab)
	pickingType    bool // type picker (transfers tab)
	categoryCursor int
	typeCursor     int
}

func NewReview(expenses, transfers []domain.Transaction, buckets []string) ReviewModel {
	return ReviewModel{expenses: expenses, transfers: transfers, buckets: buckets}
}

func (m ReviewModel) SetSize(height int) ReviewModel {
	m.height = height
	return m
}

func (m ReviewModel) currentTxs() []domain.Transaction {
	if m.tab == 0 {
		return m.expenses
	}
	return m.transfers
}

func (m ReviewModel) Update(msg tea.Msg) (ReviewModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.picking {
		return m.updatePickingCategory(keyMsg)
	}
	if m.pickingType {
		return m.updatePickingType(keyMsg)
	}
	return m.updateBrowsing(keyMsg)
}

func (m ReviewModel) updateBrowsing(msg tea.KeyMsg) (ReviewModel, tea.Cmd) {
	txs := m.currentTxs()

	switch msg.String() {
	case "tab":
		other := 1 - m.tab
		otherTxs := m.expenses
		if other == 1 {
			otherTxs = m.transfers
		}
		if len(otherTxs) > 0 {
			m.tab = other
			m.cursor = 0
			m.offset = 0
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(txs)-1 {
			m.cursor++
			if visible := m.visibleRows(); m.cursor >= m.offset+visible {
				m.offset = m.cursor - visible + 1
			}
		}
	case "pgup":
		visible := m.visibleRows()
		m.cursor -= visible
		if m.cursor < 0 {
			m.cursor = 0
		}
		m.offset = m.cursor
	case "pgdown":
		visible := m.visibleRows()
		last := len(txs) - 1
		m.cursor += visible
		if m.cursor > last {
			m.cursor = last
		}
		if m.cursor >= m.offset+visible {
			m.offset = m.cursor - visible + 1
		}
	case "g":
		m.cursor = 0
		m.offset = 0
	case "G":
		visible := m.visibleRows()
		m.cursor = len(txs) - 1
		m.offset = m.cursor - visible + 1
		if m.offset < 0 {
			m.offset = 0
		}
	case "enter":
		if len(txs) > 0 {
			if m.tab == 0 {
				m.picking = true
				m.categoryCursor = 0
			} else {
				m.pickingType = true
				m.typeCursor = 0
			}
		}
	case "i":
		if m.tab == 0 && len(txs) > 0 {
			pattern := txs[m.cursor].Description
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Type: domain.Income} }
		}
	case "T":
		if m.tab == 0 && len(txs) > 0 {
			pattern := txs[m.cursor].Description
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Type: domain.Transfer} }
		}
	case "I":
		if m.tab == 0 && len(txs) > 0 {
			pattern := txs[m.cursor].Description
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Type: domain.Investment} }
		}
	case "n":
		if len(txs) > 0 {
			pattern := txs[m.cursor].Description
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern} }
		}
	}
	return m, nil
}

func (m ReviewModel) updatePickingCategory(msg tea.KeyMsg) (ReviewModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.categoryCursor > 0 {
			m.categoryCursor--
		}
	case "down", "j":
		if m.categoryCursor < len(m.buckets)-1 {
			m.categoryCursor++
		}
	case "enter":
		pattern := m.expenses[m.cursor].Description
		category := m.buckets[m.categoryCursor]
		m.picking = false
		return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Category: category} }
	case "esc":
		m.picking = false
	}
	return m, nil
}

func (m ReviewModel) updatePickingType(msg tea.KeyMsg) (ReviewModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.typeCursor > 0 {
			m.typeCursor--
		}
	case "down", "j":
		if m.typeCursor < len(reviewTypes)-1 {
			m.typeCursor++
		}
	case "enter":
		pattern := m.transfers[m.cursor].Description
		t := reviewTypes[m.typeCursor]
		m.pickingType = false
		return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Type: t} }
	case "esc":
		m.pickingType = false
	}
	return m, nil
}

func (m ReviewModel) visibleRows() int {
	rows := m.height - 9 // header(3) + tabs(2) + hint(2) + padding
	if rows < 5 {
		rows = 5
	}
	return rows
}

func (m ReviewModel) View() string {
	allEmpty := len(m.expenses) == 0 && len(m.transfers) == 0
	if allEmpty {
		return "\n  All transactions reviewed!\n\n  Press 's' to view the summary.\n"
	}

	var sb strings.Builder

	// Tab headers
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	inactiveStyle := lipgloss.NewStyle().Faint(true)

	expLabel := fmt.Sprintf("Expenses (%d)", len(m.expenses))
	trLabel := fmt.Sprintf("Transfers (%d)", len(m.transfers))
	if m.tab == 0 {
		expLabel = activeStyle.Render(expLabel)
		trLabel = inactiveStyle.Render(trLabel)
	} else {
		expLabel = inactiveStyle.Render(expLabel)
		trLabel = activeStyle.Render(trLabel)
	}
	fmt.Fprintf(&sb, "\n  %s    %s\n\n", expLabel, trLabel)

	txs := m.currentTxs()
	if len(txs) == 0 {
		sb.WriteString("  Nothing to review here.\n")
	} else {
		visible := m.visibleRows()
		end := m.offset + visible
		if end > len(txs) {
			end = len(txs)
		}

		var left, right strings.Builder
		isPicking := m.picking || m.pickingType
		for i, tx := range txs[m.offset:end] {
			abs := m.offset + i
			isSelected := abs == m.cursor

			cursor := "  "
			if isSelected {
				cursor = "> "
			}

			desc := tx.Description
			if len(desc) > 38 {
				desc = desc[:35] + "..."
			}

			line := fmt.Sprintf("%s%-38s  %8.2f  %s", cursor, desc, tx.Amount, tx.Date.Format("2006-01-02"))
			if isSelected {
				line = lipgloss.NewStyle().Bold(true).Render(line)
			}
			fmt.Fprintln(&left, line)
			if isSelected && !isPicking {
				faint := lipgloss.NewStyle().Faint(true)
				var hint string
				if m.tab == 0 {
					hint = "  [enter] categorize  [i] income  [T] transfer  [I] invest  [n] skip"
				} else {
					hint = "  [enter] pick type  [n] skip"
				}
				fmt.Fprintln(&left, faint.Render(hint))
			}
		}
		if end < len(txs) {
			fmt.Fprintf(&left, "  ↓ %d more\n", len(txs)-end)
		}

		if m.picking {
			fmt.Fprintf(&right, "  Pick a category\n\n")
			for i, b := range m.buckets {
				c := "  "
				if i == m.categoryCursor {
					c = "> "
				}
				fmt.Fprintf(&right, "%s%s\n", c, b)
			}
		} else if m.pickingType {
			fmt.Fprintf(&right, "  Pick a type\n\n")
			for i, t := range reviewTypes {
				c := "  "
				if i == m.typeCursor {
					c = "> "
				}
				fmt.Fprintf(&right, "%s%s\n", c, string(t))
			}
		}

		sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(68).Render(left.String()),
			lipgloss.NewStyle().Width(30).Render(right.String()),
		))
	}

	var hint string
	switch {
	case m.picking:
		hint = "\n  ↑/↓ navigate   enter confirm   esc cancel"
	case m.pickingType:
		hint = "\n  ↑/↓ navigate   enter confirm   esc cancel"
	case m.tab == 0:
		hint = "\n  ↑/↓ navigate   enter categorize   i income   T transfer   I invest   n skip   tab switch"
	default:
		hint = "\n  ↑/↓ navigate   enter pick type   n skip   tab switch"
	}
	sb.WriteString(hint + "\n" + globalHint("review"))

	return sb.String()
}

func (m ReviewModel) SetQueues(expenses, transfers []domain.Transaction) ReviewModel {
	m.expenses = expenses
	m.transfers = transfers

	// If current tab is now empty but the other has items, switch over.
	if len(m.currentTxs()) == 0 {
		other := 1 - m.tab
		var otherTxs []domain.Transaction
		if other == 0 {
			otherTxs = expenses
		} else {
			otherTxs = transfers
		}
		if len(otherTxs) > 0 {
			m.tab = other
			m.cursor = 0
			m.offset = 0
			return m
		}
	}

	// Clamp cursor within current tab's list.
	txs := m.currentTxs()
	if len(txs) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(txs) {
		m.cursor = len(txs) - 1
	}
	return m
}
