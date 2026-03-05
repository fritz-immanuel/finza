package domain

import "time"

type EntryType string

const (
	EntryTypeExpense  EntryType = "expense"
	EntryTypeIncome   EntryType = "income"
	EntryTypeTransfer EntryType = "transfer"
	EntryTypeUnknown  EntryType = "unknown"
)

type Entry struct {
	ID          int64
	UserID      int64
	ChatID      int64
	MessageID   int
	Timestamp   time.Time
	Amount      int64
	Currency    string
	Type        EntryType
	Category    string
	Description string
	RawText     string
	Confidence  float64
	CreatedAt   time.Time
}
