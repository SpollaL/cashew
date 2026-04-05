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

type ReviewModel struct {
	txs            []domain.Transaction
	cursor         int
	offset         int // first visible row
	height         int
	buckets        []string
	picking        bool
	categoryCursor int
}

func NewReview(txs []domain.Transaction, buckets []string) ReviewModel {
	return ReviewModel{txs: txs, buckets: buckets}
}

func (m ReviewModel) SetSize(height int) ReviewModel {
	m.height = height
	return m
}

func (m ReviewModel) Update(msg tea.Msg) (ReviewModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	if m.picking {
		return m.updatePicking(keyMsg)
	}
	return m.updateBrowsing(keyMsg)
}

func (m ReviewModel) updateBrowsing(msg tea.KeyMsg) (ReviewModel, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.txs)-1 {
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
		last := len(m.txs) - 1
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
		m.cursor = len(m.txs) - 1
		m.offset = m.cursor - visible + 1
		if m.offset < 0 {
			m.offset = 0
		}
	case "enter":
		if len(m.txs) > 0 {
			m.picking = true
			m.categoryCursor = 0
		}
	case "i", "T", "I":
		if len(m.txs) > 0 {
			pattern := m.txs[m.cursor].Description
			txType := map[string]domain.TransactionType{
				"i": domain.Income,
				"T": domain.Transfer,
				"I": domain.Investment,
			}[msg.String()]
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Type: txType} }
		}
	case "n":
		if len(m.txs) > 0 {
			// Save a pattern-only rule to acknowledge this description
			// without assigning a category. It won't appear in review again.
			pattern := m.txs[m.cursor].Description
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern} }
		}
	}
	return m, nil
}

func (m ReviewModel) updatePicking(msg tea.KeyMsg) (ReviewModel, tea.Cmd) {
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
		pattern := m.txs[m.cursor].Description
		category := m.buckets[m.categoryCursor]
		m.picking = false
		return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Category: category} }
	case "esc":
		m.picking = false
	}
	return m, nil
}

func (m ReviewModel) visibleRows() int {
	// Reserve lines: header(3) + hint(2) + some padding
	rows := m.height - 7
	if rows < 5 {
		rows = 5
	}
	return rows
}

func (m ReviewModel) View() string {
	if len(m.txs) == 0 {
		return "\n  All transactions categorised!\n\n  Press 's' to view the summary.\n"
	}

	var left, right strings.Builder

	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.txs) {
		end = len(m.txs)
	}

	fmt.Fprintf(&left, "  Uncategorised (%d)\n\n", len(m.txs))
	for i, tx := range m.txs[m.offset:end] {
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

		amountStr := fmt.Sprintf("%8.2f", tx.Amount)
		dateStr := tx.Date.Format("2006-01-02")

		line := fmt.Sprintf("%s%-38s  %s  %s", cursor, desc, amountStr, dateStr)
		if isSelected {
			line = lipgloss.NewStyle().Bold(true).Render(line)
		}
		fmt.Fprintln(&left, line)
		if isSelected && !m.picking {
			faint := lipgloss.NewStyle().Faint(true)
			fmt.Fprintln(&left, faint.Render("  [enter] expense  [i] income  [T] transfer  [I] invest  [n] skip"))
		}
	}
	if end < len(m.txs) {
		fmt.Fprintf(&left, "  ↓ %d more\n", len(m.txs)-end)
	}

	if m.picking {
		fmt.Fprintf(&right, "  Pick a category\n\n")
		for i, b := range m.buckets {
			cursor := "  "
			if i == m.categoryCursor {
				cursor = "> "
			}
			fmt.Fprintf(&right, "%s%s\n", cursor, b)
		}
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(68).Render(left.String()),
		lipgloss.NewStyle().Width(30).Render(right.String()),
	)

	hint := "\n  ↑/↓ navigate   enter expense   i income   T transfer   I investment   n skip\n" + globalHint("review")
	if m.picking {
		hint = "\n  ↑/↓ navigate   enter confirm   esc cancel"
	}

	return body + hint
}

func (m ReviewModel) SetDescriptions(txs []domain.Transaction) ReviewModel {
	m.txs = txs
	if len(txs) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(txs) {
		m.cursor = len(txs) - 1
	}
	return m
}
