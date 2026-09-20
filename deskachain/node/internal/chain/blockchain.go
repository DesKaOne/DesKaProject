package chain

import (
	"context"
	"errors"
	"fmt"

	"deskachain/internal/arith"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/mempool"
	"deskachain/internal/state"
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

	if !has {
		genesis := GenesisBlockForNetwork(profile)
		snapshot, snapshotErr := state.SnapshotForBlocks([]types.Block{genesis}, profile.Consensus, profile)
		if snapshotErr != nil {
			return snapshotErr
		}
		if bss, ok := bc.store.(storage.BlockStateStore); ok {
			return bss.SaveBlockAndState(genesis, snapshot)
		}
		return errors.New("production chain requires atomic block-state storage")
	}

	ss, ok := bc.store.(storage.StateStore)
	if !ok {
		return nil
	}

	tip, err := bc.store.Tip()
	if err != nil {
		return err
	}
	snapshot, loadErr := ss.LoadState()
	if loadErr != nil {
		return fmt.Errorf("persisted state unavailable: %w", loadErr)
	}
	if !snapshotMatchesTip(snapshot, tip) {
		return errors.New("persisted state does not match canonical tip")
	}
	q, ok := bc.store.(storage.StateQueryStore)
	if !ok {
		return errors.New("state query store is not available")
	}
	if err := q.ValidateChainStateConsistency(); err != nil {
		return err
	}
	return nil
}

// RebuildStateWithProfile is an explicit recovery operation for operators.
// Runtime initialization must never silently rewrite state derived from the
// canonical chain.
func (bc *Blockchain) RebuildStateWithProfile(profile config.NetworkConfig) error {
	ss, ok := bc.store.(storage.StateStore)
	if !ok {
		return errors.New("state store is not available")
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		return errors.New("chain is not initialized")
	}
	snapshot, err := state.SnapshotForBlocks(blocks, profile.Consensus, profile)
	if err != nil {
		return err
	}
	return ss.SaveState(snapshot)
}

func snapshotMatchesTip(snapshot state.Snapshot, tip types.Block) bool {
	if snapshot.Version != state.SnapshotVersion || snapshot.Height != tip.Height {
		return false
	}
	if tip.ProtocolVersion() == types.BlockVersionCanonical {
		return tip.StateRoot != "" && tip.StateRoot == snapshot.StateRoot
	}
	return true
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
	return bc.ReplaceFromHeightWithNetwork(from, blocks, config.Localnet())
}

func (bc *Blockchain) ReplaceFromHeightWithNetwork(from uint64, blocks []types.Block, profile config.NetworkConfig) error {
	existing, err := bc.store.Blocks()
	if err != nil {
		return err
	}
	prefix := make([]types.Block, 0, len(existing))
	for _, block := range existing {
		if block.Height < from {
			prefix = append(prefix, block)
		}
	}
	full := make([]types.Block, 0, len(prefix)+len(blocks))
	full = append(full, prefix...)
	full = append(full, blocks...)

	if ss, ok := bc.store.(storage.StateStore); ok {
		snapshot, snapshotErr := state.SnapshotForBlocks(full, profile.Consensus, profile)
		if snapshotErr != nil {
			return snapshotErr
		}
		if bss, ok := bc.store.(storage.BlockStateStore); ok {
			return bss.ReplaceFromHeightAndState(from, blocks, snapshot)
		}
		return errors.New("production chain requires atomic block-state storage")
	}

	return errors.New("production chain requires persistent state storage")
}

// BalanceDetailsForWithProfile prefers the persistent state indexes and
// falls back to deterministic replay when the state DB is unavailable or stale.
type StateValidationResult struct {
	Valid            bool   `json:"valid"`
	Height           uint64 `json:"height"`
	StateRoot        string `json:"state_root"`
	Accounts         int    `json:"accounts"`
	Stakes           int    `json:"stakes"`
	PendingCoinbases int    `json:"pending_coinbases"`
	Error            string `json:"error,omitempty"`
}

type StateStatusResult struct {
	Available  bool   `json:"available"`
	Current    bool   `json:"current"`
	Version    uint8  `json:"version"`
	Height     uint64 `json:"height"`
	StateRoot  string `json:"state_root"`
	TipHeight  uint64 `json:"tip_height"`
	TipHash    string `json:"tip_hash"`
}

