package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type View string

const (
	ViewSummary      View = "summary"
	ViewTransactions View = "transactions"
)

type model struct {
	monthlySummaries []MonthlySummary
	transactions     []Transaction
	activeView       View
	offset           int
	height           int
	err              error
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "s" {
			m.activeView = ViewSummary
			return m, nil
		}
		if msg.String() == "t" {
			m.activeView = ViewTransactions
			return m, nil
		}
		if msg.String() == "up" || msg.String() == "k" {
			m.offset = max(0, m.offset-1)
		}
		if msg.String() == "down" || msg.String() == "j" {
			m.offset = min(max(0, len(m.transactions)-m.height), m.offset+1)
		}
	case tea.WindowSizeMsg:
		m.height = msg.Height
	}
	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	switch m.activeView {
	case ViewSummary:
		return m.renderSummary()
	case ViewTransactions:
		return m.renderTransactions()
	}
	return "Unknown view"
}

func (m model) renderSummary() string {
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
	result.WriteString("\nPress 't' to view transactions, 'q' to quit.")
	return result.String()
}

func (m model) renderTransactions() string {
	var result strings.Builder
	result.WriteString("Transactions:\n")
	for _, transaction := range m.transactions[m.offset:min(m.offset+m.height, len(m.transactions))] {
		row := fmt.Sprintf(
			"%s: %s %.2f %s\n",
			transaction.Date.Format("2006-01-02"),
			transaction.Description,
			transaction.Amount,
			transaction.Currency,
		)
		result.WriteString(row)
	}
	result.WriteString("\nPress 's' to view summary, 'q' to quit.")
	return result.String()
}
