package asset

import (
	"errors"
	"fmt"
	"strings"

	"indochain/internal/config"
)

const (
	NativeAssetID        = config.NativeAssetID
	NativeSymbol         = config.NativeAssetSymbol
	NativeDecimals uint8 = config.NativeAssetDecimals
	KindFungible         = "fungible"
	StatusActive         = "active"
	StatusFrozen         = "frozen"
)

// Definition describes an issued fungible asset. Stablecoin backing and
// redemption are external to chain consensus.
type Definition struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	Decimals     uint8  `json:"decimals"`
	Kind         string `json:"kind"`
	Issuer       string `json:"issuer"`
	MaxSupply    uint64 `json:"max_supply,omitempty"`
	TotalSupply  uint64 `json:"total_supply"`
	Mintable     bool   `json:"mintable"`
	Burnable     bool   `json:"burnable"`
	Pausable     bool   `json:"pausable"`
	Permissioned bool   `json:"permissioned"`
	Status       string `json:"status"`
}

func ValidateDefinition(d Definition) error {
	if strings.TrimSpace(d.ID) == "" || strings.EqualFold(d.ID, NativeAssetID) {
		return errors.New("invalid issued asset id")
	}
	if strings.TrimSpace(d.Name) == "" {
		return errors.New("asset name is required")
	}
	symbol := strings.TrimSpace(d.Symbol)
	if len(symbol) == 0 || len(symbol) > 16 {
		return errors.New("asset symbol must be 1..16 characters")
	}
	if d.Decimals > 18 {
		return fmt.Errorf("asset decimals too large: %d", d.Decimals)
	}
	if d.Kind != "" && d.Kind != KindFungible {
		return errors.New("unsupported asset kind")
	}
	if strings.TrimSpace(d.Issuer) == "" {
		return errors.New("asset issuer is required")
	}
	if d.MaxSupply > 0 && d.TotalSupply > d.MaxSupply {
		return errors.New("asset total supply exceeds max supply")
	}
	if d.Status == "" {
		d.Status = StatusActive
	}
	if d.Status != StatusActive && d.Status != StatusFrozen {
		return errors.New("invalid asset status")
	}
	return nil
}

// DerivedID deterministically binds an issued asset to its create transaction.
func DerivedID(txID string) string {
	return "asset:" + strings.ToLower(strings.TrimSpace(txID))
}

func IsNative(id string) bool {
	return strings.EqualFold(strings.TrimSpace(id), NativeAssetID)
}
