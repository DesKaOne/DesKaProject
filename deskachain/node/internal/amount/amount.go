package amount

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"deskachain/internal/config"
)

func Parse(value string) (uint64, error) {
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
	if wholeUnits > (^uint64(0))/config.UnitsPerCoin {
		return 0, errors.New("amount overflows uint64")
	}
	total := wholeUnits * config.UnitsPerCoin
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > config.Decimals {
			return 0, fmt.Errorf("amount has more than %d decimals", config.Decimals)
		}
		for len(fraction) < config.Decimals {
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
	whole := units / config.UnitsPerCoin
	fraction := units % config.UnitsPerCoin
	value := fmt.Sprintf("%d.%08d", whole, fraction)
	return strings.TrimRight(strings.TrimRight(value, "0"), ".")
}
