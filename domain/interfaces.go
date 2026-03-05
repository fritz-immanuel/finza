package domain

import (
	"context"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Parser parses free-form text into a financial Entry.
// Returns: entry, ok (is it financial?), confidence [0.0-1.0], error.
type Parser interface {
	Parse(ctx context.Context, text string) (Entry, bool, float64, error)
}

// Store is the persistence layer.
type Store interface {
	// Users
	UpsertUser(ctx context.Context, user User) error
	GetUser(ctx context.Context, userID int64) (User, error)

	// Messages
	SaveMessage(ctx context.Context, msg RawMessage) (int64, error)

	// Entries
	SaveEntry(ctx context.Context, entry Entry) (int64, error)
	UpdateEntry(ctx context.Context, entry Entry) error
	DeleteEntry(ctx context.Context, entryID int64) error
	GetEntryByID(ctx context.Context, entryID int64) (Entry, bool, error)
	GetEntryByMessageID(ctx context.Context, userID int64, messageID int) (Entry, bool, error)
	GetLastEntries(ctx context.Context, userID int64, limit int) ([]Entry, error)
	GetEntriesByDateRange(ctx context.Context, userID int64, from, to time.Time) ([]Entry, error)
	GetCategoryTotals(ctx context.Context, userID int64, from, to time.Time) (map[string]int64, error)
}

// Clock abstracts time for testability.
type Clock interface {
	Now() time.Time
}

// Metrics can be implemented by observability backends.
type Metrics interface {
	IncCounter(name string)
	ObserveDuration(name string, d time.Duration)
}

// BotClient abstracts Telegram API calls.
type BotClient interface {
	Send(msg tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	SetWebhook(config tgbotapi.WebhookConfig) error
}
