package p2p

import (
	"deskachain/internal/chain"
	"deskachain/internal/types"
)

const Network = "deskachain-local"

type Status struct {
	Network            string `json:"network"`
	NetworkID          string `json:"network_id"`
	ChainID            uint64 `json:"chain_id"`
	GenesisHash        string `json:"genesis_hash"`
	ProtocolVersion    uint32 `json:"protocol_version"`
	P2PProtocolVersion string `json:"p2p_protocol_version"`
	Height             uint64 `json:"height"`
	TipHash            string `json:"tip_hash"`
	Difficulty         uint32 `json:"difficulty"`
	TipDifficulty      uint32 `json:"tip_difficulty"`
	NextDifficulty     uint32 `json:"next_difficulty"`
	CumulativeWork     uint64 `json:"cumulative_work"`
	TotalSupply        string `json:"total_supply"`
	MempoolCount       int    `json:"mempool_count"`
	AdvertiseP2P       string `json:"advertise_p2p,omitempty"`
	KnownPeerCount     int    `json:"known_peer_count,omitempty"`
	ActivePeerCount    int    `json:"active_peer_count,omitempty"`
}

type Handshake struct {
	NetworkName        string     `json:"network_name"`
	NetworkID          string     `json:"network_id"`
	ChainID            uint64     `json:"chain_id"`
	ProtocolVersion    uint32     `json:"protocol_version"`
	P2PProtocolVersion string     `json:"p2p_protocol_version"`
	MinProtocolVersion uint32     `json:"min_protocol_version"`
	GenesisHash        string     `json:"genesis_hash"`
	Height             uint64     `json:"height"`
	TipHash            string     `json:"tip_hash"`
	CumulativeWork     uint64     `json:"cumulative_work"`
	NodeID             string     `json:"node_id"`
	P2PListen          string     `json:"p2p_listen"`
	P2PAdvertise       string     `json:"p2p_advertise"`
	Services           []string   `json:"services,omitempty"`
	KnownPeers         []PeerView `json:"known_peers,omitempty"`
}

type PeerView struct {
	URL             string `json:"url"`
	Source          string `json:"source,omitempty"`
	Status          string `json:"status,omitempty"`
	Score           int    `json:"score"`
	LastHeight      uint64 `json:"last_height,omitempty"`
	LastTipHash     string `json:"last_tip_hash,omitempty"`
	LastNetworkID   string `json:"last_network_id,omitempty"`
	LastChainID     uint64 `json:"last_chain_id,omitempty"`
	LastGenesisHash string `json:"last_genesis_hash,omitempty"`
	LastSuccess     string `json:"last_success,omitempty"`
	LastFailure     string `json:"last_failure,omitempty"`
	FailureCount    int    `json:"failure_count,omitempty"`
	SuccessCount    int    `json:"success_count,omitempty"`
	LastLatencyMS   int64  `json:"last_latency_ms,omitempty"`
	CooldownUntil   string `json:"cooldown_until,omitempty"`
	LastError       string `json:"last_error,omitempty"`
	Version         string `json:"version,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	Services        string `json:"services,omitempty"`
	NextRetryAt     string `json:"next_retry_at,omitempty"`
}

type PeersResponse struct {
	Network        string     `json:"network"`
	NetworkID      string     `json:"network_id"`
	ChainID        uint64     `json:"chain_id"`
	GenesisHash    string     `json:"genesis_hash"`
	AdvertiseP2P   string     `json:"advertise_p2p,omitempty"`
	KnownPeers     []PeerView `json:"known_peers"`
	KnownCount     int        `json:"known_peer_count"`
	ActiveCount    int        `json:"active_peer_count"`
	SeedCount      int        `json:"seed_count"`
	MaxPeerEntries int        `json:"max_peer_entries"`
}

type PeerReputation struct {
	URL              string `json:"url"`
	Source           string `json:"source,omitempty"`
	Status           string `json:"status"`
	Score            int    `json:"score"`
	EffectiveScore   int    `json:"effective_score"`
	UptimeScore      int    `json:"uptime_score"`
	LatencyScore     int    `json:"latency_score"`
	ReliabilityScore int    `json:"reliability_score"`
	FailurePenalty   int    `json:"failure_penalty"`
	SuccessCount     int    `json:"success_count,omitempty"`
	FailureCount     int    `json:"failure_count,omitempty"`
	LastLatencyMS    int64  `json:"last_latency_ms,omitempty"`
	LastSuccess      string `json:"last_success,omitempty"`
	LastFailure      string `json:"last_failure,omitempty"`
	CooldownUntil    string `json:"cooldown_until,omitempty"`
	LastScoreReason  string `json:"last_score_reason,omitempty"`
	LastError        string `json:"last_error,omitempty"`
}

type PeerIntroduction struct {
	URL       string `json:"url"`
	NodeID    string `json:"node_id"`
	NetworkID string `json:"network_id"`
	ChainID   uint64 `json:"chain_id"`
	Version   string `json:"version,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Services  string `json:"services,omitempty"`
}

