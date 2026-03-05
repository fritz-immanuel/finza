package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yourusername/moneytracker/domain"
)

type MySQLStore struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) UpsertUser(ctx context.Context, user domain.User) error {
	if user.Timezone == "" {
		user.Timezone = "Asia/Jakarta"
	}

	_, err := s.db.ExecContext(ctx, qUpsertUser,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.Timezone,
	)
	if err != nil {
		return fmt.Errorf("UpsertUser exec: %w", err)
	}
	return nil
}

func (s *MySQLStore) GetUser(ctx context.Context, userID int64) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRowContext(ctx, qGetUser, userID).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.Timezone,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return domain.User{}, fmt.Errorf("GetUser scan: %w", err)
	}
	return user, nil
}

func (s *MySQLStore) SaveMessage(ctx context.Context, msg domain.RawMessage) (int64, error) {
	res, err := s.db.ExecContext(ctx, qSaveMessage,
		msg.UserID,
		msg.ChatID,
		msg.MessageID,
		msg.Text,
		msg.CreatedAt,
	)
	if err != nil {
		return 0, fmt.Errorf("SaveMessage exec: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("SaveMessage LastInsertId: %w", err)
	}
	return id, nil
}

func (s *MySQLStore) SaveEntry(ctx context.Context, entry domain.Entry) (int64, error) {
	res, err := s.db.ExecContext(ctx, qSaveEntry,
		entry.UserID,
		entry.ChatID,
		entry.MessageID,
		entry.Timestamp,
		entry.Amount,
		entry.Currency,
		string(entry.Type),
		entry.Category,
		entry.Description,
		entry.RawText,
		entry.Confidence,
	)
	if err != nil {
		return 0, fmt.Errorf("SaveEntry exec: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("SaveEntry LastInsertId: %w", err)
	}
	return id, nil
}

func (s *MySQLStore) UpdateEntry(ctx context.Context, entry domain.Entry) error {
	_, err := s.db.ExecContext(ctx, qUpdateEntry,
		entry.Amount,
		entry.Currency,
		string(entry.Type),
		entry.Category,
		entry.Description,
		entry.RawText,
		entry.Confidence,
		entry.ID,
	)
	if err != nil {
		return fmt.Errorf("UpdateEntry exec: %w", err)
	}
	return nil
}

func (s *MySQLStore) DeleteEntry(ctx context.Context, entryID int64) error {
	_, err := s.db.ExecContext(ctx, qDeleteEntry, entryID)
	if err != nil {
		return fmt.Errorf("DeleteEntry exec: %w", err)
	}
	return nil
}

func (s *MySQLStore) GetEntryByID(ctx context.Context, entryID int64) (domain.Entry, bool, error) {
	row := s.db.QueryRowContext(ctx, qGetEntryByID, entryID)
	entry, ok, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return domain.Entry{}, false, nil
	}
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("GetEntryByID scan: %w", err)
	}
	return entry, ok, nil
}

func (s *MySQLStore) GetEntryByMessageID(ctx context.Context, userID int64, messageID int) (domain.Entry, bool, error) {
	row := s.db.QueryRowContext(ctx, qGetEntryByMessageID, userID, messageID)
	entry, ok, err := scanEntry(row)
	if err == sql.ErrNoRows {
		return domain.Entry{}, false, nil
	}
	if err != nil {
		return domain.Entry{}, false, fmt.Errorf("GetEntryByMessageID scan: %w", err)
	}
	return entry, ok, nil
}

func (s *MySQLStore) GetLastEntries(ctx context.Context, userID int64, limit int) ([]domain.Entry, error) {
	rows, err := s.db.QueryContext(ctx, qGetLastEntries, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("GetLastEntries query: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

func (s *MySQLStore) GetEntriesByDateRange(ctx context.Context, userID int64, from, to time.Time) ([]domain.Entry, error) {
	rows, err := s.db.QueryContext(ctx, qGetEntriesByDateRange, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("GetEntriesByDateRange query: %w", err)
	}
	defer rows.Close()

	return scanEntries(rows)
}

func (s *MySQLStore) GetCategoryTotals(ctx context.Context, userID int64, from, to time.Time) (map[string]int64, error) {
	rows, err := s.db.QueryContext(ctx, qGetCategoryTotals, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("GetCategoryTotals query: %w", err)
	}
	defer rows.Close()

	out := make(map[string]int64)
	for rows.Next() {
		var category string
		var total int64
		if err := rows.Scan(&category, &total); err != nil {
			return nil, fmt.Errorf("GetCategoryTotals scan: %w", err)
		}
		out[category] = total
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetCategoryTotals rows: %w", err)
	}

	return out, nil
}

type scanner interface {
	Scan(dest ...interface{}) error
}

type rowsScanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Err() error
}

func scanEntry(s scanner) (domain.Entry, bool, error) {
	var entry domain.Entry
	var typ string
	if err := s.Scan(
		&entry.ID,
		&entry.UserID,
		&entry.ChatID,
		&entry.MessageID,
		&entry.Timestamp,
		&entry.Amount,
		&entry.Currency,
		&typ,
		&entry.Category,
		&entry.Description,
		&entry.RawText,
		&entry.Confidence,
		&entry.CreatedAt,
	); err != nil {
		return domain.Entry{}, false, err
	}
	entry.Type = domain.EntryType(typ)
	return entry, true, nil
}

func scanEntries(rows rowsScanner) ([]domain.Entry, error) {
	entries := make([]domain.Entry, 0)
	for rows.Next() {
		var entry domain.Entry
		var typ string
		if err := rows.Scan(
			&entry.ID,
			&entry.UserID,
			&entry.ChatID,
			&entry.MessageID,
			&entry.Timestamp,
			&entry.Amount,
			&entry.Currency,
			&typ,
			&entry.Category,
			&entry.Description,
			&entry.RawText,
			&entry.Confidence,
			&entry.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanEntries scan: %w", err)
		}
		entry.Type = domain.EntryType(typ)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanEntries rows: %w", err)
	}
	return entries, nil
}
