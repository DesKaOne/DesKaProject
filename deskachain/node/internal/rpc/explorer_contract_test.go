package rpc

import "testing"

func TestExplorerContractVersionAndPaginationShape(t *testing.T) {
	if ExplorerAPIVersion != "v1" { t.Fatalf("ExplorerAPIVersion = %q, want v1", ExplorerAPIVersion) }
	page := ExplorerPage{Limit: 20, Offset: 0, Count: 1, TotalCount: 1}
	if page.Limit != 20 || page.Offset != 0 || page.Count != 1 || page.TotalCount != 1 { t.Fatalf("unexpected explorer page contract: %#v", page) }
}

func TestExplorerIndexerStatusIsDerivedOnly(t *testing.T) {
	status := ExplorerIndexerStatus{Mode: "scan", IndexedHeight: 10, ChainHeight: 12, Lag: 2, Ready: false, SchemaVersion: "v1"}
	if status.Lag != status.ChainHeight-status.IndexedHeight { t.Fatalf("lag contract mismatch: %#v", status) }
	if status.Ready { t.Fatal("lagging index must not report ready") }
}