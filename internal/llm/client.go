package llm

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// LLMDebugMsg carries a single debug line describing an LLM roundtrip event.
type LLMDebugMsg struct {
	Text string
}

// LLMErrorMsg is returned when the LLM call fails.
type LLMErrorMsg struct {
	Err error
}

const systemPromptTemplate = `You are a personal finance assistant with access to the user's bank transactions.
Today's date is %s.

CRITICAL: You must call tools to act. Never describe what you would do — always do it.

CRITICAL: Never invent category names. You must always use one of the categories returned by get_categories. If a transaction does not fit any category perfectly, assign the closest available one.

Tool guide — pick the right one:
- get_monthly_summary: totals per month (income, expenses, investments). Use for "how much did I spend in month X" questions.
- get_transactions: individual transactions with exact amounts. Use for "biggest expenses", "list my purchases", or any question needing line-item detail.
- get_categories: list available category names.
- get_uncategorized_transactions: list transactions not yet assigned a category.
- bulk_save_category_rules: save an array of pattern→category rules. Always prefer this over save_category_rule.
- save_category_rule: save a single pattern→category rule.

When the user asks to categorize transactions, follow this exact sequence:
1. Call get_categories to see available categories.
2. Call get_uncategorized_transactions (offset=0) to get the first batch of up to 20 transactions.
3. Call bulk_save_category_rules with rules for every transaction in that batch.
   Use a distinctive keyword from each description as the pattern.
4. If the result says more batches remain, call get_uncategorized_transactions again with the next offset, then bulk_save_category_rules for that batch. Repeat until the result says "last batch".
5. Reply with a short summary of the total number of transactions categorized.

Do NOT produce any prose or analysis before all batches are processed.
Do NOT ask the user for confirmation. Just call the tools and report when finished.`

// Client manages the Ollama conversation and executes tool calls.
type Client struct {
	ollama  *api.Client
	model   string
	tools   Tools
	history []api.Message
	debug   bool
}

