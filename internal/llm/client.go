package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ollama/ollama/api"
)

// LLMStreamStartMsg is returned by Ask; it carries the channel that delivers
// subsequent stream events (LLMStreamMsg / LLMStreamDoneMsg / LLMErrorMsg).
type LLMStreamStartMsg struct {
	Ch <-chan tea.Msg
}

// LLMStreamMsg carries one chunk of text from the model's final response.
type LLMStreamMsg struct {
	Content string
}

// LLMStreamDoneMsg signals the stream finished successfully.
type LLMStreamDoneMsg struct{}

// LLMErrorMsg is returned when the LLM call fails.
type LLMErrorMsg struct {
	Err error
}

const systemPrompt = `You are a personal finance assistant with access to the user's bank transactions.

CRITICAL: You must call tools to act. Never describe what you would do — always do it.

When the user asks to categorize transactions you MUST follow this exact sequence:
1. Call get_categories to see available categories.
2. Call get_uncategorized_transactions to get the list.
3. Call bulk_save_category_rules ONCE with ALL rules as an array — do NOT call save_category_rule in a loop.
   For each transaction pick a short keyword from the description as pattern and the best matching category.
4. Reply with a short summary of what was categorized.

Do NOT produce any prose, tables, or analysis before bulk_save_category_rules is called.
Do NOT ask the user for confirmation. Just call the tools and report when finished.`

// Client manages the Ollama conversation and executes tool calls.
type Client struct {
	ollama  *api.Client
	model   string
	tools   Tools
	history []api.Message
}

// NewClient creates a Client connected to the local Ollama instance.
func NewClient(model string, tools Tools) *Client {
	c, err := api.ClientFromEnvironment()
	if err != nil {
		u, _ := url.Parse("http://localhost:11434")
		c = api.NewClient(u, http.DefaultClient)
	}
	return &Client{
		ollama: c,
		model:  model,
		tools:  tools,
		history: []api.Message{
			{Role: "system", Content: systemPrompt},
		},
	}
}

// WaitForStream returns a Cmd that reads the next message from ch.
func WaitForStream(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// Ask appends the user question to history and returns a Cmd that immediately
// delivers LLMStreamStartMsg. Content tokens arrive as LLMStreamMsg events,
// terminated by LLMStreamDoneMsg or LLMErrorMsg.
func (c *Client) Ask(question string) tea.Cmd {
	ch := make(chan tea.Msg, 50)
	return func() tea.Msg {
		go c.stream(question, ch)
		return LLMStreamStartMsg{Ch: ch}
	}
}

// stream runs the tool-calling loop in a goroutine, writing content chunks and
// terminal events to ch.
func (c *Client) stream(question string, ch chan<- tea.Msg) {
	c.history = append(c.history, api.Message{Role: "user", Content: question})

	for {
		msg, err := c.roundtrip(ch)
		if err != nil {
			ch <- LLMErrorMsg{Err: err}
			close(ch)
			return
		}

		// No tool calls → final answer was already streamed to ch.
		if len(msg.ToolCalls) == 0 {
			c.history = append(c.history, api.Message{Role: "assistant", Content: msg.Content})
			ch <- LLMStreamDoneMsg{}
			close(ch)
			return
		}

		// Record the assistant turn with tool calls and execute each one.
		c.history = append(c.history, api.Message{
			Role:      "assistant",
			ToolCalls: msg.ToolCalls,
		})
		for _, call := range msg.ToolCalls {
			result := c.execute(call)
			c.history = append(c.history, api.Message{
				Role:    "tool",
				Content: result,
			})
		}
	}
}

// roundtrip sends the current history to Ollama with streaming enabled.
// Content tokens are forwarded to ch as LLMStreamMsg events.
// Tool-call rounds produce no content, so nothing is sent on those rounds.
func (c *Client) roundtrip(ch chan<- tea.Msg) (api.Message, error) {
	stream := true
	var contentBuf strings.Builder
	var toolCalls []api.ToolCall

	err := c.ollama.Chat(context.Background(), &api.ChatRequest{
		Model:    c.model,
		Messages: c.history,
		Tools:    c.tools.schemas(),
		Stream:   &stream,
	}, func(resp api.ChatResponse) error {
		if resp.Message.Content != "" {
			contentBuf.WriteString(resp.Message.Content)
			ch <- LLMStreamMsg{Content: resp.Message.Content}
		}
		toolCalls = append(toolCalls, resp.Message.ToolCalls...)
		return nil
	})

	return api.Message{
		Content:   contentBuf.String(),
		ToolCalls: toolCalls,
	}, err
}

// execute dispatches a tool call to the right callback and returns a result string.
func (c *Client) execute(call api.ToolCall) string {
	args := call.Function.Arguments

	switch call.Function.Name {
	case "bulk_save_category_rules":
		v, ok := args.Get("rules")
		if !ok {
			return "Error: rules parameter is required."
		}
		items, ok := v.([]any)
		if !ok {
			return "Error: rules must be an array."
		}
		var rules []CategoryRule
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			p, _ := m["pattern"].(string)
			cat, _ := m["category"].(string)
			if p != "" && cat != "" {
				rules = append(rules, CategoryRule{Pattern: p, Category: cat})
			}
		}
		saved, err := c.tools.BulkSaveCategoryRules(rules)
		if err != nil {
			return fmt.Sprintf("Saved %d rules, then hit an error: %v", saved, err)
		}
		return fmt.Sprintf("Saved %d rules.", saved)

	case "get_uncategorized_transactions":
		rows := c.tools.GetUncategorizedTransactions()
		if len(rows) == 0 {
			return "All transactions are categorized."
		}
		return strings.Join(rows, "\n")

	case "get_transactions":
		filters := TransactionFilters{}
		if v, ok := args.Get("category"); ok {
			filters.Category, _ = v.(string)
		}
		if v, ok := args.Get("month"); ok {
			filters.Month, _ = v.(string)
		}
		if v, ok := args.Get("type"); ok {
			filters.Type, _ = v.(string)
		}
		rows := c.tools.GetTransactions(filters)
		if len(rows) == 0 {
			return "No transactions found."
		}
		return strings.Join(rows, "\n")

	case "get_monthly_summary":
		var months []string
		if v, ok := args.Get("months"); ok {
			if ms, ok := v.([]any); ok {
				for _, m := range ms {
					if s, ok := m.(string); ok {
						months = append(months, s)
					}
				}
			}
		}
		rows := c.tools.GetMonthlySummary(months)
		if len(rows) == 0 {
			return "No summaries found."
		}
		return strings.Join(rows, "\n")

	case "get_categories":
		cats := c.tools.GetCategories()
		if len(cats) == 0 {
			return "No categories defined."
		}
		return strings.Join(cats, ", ")

	case "save_category_rule":
		var p, cat string
		if v, ok := args.Get("pattern"); ok {
			p, _ = v.(string)
		}
		if v, ok := args.Get("category"); ok {
			cat, _ = v.(string)
		}
		if p == "" || cat == "" {
			return "Error: pattern and category are required."
		}
		if err := c.tools.SaveCategoryRule(p, cat); err != nil {
			return fmt.Sprintf("Error saving rule: %v", err)
		}
		return fmt.Sprintf("Rule saved: '%s' → '%s'", p, cat)

	default:
		return fmt.Sprintf("Unknown tool: %s", call.Function.Name)
	}
}
