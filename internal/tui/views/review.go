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
	descriptions   []string
	cursor         int
	offset         int // first visible row
	height         int
	buckets        []string
	picking        bool
	categoryCursor int
}

func NewReview(descriptions, buckets []string) ReviewModel {
	return ReviewModel{descriptions: descriptions, buckets: buckets}
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
		if m.cursor < len(m.descriptions)-1 {
			m.cursor++
			if visible := m.visibleRows(); m.cursor >= m.offset+visible {
				m.offset = m.cursor - visible + 1
			}
		}
	case "enter":
		if len(m.descriptions) > 0 {
			m.picking = true
			m.categoryCursor = 0
		}
	case "i", "x", "v":
		if len(m.descriptions) > 0 {
			pattern := m.descriptions[m.cursor]
			txType := map[string]domain.TransactionType{
				"i": domain.Income,
				"x": domain.Transfer,
				"v": domain.Investment,
			}[msg.String()]
			return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Type: txType} }
		}
	case "n":
		if len(m.descriptions) > 0 {
			// Save a pattern-only rule to acknowledge this description
			// without assigning a category. It won't appear in review again.
			pattern := m.descriptions[m.cursor]
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
		pattern := m.descriptions[m.cursor]
		category := m.buckets[m.categoryCursor]
		m.picking = false
		return m, func() tea.Msg { return RuleSavedMsg{Pattern: pattern, Category: category} }
	case "esc":
		m.picking = false
	}
	return m, nil
}

func (m ReviewModel) visibleRows() int {
	// Reserve lines: header(2) + hint(2) + some padding
	rows := m.height - 6
	if rows < 5 {
		rows = 5
	}
	return rows
}

func (m ReviewModel) View() string {
	if len(m.descriptions) == 0 {
		return "\n  All transactions categorised!\n\n  Press 's' to view the summary.\n"
	}

	var left, right strings.Builder

	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.descriptions) {
		end = len(m.descriptions)
	}

	fmt.Fprintf(&left, "  Uncategorised (%d)\n\n", len(m.descriptions))
	for i, d := range m.descriptions[m.offset:end] {
		abs := m.offset + i
		cursor := "  "
		if abs == m.cursor {
			cursor = "> "
		}
		line := cursor + d
		if len(line) > 52 {
			line = line[:49] + "..."
		}
		fmt.Fprintln(&left, line)
	}
	if end < len(m.descriptions) {
		fmt.Fprintf(&left, "  ↓ %d more\n", len(m.descriptions)-end)
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
		lipgloss.NewStyle().Width(56).Render(left.String()),
		lipgloss.NewStyle().Width(30).Render(right.String()),
	)

	hint := "\n  ↑/↓ navigate   enter category   i income   x transfer   v investment   n no category\n" + globalHint()
	if m.picking {
		hint = "\n  ↑/↓ navigate   enter confirm   esc cancel"
	}

	return body + hint
}

func (m ReviewModel) SetDescriptions(descriptions []string) ReviewModel {
	m.descriptions = descriptions
	if len(descriptions) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(descriptions) {
		m.cursor = len(descriptions) - 1
	}
	return m
}
