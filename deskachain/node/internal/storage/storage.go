package storage

import "deskachain/internal/types"

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
