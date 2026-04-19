package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var navViews = []struct {
	key  string
	name string
}{
	{"s", "summary"},
	{"c", "categories"},
	{"t", "transactions"},
	{"r", "review"},
	{"a", "chat"},
	{"q", "quit"},
}

// globalHint renders the navigation bar, highlighting the active view.
func globalHint(active string) string {
	var parts []string
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	faintStyle := lipgloss.NewStyle().Faint(true)

	for _, v := range navViews {
		entry := fmt.Sprintf("%s %s", v.key, v.name)
		if v.name == active {
			parts = append(parts, activeStyle.Render(entry))
		} else {
			parts = append(parts, faintStyle.Render(entry))
		}
	}
	return "  " + strings.Join(parts, faintStyle.Render("   "))
}
