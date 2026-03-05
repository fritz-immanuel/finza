package report

import (
	"fmt"
	"strings"
	"time"
)

func FormatAmount(currency string, amount int64) string {
	major := amount
	suffix := ""
	switch currency {
	case "USD", "EUR", "SGD", "GBP":
		major = amount / 100
		cents := amount % 100
		suffix = fmt.Sprintf(".%02d", cents)
	}

	return fmt.Sprintf("%s %s%s", currencySymbol(currency), formatThousands(major), suffix)
}

func currencySymbol(code string) string {
	switch strings.ToUpper(code) {
	case "IDR":
		return "Rp"
	case "USD":
		return "$"
	case "JPY":
		return "¥"
	case "SGD":
		return "S$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "MYR":
		return "RM"
	case "THB":
		return "฿"
	default:
		return strings.ToUpper(code)
	}
}

func formatThousands(v int64) string {
	s := fmt.Sprintf("%d", v)
	if len(s) <= 3 {
		return s
	}
	var out []byte
	rem := len(s) % 3
	if rem > 0 {
		out = append(out, s[:rem]...)
		if len(s) > rem {
			out = append(out, ',')
		}
	}
	for i := rem; i < len(s); i += 3 {
		out = append(out, s[i:i+3]...)
		if i+3 < len(s) {
			out = append(out, ',')
		}
	}
	return string(out)
}

func FormatSummary(title string, date time.Time, totals Totals) string {
	expense := totals.ExpenseByCurrency["IDR"]
	income := totals.IncomeByCurrency["IDR"]
	transfer := totals.TransferByCurrency["IDR"]

	var b strings.Builder
	b.WriteString(fmt.Sprintf("📊 %s — %s\n\n", title, date.Format("Mon, 02 Jan 2006")))
	b.WriteString(fmt.Sprintf("💸 Expenses:    %s\n", FormatAmount("IDR", expense)))
	b.WriteString(fmt.Sprintf("💰 Income:      %s\n", FormatAmount("IDR", income)))
	b.WriteString(fmt.Sprintf("🔄 Transfer:    %s\n", FormatAmount("IDR", transfer)))
	b.WriteString("─────────────────────\n")
	b.WriteString("📁 Breakdown:\n")

	for _, k := range SortedCategories(totals.CategoryTotals) {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", k, FormatAmount("IDR", totals.CategoryTotals[k])))
	}
	return strings.TrimSpace(b.String())
}
