package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	CoinName = "IndoChain"
	// Ticker/Decimals/UnitsPerCoin are retained for legacy v1/v2 compatibility.
	// v3 uses the explicit native asset model below.
	Ticker   = "dIDR"
	Decimals = 8

	NativeAssetID             = "dIDR"
	NativeAssetSymbol         = "dIDR"
	NativeAssetDecimals uint8 = Decimals
	FeeAssetID                = NativeAssetID

	UnitsPerCoin         uint64 = 100000000
	InitialBlockReward   uint64 = 50 * UnitsPerCoin
	DefaultMaxBlockBytes uint64 = 1 << 20
	DefaultMaxTxBytes    uint64 = 128 << 10
	DefaultMaxTxCount    uint64 = 2000
	DefaultDataDir              = "data"
	DefaultDBPath               = "data/chain.db"
	DefaultWalletPath           = "data/wallets.json"
	DefaultMempoolPath          = "data/mempool.json"
	InitialDifficulty    uint32 = 4
	BlockTimeTargetSecs  int64  = 10

	GenesisTimestamp int64 = 1717200000
	GenesisMessage         = "IndoChain Genesis - fair CPU mining starts here"

	// MainnetGenesisHash is the frozen deterministic genesis hash for the
	// production mainnet profile. Any mismatch must stop startup rather than
	// silently accepting a different mainnet history.
	MainnetGenesisHash = "b01cbf6a3b04f3a9e374cad6adfb5cd8d24f5b2ef9723c3f7f440eb7fe53bde4"
)

type Paths struct {
	DataDir           string
	DB                string
	Wallets           string
	Mempool           string
	Peers             string
	FaucetState       string
	Lock              string
	NodeID            string
	ServiceNodes      string
	ServiceChallenges string
	ServiceRewards    string
	NetworkMetadata   string
}

func NewPaths(datadir string) Paths {
	if datadir == "" {
		datadir = DefaultDataDir
	}
	clean := filepath.Clean(datadir)
	return Paths{
		DataDir:           clean,
		DB:                filepath.Join(clean, "chain.db"),
		Wallets:           filepath.Join(clean, "wallets.json"),
		Mempool:           filepath.Join(clean, "mempool.json"),
		Peers:             filepath.Join(clean, "peers.json"),
		FaucetState:       filepath.Join(clean, "faucet_state.json"),
		Lock:              filepath.Join(clean, "node.lock"),
		NodeID:            filepath.Join(clean, "node_id"),
		ServiceNodes:      filepath.Join(clean, "service_nodes.json"),
		ServiceChallenges: filepath.Join(clean, "service_challenges.json"),
		ServiceRewards:    filepath.Join(clean, "service_rewards.json"),
		NetworkMetadata:   filepath.Join(clean, "network.json"),
	}
}

func (p Paths) DataDirPath() string {
	return p.DataDir
}

func (p Paths) ChainDBPath() string {
	return p.DB
}

func (p Paths) WalletPath() string {
	return p.Wallets
}

func (p Paths) MempoolPath() string {
	return p.Mempool
}

func (p Paths) PeersPath() string {
	return p.Peers
}

func (p Paths) FaucetStatePath() string {
	return p.FaucetState
}

func (p Paths) LockPath() string {
	return p.Lock
}

func (p Paths) NodeIDPath() string {
	return p.NodeID
}

func (p Paths) ServiceNodesPath() string {
	return p.ServiceNodes
}

func (p Paths) ServiceChallengesPath() string {
	return p.ServiceChallenges
}

func (p Paths) ServiceRewardsPath() string {
	return p.ServiceRewards
}

func (p Paths) NetworkMetadataPath() string {
	return p.NetworkMetadata
}

type NetworkMetadata struct {
	Network     string `json:"network"`
	NetworkID   string `json:"network_id"`
	ChainID     uint64 `json:"chain_id"`
	GenesisHash string `json:"genesis_hash"`
}

func MetadataForNetwork(profile NetworkConfig, genesisHash string) NetworkMetadata {
	return NetworkMetadata{
		Network:     profile.Name,
		NetworkID:   profile.NetworkID,
		ChainID:     profile.ChainID,
		GenesisHash: genesisHash,
	}
}

