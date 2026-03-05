package telegram

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/moneytracker/app"
	"github.com/yourusername/moneytracker/domain"
)

type fakeBot struct {
	sent []string
}

func (b *fakeBot) Send(msg tgbotapi.Chattable) (tgbotapi.Message, error) {
	switch m := msg.(type) {
	case tgbotapi.MessageConfig:
		b.sent = append(b.sent, m.Text)
	case tgbotapi.DocumentConfig:
		b.sent = append(b.sent, "[document]")
	}
	return tgbotapi.Message{}, nil
}
func (b *fakeBot) Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	return &tgbotapi.APIResponse{Ok: true}, nil
}
func (b *fakeBot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return nil
}
func (b *fakeBot) SetWebhook(config tgbotapi.WebhookConfig) error { return nil }

type fakeParser struct{}

func (fakeParser) Parse(context.Context, string) (domain.Entry, bool, float64, error) {
	return domain.Entry{}, false, 0, nil
}

type fakeClock struct{}

func (fakeClock) Now() time.Time { return time.Date(2026, 3, 5, 12, 0, 0, 0, time.UTC) }

type fakeStore struct {
	lastLimit int
	from      time.Time
	to        time.Time
	entries   []domain.Entry
}

func (s *fakeStore) UpsertUser(context.Context, domain.User) error { return nil }
func (s *fakeStore) GetUser(context.Context, int64) (domain.User, error) {
	return domain.User{ID: 1, Timezone: "Asia/Jakarta"}, nil
}
func (s *fakeStore) SaveMessage(context.Context, domain.RawMessage) (int64, error) { return 1, nil }
func (s *fakeStore) SaveEntry(context.Context, domain.Entry) (int64, error)        { return 1, nil }
func (s *fakeStore) UpdateEntry(context.Context, domain.Entry) error               { return nil }
func (s *fakeStore) DeleteEntry(context.Context, int64) error                      { return nil }
func (s *fakeStore) GetEntryByID(context.Context, int64) (domain.Entry, bool, error) {
	return domain.Entry{ID: 1, UserID: 1}, true, nil
}
func (s *fakeStore) GetEntryByMessageID(context.Context, int64, int) (domain.Entry, bool, error) {
	return domain.Entry{}, false, nil
}
func (s *fakeStore) GetLastEntries(_ context.Context, _ int64, limit int) ([]domain.Entry, error) {
	s.lastLimit = limit
	return s.entries, nil
}
func (s *fakeStore) GetEntriesByDateRange(_ context.Context, _ int64, from, to time.Time) ([]domain.Entry, error) {
	s.from, s.to = from, to
	return nil, nil
}
func (s *fakeStore) GetCategoryTotals(context.Context, int64, time.Time, time.Time) (map[string]int64, error) {
	return map[string]int64{}, nil
}

func newTestHandler(store *fakeStore, bot *fakeBot) *Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	exp := app.NewExpenseService(store, fakeParser{}, fakeClock{})
	rep := app.NewReportService(store, fakeClock{})
	return NewHandler(bot, store, exp, rep, log)
}

func updateWithCommand(cmd string) tgbotapi.Update {
	parts := strings.SplitN(cmd, " ", 2)
	entLen := len(parts[0])
	return tgbotapi.Update{Message: &tgbotapi.Message{
		Text:     cmd,
		Chat:     &tgbotapi.Chat{ID: 11},
		From:     &tgbotapi.User{ID: 1},
		Entities: []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: entLen}},
	}}
}

func TestCommandRouting(t *testing.T) {
	t.Run("/start dispatches", func(t *testing.T) {
		store := &fakeStore{}
		bot := &fakeBot{}
		h := newTestHandler(store, bot)

		called := false
		h.onStart = func(context.Context, tgbotapi.Update) error {
			called = true
			return nil
		}

		if err := h.handleCommand(context.Background(), updateWithCommand("/start")); err != nil {
			t.Fatalf("handleCommand: %v", err)
		}
		if !called {
			t.Fatal("start handler was not called")
		}
	})

	t.Run("/help dispatches", func(t *testing.T) {
		store := &fakeStore{}
		bot := &fakeBot{}
		h := newTestHandler(store, bot)

		called := false
		h.onHelp = func(context.Context, tgbotapi.Update) error {
			called = true
			return nil
		}

		if err := h.handleCommand(context.Background(), updateWithCommand("/help")); err != nil {
			t.Fatalf("handleCommand: %v", err)
		}
		if !called {
			t.Fatal("help handler was not called")
		}
	})

	t.Run("/last 10 parses n", func(t *testing.T) {
		store := &fakeStore{}
		bot := &fakeBot{}
		h := newTestHandler(store, bot)

		if err := h.handleCommand(context.Background(), updateWithCommand("/last 10")); err != nil {
			t.Fatalf("handleCommand: %v", err)
		}
		if store.lastLimit != 10 {
			t.Fatalf("limit = %d, want 10", store.lastLimit)
		}
	})

	t.Run("/delete opens picker when no id", func(t *testing.T) {
		store := &fakeStore{
			entries: []domain.Entry{
				{ID: 42, Description: "Coffee", Amount: 25000, Currency: "IDR"},
				{ID: 41, Description: "Grab", Amount: 28000, Currency: "IDR"},
			},
		}
		bot := &fakeBot{}
		h := newTestHandler(store, bot)

		if err := h.handleCommand(context.Background(), updateWithCommand("/delete")); err != nil {
			t.Fatalf("handleCommand: %v", err)
		}
		if store.lastLimit != 50 {
			t.Fatalf("limit = %d, want 50", store.lastLimit)
		}
		if len(bot.sent) == 0 || !strings.Contains(bot.sent[len(bot.sent)-1], "Select entry to delete") {
			t.Fatalf("unexpected response: %+v", bot.sent)
		}
	})

	t.Run("/report 2026-03 parses year month", func(t *testing.T) {
		store := &fakeStore{}
		bot := &fakeBot{}
		h := newTestHandler(store, bot)

		if err := h.handleCommand(context.Background(), updateWithCommand("/report 2026-03")); err != nil {
			t.Fatalf("handleCommand: %v", err)
		}

		wantFrom := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
		wantTo := wantFrom.AddDate(0, 1, 0)
		if !store.from.Equal(wantFrom) || !store.to.Equal(wantTo) {
			t.Fatalf("range = %s - %s, want %s - %s", store.from, store.to, wantFrom, wantTo)
		}
	})

	t.Run("unknown command replies with help hint", func(t *testing.T) {
		store := &fakeStore{}
		bot := &fakeBot{}
		h := newTestHandler(store, bot)

		if err := h.handleCommand(context.Background(), updateWithCommand("/unknown")); err != nil {
			t.Fatalf("handleCommand: %v", err)
		}

		if len(bot.sent) == 0 || !strings.Contains(bot.sent[len(bot.sent)-1], "Unknown command. Use /help") {
			t.Fatalf("unexpected response: %+v", bot.sent)
		}
	})
}
