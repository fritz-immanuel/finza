package app

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/moneytracker/domain"
	reportpkg "github.com/yourusername/moneytracker/report"
)

type ReportService struct {
	store domain.Store
	clock domain.Clock
}

func NewReportService(store domain.Store, clock domain.Clock) *ReportService {
	return &ReportService{store: store, clock: clock}
}

func (s *ReportService) Summary(ctx context.Context, userID int64, title string, from, to time.Time, loc *time.Location) (string, error) {
	entries, err := s.store.GetEntriesByDateRange(ctx, userID, from.UTC(), to.UTC())
	if err != nil {
		return "", fmt.Errorf("store.GetEntriesByDateRange: %w", err)
	}
	totals := reportpkg.Aggregate(entries)
	return reportpkg.FormatSummary(title, s.clock.Now().In(loc), totals), nil
}

func (s *ReportService) Last(ctx context.Context, userID int64, n int, loc *time.Location) ([]domain.Entry, error) {
	return s.store.GetLastEntries(ctx, userID, n)
}

func (s *ReportService) Categories(ctx context.Context, userID int64, from, to time.Time) (map[string]int64, error) {
	totals, err := s.store.GetCategoryTotals(ctx, userID, from.UTC(), to.UTC())
	if err != nil {
		return nil, fmt.Errorf("store.GetCategoryTotals: %w", err)
	}
	return totals, nil
}

func (s *ReportService) ExportCSV(ctx context.Context, userID int64) ([]byte, error) {
	from := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	to := s.clock.Now().UTC().AddDate(30, 0, 0)
	entries, err := s.store.GetEntriesByDateRange(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("store.GetEntriesByDateRange: %w", err)
	}
	data, err := reportpkg.BuildCSV(entries)
	if err != nil {
		return nil, fmt.Errorf("report.BuildCSV: %w", err)
	}
	return data, nil
}
