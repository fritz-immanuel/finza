package rule

import (
	"context"
	"testing"

	"github.com/yourusername/moneytracker/domain"
)

func TestRuleParser_Parse(t *testing.T) {
	p := NewParser()

	tests := []struct {
		input        string
		wantType     domain.EntryType
		wantAmount   int64
		wantCurrency string
		wantOK       bool
	}{
		{"coffee 50k", domain.EntryTypeExpense, 50000, "IDR", true},
		{"salary 10jt", domain.EntryTypeIncome, 10000000, "IDR", true},
		{"grab 28k", domain.EntryTypeExpense, 28000, "IDR", true},
		{"dinner Rp 75.000", domain.EntryTypeExpense, 75000, "IDR", true},
		{"refund 50k", domain.EntryTypeIncome, 50000, "IDR", true},
		{"paid me 200k", domain.EntryTypeIncome, 200000, "IDR", true},
		{"transfer 500k to budi", domain.EntryTypeTransfer, 500000, "IDR", true},
		{"lunch $12", domain.EntryTypeExpense, 1200, "USD", true},
		{"bonus ¥5000", domain.EntryTypeIncome, 5000, "JPY", true},
		{"topup gopay 100rb", domain.EntryTypeTransfer, 100000, "IDR", true},
		{"hello world", "", 0, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok, _, err := p.Parse(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("Parse() ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got.Type != tt.wantType {
				t.Fatalf("type = %s, want %s", got.Type, tt.wantType)
			}
			if got.Amount != tt.wantAmount {
				t.Fatalf("amount = %d, want %d", got.Amount, tt.wantAmount)
			}
			if got.Currency != tt.wantCurrency {
				t.Fatalf("currency = %s, want %s", got.Currency, tt.wantCurrency)
			}
		})
	}
}