func WriteNetworkMetadata(paths Paths, profile NetworkConfig, genesisHash string) error {
	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(MetadataForNetwork(profile, genesisHash), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(paths.NetworkMetadata, raw, 0644)
}

func ReadNetworkMetadata(paths Paths) (NetworkMetadata, bool, error) {
	raw, err := os.ReadFile(paths.NetworkMetadata)
	if os.IsNotExist(err) {
		return NetworkMetadata{}, false, nil
	}
	if err != nil {
		return NetworkMetadata{}, false, err
	}
	var metadata NetworkMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return NetworkMetadata{}, true, err
	}
	return metadata, true, nil
}

func EnsureNetworkMatches(paths Paths, profile NetworkConfig) error {
	metadata, ok, err := ReadNetworkMetadata(paths)
	if err != nil {
		return err
	}
	if !ok {
		if profile.Name != "localnet" {
			return fmt.Errorf("datadir network metadata missing; refusing to use as %s", profile.Name)
		}
		return nil
	}
	if metadata.Network != profile.Name || metadata.NetworkID != profile.NetworkID || metadata.ChainID != profile.ChainID {
		return fmt.Errorf("datadir initialized for %s, cannot start as %s", metadata.Network, profile.Name)
	}
	if profile.GenesisHash != "" {
		if metadata.GenesisHash == "" {
			return fmt.Errorf("datadir genesis metadata missing; refusing to use as %s", profile.Name)
		}
		if metadata.GenesisHash != profile.GenesisHash {
			return fmt.Errorf("datadir genesis mismatch for %s", profile.Name)
		}
	}
	return nil
}

type NetworkLimits struct {
	MaxHeaderBatch      uint64 `json:"max_header_batch"`
	MaxSyncBlocks       uint64 `json:"max_sync_blocks"`
	MaxReorgFetchBlocks uint64 `json:"max_reorg_fetch_blocks"`
}

type NetworkConfig struct {
	NetworkName              string           `json:"network_name"`
	Name                     string           `json:"name"`
	NetworkID                string           `json:"network_id"`
	ChainID                  uint64           `json:"chain_id"`
	AddressPrefix            string           `json:"address_prefix"`
	AddressVersion           byte             `json:"address_version"`
	LegacyAddressAllowed     bool             `json:"legacy_address_allowed"`
	DefaultRPCPort           int              `json:"default_rpc_port"`
	DefaultP2PPort           int              `json:"default_p2p_port"`
	ProtocolVersion          uint32           `json:"protocol_version"`
	MinProtocolVersion       uint32           `json:"min_protocol_version"`
	RPCAPIVersion            string           `json:"rpc_api_version"`
	P2PProtocolVersion       string           `json:"p2p_protocol_version"`
	BlockVersion             uint32           `json:"block_version"`
	TxVersion                uint32           `json:"tx_version"`
	Difficulty               DifficultyParams `json:"difficulty"`
	Consensus                ConsensusParams  `json:"consensus"`
	Asset                    AssetParams      `json:"asset"`
	Fee                      FeeParams        `json:"fee"`
	Economic                 EconomicParams   `json:"economic"`
	MaxPeers                 int              `json:"max_peers"`
	MaxReorgDepth            uint64           `json:"max_reorg_depth"`
	MinMiningPeers           int              `json:"min_mining_peers"`
	AllowIsolatedMining      bool             `json:"allow_isolated_mining"`
	MinWritePeers            int              `json:"min_write_peers"`
	AllowIsolatedWrites      bool             `json:"allow_isolated_writes"`
	RequireAuthenticatedNode bool             `json:"require_authenticated_node"`
	SeedPeers                []string         `json:"seed_peers"`
	GenesisHash              string           `json:"genesis_hash"`
	NetworkLimits            NetworkLimits    `json:"network_limits"`
}

type DifficultyParams struct {
	InitialDifficulty      uint32 `json:"initial_difficulty"`
	MinDifficulty          uint32 `json:"min_difficulty"`
	MaxDifficulty          uint32 `json:"max_difficulty"`
	TargetBlockTimeSeconds int64  `json:"target_block_time_seconds"`
	RetargetWindow         uint64 `json:"retarget_window"`
	MaxFutureDriftSeconds  int64  `json:"max_future_drift_seconds"`
}

