package views

import (
	"cashew/internal/domain"
	"cashew/internal/ledger"
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DrillDownMsg is emitted when the user presses Enter on a cell.
// Period is the specific period selected; nil means all time.
type DrillDownMsg struct {
	Category string
	Period   *domain.Period
}

type CategoriesModel struct {
	pivot       categoryPivot
	granularity domain.Granularity
	rowCursor   int
	colCursor   int // absolute index into pivot.periods
	colOffset   int // first visible period column
	width       int
	height      int
}

type categoryPivot struct {
	categories   []string
	periods      []domain.Period
	data         map[string]map[string]float64 // [category][period.Label]
	catTotals    map[string]float64
	periodTotals map[string]float64
	grandTotal   float64
}

func NewCategories(summaries []ledger.Summary, g domain.Granularity) CategoriesModel {
	return CategoriesModel{pivot: buildPivot(summaries), granularity: g}
}

func (m CategoriesModel) SetData(summaries []ledger.Summary, g domain.Granularity) CategoriesModel {
	m.pivot = buildPivot(summaries)
	m.granularity = g
	m.colOffset = 0
	m.colCursor = 0
	if m.rowCursor >= len(m.pivot.categories) {
		m.rowCursor = 0
	}
	return m
}

func (m CategoriesModel) SetSize(width, height int) CategoriesModel {
	m.width = width
	m.height = height
	return m
}

func (m CategoriesModel) Update(msg tea.Msg) (CategoriesModel, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	visible := m.colsVisible()

	switch keyMsg.String() {
	case "up", "k":
		if m.rowCursor > 0 {
			m.rowCursor--
		}
	case "down", "j":
		if m.rowCursor < len(m.pivot.categories)-1 {
			m.rowCursor++
		}
	case "left", "h":
		if m.colCursor > 0 {
			m.colCursor--
			if m.colCursor < m.colOffset {
				m.colOffset = m.colCursor
			}
		}
	case "right", "l":
		if m.colCursor < len(m.pivot.periods)-1 {
			m.colCursor++
			if m.colCursor >= m.colOffset+visible {
				m.colOffset = m.colCursor - visible + 1
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
	case "enter":
		if len(m.pivot.categories) == 0 || len(m.pivot.periods) == 0 {
			break
		}
		cat := m.pivot.categories[m.rowCursor]
		p := m.pivot.periods[m.colCursor] // copy
		return m, func() tea.Msg { return DrillDownMsg{Category: cat, Period: &p} }
	}
	return m, nil
}

func (m CategoriesModel) View() string {
	const labelW = 18
	const colW = 9

	var sb strings.Builder
	bold := lipgloss.NewStyle().Bold(true)
	fmt.Fprintf(&sb, "\n  %s — %s\n\n", bold.Render("Categories"), m.granularity.String())

	if len(m.pivot.categories) == 0 {
		sb.WriteString("  No expense data.\n")
		sb.WriteString("\n  s summary  t transactions  r review  q quit\n")
		return sb.String()
	}

	visible := m.colsVisible()
	maxOffset := len(m.pivot.periods) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	colOffset := m.colOffset
	if colOffset > maxOffset {
		colOffset = maxOffset
	}
	endCol := colOffset + visible
	if endCol > len(m.pivot.periods) {
		endCol = len(m.pivot.periods)
	}
	visiblePeriods := m.pivot.periods[colOffset:endCol]
	nPeriods := len(m.pivot.periods)

	tableWidth := labelW + len(visiblePeriods)*colW + colW*2

	// Column headers — highlight the selected column
	selectedColStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	fmt.Fprintf(&sb, "  %-*s", labelW, "")
	for colIdx, p := range visiblePeriods {
		absIdx := colOffset + colIdx
		label := p.Label
		if len(label) > colW-1 {
			label = label[:colW-1]
		}
		cell := fmt.Sprintf("%*s", colW, label)
		if absIdx == m.colCursor {
			cell = selectedColStyle.Render(cell)
		}
		sb.WriteString(cell)
	}
	fmt.Fprintf(&sb, "%*s%*s\n", colW, "Total", colW, "Avg/p")
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", tableWidth))

	// Max value per visible column, for bold highlighting
	maxInCol := map[string]float64{}
	for _, p := range visiblePeriods {
		for _, cat := range m.pivot.categories {
			if v := m.pivot.data[cat][p.Label]; v > maxInCol[p.Label] {
				maxInCol[p.Label] = v
			}
		}
	}

	rowStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	cellStyle := lipgloss.NewStyle().Reverse(true).Bold(true) // selected cell

	for i, cat := range m.pivot.categories {
		isSelectedRow := i == m.rowCursor

		prefix := "  "
		if isSelectedRow {
			prefix = "> "
		}

		label := cat
		if len(label) > labelW-2 {
			label = label[:labelW-2]
		}

		var row strings.Builder
		labelStr := fmt.Sprintf("%-*s", labelW-2, label)
		if isSelectedRow {
			fmt.Fprintf(&row, "%s%s", prefix, rowStyle.Render(labelStr))
		} else {
			fmt.Fprintf(&row, "%s%s", prefix, labelStr)
		}

		for colIdx, p := range visiblePeriods {
			absIdx := colOffset + colIdx
			v := m.pivot.data[cat][p.Label]
			raw := fmt.Sprintf("%*.0f€", colW-1, v)

			switch {
			case isSelectedRow && absIdx == m.colCursor:
				row.WriteString(cellStyle.Render(raw))
			case isSelectedRow:
				row.WriteString(rowStyle.Render(raw))
			case v > 0 && v == maxInCol[p.Label]:
				row.WriteString(lipgloss.NewStyle().Bold(true).Render(raw))
			default:
				row.WriteString(raw)
			}
		}

		// Total and Avg columns (not selectable)
		total := m.pivot.catTotals[cat]
		avg := 0.0
		if nPeriods > 0 {
			avg = total / float64(nPeriods)
		}
		suffix := fmt.Sprintf("%*.0f€%*.0f€", colW-1, total, colW-1, avg)
		if isSelectedRow {
			suffix = rowStyle.Render(suffix)
		}
		row.WriteString(suffix)

		sb.WriteString(row.String() + "\n")
	}

	// Totals row
	fmt.Fprintf(&sb, "  %s\n", strings.Repeat("─", tableWidth))
	var totalsRow strings.Builder
	fmt.Fprintf(&totalsRow, "  %-*s", labelW, "Total")
	for _, p := range visiblePeriods {
		fmt.Fprintf(&totalsRow, "%*.0f€", colW-1, m.pivot.periodTotals[p.Label])
	}
	avgTotal := 0.0
	if nPeriods > 0 {
		avgTotal = m.pivot.grandTotal / float64(nPeriods)
	}
	fmt.Fprintf(&totalsRow, "%*.0f€%*.0f€", colW-1, m.pivot.grandTotal, colW-1, avgTotal)
	sb.WriteString(bold.Render(totalsRow.String()) + "\n")

	// Status line: show what Enter will drill into
	if len(m.pivot.periods) > 0 {
		cat := m.pivot.categories[m.rowCursor]
		p := m.pivot.periods[m.colCursor]
		status := fmt.Sprintf("selected: %s / %s", cat, p.Label)
		fmt.Fprintf(&sb, "\n  %s\n", lipgloss.NewStyle().Faint(true).Render(status))
	}

	if len(m.pivot.periods) > visible {
		fmt.Fprintf(&sb, "  ←/→ scroll periods (%d–%d of %d)\n", colOffset+1, endCol, nPeriods)
	}

	sb.WriteString("\n  ↑/↓/←/→ navigate   enter drill into transactions")
	sb.WriteString("   a y M w d granularity   s summary   t transactions   r review   q quit\n")
	return sb.String()
}

func (m CategoriesModel) colsVisible() int {
	const labelW = 18
	const colW = 9
	const fixedCols = 2
	available := m.width - labelW - fixedCols*colW - 2
	if available < colW {
		return 1
	}
	return available / colW
}

func buildPivot(summaries []ledger.Summary) categoryPivot {
	p := categoryPivot{
		data:         map[string]map[string]float64{},
		catTotals:    map[string]float64{},
		periodTotals: map[string]float64{},
	}

	catSet := map[string]bool{}
	for _, s := range summaries {
		p.periods = append(p.periods, s.Period)
		for cat, amount := range s.ByCategory {
			if p.data[cat] == nil {
				p.data[cat] = map[string]float64{}
			}
			p.data[cat][s.Period.Label] += amount
			p.catTotals[cat] += amount
			p.periodTotals[s.Period.Label] += amount
			p.grandTotal += amount
			catSet[cat] = true
		}
	}

	for cat := range catSet {
		p.categories = append(p.categories, cat)
	}
	sort.Slice(p.categories, func(i, j int) bool {
		return p.catTotals[p.categories[i]] > p.catTotals[p.categories[j]]
	})

	return p
}
