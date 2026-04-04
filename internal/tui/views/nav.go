package views

import "github.com/charmbracelet/lipgloss"

// globalHint renders the navigation bar that is the same across all views.
func globalHint() string {
	return lipgloss.NewStyle().Faint(true).Render(
		"  s summary   c categories   t transactions   r review   q quit",
	)
}