func (bc *Blockchain) StateStatusWithNetwork(profile config.NetworkConfig) (StateStatusResult, error) {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	tip, err := bc.Tip()
	if err != nil {
		return StateStatusResult{}, err
	}
	result := StateStatusResult{TipHeight: tip.Height, TipHash: tip.Hash}
	q, ok := bc.store.(storage.StateQueryStore)
	if !ok {
		return result, nil
	}
	version, height, root, err := q.GetStateMetadata()
	if err != nil {
		return result, nil
	}
	result.Available = true
	result.Version = version
	result.Height = height
	result.StateRoot = root
	result.Current = version == state.SnapshotVersion && height == tip.Height
	if result.Current && tip.ProtocolVersion() == types.BlockVersionCanonical {
		result.Current = tip.StateRoot != "" && tip.StateRoot == root
	}
	return result, nil
}

func (bc *Blockchain) ValidateStateWithNetwork(profile config.NetworkConfig) (StateValidationResult, error) {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	ss, ok := bc.store.(storage.StateStore)
	if !ok {
		return StateValidationResult{}, errors.New("state store is not available")
	}
	tip, err := bc.Tip()
	if err != nil {
		return StateValidationResult{}, err
	}
	q, ok := bc.store.(storage.StateQueryStore)
	if !ok {
		return StateValidationResult{}, errors.New("state query store is not available")
	}
	if err := q.ValidateStateIndexes(); err != nil {
		return StateValidationResult{
			Valid:  false,
			Height: tip.Height,
			Error:  err.Error(),
		}, err
	}
	persisted, err := ss.LoadState()
	if err != nil {
		return StateValidationResult{}, err
	}
	blocks, err := bc.Blocks()
	if err != nil {
		return StateValidationResult{}, err
	}
	expected, err := state.SnapshotForBlocks(blocks, profile.Consensus, profile)
	if err != nil {
		return StateValidationResult{
			Valid:     false,
			Height:    tip.Height,
			StateRoot: persisted.StateRoot,
		}, err
	}
	result := StateValidationResult{
		Valid:            state.Equivalent(persisted, expected),
		Height:           persisted.Height,
		StateRoot:        persisted.StateRoot,
		Accounts:         len(persisted.Accounts),
		Stakes:           len(persisted.Stakes),
		PendingCoinbases: len(persisted.Coinbases),
	}
	if !result.Valid {
		result.Error = "state snapshot does not match canonical chain"
		return result, errors.New(result.Error)
	}
	return result, nil
}

func (bc *Blockchain) BalanceDetailsForWithProfile(address string, pending []types.Transaction, profile config.NetworkConfig) (ledger.BalanceDetails, error) {
	tip, err := bc.Tip()
	if err != nil {
		return ledger.BalanceDetails{}, err
	}

	if q, ok := bc.store.(storage.StateQueryStore); ok {
		version, height, stateRoot, metaErr := q.GetStateMetadata()
		stateCurrent := metaErr == nil &&
			version == state.SnapshotVersion &&
			height == tip.Height
		if stateCurrent && tip.ProtocolVersion() == types.BlockVersionCanonical {
			stateCurrent = tip.StateRoot != "" && tip.StateRoot == stateRoot
		}
		if stateCurrent {
			account, _, accountErr := q.GetStateAccount(address)
			if accountErr != nil {
				return ledger.BalanceDetails{}, accountErr
			}
			stakes, stakesErr := q.GetStateStakesForAddress(address)
			if stakesErr != nil {
				return ledger.BalanceDetails{}, stakesErr
			}
			return ledger.BalanceDetailsFromState(
				address,
				account,
				stakes,
				pending,
				profile.Consensus.CoinbaseMaturity,
				tip.Height,
			), nil
		}
	}

	blocks, err := bc.Blocks()
	if err != nil {
		return ledger.BalanceDetails{}, err
	}
	return ledger.BalanceDetailsForWithProfile(address, blocks, pending, profile.Consensus, profile)
}

