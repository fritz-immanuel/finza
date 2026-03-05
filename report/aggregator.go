package report

import (
	"sort"

	"github.com/yourusername/moneytracker/domain"
)

type Totals struct {
	ExpenseByCurrency  map[string]int64
	IncomeByCurrency   map[string]int64
	TransferByCurrency map[string]int64
	CategoryTotals     map[string]int64
}

func Aggregate(entries []domain.Entry) Totals {
	t := Totals{
		ExpenseByCurrency:  map[string]int64{},
		IncomeByCurrency:   map[string]int64{},
		TransferByCurrency: map[string]int64{},
		CategoryTotals:     map[string]int64{},
	}

	for _, e := range entries {
		switch e.Type {
		case domain.EntryTypeExpense:
			t.ExpenseByCurrency[e.Currency] += e.Amount
		case domain.EntryTypeIncome:
			t.IncomeByCurrency[e.Currency] += e.Amount
		case domain.EntryTypeTransfer:
			t.TransferByCurrency[e.Currency] += e.Amount
		}
		t.CategoryTotals[e.Category] += e.Amount
	}

	return t
}

func SortedCategories(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
