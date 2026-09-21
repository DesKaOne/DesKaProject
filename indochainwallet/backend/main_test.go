package main

import "testing"

func TestMapToTxPreservesFields(t *testing.T){ in:=map[string]any{"from":"a","to":"b","amount":1}; out:=mapToTx(in); if out["from"]!="a"||out["to"]!="b"{t.Fatal("transaction fields changed")} }
