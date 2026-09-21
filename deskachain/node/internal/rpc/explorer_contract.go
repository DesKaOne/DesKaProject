package rpc

const ExplorerAPIVersion = "v1"

type ExplorerPage struct {
	Limit int `json:"limit"`
	Offset int `json:"offset"`
	Count int `json:"count"`
	TotalCount int `json:"total_count"`
	NextOffset *int `json:"next_offset,omitempty"`
	PrevOffset *int `json:"prev_offset,omitempty"`
}

type ExplorerSearchResult struct {
	Type string `json:"type"`
	ID string `json:"id"`
	Label string `json:"label"`
	Path string `json:"path"`
	APIPath string `json:"api_path"`
}

type ExplorerIndexerStatus struct {
	Mode string `json:"mode"`
	IndexedHeight uint64 `json:"indexed_height"`
	ChainHeight uint64 `json:"chain_height"`
	Lag uint64 `json:"lag"`
	Ready bool `json:"ready"`
	RebuildRequired bool `json:"rebuild_required"`
	LastIndexedHash string `json:"last_indexed_hash,omitempty"`
	SchemaVersion string `json:"schema_version"`
}

type ExplorerIndexerStats struct {
	BlockCount          int     `json:"block_count"`
	TransactionCount    int     `json:"transaction_count"`
	AddressHistoryCount int     `json:"address_history_count"`
	AssetEventCount     int     `json:"asset_event_count"`
	IndexedHeight       uint64  `json:"indexed_height"`
	ChainHeight         uint64  `json:"chain_height"`
	Lag                 uint64  `json:"lag"`
	Ready               bool    `json:"ready"`
	SyncStatus          string  `json:"sync_status"`
	SchemaVersion       string  `json:"schema_version"`
	SyncCount           uint64  `json:"sync_count"`
	LastSyncAtUnix      int64   `json:"last_sync_at_unix"`
	LastSyncDurationMs  int64   `json:"last_sync_duration_ms"`
	LastSyncBlockCount  int     `json:"last_sync_block_count"`
	BlocksPerSecond     float64 `json:"blocks_per_second"`
}

type ExplorerContract struct {
	APIVersion string `json:"api_version"`
	Network string `json:"network"`
	Indexer ExplorerIndexerStatus `json:"indexer"`
}