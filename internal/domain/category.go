package domain

type Rule struct {
	Pattern  string          // case-insensitive substring
	Type     TransactionType // empty = don't override
	Category string          // empty = don't override
}