type AssetParams struct {
	NativeAssetID           string `json:"native_asset_id"`
	NativeAssetSymbol       string `json:"native_asset_symbol"`
	NativeAssetDecimals     uint8  `json:"native_asset_decimals"`
	FeeAssetID              string `json:"fee_asset_id"`
	TokenTransfersEnabled   bool   `json:"token_transfers_enabled"`
	UserIssuedTokensEnabled bool   `json:"user_issued_tokens_enabled"`
	PaymasterEnabled        bool   `json:"paymaster_enabled"`
}

type FeeParams struct {
	Enabled              bool   `json:"enabled"`
	MinFee               uint64 `json:"min_fee"`
	MinGasPrice          uint64 `json:"min_gas_price"`
	BytesPerGas          uint64 `json:"bytes_per_gas"`
	MaxGasPerTx          uint64 `json:"max_gas_per_tx"`
	BaseGasTransfer      uint64 `json:"base_gas_transfer"`
	BaseGasAssetTransfer uint64 `json:"base_gas_asset_transfer"`
	BaseGasStakeLock     uint64 `json:"base_gas_stake_lock"`
	BaseGasStakeUnlock   uint64 `json:"base_gas_stake_unlock"`
	BaseGasAssetCreate   uint64 `json:"base_gas_asset_create"`
	BaseGasAssetMint     uint64 `json:"base_gas_asset_mint"`
	BaseGasAssetBurn     uint64 `json:"base_gas_asset_burn"`
}

type EconomicParams struct {
	BlockSubsidy  uint64 `json:"block_subsidy"`
	FeeOnlyBlocks bool   `json:"fee_only_blocks"`
}

type ConsensusParams struct {
	CoinbaseMaturity uint64        `json:"coinbase_maturity"`
	MaxBlockBytes    uint64        `json:"max_block_bytes,omitempty"`
	MaxTxBytes       uint64        `json:"max_tx_bytes,omitempty"`
	MaxTxCount       uint64        `json:"max_tx_count,omitempty"`
	MaxGasPerBlock   uint64        `json:"max_gas_per_block,omitempty"`
	Staking          StakingParams `json:"staking"`
}

type StakingParams struct {
	Enabled                       bool   `json:"enabled"`
	MinServiceStake               uint64 `json:"min_service_stake"`
	MinStakeAmount                uint64 `json:"min_stake_amount"`
	UnbondingPeriodBlocks         uint64 `json:"unbonding_period_blocks"`
	MaxActiveStakesPerAddress     int    `json:"max_active_stakes_per_address"`
	RequireStakeForServiceRewards bool   `json:"require_stake_for_service_rewards"`
}

func Localnet() NetworkConfig {
	return NetworkConfig{
		NetworkName:          "localnet",
		Name:                 "localnet",
		NetworkID:            "ind-local-1",
		ChainID:              777001,
		AddressPrefix:        "iND",
		AddressVersion:       0x1E,
		LegacyAddressAllowed: true,
		DefaultRPCPort:       8331,
		DefaultP2PPort:       9331,
		ProtocolVersion:      1,
		MinProtocolVersion:   1,
		RPCAPIVersion:        "v1",
		P2PProtocolVersion:   "ind-p2p/1",
		BlockVersion:         1,
		TxVersion:            3,
		Difficulty: DifficultyParams{
			InitialDifficulty:      4,
			MinDifficulty:          1,
			MaxDifficulty:          8,
			TargetBlockTimeSeconds: 10,
			RetargetWindow:         10,
			MaxFutureDriftSeconds:  900,
		},
		Consensus: ConsensusParams{
			CoinbaseMaturity: 10,
			MaxBlockBytes:    DefaultMaxBlockBytes,
			MaxTxBytes:       DefaultMaxTxBytes,
			MaxTxCount:       DefaultMaxTxCount,
			MaxGasPerBlock:   200000,
			Staking: StakingParams{
				Enabled:                       true,
				MinServiceStake:               100 * UnitsPerCoin,
				MinStakeAmount:                10 * UnitsPerCoin,
				UnbondingPeriodBlocks:         10,
				MaxActiveStakesPerAddress:     10,
				RequireStakeForServiceRewards: true,
			},
		},
		Asset: AssetParams{
			NativeAssetID:           NativeAssetID,
			NativeAssetSymbol:       NativeAssetSymbol,
			NativeAssetDecimals:     NativeAssetDecimals,
			FeeAssetID:              FeeAssetID,
			TokenTransfersEnabled:   true,
			UserIssuedTokensEnabled: true,
			PaymasterEnabled:        true,
		},
		Fee: FeeParams{
			Enabled:              true,
			MinFee:               1,
			MinGasPrice:          0,
			BytesPerGas:          32,
			MaxGasPerTx:          100000,
			BaseGasTransfer:      10,
			BaseGasAssetTransfer: 12,
			BaseGasStakeLock:     12,
			BaseGasStakeUnlock:   8,
			BaseGasAssetCreate:   50,
			BaseGasAssetMint:     30,
			BaseGasAssetBurn:     25,
		},
		Economic: EconomicParams{
			BlockSubsidy:  0,
			FeeOnlyBlocks: true,
		},
		MaxPeers:            32,
		MaxReorgDepth:       64,
		MinMiningPeers:      0,
		AllowIsolatedMining: true,
		MinWritePeers:       0,
		AllowIsolatedWrites: true,
		SeedPeers:           nil,
		GenesisHash:         "",
		NetworkLimits: NetworkLimits{
			MaxHeaderBatch:      500,
			MaxSyncBlocks:       500,
			MaxReorgFetchBlocks: 512,
		},
	}
}

