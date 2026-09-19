package chain

import (
	"context"
	"errors"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/storage"
	"deskachain/internal/types"
)

type Blockchain struct {
	store storage.Store
}

func New(store storage.Store) *Blockchain {
	return &Blockchain{store: store}
}

func (bc *Blockchain) HasChain() (bool, error) {
	return bc.store.HasChain()
}

func (bc *Blockchain) Init() error {
	return bc.InitWithProfile(config.Localnet())
}

func (bc *Blockchain) InitWithProfile(profile config.NetworkConfig) error {
	has, err := bc.store.HasChain()
	if err != nil {
		return err
	}
	if has {
		return nil
	}
	return bc.store.SaveBlock(GenesisBlockForNetwork(profile))
}

func (bc *Blockchain) Blocks() ([]types.Block, error) {
	return bc.store.Blocks()
}

func (bc *Blockchain) Tip() (types.Block, error) {
	return bc.store.Tip()
}

func (bc *Blockchain) GetBlockByHeight(height uint64) (types.Block, error) {
	return bc.store.GetBlockByHeight(height)
}

func (bc *Blockchain) ReplaceFromHeight(from uint64, blocks []types.Block) error {
	for h, err := bc.store.GetHeight(); err == nil && h >= from; h-- {
		if err := bc.store.DeleteBlockByHeight(h); err != nil {
			return err
		}
		if h == 0 {
			break
		}
	}
	for _, block := range blocks {
		if err := bc.store.SaveBlock(block); err != nil {
			return err
		}
	}
	return nil
}

func (bc *Blockchain) Ledger() (*ledger.Ledger, error) {
	blocks, err := bc.Blocks()
	if err != nil {
		return nil, err
	}
	return ledger.Replay(blocks)
}

func (bc *Blockchain) AddBlock(block types.Block) error {
	return bc.AddBlockWithNetwork(block, config.Localnet())
}

func (bc *Blockchain) AddBlockWithNetwork(block types.Block, profile config.NetworkConfig) error {
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return errors.New("chain is not initialized")
	}
	tip := blocks[len(blocks)-1]
	if err := ValidateNextBlockWithNetwork(block, tip, blocks, profile.Difficulty, profile.Consensus, profile); err != nil {
		return err
	}
	return bc.store.SaveBlock(block)
}

func (bc *Blockchain) MineBlock(miner string, pending []types.Transaction) (types.Block, error) {
	return bc.MineBlockWithContext(context.Background(), miner, pending, MineOptions{})
}

func (bc *Blockchain) MineBlockWithContext(ctx context.Context, miner string, pending []types.Transaction, opts MineOptions) (types.Block, error) {
	return bc.MineBlockWithContextAndNetwork(ctx, miner, pending, opts, config.Localnet())
}

func (bc *Blockchain) MineBlockWithContextAndNetwork(ctx context.Context, miner string, pending []types.Transaction, opts MineOptions, profile config.NetworkConfig) (types.Block, error) {
	blocks, err := bc.Blocks()
	if err != nil {
		return types.Block{}, err
	}
	if len(blocks) == 0 {
		return types.Block{}, errors.New("chain is not initialized")
	}
	l, err := ledger.ReplayMatureWithProfile(blocks, profile.Consensus, profile)
	if err != nil {
		return types.Block{}, err
	}
	workLedger := l.Clone()
	validPending := make([]types.Transaction, 0, len(pending))
	totalFees := uint64(0)
	height := blocks[len(blocks)-1].Height + 1
	for _, tx := range pending {
		if tx.Coinbase {
			continue
		}
		if err := workLedger.ApplyTransactionAtHeight(tx, height); err != nil {
			continue
		}
		validPending = append(validPending, tx)
		if tx.TxType() == types.TxTypeTransfer {
			totalFees += tx.Fee
		}
	}
	tip := blocks[len(blocks)-1]
	txs := append([]types.Transaction{types.NewCoinbaseTransaction(miner, config.InitialBlockReward+totalFees, height)}, validPending...)
	block := types.NewBlock(height, tip.Hash, miner, CalculateNextDifficultyWithParams(blocks, profile.Difficulty), txs)
	return MineWithContext(ctx, block, opts)
}
