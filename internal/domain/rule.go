package domain

type Rule struct {
	Pattern  string          // single pattern, kept for backward compat
	Patterns []string        // multiple patterns, OR logic
	Regex    bool            // if true, treat pattern(s) as Go regexp
	Type     TransactionType // empty = don't override
	Category string          // empty = don't override
}

// AllPatterns returns all patterns from both Pattern and Patterns fields.
// Pattern is listed first so single-pattern rules keep their position.
func (r Rule) AllPatterns() []string {
	var out []string
	if r.Pattern != "" {
		out = append(out, r.Pattern)
	}
	out = append(out, r.Patterns...)
	return out
}