func Testnet() NetworkConfig {
	return NetworkConfig{
		NetworkName:          "testnet",
		Name:                 "testnet",
		NetworkID:            "ind-testnet-1",
		ChainID:              777101,
		AddressPrefix:        "iND",
		AddressVersion:       0x1F,
		LegacyAddressAllowed: false,
		DefaultRPCPort:       18331,
		DefaultP2PPort:       19331,
		ProtocolVersion:      1,
		MinProtocolVersion:   1,
		RPCAPIVersion:        "v1",
		P2PProtocolVersion:   "ind-p2p/1",
		BlockVersion:         1,
		TxVersion:            3,
		Difficulty: DifficultyParams{
			InitialDifficulty:      4,
			MinDifficulty:          1,
			MaxDifficulty:          12,
			TargetBlockTimeSeconds: 30,
			RetargetWindow:         30,
			MaxFutureDriftSeconds:  900,
		},
		Consensus: ConsensusParams{
			CoinbaseMaturity: 100,
			MaxBlockBytes:    DefaultMaxBlockBytes,
			MaxTxBytes:       DefaultMaxTxBytes,
			MaxTxCount:       DefaultMaxTxCount,
			MaxGasPerBlock:   200000,
			Staking: StakingParams{
				Enabled:                       true,
				MinServiceStake:               1000 * UnitsPerCoin,
				MinStakeAmount:                100 * UnitsPerCoin,
				UnbondingPeriodBlocks:         100,
				MaxActiveStakesPerAddress:     20,
				RequireStakeForServiceRewards: true,
			},
		},
		Asset: AssetParams{
			NativeAssetID:           NativeAssetID,
			NativeAssetSymbol:       NativeAssetSymbol,
			NativeAssetDecimals:     NativeAssetDecimals,
			FeeAssetID:              FeeAssetID,
			TokenTransfersEnabled:   true,
			UserIssuedTokensEnabled: true,
			PaymasterEnabled:        true,
		},
		Fee: FeeParams{
			Enabled:              true,
			MinFee:               1,
			MinGasPrice:          0,
			BytesPerGas:          32,
			MaxGasPerTx:          100000,
			BaseGasTransfer:      10,
			BaseGasAssetTransfer: 12,
			BaseGasStakeLock:     12,
			BaseGasStakeUnlock:   8,
			BaseGasAssetCreate:   50,
			BaseGasAssetMint:     30,
			BaseGasAssetBurn:     25,
		},
		Economic: EconomicParams{
			BlockSubsidy:  0,
			FeeOnlyBlocks: true,
		},
		MaxPeers:                 128,
		MaxReorgDepth:            128,
		MinMiningPeers:           1,
		AllowIsolatedMining:      false,
		MinWritePeers:            1,
		AllowIsolatedWrites:      false,
		SeedPeers:                nil,
		GenesisHash:              "",
		RequireAuthenticatedNode: true,
		NetworkLimits: NetworkLimits{
			MaxHeaderBatch:      500,
			MaxSyncBlocks:       500,
			MaxReorgFetchBlocks: 512,
		},
	}
}