// NewClient creates a Client connected to the local Ollama instance.
func NewClient(model string, tools Tools, debug bool) *Client {
	c, err := api.ClientFromEnvironment()
	if err != nil {
		u, _ := url.Parse("http://localhost:11434")
		c = api.NewClient(u, http.DefaultClient)
	}
	today := time.Now().Format("2006-01-02")
	return &Client{
		ollama: c,
		model:  model,
		tools:  tools,
		debug:  debug,
		history: []api.Message{
			{Role: "system", Content: fmt.Sprintf(systemPromptTemplate, today)},
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

const maxRoundtrips = 10

// stream runs the tool-calling loop in a goroutine, writing content chunks and
// terminal events to ch.
func (c *Client) stream(question string, ch chan<- tea.Msg) {
	c.history = append(c.history, api.Message{Role: "user", Content: question})

	for round := range maxRoundtrips {
		if c.debug {
			roles := make([]string, len(c.history))
			for i, m := range c.history {
				roles[i] = m.Role
			}
			ch <- LLMDebugMsg{Text: fmt.Sprintf("[round %d/%d] → %d msgs: [%s]",
				round+1, maxRoundtrips, len(c.history), strings.Join(roles, ", "))}
		}

		msg, chunks, err := c.roundtrip()
		if err != nil {
			ch <- LLMErrorMsg{Err: err}
			close(ch)
			return
		}

		// No tool calls → this is the final answer; replay buffered chunks to ch.
		if len(msg.ToolCalls) == 0 {
			if c.debug {
				preview := msg.Content
				if len([]rune(preview)) > 80 {
					preview = string([]rune(preview)[:80]) + "…"
				}
				ch <- LLMDebugMsg{Text: fmt.Sprintf("[round %d] ← FINAL (%d chars): %q",
					round+1, len(msg.Content), preview)}
			}
			c.history = append(c.history, api.Message{Role: "assistant", Content: msg.Content})
			for _, chunk := range chunks {
				ch <- chunk
			}
			ch <- LLMStreamDoneMsg{}
			close(ch)
			return
		}

		if c.debug {
			names := make([]string, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				names[i] = tc.Function.Name
			}
			ch <- LLMDebugMsg{Text: fmt.Sprintf("[round %d] ← %d tool call(s): [%s]",
				round+1, len(msg.ToolCalls), strings.Join(names, ", "))}
		}

		if round == maxRoundtrips-1 {
			ch <- LLMErrorMsg{Err: fmt.Errorf("model did not produce a final answer after %d tool-call rounds", maxRoundtrips)}
			close(ch)
			return
		}

		// Tool-call round: discard any content the model produced before the tool
		// call (reasoning text, preamble, etc.) so it doesn't appear as a fake
		// final answer in the UI. Record the assistant turn and execute each tool.
		c.history = append(c.history, api.Message{
			Role:      "assistant",
			ToolCalls: msg.ToolCalls,
		})
		for _, call := range msg.ToolCalls {
			result := c.execute(call)
			if c.debug {
				argsStr := fmt.Sprintf("%v", call.Function.Arguments)
				if len([]rune(argsStr)) > 60 {
					argsStr = string([]rune(argsStr)[:60]) + "…"
				}
				resultPreview := result
				if len([]rune(resultPreview)) > 80 {
					resultPreview = string([]rune(resultPreview)[:80]) + "…"
				}
				ch <- LLMDebugMsg{Text: fmt.Sprintf("  ↳ %s(%s) → %s",
					call.Function.Name, argsStr, resultPreview)}
			}
			c.history = append(c.history, api.Message{
				Role:    "tool",
				Content: result,
			})
		}
	}
}

// roundtrip sends the current history to Ollama with streaming enabled.
// Content tokens are buffered and returned as []LLMStreamMsg so the caller can
// decide whether to forward them to the UI (final answer only) or discard them
// (tool-call rounds, where the model often produces preamble text before the
// tool call that would otherwise look like a premature final answer).
func (c *Client) roundtrip() (api.Message, []LLMStreamMsg, error) {
	stream := true
	var contentBuf strings.Builder
	var toolCalls []api.ToolCall
	var chunks []LLMStreamMsg

	err := c.ollama.Chat(context.Background(), &api.ChatRequest{
		Model:    c.model,
		Messages: c.history,
		Tools:    c.tools.schemas(),
		Stream:   &stream,
	}, func(resp api.ChatResponse) error {
		if resp.Message.Content != "" {
			contentBuf.WriteString(resp.Message.Content)
			chunks = append(chunks, LLMStreamMsg{Content: resp.Message.Content})
		}
		toolCalls = append(toolCalls, resp.Message.ToolCalls...)
		return nil
	})

	return api.Message{
		Content:   contentBuf.String(),
		ToolCalls: toolCalls,
	}, chunks, err
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

		// Build a case-insensitive lookup of valid categories.
		canonical := make(map[string]string)
		for _, c := range c.tools.GetCategories() {
			canonical[strings.ToLower(c)] = c
		}

		var rules []CategoryRule
		var invalid []string
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			p, _ := m["pattern"].(string)
			cat, _ := m["category"].(string)
			if p == "" || cat == "" {
				continue
			}
			norm, ok := canonical[strings.ToLower(cat)]
			if !ok {
				invalid = append(invalid, cat)
				continue
			}
			rules = append(rules, CategoryRule{Pattern: p, Category: norm})
		}
		saved, err := c.tools.BulkSaveCategoryRules(rules)
		if err != nil {
			return fmt.Sprintf("Saved %d rules, then hit an error: %v", saved, err)
		}
		if len(invalid) > 0 {
			valid := strings.Join(c.tools.GetCategories(), ", ")
			return fmt.Sprintf("Saved %d rules. WARNING: skipped %d rules with unknown categories [%s] — valid categories are: %s. Use only valid categories.",
				saved, len(invalid), strings.Join(invalid, ", "), valid)
		}
		return fmt.Sprintf("Saved %d rules.", saved)

	case "get_uncategorized_transactions":
		offset := 0
		if v, ok := args.Get("offset"); ok {
			switch n := v.(type) {
			case float64:
				offset = int(n)
			case int:
				offset = n
			}
		}
		rows := c.tools.GetUncategorizedTransactions(offset)
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
