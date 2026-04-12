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

// LLMResponseMsg is returned when the LLM produces a final answer.
type LLMResponseMsg struct {
	Content string
}

// LLMErrorMsg is returned when the LLM call fails.
type LLMErrorMsg struct {
	Err error
}

const systemPrompt = `You are a personal finance assistant with access to the user's bank transactions.

CRITICAL: You must call tools to act. Never describe what you would do — always do it.

When the user asks to categorize transactions you MUST follow this exact sequence:
1. Call get_categories to see available categories.
2. Call get_uncategorized_transactions to get the list.
3. For EACH transaction in the list, call save_category_rule immediately with:
   - pattern: a short keyword from the description (e.g. "Amazon", "Starbucks")
   - category: the most fitting category from the list
4. After ALL rules are saved, reply with a short summary of what was categorized.

Do NOT produce any prose, tables, or analysis before all save_category_rule calls are done.
Do NOT ask the user for confirmation. Just call the tools and report when finished.`

// Client manages the Ollama conversation and executes tool calls.
type Client struct {
	ollama  *api.Client
	model   string
	tools   Tools
	history []api.Message
}

// NewClient creates a Client connected to the local Ollama instance.
// It never fails: if OLLAMA_HOST is unset or malformed it falls back to
// the default http://localhost:11434.
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

// Ask appends the user question to history and returns a tea.Cmd that drives
// the conversation loop in a goroutine: send → tool calls → send → … → answer.
func (c *Client) Ask(question string) tea.Cmd {
	return func() tea.Msg {
		c.history = append(c.history, api.Message{Role: "user", Content: question})

		for {
			msg, err := c.roundtrip()
			if err != nil {
				return LLMErrorMsg{Err: err}
			}

			// No tool calls → final answer.
			if len(msg.ToolCalls) == 0 {
				c.history = append(c.history, api.Message{Role: "assistant", Content: msg.Content})
				return LLMResponseMsg{Content: msg.Content}
			}

			// Record the assistant turn that contains the tool calls.
			c.history = append(c.history, api.Message{
				Role:      "assistant",
				ToolCalls: msg.ToolCalls,
			})

			// Execute each tool and feed the results back.
			for _, call := range msg.ToolCalls {
				result := c.execute(call)
				c.history = append(c.history, api.Message{
					Role:    "tool",
					Content: result,
				})
			}
		}
	}
}

// roundtrip sends the current history to Ollama and returns the response message.
func (c *Client) roundtrip() (api.Message, error) {
	stream := false
	var response api.Message
	err := c.ollama.Chat(context.Background(), &api.ChatRequest{
		Model:    c.model,
		Messages: c.history,
		Tools:    c.tools.schemas(),
		Stream:   &stream,
	}, func(resp api.ChatResponse) error {
		response = resp.Message
		return nil
	})
	return response, err
}

// execute dispatches a tool call to the right callback and returns a result string.
func (c *Client) execute(call api.ToolCall) string {
	args := call.Function.Arguments

	switch call.Function.Name {
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
