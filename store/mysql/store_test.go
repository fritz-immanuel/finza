package mysql

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yourusername/moneytracker/domain"
)

func TestSaveEntryRoundTrip(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	now := time.Now().UTC()

	entry := domain.Entry{
		UserID:      1,
		ChatID:      2,
		MessageID:   3,
		Timestamp:   now,
		Amount:      50000,
		Currency:    "IDR",
		Type:        domain.EntryTypeExpense,
		Category:    "Food & Drink",
		Description: "coffee",
		RawText:     "coffee 50k",
		Confidence:  0.75,
	}

	mock.ExpectExec(regexp.QuoteMeta(qSaveEntry)).
		WithArgs(entry.UserID, entry.ChatID, entry.MessageID, entry.Timestamp, entry.Amount, entry.Currency, string(entry.Type), entry.Category, entry.Description, entry.RawText, entry.Confidence).
		WillReturnResult(sqlmock.NewResult(42, 1))

	id, err := store.SaveEntry(context.Background(), entry)
	if err != nil {
		t.Fatalf("SaveEntry: %v", err)
	}
	if id != 42 {
		t.Fatalf("SaveEntry id = %d, want 42", id)
	}

	rows := sqlmock.NewRows([]string{"id", "user_id", "chat_id", "message_id", "timestamp", "amount", "currency", "type", "category", "description", "raw_text", "confidence", "created_at"}).
		AddRow(42, 1, 2, 3, now, 50000, "IDR", "expense", "Food & Drink", "coffee", "coffee 50k", 0.75, now)

	mock.ExpectQuery(regexp.QuoteMeta(qGetEntryByID)).WithArgs(int64(42)).WillReturnRows(rows)

	got, ok, err := store.GetEntryByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetEntryByID: %v", err)
	}
	if !ok || got.ID != 42 || got.Amount != 50000 {
		t.Fatalf("GetEntryByID mismatch: %+v", got)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestGetEntriesByDateRange(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	store := NewStore(db)
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	now := from.Add(2 * time.Hour)

	rows := sqlmock.NewRows([]string{"id", "user_id", "chat_id", "message_id", "timestamp", "amount", "currency", "type", "category", "description", "raw_text", "confidence", "created_at"}).
		AddRow(1, 1, 1, 10, now, 28000, "IDR", "expense", "Transport", "Grab", "grab 28k", 0.9, now)

	mock.ExpectQuery(regexp.QuoteMeta(qGetEntriesByDateRange)).
		WithArgs(int64(1), from, to).
		WillReturnRows(rows)

	got, err := store.GetEntriesByDateRange(context.Background(), 1, from, to)
	if err != nil {
		t.Fatalf("GetEntriesByDateRange: %v", err)
	}
	if len(got) != 1 || got[0].Category != "Transport" {
		t.Fatalf("unexpected entries: %+v", got)
	}
}

func TestGetCategoryTotals(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	store := NewStore(db)
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)

	rows := sqlmock.NewRows([]string{"category", "total"}).
		AddRow("Food & Drink", 75000).
		AddRow("Transport", 45000)

	mock.ExpectQuery(regexp.QuoteMeta(qGetCategoryTotals)).
		WithArgs(int64(1), from, to).
		WillReturnRows(rows)

	got, err := store.GetCategoryTotals(context.Background(), 1, from, to)
	if err != nil {
		t.Fatalf("GetCategoryTotals: %v", err)
	}
	if got["Food & Drink"] != 75000 || got["Transport"] != 45000 {
		t.Fatalf("unexpected totals: %+v", got)
	}
}

func TestUpsertUserIdempotency(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	store := NewStore(db)
	user := domain.User{ID: 1, Username: "u", FirstName: "f", LastName: "l", Timezone: "Asia/Jakarta"}

	mock.ExpectExec(regexp.QuoteMeta(qUpsertUser)).
		WithArgs(user.ID, user.Username, user.FirstName, user.LastName, user.Timezone).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(qUpsertUser)).
		WithArgs(user.ID, user.Username, user.FirstName, user.LastName, user.Timezone).
		WillReturnResult(sqlmock.NewResult(1, 1))

	if err := store.UpsertUser(context.Background(), user); err != nil {
		t.Fatalf("first UpsertUser: %v", err)
	}
	if err := store.UpsertUser(context.Background(), user); err != nil {
		t.Fatalf("second UpsertUser: %v", err)
	}
}
