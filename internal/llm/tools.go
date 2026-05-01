package llm

import (
	"fmt"
	"strings"

	"github.com/ollama/ollama/api"
)

// TransactionFilters holds optional filters for the GetTransactions tool.
type TransactionFilters struct {
	Category string // filter by category name
	Month    string // "YYYY-MM" format, e.g. "2024-01"
	Type     string // "expense", "income", or "investment"
}

// Tools holds callback functions the LLM can invoke.
// Each field is wired up by the caller (app.go) which owns the data.
type Tools struct {
	GetUncategorizedTransactions func(offset int) []string
	GetTransactions              func(filters TransactionFilters) []string
	GetMonthlySummary            func(months []string) []string
	GetCategories                func() []string
	SaveCategoryRule             func(pattern, category string) error
	BulkSaveCategoryRules        func(rules []CategoryRule) (saved int, err error)
}

// CategoryRule is a single pattern → category mapping used by BulkSaveCategoryRules.
type CategoryRule struct {
	Pattern  string
	Patterns []string
	Regex    bool
	Category string
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

	uncatProps := api.NewToolPropertiesMap()
	uncatProps.Set("offset", api.ToolProperty{
		Type:        api.PropertyType{"integer"},
		Description: "Number of transactions to skip (default 0). Use to fetch subsequent batches.",
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

	validCats := strings.Join(t.GetCategories(), ", ")
	bulkRuleItemProps := api.NewToolPropertiesMap()
	bulkRuleItemProps.Set("pattern", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: "Single keyword to match in the transaction description (case-insensitive substring).",
	})
	bulkRuleItemProps.Set("patterns", api.ToolProperty{
		Type:        api.PropertyType{"array"},
		Description: "Alternative to 'pattern': list of keywords matched with OR logic. Use to group similar merchants under one rule.",
	})
	bulkRuleItemProps.Set("regex", api.ToolProperty{
		Type:        api.PropertyType{"boolean"},
		Description: "If true, treat pattern/patterns as Go regexp matched against the lowercase description. Use ^ to anchor to the start.",
	})
	bulkRuleItemProps.Set("category", api.ToolProperty{
		Type:        api.PropertyType{"string"},
		Description: fmt.Sprintf("Category to assign. Must be exactly one of: %s", validCats),
	})
	bulkSaveProps := api.NewToolPropertiesMap()
	bulkSaveProps.Set("rules", api.ToolProperty{
		Type:        api.PropertyType{"array"},
		Description: "List of rule objects to save. Each rule has: pattern (string), patterns (array, optional), regex (bool, optional), category (string).",
	})

	return api.Tools{
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "bulk_save_category_rules",
				Description: "Save multiple categorization rules at once. Use this when categorizing many transactions — pass all rules in a single call instead of calling save_category_rule repeatedly.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: bulkSaveProps,
					Required:   []string{"rules"},
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "get_uncategorized_transactions",
				Description: "Fetch a batch of uncategorized transactions. Returns up to 20 at a time. If the result says more remain, call again with the next offset.",
				Parameters: api.ToolFunctionParameters{
					Type:       "object",
					Properties: uncatProps,
				},
			},
		},
		{
			Type: "function",
			Function: api.ToolFunction{
				Name:        "get_transactions",
				Description: "Fetch individual transactions with their exact amounts and descriptions. Use this to find the biggest or smallest expenses, list specific purchases, or drill into details. Optionally filter by category, month (YYYY-MM), or type (expense/income/investment).",
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
				Description: "Get high-level totals per month: total income, total expenses, total investments. Use this for questions like 'how much did I spend in January' or 'show me all months'. Do NOT use this to find individual transactions or biggest expenses — use get_transactions for that.",
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
