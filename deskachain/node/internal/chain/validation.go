package chain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"deskachain/internal/arith"
	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/fees"
	"deskachain/internal/ledger"
	"deskachain/internal/state"
	"deskachain/internal/types"
)

type ValidationResult struct {
	Height      uint64
	Blocks      int
	TotalSupply uint64
}

func ValidateChain(blocks []types.Block) (ValidationResult, error) {
	net := config.Localnet()
	return ValidateChainWithNetwork(blocks, net)
}

func ValidateChainWithParams(blocks []types.Block, params config.DifficultyParams) (ValidationResult, error) {
	return validateChain(blocks, params, config.Localnet().Consensus, config.Localnet())
}

func ValidateChainWithNetwork(blocks []types.Block, profile config.NetworkConfig) (ValidationResult, error) {
	return validateChain(blocks, profile.Difficulty, profile.Consensus, profile)
}

func validateChain(blocks []types.Block, params config.DifficultyParams, consensus config.ConsensusParams, profile config.NetworkConfig) (ValidationResult, error) {
	if len(blocks) == 0 {
		return ValidationResult{}, errors.New("chain is empty")
	}
	genesis := GenesisBlockForNetwork(profile)
	if blocks[0].Hash != genesis.Hash ||
		blocks[0].Height != genesis.Height ||
		blocks[0].PreviousHash != genesis.PreviousHash ||
		blocks[0].Timestamp != genesis.Timestamp ||
		blocks[0].MerkleRoot != genesis.MerkleRoot ||
		blocks[0].GenesisMarker != genesis.GenesisMarker {
		return ValidationResult{}, errors.New("genesis block is not deterministic")
	}
	if blocks[0].Hash != blocks[0].CalculateHash() {
		return ValidationResult{}, errors.New("block 0 hash mismatch")
	}
	seenTxIDs := make(map[string]struct{})
	prior := make([]types.Block, 0, len(blocks))
	prior = append(prior, blocks[0])
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]
		prev := blocks[i-1]
		if block.PreviousHash != prev.Hash {
			return ValidationResult{}, fmt.Errorf("block %d previous_hash mismatch", block.Height)
		}
		expectedHeight, heightErr := arith.Add(prev.Height, 1)
		if heightErr != nil {
			return ValidationResult{}, errors.New("block height overflow")
		}
		if block.Height != expectedHeight {
			return ValidationResult{}, fmt.Errorf("block %d height mismatch", block.Height)
		}
		if err := validateBlock(block, prior, params, consensus, profile); err != nil {
			if strings.HasPrefix(err.Error(), "invalid difficulty at height") {
				return ValidationResult{}, err
			}
			return ValidationResult{}, fmt.Errorf("block %d %w", block.Height, err)
		}
		for _, tx := range block.Transactions {
			if tx.ID == "" {
				return ValidationResult{}, fmt.Errorf("block %d empty tx id", block.Height)
			}
			if _, ok := seenTxIDs[tx.ID]; ok {
				return ValidationResult{}, fmt.Errorf("block %d duplicate tx id", block.Height)
			}
			seenTxIDs[tx.ID] = struct{}{}
		}
		prior = append(prior, block)
	}
	return ValidationResult{
		Height:      blocks[len(blocks)-1].Height,
		Blocks:      len(blocks),
		TotalSupply: ledger.TotalSupplyWithProfile(blocks, profile),
	}, nil
}

func ValidateNextBlock(block types.Block, tip types.Block, prior []types.Block) error {
	return ValidateNextBlockWithParams(block, tip, prior, config.Localnet().Difficulty)
}

func ValidateNextBlockWithParams(block types.Block, tip types.Block, prior []types.Block, params config.DifficultyParams) error {
	return ValidateNextBlockWithConsensus(block, tip, prior, params, config.Localnet().Consensus)
}

func ValidateNextBlockWithConsensus(block types.Block, tip types.Block, prior []types.Block, params config.DifficultyParams, consensus config.ConsensusParams) error {
	return ValidateNextBlockWithNetwork(block, tip, prior, params, consensus, config.Localnet())
}

func ValidateNextBlockWithNetwork(block types.Block, tip types.Block, prior []types.Block, params config.DifficultyParams, consensus config.ConsensusParams, profile config.NetworkConfig) error {
	expectedHeight, heightErr := arith.Add(tip.Height, 1)
	if heightErr != nil {
		return errors.New("block height overflow")
	}
	if block.Height != expectedHeight {
		return fmt.Errorf("invalid block height: got %d want %d", block.Height, expectedHeight)
	}
	if block.PreviousHash != tip.Hash {
		return errors.New("previous hash does not match tip")
	}
	return validateBlock(block, prior, params, consensus, profile)
}

func ValidateBlock(block types.Block, prior []types.Block) error {
	return ValidateBlockWithParams(block, prior, config.Localnet().Difficulty)
}

func ValidateBlockWithParams(block types.Block, prior []types.Block, params config.DifficultyParams) error {
	return validateBlock(block, prior, params, config.Localnet().Consensus, config.Localnet())
}

func ValidateBlockWithConsensus(block types.Block, prior []types.Block, params config.DifficultyParams, consensus config.ConsensusParams) error {
	return validateBlock(block, prior, params, consensus, config.Localnet())
}

func ValidateBlockWithNetwork(block types.Block, prior []types.Block, profile config.NetworkConfig) error {
	return validateBlock(block, prior, profile.Difficulty, profile.Consensus, profile)
}