func Mainnet() NetworkConfig {
	return NetworkConfig{
		NetworkName:          "mainnet",
		Name:                 "mainnet",
		NetworkID:            "ind-main-1",
		ChainID:              777000,
		AddressPrefix:        "iND",
		AddressVersion:       0x20,
		LegacyAddressAllowed: false,
		DefaultRPCPort:       8333,
		DefaultP2PPort:       9333,
		ProtocolVersion:      1,
		MinProtocolVersion:   1,
		RPCAPIVersion:        "v1",
		P2PProtocolVersion:   "ind-p2p/1",
		BlockVersion:         1,
		TxVersion:            3,
		Difficulty: DifficultyParams{
			InitialDifficulty:      6,
			MinDifficulty:          1,
			MaxDifficulty:          24,
			TargetBlockTimeSeconds: 60,
			RetargetWindow:         60,
			MaxFutureDriftSeconds:  900,
		},
		Consensus: ConsensusParams{
			CoinbaseMaturity: 100,
			MaxBlockBytes:    DefaultMaxBlockBytes,
			MaxTxBytes:       DefaultMaxTxBytes,
			MaxTxCount:       DefaultMaxTxCount,
			MaxGasPerBlock:   200000,
			Staking: StakingParams{
				Enabled:                       false,
				MinServiceStake:               0,
				MinStakeAmount:                0,
				UnbondingPeriodBlocks:         0,
				MaxActiveStakesPerAddress:     0,
				RequireStakeForServiceRewards: true,
			},
		},
		Asset: AssetParams{
			NativeAssetID:           NativeAssetID,
			NativeAssetSymbol:       NativeAssetSymbol,
			NativeAssetDecimals:     NativeAssetDecimals,
			FeeAssetID:              FeeAssetID,
			TokenTransfersEnabled:   true,
			UserIssuedTokensEnabled: true,
			PaymasterEnabled:        true,
		},
		Fee: FeeParams{
			Enabled:              true,
			MinFee:               1,
			MinGasPrice:          0,
			BytesPerGas:          32,
			MaxGasPerTx:          100000,
			BaseGasTransfer:      10,
			BaseGasAssetTransfer: 12,
			BaseGasStakeLock:     12,
			BaseGasStakeUnlock:   8,
			BaseGasAssetCreate:   50,
			BaseGasAssetMint:     30,
			BaseGasAssetBurn:     25,
		},
		Economic: EconomicParams{
			BlockSubsidy:  0,
			FeeOnlyBlocks: true,
		},
		MaxPeers:                 256,
		MaxReorgDepth:            64,
		MinMiningPeers:           1,
		AllowIsolatedMining:      false,
		MinWritePeers:            1,
		AllowIsolatedWrites:      false,
		SeedPeers:                nil,
		GenesisHash:              MainnetGenesisHash,
		RequireAuthenticatedNode: true,
		NetworkLimits: NetworkLimits{
			MaxHeaderBatch:      500,
			MaxSyncBlocks:       500,
			MaxReorgFetchBlocks: 512,
		},
	}
}