func (bc *Blockchain) AccountNonceWithProfile(address string, profile config.NetworkConfig) (uint64, error) {
	tip, err := bc.Tip()
	if err != nil {
		return 0, err
	}
	if q, ok := bc.store.(storage.StateQueryStore); ok {
		version, height, stateRoot, metaErr := q.GetStateMetadata()
		stateCurrent := metaErr == nil &&
			version == state.SnapshotVersion &&
			height == tip.Height
		if stateCurrent && tip.ProtocolVersion() == types.BlockVersionCanonical {
			stateCurrent = tip.StateRoot != "" && tip.StateRoot == stateRoot
		}
		if stateCurrent {
			account, _, accountErr := q.GetStateAccount(address)
			if accountErr != nil {
				return 0, accountErr
			}
			return account.Nonce, nil
		}
	}
	l, err := bc.Ledger()
	if err != nil {
		return 0, err
	}
	return l.Nonce(address), nil
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

	if ss, ok := bc.store.(storage.StateStore); ok {
		var snapshot state.Snapshot
		var snapshotErr error
		if current, loadErr := ss.LoadState(); loadErr == nil && snapshotMatchesTip(current, tip) {
			snapshot, snapshotErr = state.SnapshotAfterBlock(current, block, profile.Consensus, profile)
		} else {
			nextBlocks := make([]types.Block, 0, len(blocks)+1)
			nextBlocks = append(nextBlocks, blocks...)
			nextBlocks = append(nextBlocks, block)
			snapshot, snapshotErr = state.SnapshotForBlocks(nextBlocks, profile.Consensus, profile)
		}
		if snapshotErr != nil {
			return snapshotErr
		}
		if bss, ok := bc.store.(storage.BlockStateStore); ok {
			return bss.SaveBlockAndState(block, snapshot)
		}
		return errors.New("production chain requires atomic block-state storage")
	}
	return errors.New("production chain requires persistent state storage")
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
	height, heightErr := arith.Add(blocks[len(blocks)-1].Height, 1)
	if heightErr != nil {
		return types.Block{}, errors.New("block height overflow")
	}
	nextNonce := make(map[string]uint64)
	for _, tx := range pending {
		if tx.Coinbase {
			continue
		}
		if _, ok := nextNonce[tx.From]; ok {
			continue
		}
		nonce := workLedger.Nonce(tx.From)
		if nonce == ^uint64(0) {
			return types.Block{}, errors.New("transaction nonce overflow")
		}
		nextNonce[tx.From] = nonce + 1
	}
	selected, err := mempool.SelectWithNonces(pending, profile, profile.Consensus.MaxGasPerBlock, nextNonce)
	if err != nil {
		return types.Block{}, err
	}
	validPending := make([]types.Transaction, 0, len(selected))
	totalFees := uint64(0)
	for _, tx := range selected {
		if err := ValidateTransactionSize(tx, profile.Consensus); err != nil {
			continue
		}
		if err := workLedger.ApplyTransactionAtHeight(tx, height); err != nil {
			continue
		}
		validPending = append(validPending, tx)
		if profile.TxVersion >= types.TxVersionAsset || tx.TxType() == types.TxTypeTransfer {
			nextFees, feeErr := arith.Add(totalFees, tx.Fee)
			if feeErr != nil {
				return types.Block{}, errors.New("transaction fees overflow")
			}
			totalFees = nextFees
		}
		if profile.Consensus.MaxTxCount > 0 && uint64(len(validPending)) >= profile.Consensus.MaxTxCount {
			break
		}
	}
	tip := blocks[len(blocks)-1]
	reward := profile.Economic.BlockSubsidy
	if profile.TxVersion < types.TxVersionAsset {
		var rewardErr error
		reward, rewardErr = arith.Add(config.InitialBlockReward, totalFees)
		if rewardErr != nil {
			return types.Block{}, errors.New("block reward overflow")
		}
	}
	coinbase := types.NewCoinbaseTransactionWithVersion(miner, reward, height, profile.TxVersion)
	if coinbase.ProtocolVersion() >= types.TxVersionCanonical {
		if err := coinbase.RefreshIDForChainID(profile.ChainID); err != nil {
			return types.Block{}, err
		}
	}
	txs := append([]types.Transaction{coinbase}, validPending...)
	block := types.NewBlockWithVersion(height, tip.Hash, miner, CalculateNextDifficultyWithParams(blocks, profile.Difficulty), txs, profile.BlockVersion)
	if block.ProtocolVersion() == types.BlockVersionCanonical {
		candidate := l.Clone()
		if err := candidate.ApplyBlock(block); err != nil {
			return types.Block{}, err
		}
		root, err := state.RootForLedger(candidate)
		if err != nil {
			return types.Block{}, err
		}
		block.StateRoot = root
	}
	return MineWithContext(ctx, block, opts)
}
