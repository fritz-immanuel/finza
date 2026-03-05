package rule

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var nonAmountChars = regexp.MustCompile(`[^0-9.,kKmMrRbBjJtTuUaA]`)

func NormalizeAmount(raw, currency string) (int64, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "rp", "")
	s = strings.ReplaceAll(s, "idr", "")
	s = strings.ReplaceAll(s, "usd", "")
	s = strings.ReplaceAll(s, "sgd", "")
	s = strings.ReplaceAll(s, "eur", "")
	s = strings.ReplaceAll(s, "gbp", "")
	s = strings.ReplaceAll(s, "jpy", "")
	s = strings.ReplaceAll(s, "myr", "")
	s = strings.ReplaceAll(s, "thb", "")
	s = strings.ReplaceAll(s, "s$", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "¥", "")
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "£", "")
	s = strings.ReplaceAll(s, "฿", "")
	s = nonAmountChars.ReplaceAllString(s, "")

	if s == "" {
		return 0, fmt.Errorf("empty amount")
	}

	multiplier := 1.0
	switch {
	case strings.HasSuffix(s, "rb") || strings.HasSuffix(s, "ribu") || strings.HasSuffix(s, "k"):
		multiplier = 1_000
		s = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "ribu"), "rb"), "k")
	case strings.HasSuffix(s, "jt") || strings.HasSuffix(s, "juta") || strings.HasSuffix(s, "m"):
		multiplier = 1_000_000
		s = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(s, "juta"), "jt"), "m")
	}

	value, err := parseNumber(s)
	if err != nil {
		return 0, err
	}

	major := value * multiplier
	if usesMinorUnits(currency) {
		major = major * 100
	}

	return int64(math.Round(major)), nil
}

func parseNumber(s string) (float64, error) {
	if strings.Count(s, ".") > 1 && !strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
	}
	if strings.Count(s, ",") > 1 && !strings.Contains(s, ".") {
		s = strings.ReplaceAll(s, ",", "")
	}

	if strings.Contains(s, ".") && strings.Contains(s, ",") {
		lastDot := strings.LastIndex(s, ".")
		lastComma := strings.LastIndex(s, ",")
		if lastDot > lastComma {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		}
	} else if strings.Contains(s, ".") {
		idx := strings.LastIndex(s, ".")
		tail := len(s) - idx - 1
		if tail == 3 {
			s = strings.ReplaceAll(s, ".", "")
		}
	} else if strings.Contains(s, ",") {
		idx := strings.LastIndex(s, ",")
		tail := len(s) - idx - 1
		if tail == 3 {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.ReplaceAll(s, ",", ".")
		}
	}

	return strconv.ParseFloat(s, 64)
}

func usesMinorUnits(currency string) bool {
	switch strings.ToUpper(currency) {
	case "USD", "EUR", "SGD", "GBP":
		return true
	default:
		return false
	}
}
