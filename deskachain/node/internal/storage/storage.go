package storage

import (
	"deskachain/internal/state"
	"deskachain/internal/types"
)

type Store interface {
	Init() error
	Close() error
	HasChain() (bool, error)
	SaveBlock(block types.Block) error
	Blocks() ([]types.Block, error)
	Tip() (types.Block, error)
	GetBlockByHeight(height uint64) (types.Block, error)
	DeleteBlockByHeight(height uint64) error
	ReplaceFromHeight(from uint64, blocks []types.Block) error
	SetTip(height uint64, hash string) error
	GetHeight() (uint64, error)
	GetTip() (uint64, string, error)
}

// StateStore persists the deterministic state derived from the canonical chain.
type StateStore interface {
	SaveState(snapshot state.Snapshot) error
	LoadState() (state.Snapshot, error)
	DeleteState() error
}

// BlockStateStore provides atomic chain + state commits for stores that can
// update both datasets inside one database transaction.
type BlockStateStore interface {
	StateStore
	SaveBlockAndState(block types.Block, snapshot state.Snapshot) error
	ReplaceFromHeightAndState(from uint64, blocks []types.Block, snapshot state.Snapshot) error
}
