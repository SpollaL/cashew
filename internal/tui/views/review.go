package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// RuleSavedMsg is emitted when the user picks a category.
// The App handles this by saving the rule and refreshing all state.
type RuleSavedMsg struct {
	Pattern  string
	Category string
}

type ReviewModel struct {
	descriptions   []string
	cursor         int
	buckets        []string
	picking        bool
	categoryCursor int
}

func NewReview(descriptions, buckets []string) ReviewModel {
	return ReviewModel{descriptions: descriptions, buckets: buckets}
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
		}
	case "down", "j":
		if m.cursor < len(m.descriptions)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.descriptions) > 0 {
			m.picking = true
			m.categoryCursor = 0
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

func (m ReviewModel) View() string {
	if len(m.descriptions) == 0 {
		return "\n  All transactions categorised!\n\n  Press 's' to view the summary.\n"
	}

	var left, right strings.Builder

	fmt.Fprintf(&left, "  Uncategorised (%d)\n\n", len(m.descriptions))
	for i, d := range m.descriptions {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		line := cursor + d
		if len(line) > 52 {
			line = line[:49] + "..."
		}
		fmt.Fprintln(&left, line)
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

	hint := "\n  ↑/↓ navigate   enter pick category   s summary   t transactions   q quit"
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
