package wallet

import (
	"fmt"
	"indochain/internal/config"
)

func RequireNetwork(profile config.NetworkConfig, expectedName, expectedNetworkID string, expectedChainID uint64) error {
	if expectedName == "" || expectedNetworkID == "" || expectedChainID == 0 {
		return fmt.Errorf("expected wallet network identity is incomplete")
	}
	if profile.Name != expectedName || profile.NetworkID != expectedNetworkID || profile.ChainID != expectedChainID {
		return fmt.Errorf("wallet network mismatch: selected=%s/%s/%d expected=%s/%s/%d",
			profile.Name, profile.NetworkID, profile.ChainID,
			expectedName, expectedNetworkID, expectedChainID)
	}
	return nil
}

func RequireTestnet(profile config.NetworkConfig) error {
	testnet := config.Testnet()
	return RequireNetwork(profile, testnet.Name, testnet.NetworkID, testnet.ChainID)
}
