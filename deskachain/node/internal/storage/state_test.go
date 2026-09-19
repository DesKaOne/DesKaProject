package storage

import (
	"errors"
	"strings"
	"testing"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/state"
	"deskachain/internal/types"
)

func emptySnapshot(t *testing.T) state.Snapshot {
	t.Helper()
	l := ledger.NewMatureWithProfile(config.ConsensusParams{}, config.Localnet())
	snapshot, err := state.SnapshotForLedger(l)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestStateStorePersistsMetadataAndIndexes(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	snapshot.Accounts = append(snapshot.Accounts, ledger.StateAccount{
		Address:   "DKC-test",
		Confirmed: 10,
		Mature:    10,
	})
	snapshot.StateRoot, err = state.RootForCollections(snapshot.Accounts, snapshot.Stakes)
	if err != nil {
		t.Fatal(err)
	}

	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
	got, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.StateRoot != snapshot.StateRoot || len(got.Accounts) != 1 {
		t.Fatalf("unexpected restored snapshot: %#v", got)
	}
	if got.Accounts[0].Address != "DKC-test" || got.Accounts[0].Confirmed != 10 {
		t.Fatalf("unexpected account index contents: %#v", got.Accounts)
	}
}

func TestStateStoreMissingStateIsExplicit(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.LoadState()
	if !errors.Is(err, ErrStateNotInitialized) {
		t.Fatalf("expected ErrStateNotInitialized, got %v", err)
	}
}

func TestStateQueryIndexesReadWithoutLoadingFullSnapshot(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	snapshot.Height = 12
	snapshot.Accounts = []ledger.StateAccount{
		{Address: "DKC-alice", Confirmed: 100, Mature: 90, Nonce: 4},
		{Address: "DKC-bob", Confirmed: 25, Mature: 25, Nonce: 2},
	}
	snapshot.Stakes = []staking.Record{
		{StakeID: "stake-2", OwnerAddress: "DKC-alice", Amount: 20, Status: staking.StatusActive},
		{StakeID: "stake-1", OwnerAddress: "DKC-alice", Amount: 30, Status: staking.StatusUnlocking, ReleaseHeight: 20},
		{StakeID: "stake-3", OwnerAddress: "DKC-bob", Amount: 10, Status: staking.StatusActive},
	}
	snapshot.StateRoot, err = state.RootForCollections(snapshot.Accounts, snapshot.Stakes)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}

	version, height, root, err := store.GetStateMetadata()
	if err != nil {
		t.Fatal(err)
	}
	if version != state.SnapshotVersion || height != 12 || root != snapshot.StateRoot {
		t.Fatalf("unexpected state metadata: version=%d height=%d root=%s", version, height, root)
	}

	account, found, err := store.GetStateAccount("DKC-alice")
	if err != nil {
		t.Fatal(err)
	}
	if !found || account.Confirmed != 100 || account.Nonce != 4 {
		t.Fatalf("unexpected account query: found=%v account=%#v", found, account)
	}
	if _, found, err := store.GetStateAccount("DKC-missing"); err != nil || found {
		t.Fatalf("unexpected missing account query: found=%v err=%v", found, err)
	}

	record, found, err := store.GetStateStake("stake-2")
	if err != nil {
		t.Fatal(err)
	}
	if !found || record.OwnerAddress != "DKC-alice" || record.Amount != 20 {
		t.Fatalf("unexpected stake query: found=%v record=%#v", found, record)
	}

	aliceStakes, err := store.GetStateStakesForAddress("DKC-alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(aliceStakes) != 2 {
		t.Fatalf("alice stake count = %d, want 2", len(aliceStakes))
	}
	if aliceStakes[0].StakeID != "stake-1" || aliceStakes[1].StakeID != "stake-2" {
		t.Fatalf("unexpected owner stake ordering: %#v", aliceStakes)
	}
	for _, stake := range aliceStakes {
		if !strings.HasPrefix(stake.OwnerAddress, "DKC-") {
			t.Fatalf("unexpected owner in index: %#v", stake)
		}
	}
}

func TestStateQueryIndexesDisappearWithStateReset(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	snapshot.Accounts = []ledger.StateAccount{{Address: "DKC-alice", Confirmed: 1, Mature: 1}}
	snapshot.Stakes = []staking.Record{{StakeID: "stake-1", OwnerAddress: "DKC-alice", Amount: 1, Status: staking.StatusActive}}
	snapshot.StateRoot, err = state.RootForCollections(snapshot.Accounts, snapshot.Stakes)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteState(); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.GetStateAccount("DKC-alice"); !errors.Is(err, ErrStateNotInitialized) || found {
		t.Fatalf("expected cleared account index, found=%v err=%v", found, err)
	}
	if _, err := store.GetStateStakesForAddress("DKC-alice"); !errors.Is(err, ErrStateNotInitialized) {
		t.Fatalf("expected cleared owner index, got %v", err)
	}
}

func TestSaveBlockAndStatePersistsBothDatasets(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	block := types.Block{Height: 0, Hash: "genesis"}
	if err := store.SaveBlockAndState(block, snapshot); err != nil {
		t.Fatal(err)
	}

	has, err := store.HasChain()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("chain was not persisted")
	}
	got, err := store.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if got.Height != 0 || got.StateRoot != snapshot.StateRoot {
		t.Fatalf("unexpected persisted state: %#v", got)
	}
}
