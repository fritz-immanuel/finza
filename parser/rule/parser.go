package rule

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/yourusername/moneytracker/domain"
)

type RuleParser struct{}

func NewParser() *RuleParser {
	return &RuleParser{}
}

var amountPattern = regexp.MustCompile(`(?i)(?:rp|idr|usd|sgd|eur|gbp|jpy|myr|thb|s\$|rm|\$|¥|€|£|฿)?\s*\d+(?:[\.,]\d+)?(?:\s*(?:k|rb|ribu|jt|juta|m))?`)

func (p *RuleParser) Parse(_ context.Context, text string) (domain.Entry, bool, float64, error) {
	clean := strings.TrimSpace(text)
	currency, hasCurrency := DetectCurrency(clean)

	amountRaw := amountPattern.FindString(clean)
	if amountRaw == "" {
		return domain.Entry{}, false, 0, nil
	}

	amount, err := NormalizeAmount(amountRaw, currency)
	if err != nil || amount <= 0 {
		return domain.Entry{}, false, 0, nil
	}

	income, expense, transfer := scoreKeywords(clean)
	entryType := resolveType(income, expense, transfer)
	category := inferCategory(clean, entryType)

	keywordMatched := income > 0 || expense > 0 || transfer > 0
	confidence := 0.50
	if keywordMatched {
		confidence = 0.75
	}
	if hasCurrency && keywordMatched {
		confidence = 0.95
	}

	entry := domain.Entry{
		Timestamp:   time.Now().UTC(),
		Amount:      amount,
		Currency:    currency,
		Type:        entryType,
		Category:    category,
		Description: inferDescription(clean),
		RawText:     clean,
		Confidence:  confidence,
	}

	return entry, true, confidence, nil
}

func scoreKeywords(text string) (int, int, int) {
	s := strings.ToLower(text)

	incomeWords := []string{"salary", "gaji", "bonus", "income", "received", "paid me", "dividend", "interest", "thr", "komisi", "pendapatan", "refund"}
	expenseWords := []string{"bought", "paid", "spent", "beli", "bayar", "makan", "ngopi", "lunch", "dinner", "breakfast", "grab", "gojek", "gocar", "shopee", "tokopedia", "lazada", "coffee", "snack", "belanja"}
	transferWords := []string{"transfer", "tf", "send", "kirim", "tarik", "withdraw", "top up", "topup", "deposit"}

	income, expense, transfer := 0, 0, 0
	for _, w := range incomeWords {
		if strings.Contains(s, w) {
			if w == "paid me" {
				income += 4
			} else {
				income += 2
			}
		}
	}
	for _, w := range expenseWords {
		if strings.Contains(s, w) {
			expense += 2
		}
	}
	for _, w := range transferWords {
		if strings.Contains(s, w) {
			transfer += 3
		}
	}

	return income, expense, transfer
}

func resolveType(income, expense, transfer int) domain.EntryType {
	if transfer >= income+1 && transfer >= expense+1 {
		return domain.EntryTypeTransfer
	}
	if income-expense >= 1 {
		return domain.EntryTypeIncome
	}
	if expense-income >= 1 {
		return domain.EntryTypeExpense
	}
	return domain.EntryTypeUnknown
}

func inferCategory(text string, entryType domain.EntryType) string {
	s := strings.ToLower(text)

	switch {
	case containsAny(s, "coffee", "ngopi", "cafe", "makan", "lunch", "dinner", "breakfast", "resto", "warung"):
		return "Food & Drink"
	case containsAny(s, "grab", "gojek", "gocar", "taxi", "commute", "busway", "mrt", "krl"):
		return "Transport"
	case containsAny(s, "shopee", "tokopedia", "lazada", "mall", "belanja"):
		return "Shopping"
	case containsAny(s, "listrik", "pln", "air", "wifi", "internet", "pulsa", "token"):
		return "Utilities"
	case containsAny(s, "gym", "dokter", "apotik", "obat", "rumah sakit"):
		return "Health"
	case containsAny(s, "kos", "kontrakan", "rent", "sewa"):
		return "Housing"
	case containsAny(s, "salary", "gaji", "bonus", "thr"):
		return "Income"
	case containsAny(s, "transfer", "tf", "topup", "top up"):
		return "Transfer"
	default:
		if entryType == domain.EntryTypeIncome {
			return "Income"
		}
		if entryType == domain.EntryTypeTransfer {
			return "Transfer"
		}
		return "General"
	}
}

func containsAny(s string, words ...string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

func inferDescription(text string) string {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return "Transaction"
	}
	parts := strings.Fields(clean)
	if len(parts) > 5 {
		parts = parts[:5]
	}
	return strings.Join(parts, " ")
}
