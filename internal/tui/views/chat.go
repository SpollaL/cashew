package views

import (
	"cashew/internal/llm"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type chatEntry struct {
	role    string // "user", "assistant", "error"
	content string
}

// ChatModel is the chat view. It holds a text input and a scrollable
// conversation history backed by a viewport.
type ChatModel struct {
	client   *llm.Client
	input    textinput.Model
	vp       viewport.Model
	history  []chatEntry
	thinking bool
	streamCh <-chan tea.Msg
	width    int
	height   int
}

// reserved lines: title (2) + gap before input (1) + input (1) + gap (1) + nav (1) = 6
const reservedLines = 6

func NewChat(client *llm.Client) ChatModel {
	ti := textinput.New()
	ti.Placeholder = "Ask about your finances…"
	ti.Focus()
	ti.CharLimit = 500
	ti.Width = 60

	vp := viewport.New(0, 0)
	return ChatModel{client: client, input: ti, vp: vp}
}

func (m ChatModel) SetSize(width, height int) ChatModel {
	m.width = width
	m.height = height
	m.input.Width = width - 6
	m.vp.Width = width
	m.vp.Height = height - reservedLines
	m.vp.SetContent(m.renderHistory())
	m.vp.GotoBottom()
	return m
}

func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" {
			if m.thinking || strings.TrimSpace(m.input.Value()) == "" {
				return m, nil
			}
			question := m.input.Value()
			m.history = append(m.history, chatEntry{role: "user", content: question})
			m.input.SetValue("")
			m.thinking = true
			m.vp.SetContent(m.renderHistory())
			m.vp.GotoBottom()
			return m, m.client.Ask(question)
		}

	case llm.LLMStreamStartMsg:
		m.streamCh = msg.Ch
		return m, llm.WaitForStream(m.streamCh)

	case llm.LLMStreamMsg:
		if m.thinking {
			// First content chunk: clear the thinking indicator and open a new entry.
			m.thinking = false
			m.history = append(m.history, chatEntry{role: "assistant", content: msg.Content})
		} else {
			last := len(m.history) - 1
			if last >= 0 && m.history[last].role == "assistant" {
				m.history[last].content += msg.Content
			} else {
				m.history = append(m.history, chatEntry{role: "assistant", content: msg.Content})
			}
		}
		m.vp.SetContent(m.renderHistory())
		m.vp.GotoBottom()
		return m, llm.WaitForStream(m.streamCh)

	case llm.LLMStreamDoneMsg:
		m.thinking = false
		m.streamCh = nil
		return m, nil

	case llm.LLMErrorMsg:
		m.history = append(m.history, chatEntry{
			role:    "error",
			content: fmt.Sprintf("error: %v", msg.Err),
		})
		m.thinking = false
		m.streamCh = nil
		m.vp.SetContent(m.renderHistory())
		m.vp.GotoBottom()
		return m, nil
	}

	// Scroll keys go to the viewport only; everything else goes to the text input.
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "up", "k", "down", "j", "pgup", "pgdown", "ctrl+u", "ctrl+d":
			var vpCmd tea.Cmd
			m.vp, vpCmd = m.vp.Update(msg)
			return m, vpCmd
		}
	}

	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	return m, inputCmd
}

func (m ChatModel) View() string {
	bold := lipgloss.NewStyle().Bold(true)
	title := "\n  " + bold.Render("Chat") + "\n"
	input := "\n  " + m.input.View() + "\n"
	nav := globalHint("chat") + "\n"
	return title + m.vp.View() + input + nav
}

// renderHistory builds the full content string for the viewport.
func (m ChatModel) renderHistory() string {
	userStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	assistantStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	// Content width: viewport width minus "  llm: " (7 chars) indent.
	contentWidth := m.vp.Width - 7
	if contentWidth < 20 {
		contentWidth = 20
	}

	var sb strings.Builder
	for _, e := range m.history {
		switch e.role {
		case "user":
			sb.WriteString("  " + userStyle.Render("you: ") + e.content + "\n")
		case "assistant":
			prefix := "  " + assistantStyle.Render("llm: ")
			indent := "       " // 7 spaces, matching "  llm: "
			lines := wordWrap(e.content, contentWidth)
			for i, line := range lines {
				if i == 0 {
					sb.WriteString(prefix + line + "\n")
				} else {
					sb.WriteString(indent + line + "\n")
				}
			}
		case "error":
			sb.WriteString("  " + errorStyle.Render(e.content) + "\n")
		}
	}
	if m.thinking {
		sb.WriteString("  " + assistantStyle.Render("llm: ") + "thinking…\n")
	}
	return sb.String()
}

// wordWrap splits text into lines of at most width runes, breaking at word boundaries.
// It preserves existing newlines in the input.
func wordWrap(text string, width int) []string {
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		out = append(out, wrapLine(paragraph, width)...)
	}
	return out
}

func wrapLine(line string, width int) []string {
	if width <= 0 || len([]rune(line)) <= width {
		return []string{line}
	}
	var lines []string
	for len([]rune(line)) > width {
		cut := width
		// Walk back to the last space so we don't break mid-word.
		if idx := strings.LastIndex(string([]rune(line)[:cut]), " "); idx > 0 {
			cut = idx
		}
		lines = append(lines, string([]rune(line)[:cut]))
		line = strings.TrimLeft(string([]rune(line)[cut:]), " ")
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}
