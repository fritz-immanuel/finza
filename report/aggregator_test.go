package report

import (
	"testing"
	"time"

	"github.com/yourusername/moneytracker/domain"
)

func TestAggregateTotalsByTypeAndCategory(t *testing.T) {
	entries := []domain.Entry{
		{Type: domain.EntryTypeExpense, Currency: "IDR", Amount: 10000, Category: "Food & Drink"},
		{Type: domain.EntryTypeExpense, Currency: "IDR", Amount: 25000, Category: "Transport"},
		{Type: domain.EntryTypeIncome, Currency: "IDR", Amount: 1000000, Category: "Income"},
		{Type: domain.EntryTypeTransfer, Currency: "IDR", Amount: 50000, Category: "Transfer"},
	}

	got := Aggregate(entries)

	if got.ExpenseByCurrency["IDR"] != 35000 {
		t.Fatalf("expense total = %d, want 35000", got.ExpenseByCurrency["IDR"])
	}
	if got.IncomeByCurrency["IDR"] != 1000000 {
		t.Fatalf("income total = %d, want 1000000", got.IncomeByCurrency["IDR"])
	}
	if got.TransferByCurrency["IDR"] != 50000 {
		t.Fatalf("transfer total = %d, want 50000", got.TransferByCurrency["IDR"])
	}
	if got.CategoryTotals["Food & Drink"] != 10000 {
		t.Fatalf("category Food & Drink total mismatch")
	}
}

func TestAggregateEmptyRange(t *testing.T) {
	got := Aggregate(nil)
	if len(got.CategoryTotals) != 0 {
		t.Fatalf("expected empty totals")
	}
}

func TestAggregateMixedCurrencies(t *testing.T) {
	entries := []domain.Entry{
		{Type: domain.EntryTypeExpense, Currency: "IDR", Amount: 1000, Category: "General", Timestamp: time.Now()},
		{Type: domain.EntryTypeExpense, Currency: "USD", Amount: 500, Category: "General", Timestamp: time.Now()},
	}

	got := Aggregate(entries)
	if got.ExpenseByCurrency["IDR"] != 1000 {
		t.Fatalf("IDR total mismatch")
	}
	if got.ExpenseByCurrency["USD"] != 500 {
		t.Fatalf("USD total mismatch")
	}
}
