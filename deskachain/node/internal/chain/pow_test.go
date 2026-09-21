package chain

import (
	"context"
	"errors"
	"testing"

	"indochain/internal/config"
	"indochain/internal/types"
	"indochain/internal/wallet"
)

func TestPoWValidationAcceptsMinedBlock(t *testing.T) {
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	genesis := GenesisBlock()
	block := types.NewBlock(1, genesis.Hash, w.Address, 1, []types.Transaction{
		types.NewCoinbaseTransaction(w.Address, config.InitialBlockReward, 1),
	})
	mined := Mine(block)
	if !ValidateProofOfWork(mined) {
		t.Fatal("mined block failed proof-of-work validation")
	}
}

func TestMineWithContextCancelAndMaxNonce(t *testing.T) {
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	block := types.NewBlock(1, GenesisBlock().Hash, w.Address, 8, []types.Transaction{
		types.NewCoinbaseTransaction(w.Address, config.InitialBlockReward, 1),
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := MineWithContext(ctx, block, MineOptions{}); err == nil || !errors.Is(err, context.Canceled) && err.Error() != "mining cancelled" {
		t.Fatalf("expected mining cancelled, got %v", err)
	}
	if _, err := MineWithContext(context.Background(), block, MineOptions{MaxNonce: 1}); !errors.Is(err, ErrMaxNonceReached) {
		t.Fatalf("expected max nonce reached, got %v", err)
	}
}

func TestPoWValidationRejectsInvalidHash(t *testing.T) {
	w, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	genesis := GenesisBlock()
	block := types.NewBlock(1, genesis.Hash, w.Address, 1, []types.Transaction{
		types.NewCoinbaseTransaction(w.Address, config.InitialBlockReward, 1),
	})
	mined := Mine(block)
	mined.Nonce++
	if ValidateProofOfWork(mined) {
		t.Fatal("proof-of-work accepted mismatched nonce/hash")
	}
}
