package views

import (
	"cashew/internal/domain"
	"cashew/internal/ledger"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GranularityChangedMsg is emitted when the user presses a granularity key.
// The App handles re-aggregation so views stay data-free.
type GranularityChangedMsg struct {
	Granularity domain.Granularity
}

type SummaryModel struct {
	summaries   []ledger.Summary
	granularity domain.Granularity
	cursor      int
	offset      int
	height      int
}

func NewSummary(summaries []ledger.Summary, g domain.Granularity) SummaryModel {
	return SummaryModel{summaries: summaries, granularity: g}
}

func (m SummaryModel) SetSize(height int) SummaryModel {
	m.height = height
	return m
}

func (m SummaryModel) SetData(summaries []ledger.Summary, g domain.Granularity) SummaryModel {
	m.summaries = summaries
	m.granularity = g
	m.offset = 0
	m.cursor = 0
	return m
}

func (m SummaryModel) Update(msg tea.Msg) (SummaryModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
	case "down", "j":
		if m.cursor < len(m.summaries)-1 {
			m.cursor++
			if m.cursor >= m.offset+m.pageSize() {
				m.offset++
			}
		}
	case "a":
		return m, func() tea.Msg { return GranularityChangedMsg{domain.AllTime} }
	case "y":
		return m, func() tea.Msg { return GranularityChangedMsg{domain.Yearly} }
	case "M":
		return m, func() tea.Msg { return GranularityChangedMsg{domain.Monthly} }
	case "w":
		return m, func() tea.Msg { return GranularityChangedMsg{domain.Weekly} }
	case "d":
		return m, func() tea.Msg { return GranularityChangedMsg{domain.Daily} }
	}
	return m, nil
}

func (m SummaryModel) View() string {
	var sb strings.Builder

	bold := lipgloss.NewStyle().Bold(true)
	fmt.Fprintf(&sb, "\n  %s — %s\n\n", bold.Render("Summary"), m.granularity.String())
	fmt.Fprintf(&sb, "  %-12s  %10s  %10s  %12s  %10s\n", "Period", "Income", "Expenses", "Investments", "Net")
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", 62))

	start := m.offset
	end := start + m.pageSize()
	if end > len(m.summaries) {
		end = len(m.summaries)
	}

	for i, s := range m.summaries[start:end] {
		idx := start + i
		netColor := lipgloss.Color("2")
		if s.Net < 0 {
			netColor = "1"
		}
		net := lipgloss.NewStyle().Foreground(netColor).Render(fmt.Sprintf("%10.2f", s.Net))

		prefix := "  "
		rowStyle := lipgloss.NewStyle()
		if idx == m.cursor {
			prefix = "> "
			rowStyle = rowStyle.Bold(true)
		}

		row := fmt.Sprintf("%s%-12s  %10.2f  %10.2f  %12.2f  %s",
			prefix, s.Period.Label, s.Income, s.Expenses, s.Investments, net)
		sb.WriteString(rowStyle.Render(row) + "\n")
	}

	if len(m.summaries) > m.pageSize() {
		fmt.Fprintf(&sb, "\n  row %d/%d  ↑/↓ scroll\n", m.cursor+1, len(m.summaries))
	}

	sb.WriteString("\n  a all-time  y yearly  M monthly  w weekly  d daily")
	sb.WriteString("   c categories  t transactions  r review  q quit\n")
	return sb.String()
}

func (m SummaryModel) pageSize() int {
	size := m.height - 10
	if size < 3 {
		return 3
	}
	return size
}
