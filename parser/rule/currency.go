package rule

import "strings"

func DetectCurrency(text string) (string, bool) {
	s := strings.ToLower(text)
	checks := []struct {
		needle string
		code   string
	}{
		{"rp", "IDR"},
		{"idr", "IDR"},
		{"s$", "SGD"},
		{"sgd", "SGD"},
		{"$", "USD"},
		{"usd", "USD"},
		{"¥", "JPY"},
		{"jpy", "JPY"},
		{"€", "EUR"},
		{"eur", "EUR"},
		{"£", "GBP"},
		{"gbp", "GBP"},
		{"rm", "MYR"},
		{"myr", "MYR"},
		{"฿", "THB"},
		{"thb", "THB"},
	}

	for _, c := range checks {
		if strings.Contains(s, c.needle) {
			return c.code, true
		}
	}
	return "IDR", false
}
