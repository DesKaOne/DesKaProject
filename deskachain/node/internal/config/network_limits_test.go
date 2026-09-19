package config

import "testing"

func TestNetworkLimitsAreSeparateFromConsensusLimits(t *testing.T) {
	for _, profile := range []NetworkConfig{Localnet(), Testnet(), Mainnet()} {
		if profile.NetworkLimits.MaxHeaderBatch == 0 || profile.NetworkLimits.MaxSyncBlocks == 0 || profile.NetworkLimits.MaxReorgFetchBlocks == 0 {
			t.Fatalf("%s network limits not initialized: %#v", profile.Name, profile.NetworkLimits)
		}
		if profile.Consensus.MaxBlockBytes == 0 || profile.Consensus.MaxTxBytes == 0 || profile.Consensus.MaxTxCount == 0 {
			t.Fatalf("%s consensus resource limits not initialized", profile.Name)
		}
	}
}
