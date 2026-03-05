package domain

import "time"

type RawMessage struct {
	ID        int64
	UserID    int64
	ChatID    int64
	MessageID int
	Text      string
	CreatedAt time.Time
}
