package amount

import (
	"testing"

	"indochain/internal/config"
)

func TestParseHandlesDecimalsSafely(t *testing.T) {
	valid := map[string]uint64{
		"1":          config.UnitsPerCoin,
		"1.0":        config.UnitsPerCoin,
		"1.00000000": config.UnitsPerCoin,
		"1.25":       125000000,
		"0.00000001": 1,
	}
	for input, want := range valid {
		got, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q) returned error: %v", input, err)
		}
		if got != want {
			t.Fatalf("Parse(%q) = %d, want %d", input, got, want)
		}
	}
	invalid := []string{"", "-1", "abc", "1.2.3", "0", "0.000000001", "1e3"}
	for _, input := range invalid {
		if _, err := Parse(input); err == nil {
			t.Fatalf("Parse(%q) unexpectedly succeeded", input)
		}
	}
}
