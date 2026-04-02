package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type View string

const (
	ViewReview       View = "review"
	ViewSummary      View = "summary"
	ViewTransactions View = "transactions"
)

type model struct {
	monthlySummaries []MonthlySummary
	transactions     []Transaction
	activeView       View
	categories       []string
	reviewQueue      []ReviewItem
	reviewCursor     int
	pickingCategory  bool
	categoryCursor   int
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
			switch m.activeView {
			case ViewReview:
				if m.pickingCategory {
					m.categoryCursor = max(0, m.categoryCursor-1)
				} else {
					m.reviewCursor = max(0, m.reviewCursor-1)
				}
			case ViewTransactions:
				m.offset = max(0, m.offset-1)
			}
		}
		if msg.String() == "down" || msg.String() == "j" {
			switch m.activeView {
			case ViewReview:
				if m.pickingCategory {
					m.categoryCursor = min(len(m.categories)-1, m.categoryCursor+1)
				} else {
					m.reviewCursor = min(len(m.reviewQueue)-1, m.reviewCursor+1)
				}
			case ViewTransactions:
				m.offset = min(max(0, len(m.transactions)-(m.height-3)), m.offset+1)
			}
		}
		if msg.String() == "enter" {
			if !m.pickingCategory && m.activeView == ViewReview {
				m.pickingCategory = true
				m.categoryCursor = 0
			} else if m.pickingCategory && m.activeView == ViewReview {
				item := m.reviewQueue[m.reviewCursor]
				err := saveCategoryRule("rules.toml", item.label, m.categories[m.categoryCursor])
				if err != nil {
					m.err = fmt.Errorf("failed to save category rule: %w", err)
				}
				m.reviewQueue = append(m.reviewQueue[:m.reviewCursor], m.reviewQueue[m.reviewCursor+1:]...)
				m.reviewCursor = max(0, m.reviewCursor-1)
				m.pickingCategory = false
			}
		}
		if msg.String() == "esc" {
			if m.pickingCategory && m.activeView == ViewReview {
				m.pickingCategory = false
			}
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
	case ViewReview:
		return m.renderReview()
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
	for _, transaction := range m.transactions[m.offset:min(m.offset+m.height-3, len(m.transactions))] {
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

func (m model) renderReview() string {
	var result strings.Builder
	result.WriteString("Review Queue:\n")
	for i, item := range m.reviewQueue[m.reviewCursor:min(m.reviewCursor+m.height-3, len(m.reviewQueue))] {
		if i == 0 {
			result.WriteString("> ")
		} else {
			result.WriteString("  ")
		}
		row := fmt.Sprintf("%s\n", item.label)
		result.WriteString(row)
	}
	result.WriteString("\nPress 's' to view summary, 't' to view transactions, 'q' to quit.")
	var categoryOptions strings.Builder
	if m.pickingCategory {
		categoryOptions.WriteString("\nSelect category:\n")
		for i, category := range m.categories {
			if i == m.categoryCursor {
				categoryOptions.WriteString("> ")
			} else {
				categoryOptions.WriteString("  ")
			}
			row := fmt.Sprintf("%s\n", category)
			categoryOptions.WriteString(row)
		}
	}
	view := lipgloss.JoinHorizontal(lipgloss.Top, result.String(), categoryOptions.String())
	return view
}
