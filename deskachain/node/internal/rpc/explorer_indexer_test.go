package rpc

import (
	"context"
	"encoding/json"
	"indochain/internal/config"
	"errors"
	"indochain/internal/types"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

func TestExplorerIndexerPersistsTransactionIndex(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	db, err := x.open()
	if err != nil {
		t.Fatal(err)
	}
	block := types.Block{Height: 0, Hash: "genesis", Timestamp: 1, Transactions: []types.Transaction{{ID: "tx1", From: "INDfrom", To: "INDto", Amount: 10}}}
	err = db.Update(func(tx *bolt.Tx) error { return indexExplorerBlockTx(tx, block) })
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := x.transaction("tx1")
	if err != nil || !ok {
		t.Fatalf("transaction lookup failed: %v %v", ok, err)
	}
	if got.BlockHeight != 0 || got.BlockHash != "genesis" {
		t.Fatalf("unexpected index: %#v", got)
	}
}

func TestExplorerIndexerAddressHistoryIsAddressScoped(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	db, err := x.open()
	if err != nil {
		t.Fatal(err)
	}
	block := types.Block{Height: 1, Hash: "block1", Transactions: []types.Transaction{
		{ID: "tx-from", From: "INDfrom", To: "INDto", Amount: 10},
		{ID: "tx-to", From: "INDother", To: "INDto", Amount: 20},
	}}
	err = db.Update(func(tx *bolt.Tx) error { return indexExplorerBlockTx(tx, block) })
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	from, total, err := x.addressHistory("INDfrom", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(from) != 1 || from[0].TxID != "tx-from" {
		t.Fatalf("unexpected from history: total=%d items=%#v", total, from)
	}
	to, total, err := x.addressHistory("INDto", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(to) != 2 {
		t.Fatalf("unexpected to history: total=%d items=%#v", total, to)
	}
	missing, total, err := x.addressHistory("INDmissing", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(missing) != 0 {
		t.Fatalf("unexpected missing history: total=%d items=%#v", total, missing)
	}
}

func TestExplorerIndexerAssetEventsAreScoped(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	db, err := x.open()
	if err != nil {
		t.Fatal(err)
	}
	block := types.Block{Height: 2, Hash: "block2", Transactions: []types.Transaction{
		{ID: "tx-asset-1", From: "INDfrom", To: "INDto", Amount: 7, AssetID: "asset-a"},
		{ID: "tx-native", From: "INDfrom", To: "INDto", Amount: 9},
		{ID: "tx-asset-2", From: "INDother", To: "INDto", Amount: 11, AssetID: "asset-b"},
	}}
	err = db.Update(func(tx *bolt.Tx) error { return indexExplorerBlockTx(tx, block) })
	_ = db.Close()
	if err != nil {
		t.Fatal(err)
	}
	items, total, err := x.assetEvents("asset-a", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].TxID != "tx-asset-1" {
		t.Fatalf("unexpected asset-a history: total=%d items=%#v", total, items)
	}
	items, total, err = x.assetEvents("asset-b", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(items) != 1 || items[0].TxID != "tx-asset-2" {
		t.Fatalf("unexpected asset-b history: total=%d items=%#v", total, items)
	}
	items, total, err = x.assetEvents("asset-missing", 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(items) != 0 {
		t.Fatalf("unexpected missing asset history: total=%d items=%#v", total, items)
	}
}

func TestExplorerIndexerRunStopsOnContextCancellation(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		x.run(ctx, time.Millisecond)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("explorer indexer did not stop after context cancellation")
	}
}

func TestExplorerIndexerStatsCountsIndexedRecords(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	db, err := x.open()
	if err != nil { t.Fatal(err) }
	block := types.Block{Height: 3, Hash: "block3", Transactions: []types.Transaction{
		{ID: "tx-stats-1", From: "INDfrom", To: "INDto", Amount: 10},
		{ID: "tx-stats-2", From: "INDfrom2", To: "INDto2", Amount: 20, AssetID: "asset-stats"},
	}}
	err = db.Update(func(tx *bolt.Tx) error { return indexExplorerBlockTx(tx, block) })
	_ = db.Close()
	if err != nil { t.Fatal(err) }
	stats, err := x.stats(ExplorerIndexerStatus{IndexedHeight: 3, ChainHeight: 3, Ready: true, SchemaVersion: explorerIndexerSchemaVersion})
	if err != nil { t.Fatal(err) }
	if stats.BlockCount != 1 || stats.TransactionCount != 2 || stats.AddressHistoryCount != 4 || stats.AssetEventCount != 1 {
		t.Fatalf("unexpected index stats: %#v", stats)
	}
}


func TestExplorerIndexerStatsExposePersistedSyncMetrics(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	db, err := x.open()
	if err != nil { t.Fatal(err) }
	metrics := explorerIndexerMetrics{
		SyncCount: 7,
		LastSyncAtUnix: 123,
		LastSyncDurationMs: 250,
		LastSyncBlockCount: 5,
		BlocksPerSecond: 20,
		SyncFailureCount: 2,
		LastSyncError: "canonical parent mismatch at height 4",
	}
	err = db.Update(func(tx *bolt.Tx) error {
		raw, err := json.Marshal(metrics)
		if err != nil { return err }
		return tx.Bucket([]byte("meta")).Put([]byte("metrics"), raw)
	})
	_ = db.Close()
	if err != nil { t.Fatal(err) }
	stats, err := x.stats(ExplorerIndexerStatus{IndexedHeight: 5, ChainHeight: 5, Ready: true, SchemaVersion: explorerIndexerSchemaVersion})
	if err != nil { t.Fatal(err) }
	if stats.SyncCount != 7 || stats.LastSyncAtUnix != 123 || stats.LastSyncDurationMs != 250 || stats.LastSyncBlockCount != 5 || stats.BlocksPerSecond != 20 || stats.SyncFailureCount != 2 || stats.LastSyncError != "canonical parent mismatch at height 4" {
		t.Fatalf("unexpected sync metrics: %#v", stats)
	}
}


func TestExplorerIndexerRecordSyncFailurePersistsMetrics(t *testing.T) {
	dir := t.TempDir()
	paths := config.NewPaths(dir)
	x := newExplorerIndexer(paths, config.Localnet())
	if err := x.recordSyncFailure(errors.New("simulated sync failure")); err != nil {
		t.Fatal(err)
	}
	stats, err := x.stats(ExplorerIndexerStatus{IndexedHeight: 0, ChainHeight: 1, Ready: false, SchemaVersion: explorerIndexerSchemaVersion})
	if err != nil { t.Fatal(err) }
	if stats.SyncFailureCount != 1 || stats.LastSyncError != "simulated sync failure" {
		t.Fatalf("unexpected failure metrics: %#v", stats)
	}
}
