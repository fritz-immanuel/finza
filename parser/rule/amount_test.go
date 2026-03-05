package rule

import "testing"

func TestNormalizeAmount(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		currency string
		want     int64
	}{
		{name: "k suffix", raw: "75k", currency: "IDR", want: 75000},
		{name: "rb suffix", raw: "75rb", currency: "IDR", want: 75000},
		{name: "thousand dot", raw: "75.000", currency: "IDR", want: 75000},
		{name: "jt suffix", raw: "10jt", currency: "IDR", want: 10000000},
		{name: "juta suffix", raw: "10juta", currency: "IDR", want: 10000000},
		{name: "m suffix", raw: "10m", currency: "IDR", want: 10000000},
		{name: "decimal juta", raw: "1.5jt", currency: "IDR", want: 1500000},
		{name: "decimal k", raw: "75.5k", currency: "IDR", want: 75500},
		{name: "plain", raw: "75000", currency: "IDR", want: 75000},
		{name: "usd cents", raw: "$12", currency: "USD", want: 1200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeAmount(tt.raw, tt.currency)
			if err != nil {
				t.Fatalf("NormalizeAmount() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeAmount() = %d, want %d", got, tt.want)
			}
		})
	}
}
