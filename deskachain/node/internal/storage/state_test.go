package storage

import (
	"errors"
	"strings"
	"testing"

	"deskachain/internal/asset"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/state"
	"deskachain/internal/types"

	bolt "go.etcd.io/bbolt"
)

func snapshotRoot(snapshot state.Snapshot) (string, error) {
	if len(snapshot.Assets) > 0 {
		return state.RootForCollectionsWithAssets(snapshot.Accounts, snapshot.Stakes, snapshot.Assets, snapshot.AssetBalances)
	}
	return state.RootForCollections(snapshot.Accounts, snapshot.Stakes)
}

func syncNativeBalances(snapshot *state.Snapshot) {
	for _, account := range snapshot.Accounts {
		found := false
		for i := range snapshot.AssetBalances {
			if snapshot.AssetBalances[i].Address == account.Address && snapshot.AssetBalances[i].AssetID == asset.NativeAssetID {
				snapshot.AssetBalances[i].Amount = account.Confirmed
				found = true
				break
			}
		}
		if !found && account.Confirmed > 0 {
			snapshot.AssetBalances = append(snapshot.AssetBalances, asset.BalanceEntry{Address: account.Address, AssetID: asset.NativeAssetID, Amount: account.Confirmed})
		}
	}
}

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
		Address:   "IDR-test",
		Confirmed: 10,
		Mature:    10,
	})
	syncNativeBalances(&snapshot)
	snapshot.StateRoot, err = snapshotRoot(snapshot)
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
	if got.StateRoot != snapshot.StateRoot || len(got.Accounts) != 1 || len(got.Coinbases) != 0 {
		t.Fatalf("unexpected restored snapshot: %#v", got)
	}
	if got.Accounts[0].Address != "IDR-test" || got.Accounts[0].Confirmed != 10 {
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
		{Address: "IDR-alice", Confirmed: 100, Mature: 90, Nonce: 4},
		{Address: "IDR-bob", Confirmed: 25, Mature: 25, Nonce: 2},
	}
	snapshot.Stakes = []staking.Record{
		{StakeID: "stake-2", OwnerAddress: "IDR-alice", Amount: 20, Status: staking.StatusActive},
		{StakeID: "stake-1", OwnerAddress: "IDR-alice", Amount: 30, Status: staking.StatusUnlocking, ReleaseHeight: 20},
		{StakeID: "stake-3", OwnerAddress: "IDR-bob", Amount: 10, Status: staking.StatusActive},
	}
	snapshot.Coinbases = []ledger.StateCoinbase{{Address: "IDR-alice", Amount: 50, Height: 11}}
	syncNativeBalances(&snapshot)
	snapshot.StateRoot, err = snapshotRoot(snapshot)
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

	account, found, err := store.GetStateAccount("IDR-alice")
	if err != nil {
		t.Fatal(err)
	}
	if !found || account.Confirmed != 100 || account.Nonce != 4 {
		t.Fatalf("unexpected account query: found=%v account=%#v", found, account)
	}
	if _, found, err := store.GetStateAccount("IDR-missing"); err != nil || found {
		t.Fatalf("unexpected missing account query: found=%v err=%v", found, err)
	}

	record, found, err := store.GetStateStake("stake-2")
	if err != nil {
		t.Fatal(err)
	}
	if !found || record.OwnerAddress != "IDR-alice" || record.Amount != 20 {
		t.Fatalf("unexpected stake query: found=%v record=%#v", found, record)
	}

	aliceStakes, err := store.GetStateStakesForAddress("IDR-alice")
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
		if !strings.HasPrefix(stake.OwnerAddress, "IDR-") {
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
	snapshot.Accounts = []ledger.StateAccount{{Address: "IDR-alice", Confirmed: 1, Mature: 1}}
	snapshot.Stakes = []staking.Record{{StakeID: "stake-1", OwnerAddress: "IDR-alice", Amount: 1, Status: staking.StatusActive}}
	syncNativeBalances(&snapshot)
	snapshot.StateRoot, err = snapshotRoot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteState(); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.GetStateAccount("IDR-alice"); !errors.Is(err, ErrStateNotInitialized) || found {
		t.Fatalf("expected cleared account index, found=%v err=%v", found, err)
	}
	if _, err := store.GetStateStakesForAddress("IDR-alice"); !errors.Is(err, ErrStateNotInitialized) {
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

func TestValidateStateIndexesDetectsOwnerIndexCorruption(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	snapshot.Stakes = []staking.Record{
		{
			StakeID:      "stake-1",
			OwnerAddress: "IDR-alice",
			Amount:       25,
			Status:       staking.StatusActive,
		},
	}
	snapshot.StateRoot, err = snapshotRoot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateStateIndexes(); err != nil {
		t.Fatalf("fresh index validation failed: %v", err)
	}

	err = store.db.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		return root.Bucket(stateStakesByOwnerBucket).Delete(
			stateStakeOwnerKey("IDR-alice", "stake-1"),
		)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateStateIndexes(); err == nil || !strings.Contains(err.Error(), "owner index") {
		t.Fatalf("expected owner index corruption, got %v", err)
	}
}

func TestValidateStateIndexesDetectsCoinbaseKeyCorruption(t *testing.T) {
	store, err := OpenBolt(t.TempDir() + "/chain.db")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	snapshot := emptySnapshot(t)
	snapshot.Height = 2
	snapshot.Coinbases = []ledger.StateCoinbase{
		{Address: "IDR-alice", Amount: 10, Height: 2},
	}
	snapshot.StateRoot, err = snapshotRoot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}

	err = store.db.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		return root.Bucket(stateCoinbasesBucket).Put([]byte("bad"), []byte("{}"))
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateStateIndexes(); err == nil || !strings.Contains(err.Error(), "coinbase key length") {
		t.Fatalf("expected coinbase key corruption, got %v", err)
	}
}


func TestOpenBoltRejectsPersistedStateRootMismatch(t *testing.T) {
	path := t.TempDir() + "/chain.db"

	store, err := OpenBolt(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := emptySnapshot(t)
	block := types.Block{
		Height:    snapshot.Height,
		Hash:      "genesis-hash",
		StateRoot: snapshot.StateRoot,
	}
	if err := store.SaveBlockAndState(block, snapshot); err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		meta := root.Bucket(stateMetaBucket)
		return meta.Put(stateRootKey, []byte("tampered-state-root"))
	})
	if closeErr := db.Close(); err != nil {
		t.Fatal(err)
	} else if closeErr != nil {
		t.Fatal(closeErr)
	}

	if _, err := OpenBolt(path); err == nil || !strings.Contains(err.Error(), "state root mismatch") {
		t.Fatalf("expected startup state-root rejection, got %v", err)
	}
}
