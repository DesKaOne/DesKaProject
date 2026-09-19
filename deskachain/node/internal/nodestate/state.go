package nodestate

import (
	"errors"
	"sync"
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/mempool"
	"deskachain/internal/storage"
)

type ChainRuntimeState struct {
	Height         uint64    `json:"height"`
	TipHash        string    `json:"tip_hash"`
	Difficulty     uint32    `json:"difficulty"`
	TipDifficulty  uint32    `json:"tip_difficulty"`
	NextDifficulty uint32    `json:"next_difficulty"`
	TotalSupply    uint64    `json:"total_supply"`
	CumulativeWork uint64    `json:"cumulative_work"`
	MempoolCount   int       `json:"mempool_count"`
	Blocks         int       `json:"blocks"`
	CoinbaseBlocks int       `json:"coinbase_blocks"`
	TotalTxs       int       `json:"total_transactions"`
	CoinbaseTxs    int       `json:"coinbase_transactions"`
	NormalTxs      int       `json:"normal_transactions"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Store struct {
	mu       sync.RWMutex
	snapshot ChainRuntimeState
	profile  config.NetworkConfig
}

func New(paths config.Paths) (*Store, error) {
	return NewWithProfile(paths, config.Localnet())
}

func NewWithProfile(paths config.Paths, profile config.NetworkConfig) (*Store, error) {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	s := &Store{profile: profile}
	return s, s.Refresh(paths)
}

func (s *Store) Snapshot() ChainRuntimeState {
	if s == nil {
		return ChainRuntimeState{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

func (s *Store) Refresh(paths config.Paths) error {
	if s == nil {
		return nil
	}
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()
	bc := chain.New(store)
	has, err := bc.HasChain()
	if err != nil {
		return err
	}
	if !has {
		return errors.New("chain is not initialized")
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return nil
	}
	tip := blocks[len(blocks)-1]
	nextDifficulty := chain.CalculateNextDifficultyWithParams(blocks, s.profile.Difficulty)
	stats := chain.CalculateChainStatsWithProfile(blocks, s.profile)
	pending, _ := mempool.New(paths.Mempool).Load()
	next := ChainRuntimeState{
		Height:         tip.Height,
		TipHash:        tip.Hash,
		Difficulty:     nextDifficulty,
		TipDifficulty:  tip.Difficulty,
		NextDifficulty: nextDifficulty,
		TotalSupply:    stats.TotalSupply,
		CumulativeWork: stats.CumulativeWork,
		MempoolCount:   len(pending),
		Blocks:         stats.Blocks,
		CoinbaseBlocks: stats.CoinbaseBlocks,
		TotalTxs:       stats.TotalTransactions,
		CoinbaseTxs:    stats.CoinbaseTransactions,
		NormalTxs:      stats.NormalTransactions,
		UpdatedAt:      time.Now(),
	}
	s.mu.Lock()
	s.snapshot = next
	s.mu.Unlock()
	return nil
}
