package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	monthlySummaries []MonthlySummary
	err              error
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	var result strings.Builder
	result.WriteString("Monthly Summary:\n")
	for _, monthlySummary := range m.monthlySummaries {
		row := fmt.Sprintf(
			"%d %s: in %.2f, out %.2f, invested %.2f, net %.2f\n",
			monthlySummary.Year,
			monthlySummary.Month,
			monthlySummary.TotalIncome,
			monthlySummary.TotalExpense,
			monthlySummary.TotalInvestment,
			monthlySummary.NetAmount,
		)
		result.WriteString(row)
	}
	return result.String()
}
