package app

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/moneytracker/domain"
)

type ExpenseService struct {
	store domain.Store
	parse domain.Parser
	clock domain.Clock
}

type ParseResult struct {
	Entry           domain.Entry
	Financial       bool
	RequiresConfirm bool
	Saved           bool
}

func NewExpenseService(store domain.Store, parser domain.Parser, clock domain.Clock) *ExpenseService {
	return &ExpenseService{store: store, parse: parser, clock: clock}
}

func (s *ExpenseService) ProcessMessage(ctx context.Context, user domain.User, chatID int64, messageID int, text string) (ParseResult, error) {
	if err := s.store.UpsertUser(ctx, user); err != nil {
		return ParseResult{}, fmt.Errorf("store.UpsertUser: %w", err)
	}

	_, err := s.store.SaveMessage(ctx, domain.RawMessage{
		UserID:    user.ID,
		ChatID:    chatID,
		MessageID: messageID,
		Text:      text,
		CreatedAt: s.clock.Now().UTC(),
	})
	if err != nil {
		return ParseResult{}, fmt.Errorf("store.SaveMessage: %w", err)
	}

	entry, ok, confidence, err := s.parse.Parse(ctx, text)
	if err != nil {
		return ParseResult{}, fmt.Errorf("parser.Parse: %w", err)
	}
	if !ok {
		return ParseResult{Financial: false}, nil
	}

	entry.UserID = user.ID
	entry.ChatID = chatID
	entry.MessageID = messageID
	entry.RawText = text
	entry.Confidence = confidence
	entry.Timestamp = s.clock.Now().UTC()
	if entry.Type == "" {
		entry.Type = domain.EntryTypeUnknown
	}
	if entry.Category == "" {
		entry.Category = "General"
	}

	requiresConfirm := confidence < 0.75 || entry.Type == domain.EntryTypeUnknown
	if requiresConfirm {
		entry.Type = domain.EntryTypeUnknown
	}

	id, err := s.store.SaveEntry(ctx, entry)
	if err != nil {
		return ParseResult{}, fmt.Errorf("store.SaveEntry: %w", err)
	}
	entry.ID = id

	return ParseResult{
		Entry:           entry,
		Financial:       true,
		RequiresConfirm: requiresConfirm,
		Saved:           true,
	}, nil
}

func (s *ExpenseService) ConfirmEntryType(ctx context.Context, entryID int64, typ domain.EntryType) error {
	entry, ok, err := s.store.GetEntryByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("store.GetEntryByID: %w", err)
	}
	if !ok {
		return fmt.Errorf("entry not found")
	}

	entry.Type = typ
	if typ == domain.EntryTypeExpense && entry.Category == "Transfer" {
		entry.Category = "General"
	}
	if typ == domain.EntryTypeIncome {
		entry.Category = "Income"
	}
	if typ == domain.EntryTypeTransfer {
		entry.Category = "Transfer"
	}

	if err := s.store.UpdateEntry(ctx, entry); err != nil {
		return fmt.Errorf("store.UpdateEntry: %w", err)
	}
	return nil
}

func (s *ExpenseService) DeleteEntry(ctx context.Context, userID, entryID int64) error {
	entry, ok, err := s.store.GetEntryByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("store.GetEntryByID: %w", err)
	}
	if !ok || entry.UserID != userID {
		return fmt.Errorf("entry not found")
	}
	if err := s.store.DeleteEntry(ctx, entryID); err != nil {
		return fmt.Errorf("store.DeleteEntry: %w", err)
	}
	return nil
}

type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