// ValidateNetworkProfile enforces the protocol invariants shared by all
// built-in IndoChain networks. Keeping these checks centralized prevents a
// network profile from silently drifting away from the frozen v3 asset/fee
// model.
func ValidateNetworkProfile(profile NetworkConfig) error {
	if profile.Name == "" || profile.NetworkID == "" || profile.NetworkName == "" {
		return fmt.Errorf("network identity is incomplete")
	}
	if profile.Name != profile.NetworkName {
		return fmt.Errorf("network name mismatch: name=%q network_name=%q", profile.Name, profile.NetworkName)
	}
	if profile.ChainID == 0 {
		return fmt.Errorf("chain id must be non-zero")
	}
	if profile.ProtocolVersion != 1 || profile.MinProtocolVersion != 1 {
		return fmt.Errorf("unsupported protocol version range: %d/%d", profile.MinProtocolVersion, profile.ProtocolVersion)
	}
	if profile.RPCAPIVersion != "v1" || profile.P2PProtocolVersion != "ind-p2p/1" {
		return fmt.Errorf("unsupported transport protocol versions")
	}
	if profile.BlockVersion != 1 || profile.TxVersion != 3 {
		return fmt.Errorf("unsupported consensus versions: block=%d tx=%d", profile.BlockVersion, profile.TxVersion)
	}
	if profile.Asset.NativeAssetID != NativeAssetID || profile.Asset.NativeAssetSymbol != NativeAssetSymbol || profile.Asset.NativeAssetDecimals != NativeAssetDecimals {
		return fmt.Errorf("native asset must remain dIDR with %d decimals", NativeAssetDecimals)
	}
	if profile.Asset.FeeAssetID != NativeAssetID {
		return fmt.Errorf("fee asset must remain dIDR")
	}
	if !profile.Asset.TokenTransfersEnabled || !profile.Asset.UserIssuedTokensEnabled || !profile.Asset.PaymasterEnabled {
		return fmt.Errorf("v3 asset capabilities are incomplete")
	}
	if !profile.Fee.Enabled || profile.Fee.MinFee == 0 || profile.Fee.MaxGasPerTx == 0 || profile.Consensus.MaxGasPerBlock == 0 {
		return fmt.Errorf("fee/gas policy is incomplete")
	}
	if profile.Economic.BlockSubsidy != 0 || !profile.Economic.FeeOnlyBlocks {
		return fmt.Errorf("v3 economics must be fee-only with zero block subsidy")
	}
	if profile.Consensus.MaxBlockBytes == 0 || profile.Consensus.MaxTxBytes == 0 || profile.Consensus.MaxTxCount == 0 {
		return fmt.Errorf("consensus size limits are incomplete")
	}
	if profile.NetworkLimits.MaxHeaderBatch == 0 || profile.NetworkLimits.MaxSyncBlocks == 0 || profile.NetworkLimits.MaxReorgFetchBlocks == 0 {
		return fmt.Errorf("network fetch limits are incomplete")
	}
	if profile.Name != "localnet" && !profile.RequireAuthenticatedNode {
		return fmt.Errorf("production network requires authenticated P2P nodes")
	}
	if profile.Name == "mainnet" && profile.GenesisHash != MainnetGenesisHash {
		return fmt.Errorf("mainnet genesis hash is not frozen")
	}
	return nil
}

func NetworkByName(name string) (NetworkConfig, error) {
	var profile NetworkConfig
	switch name {
	case "", "localnet":
		profile = Localnet()
	case "testnet":
		profile = Testnet()
	case "mainnet":
		profile = Mainnet()
	default:
		return NetworkConfig{}, fmt.Errorf("unknown network: %s", name)
	}
	if err := ValidateNetworkProfile(profile); err != nil {
		return NetworkConfig{}, fmt.Errorf("invalid %s network profile: %w", profile.Name, err)
	}
	return profile, nil
}

func DefaultMaxReorgDepth(profile NetworkConfig) uint64 {
	if profile.MaxReorgDepth > 0 {
		return profile.MaxReorgDepth
	}
	switch profile.Name {
	case "testnet":
		return 128
	default:
		return 64
	}
}

func MaxReorgDepthFromEnv(profile NetworkConfig) (uint64, error) {
	value := strings.TrimSpace(os.Getenv("IND_MAX_REORG_DEPTH"))
	if value == "" {
		return DefaultMaxReorgDepth(profile), nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return 0, fmt.Errorf("invalid IND_MAX_REORG_DEPTH: %q", value)
	}
	return parsed, nil
}

func MinMiningPeersFromEnv(profile NetworkConfig) (int, error) {
	return intFromEnv("IND_MIN_MINING_PEERS", profile.MinMiningPeers)
}

func AllowIsolatedMiningFromEnv(profile NetworkConfig) (bool, error) {
	return boolFromEnv("IND_ALLOW_ISOLATED_MINING", profile.AllowIsolatedMining)
}

func MinWritePeersFromEnv(profile NetworkConfig) (int, error) {
	return intFromEnv("IND_MIN_WRITE_PEERS", profile.MinWritePeers)
}

func AllowIsolatedWritesFromEnv(profile NetworkConfig) (bool, error) {
	return boolFromEnv("IND_ALLOW_ISOLATED_WRITES", profile.AllowIsolatedWrites)
}

func intFromEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid %s: %q", name, value)
	}
	return parsed, nil
}

func boolFromEnv(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %q", name, value)
	}
	return parsed, nil
}
