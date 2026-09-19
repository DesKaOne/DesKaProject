package amount

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"deskachain/internal/config"
)

func Parse(value string) (uint64, error) {
	return ParseUnits(value, config.Decimals)
}

// ParseUnits parses a human-readable asset amount using its decimal scale.
func ParseUnits(value string, decimals uint8) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("amount is required")
	}
	if strings.HasPrefix(value, "-") {
		return 0, errors.New("amount cannot be negative")
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, errors.New("invalid decimal amount")
	}
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	wholeUnits, err := strconv.ParseUint(whole, 10, 64)
	if err != nil {
		return 0, err
	}
	unitsPerWhole := uint64(1)
	for i := uint8(0); i < decimals; i++ {
		unitsPerWhole *= 10
	}
	if wholeUnits > (^uint64(0))/unitsPerWhole {
		return 0, errors.New("amount overflows uint64")
	}
	total := wholeUnits * unitsPerWhole
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > int(decimals) {
			return 0, fmt.Errorf("amount has more than %d decimals", decimals)
		}
		for len(fraction) < int(decimals) {
			fraction += "0"
		}
		fracUnits, err := strconv.ParseUint(fraction, 10, 64)
		if err != nil {
			return 0, err
		}
		if ^uint64(0)-total < fracUnits {
			return 0, errors.New("amount overflows uint64")
		}
		total += fracUnits
	}
	if total == 0 {
		return 0, errors.New("amount must be greater than zero")
	}
	return total, nil
}

func Format(units uint64) string {
	return FormatUnits(units, config.Decimals)
}

// FormatUnits renders integer units according to an asset's decimal scale.
func FormatUnits(units uint64, decimals uint8) string {
	unitsPerWhole := uint64(1)
	for i := uint8(0); i < decimals; i++ {
		unitsPerWhole *= 10
	}
	if decimals == 0 {
		return fmt.Sprintf("%d", units)
	}
	whole := units / unitsPerWhole
	fraction := units % unitsPerWhole
	value := fmt.Sprintf("%d.%0*d", whole, int(decimals), fraction)
	return strings.TrimRight(strings.TrimRight(value, "0"), ".")
}