func validateBlock(block types.Block, prior []types.Block, params config.DifficultyParams, consensus config.ConsensusParams, profile config.NetworkConfig) error {
	if err := types.ValidateBlockVersion(block.ProtocolVersion(), profile.BlockVersion); err != nil {
		return err
	}
	if block.Hash != block.CalculateHash() {
		return errors.New("block hash mismatch")
	}
	if block.MerkleRoot != types.CalculateMerkleRoot(block.Transactions) {
		return errors.New("merkle root mismatch")
	}
	if err := ValidateBlockResourcesWithProfile(block, profile); err != nil {
		return err
	}
	if block.Height > 0 {
		expectedDifficulty := CalculateNextDifficultyWithParams(prior, params)
		if block.Difficulty != expectedDifficulty {
			return fmt.Errorf("invalid difficulty at height %d expected %d got %d", block.Height, expectedDifficulty, block.Difficulty)
		}
		if err := ValidateBlockTimestamp(block, prior[len(prior)-1], params); err != nil {
			return err
		}
		if err := crypto.ValidateAddressForNetwork(block.MinerAddress, profile); err != nil {
			return fmt.Errorf("invalid miner address: %w", err)
		}
		if !ValidateProofOfWork(block) {
			return errors.New("invalid proof of work")
		}
	}
	coinbaseCount := 0
	totalFees := uint64(0)
	for i, tx := range block.Transactions {
		if err := types.ValidateTransactionVersion(tx.ProtocolVersion(), profile.TxVersion); err != nil {
			return fmt.Errorf("tx %d %w", i, err)
		}
		if tx.Coinbase {
			coinbaseCount++
			if i != 0 {
				return errors.New("coinbase must be first transaction")
			}
			if tx.From != types.CoinbaseSender {
				return errors.New("coinbase sender must be COINBASE")
			}
			expectedID, idErr := tx.CalculateIDForChainID(profile.ChainID)
			if idErr != nil {
				return idErr
			}
			if tx.ID != expectedID {
				return errors.New("coinbase transaction id mismatch")
			}
			continue
		}
		if tx.TxType() == types.TxTypeTransfer && tx.Amount == 0 {
			return fmt.Errorf("tx %d amount must be greater than zero", i)
		}
		if tx.TxType() == types.TxTypeStakeLock && tx.Amount == 0 {
			return fmt.Errorf("tx %d amount must be greater than zero", i)
		}
		if tx.TxType() == types.TxTypeTransfer && tx.From == tx.To {
			return fmt.Errorf("tx %d sender and recipient must differ", i)
		}
		if err := fees.Validate(tx, profile); err != nil {
			return fmt.Errorf("tx %d %w", i, err)
		}
		if profile.TxVersion >= types.TxVersionAsset || tx.TxType() == types.TxTypeTransfer {
			nextFees, feeErr := arith.Add(totalFees, tx.Fee)
			if feeErr != nil {
				return errors.New("transaction fees overflow")
			}
			totalFees = nextFees
		}
	}
	if block.Height > 0 {
		if coinbaseCount != 1 {
			return errors.New("block must include exactly one coinbase transaction")
		}
		want := uint64(0)
		if profile.TxVersion >= types.TxVersionAsset {
			want = profile.Economic.BlockSubsidy
		} else {
			var rewardErr error
			want, rewardErr = arith.Add(config.InitialBlockReward, totalFees)
			if rewardErr != nil {
				return errors.New("block reward overflow")
			}
		}
		for _, tx := range block.Transactions {
			if tx.Coinbase && tx.Amount != want {
				return fmt.Errorf("invalid coinbase amount: got %d want %d", tx.Amount, want)
			}
		}
	}
	l, err := ledger.ReplayMatureWithProfile(prior, consensus, profile)
	if err != nil {
		return err
	}
	preBlockLedger := l.Clone()
	for i, tx := range block.Transactions {
		if tx.Coinbase {
			if err := l.ApplyCoinbaseAtHeight(tx, block.Height); err != nil {
				return fmt.Errorf("tx %d %w", i, err)
			}
			continue
		}
		if err := l.ApplyTransactionAtHeight(tx, block.Height); err != nil {
			if errors.Is(err, ledger.ErrImmatureBalance) {
				return fmt.Errorf("invalid transaction at height %d: spends immature coinbase", block.Height)
			}
			return fmt.Errorf("tx %d %s", i, normalizeTxValidationError(err))
		}
	}

	if block.ProtocolVersion() == types.BlockVersionCanonical {
		candidate := preBlockLedger.Clone()
		if err := candidate.ApplyBlock(block); err != nil {
			return fmt.Errorf("state root replay failed: %w", err)
		}
		expectedStateRoot, err := state.RootForLedger(candidate)
		if err != nil {
			return fmt.Errorf("state root calculation failed: %w", err)
		}
		if block.StateRoot == "" {
			return errors.New("canonical block state root is empty")
		}
		if block.StateRoot != expectedStateRoot {
			return errors.New("state root mismatch")
		}
	}
	return nil
}

func ValidateBlockTimestamp(block types.Block, parent types.Block, params config.DifficultyParams) error {
	params = normalizeDifficultyParams(params)
	if block.Timestamp < parent.Timestamp {
		return errors.New("invalid block timestamp: before parent")
	}
	if block.Timestamp > time.Now().Unix()+params.MaxFutureDriftSeconds {
		return errors.New("invalid block timestamp: too far in future")
	}
	return nil
}

func normalizeTxValidationError(err error) string {
	switch err.Error() {
	case "invalid transaction signature":
		return "invalid signature"
	case "public key does not match sender address":
		return "public key does not match sender"
	case "transaction id mismatch":
		return "tx id mismatch"
	case "insufficient balance":
		return "sender balance insufficient"
	default:
		return err.Error()
	}
}
