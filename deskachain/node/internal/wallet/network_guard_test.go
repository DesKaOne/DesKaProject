package wallet

import (
	"testing"
	"indochain/internal/config"
)

func TestRequireTestnet(t *testing.T) {
	if err := RequireTestnet(config.Testnet()); err != nil { t.Fatal(err) }
	if err := RequireTestnet(config.Localnet()); err == nil { t.Fatal("localnet unexpectedly accepted as testnet") }
	if err := RequireTestnet(config.Mainnet()); err == nil { t.Fatal("mainnet unexpectedly accepted as testnet") }
}
