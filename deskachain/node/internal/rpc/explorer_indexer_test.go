package rpc

import (
 "testing"
 bolt "go.etcd.io/bbolt"
 "deskachain/internal/config"
 "deskachain/internal/types"
 )

func TestExplorerIndexerPersistsTransactionIndex(t *testing.T) {
 dir:=t.TempDir(); paths:=config.NewPaths(dir); x:=newExplorerIndexer(paths,config.Localnet()); db,err:=x.open();if err!=nil{t.Fatal(err)}; block:=types.Block{Height:0,Hash:"genesis",Timestamp:1,Transactions:[]types.Transaction{{ID:"tx1",From:"IDRfrom",To:"IDRto",Amount:10}}};err=db.Update(func(tx *bolt.Tx)error{return indexExplorerBlockTx(tx,block)});_ = db.Close();if err!=nil{t.Fatal(err)}; got,ok,err:=x.transaction("tx1");if err!=nil||!ok{t.Fatalf("transaction lookup failed: %v %v",ok,err)};if got.BlockHeight!=0||got.BlockHash!="genesis"{t.Fatalf("unexpected index: %#v",got)}
}