type BlockHeader struct {
	Height       uint64 `json:"height"`
	Hash         string `json:"hash"`
	PreviousHash string `json:"previous_hash"`
	Timestamp    int64  `json:"timestamp"`
	Difficulty   uint32 `json:"difficulty"`
	MerkleRoot   string `json:"merkle_root"`
	TxCount      int    `json:"tx_count"`
	MinerAddress string `json:"miner_address"`
}

type HeadersResponse struct {
	Headers []BlockHeader `json:"headers"`
}

type LocatorResponse struct {
	Height  uint64                    `json:"height"`
	TipHash string                    `json:"tip_hash"`
	Locator []chain.BlockLocatorEntry `json:"locator"`
}

type CommonAncestorRequest struct {
	Locator []chain.BlockLocatorEntry `json:"locator"`
}

type CommonAncestorResponse struct {
	Found  bool   `json:"found"`
	Height uint64 `json:"height,omitempty"`
	Hash   string `json:"hash,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Tip struct {
	Height uint64 `json:"height"`
	Hash   string `json:"hash"`
}

type BlocksResponse struct {
	Blocks []types.Block `json:"blocks"`
}

type TxResponse struct {
	Accepted bool   `json:"accepted"`
	TxID     string `json:"txid,omitempty"`
	Error    string `json:"error,omitempty"`
}

type BlockResponse struct {
	Accepted    bool   `json:"accepted"`
	Height      uint64 `json:"height,omitempty"`
	Hash        string `json:"hash,omitempty"`
	Error       string `json:"error,omitempty"`
	LocalHeight uint64 `json:"local_height,omitempty"`
	LocalTip    string `json:"local_tip,omitempty"`
}

type ForkCheckResult struct {
	ForkDetected         bool   `json:"fork_detected"`
	InSync               bool   `json:"in_sync"`
	LocalHeight          uint64 `json:"local_height"`
	LocalTip             string `json:"local_tip"`
	PeerHeight           uint64 `json:"peer_height"`
	PeerTip              string `json:"peer_tip"`
	CommonAncestorFound  bool   `json:"common_ancestor_found"`
	CommonAncestorHeight uint64 `json:"common_ancestor_height,omitempty"`
	CommonAncestorHash   string `json:"common_ancestor_hash,omitempty"`
	LocalAheadBlocks     uint64 `json:"local_ahead_blocks"`
	PeerAheadBlocks      uint64 `json:"peer_ahead_blocks"`
	ReorgSupported       bool   `json:"reorg_supported"`
	Status               string `json:"status"`
	Error                string `json:"error,omitempty"`
}

type ForkInspectResult struct {
	ForkDetected         bool   `json:"fork_detected"`
	InSync               bool   `json:"in_sync"`
	LocalHeight          uint64 `json:"local_height"`
	LocalTip             string `json:"local_tip"`
	OtherHeight          uint64 `json:"other_height"`
	OtherTip             string `json:"other_tip"`
	CommonAncestorFound  bool   `json:"common_ancestor_found"`
	CommonAncestorHeight uint64 `json:"common_ancestor_height,omitempty"`
	CommonAncestorHash   string `json:"common_ancestor_hash,omitempty"`
	LocalAheadBlocks     uint64 `json:"local_ahead_blocks"`
	OtherAheadBlocks     uint64 `json:"other_ahead_blocks"`
	ReorgSupported       bool   `json:"reorg_supported"`
	Status               string `json:"status"`
	Error                string `json:"error,omitempty"`
}
