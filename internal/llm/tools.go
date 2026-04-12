package llm

import "github.com/ollama/ollama/api"

// TransactionFilters holds optional filters for the GetTransactions tool.
type TransactionFilters struct {
	Category string // filter by category name
	Month    string // "YYYY-MM" format, e.g. "2024-01"
	Type     string // "expense", "income", or "investment"
}

// Tools holds callback functions the LLM can invoke.
// Each field is wired up by the caller (app.go) which owns the data.
type Tools struct {
	GetUncategorizedTransactions func() []string
	GetTransactions              func(filters TransactionFilters) []string
	GetMonthlySummary            func(months []string) []string
	GetCategories                func() []string
	SaveCategoryRule             func(pattern, category string) error
}

// schemas returns the Ollama tool definitions so the model knows what tools exist.
func (t Tools) schemas() api.Tools {
	getTxProps := api.NewToolPropertiesMap()
	getTxProps.Set("category", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Filter by category name, e.g. 'groceries'",
	})
	getTxProps.Set("month", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Filter by month in YYYY-MM format, e.g. '2024-01'",
	})
	getTxProps.Set("type", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Filter by transaction type: 'expense', 'income', or 'investment'",
	})

	summaryProps := api.NewToolPropertiesMap()
	summaryProps.Set("months", api.ToolProperty{
		Type:        api.PropertyType{"array"},
		Description: "Months in YYYY-MM format to include. Omit to return all months.",
	})

	saveRuleProps := api.NewToolPropertiesMap()
	saveRuleProps.Set("pattern", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Merchant name or keyword to match, e.g. 'Starbucks'",
	})
	saveRuleProps.Set("category", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Category to assign to matching transactions, e.g. 'coffee'",
	})

	return api.Tools{
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "get_uncategorized_transactions",
				Description: "Fetch all transactions that have not been assigned a category yet.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: api.NewToolPropertiesMap(),
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "get_transactions",
				Description: "Fetch individual transactions. Optionally filter by category, month (YYYY-MM), or type.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: getTxProps,
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "get_monthly_summary",
				Description: "Get monthly income/expense/investment summaries. Pass specific months or omit for all.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: summaryProps,
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "get_categories",
				Description: "List all known spending categories.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: api.NewToolPropertiesMap(),
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "save_category_rule",
				Description: "Save a rule that maps a merchant keyword to a category.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: saveRuleProps,
					Required:   []string{"pattern", "category"},
				},
			},
		},
	}
}
