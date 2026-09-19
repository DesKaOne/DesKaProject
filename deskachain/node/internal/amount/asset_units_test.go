package amount

import "testing"

func TestParseUnitsAndFormatUnits(t *testing.T) {
	tests := []struct {
		input    string
		decimals uint8
		units    uint64
		formatted string
	}{
		{"1", 0, 1, "1"},
		{"1000", 0, 1000, "1000"},
		{"1.25", 2, 125, "1.25"},
		{"0.000001", 6, 1, "0.000001"},
		{"25.50", 2, 2550, "25.5"},
	}
	for _, tt := range tests {
		got, err := ParseUnits(tt.input, tt.decimals)
		if err != nil {
			t.Fatalf("ParseUnits(%q,%d): %v", tt.input, tt.decimals, err)
		}
		if got != tt.units {
			t.Fatalf("ParseUnits(%q,%d)=%d want %d", tt.input, tt.decimals, got, tt.units)
		}
		if got := FormatUnits(tt.units, tt.decimals); got != tt.formatted {
			t.Fatalf("FormatUnits(%d,%d)=%q want %q", tt.units, tt.decimals, got, tt.formatted)
		}
	}
}

func TestParseUnitsRejectsExcessPrecision(t *testing.T) {
	if _, err := ParseUnits("1.001", 2); err == nil {
		t.Fatal("expected excess precision rejection")
	}
}
