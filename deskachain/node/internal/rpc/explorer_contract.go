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

type ExplorerContract struct {
	APIVersion string `json:"api_version"`
	Network string `json:"network"`
	Indexer ExplorerIndexerStatus `json:"indexer"`
}