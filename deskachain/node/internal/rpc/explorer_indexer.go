package rpc

import (
  "encoding/binary"
  "encoding/json"
  "errors"
  "fmt"
  "os"
  "path/filepath"
  "sort"
  "sync"

  bolt "go.etcd.io/bbolt"
  "deskachain/internal/asset"
  "deskachain/internal/config"
  "deskachain/internal/storage"
  "deskachain/internal/types"
 )

const explorerIndexerSchemaVersion = "v1"
const explorerIndexerFileName = "explorer.db"

type explorerIndexCursor struct { Height uint64; Hash string }
type explorerIndexedBlock struct { Height uint64; Hash string; PreviousHash string; Timestamp int64; TxCount int }
type explorerIndexedTx struct { ID string; BlockHeight uint64; BlockHash string; Index int; From string; To string; FeePayer string; AssetID string; Amount uint64; Fee uint64; Status string }
type explorerIndexedHistory struct { TxID string; BlockHeight uint64; BlockHash string; Role string; Counterparty string; AssetID string; Amount uint64; Fee uint64 }
type explorerIndexedAssetEvent struct { TxID string; BlockHeight uint64; BlockHash string; AssetID string; From string; To string; Amount uint64; Fee uint64 }

type explorerIndexer struct { paths config.Paths; profile config.NetworkConfig; mu sync.Mutex }
func newExplorerIndexer(paths config.Paths, profile config.NetworkConfig) *explorerIndexer { return &explorerIndexer{paths: paths, profile: profile} }
func (x *explorerIndexer) dbPath() string { return filepath.Join(x.paths.DataDir, explorerIndexerFileName) }
func (x *explorerIndexer) open() (*bolt.DB, error) {
  if err:=os.MkdirAll(x.paths.DataDir,0755); err!=nil{return nil,err}; db,err:=bolt.Open(x.dbPath(),0600,nil); if err!=nil{return nil,err}
  err=db.Update(func(tx *bolt.Tx) error { for _,n:=range []string{"meta","blocks","txs","addresses","assets"} { if _,e:=tx.CreateBucketIfNotExists([]byte(n));e!=nil{return e} }; return nil }); if err!=nil{_ = db.Close();return nil,err}; return db,nil
}
func clearExplorerIndexTx(tx *bolt.Tx) error {
  for _,n:=range []string{"blocks","txs","addresses","assets"} { b:=tx.Bucket([]byte(n)); if b==nil{continue}; var keys [][]byte; if err:=b.ForEach(func(k,v []byte)error{keys=append(keys,append([]byte(nil),k...));return nil});err!=nil{return err}; for _,k:=range keys{if err:=b.Delete(k);err!=nil{return err}} }; return tx.Bucket([]byte("meta")).Delete([]byte("cursor"))
}
func uint64Key(v uint64) []byte { b:=make([]byte,8); binary.BigEndian.PutUint64(b,v); return b }
func historyKey(address string, h uint64, txid, role string) []byte { return []byte(fmt.Sprintf("%s|%020d|%s|%s",address,h,txid,role)) }
func assetEventKey(id string,h uint64,txid string) []byte { return []byte(fmt.Sprintf("%s|%020d|%s",id,h,txid)) }
func findBlockByHeight(blocks []types.Block,h uint64)*types.Block{for i:=range blocks{if blocks[i].Height==h{return &blocks[i]}};return nil}
func historyAddress(item types.Transaction, role string) string { switch role { case "from": return item.From; case "to": return item.To; case "fee_payer": return item.EffectiveFeePayer(); default: return "" } }

func indexExplorerBlockTx(tx *bolt.Tx, block types.Block) error {
  br,err:=json.Marshal(explorerIndexedBlock{Height:block.Height,Hash:block.Hash,PreviousHash:block.PreviousHash,Timestamp:block.Timestamp,TxCount:len(block.Transactions)});if err!=nil{return err}; if err=tx.Bucket([]byte("blocks")).Put(uint64Key(block.Height),br);err!=nil{return err}
  for i,item:=range block.Transactions { ix:=explorerIndexedTx{ID:item.ID,BlockHeight:block.Height,BlockHash:block.Hash,Index:i,From:item.From,To:item.To,FeePayer:item.EffectiveFeePayer(),AssetID:item.EffectiveAssetID(),Amount:item.Amount,Fee:item.Fee,Status:"confirmed"}; raw,err:=json.Marshal(ix);if err!=nil{return err};if err:=tx.Bucket([]byte("txs")).Put([]byte(item.ID),raw);err!=nil{return err}
    events:=[]explorerIndexedHistory{}; if item.From!=""{events=append(events,explorerIndexedHistory{TxID:item.ID,BlockHeight:block.Height,BlockHash:block.Hash,Role:"from",Counterparty:item.To,AssetID:item.EffectiveAssetID(),Amount:item.Amount,Fee:item.Fee})}; if item.To!=""&&item.To!=item.From{events=append(events,explorerIndexedHistory{TxID:item.ID,BlockHeight:block.Height,BlockHash:block.Hash,Role:"to",Counterparty:item.From,AssetID:item.EffectiveAssetID(),Amount:item.Amount,Fee:item.Fee})}; if ix.FeePayer!=""&&ix.FeePayer!=item.From&&ix.FeePayer!=item.To{events=append(events,explorerIndexedHistory{TxID:item.ID,BlockHeight:block.Height,BlockHash:block.Hash,Role:"fee_payer",Counterparty:item.To,AssetID:item.EffectiveAssetID(),Amount:item.Amount,Fee:item.Fee})}
    for _,e:=range events{r,er:=json.Marshal(e);if er!=nil{return er};if er=tx.Bucket([]byte("addresses")).Put(historyKey(historyAddress(item,e.Role),e.BlockHeight,e.TxID,e.Role),r);er!=nil{return er}}
    if ix.AssetID!=""&&!asset.IsNative(ix.AssetID){e:=explorerIndexedAssetEvent{TxID:item.ID,BlockHeight:block.Height,BlockHash:block.Hash,AssetID:ix.AssetID,From:item.From,To:item.To,Amount:item.Amount,Fee:item.Fee};r,er:=json.Marshal(e);if er!=nil{return er};if er=tx.Bucket([]byte("assets")).Put(assetEventKey(e.AssetID,e.BlockHeight,e.TxID),r);er!=nil{return er}}
  }; return nil
}

func (x *explorerIndexer) sync() (ExplorerIndexerStatus,error) {
 x.mu.Lock(); defer x.mu.Unlock(); store,err:=storage.OpenBolt(x.paths.DB);if err!=nil{return ExplorerIndexerStatus{},err};blocks,err:=store.Blocks();_ = store.Close();if err!=nil{return ExplorerIndexerStatus{},err};sort.Slice(blocks,func(i,j int)bool{return blocks[i].Height<blocks[j].Height});if len(blocks)==0{return ExplorerIndexerStatus{},errors.New("chain is empty")}
 db,err:=x.open();if err!=nil{return ExplorerIndexerStatus{},err};defer db.Close();var cur explorerIndexCursor;_ = db.View(func(tx *bolt.Tx)error{r:=tx.Bucket([]byte("meta")).Get([]byte("cursor"));if r!=nil{return json.Unmarshal(r,&cur)};return nil})
 rebuild:=false;if cur.Hash!=""{if b:=findBlockByHeight(blocks,cur.Height);b!=nil&&b.Hash!=cur.Hash{rebuild=true}};if cur.Height>blocks[len(blocks)-1].Height{rebuild=true};if rebuild{cur=explorerIndexCursor{}}
 err=db.Update(func(tx *bolt.Tx)error{if rebuild||cur.Hash==""{if err:=clearExplorerIndexTx(tx);err!=nil{return err}};for _,b:=range blocks{if b.Height<=cur.Height{continue};if b.Height>0{p:=findBlockByHeight(blocks,b.Height-1);if p==nil||p.Hash!=b.PreviousHash{return fmt.Errorf("canonical parent mismatch at height %d",b.Height)}};if err:=indexExplorerBlockTx(tx,b);err!=nil{return err};cur=explorerIndexCursor{Height:b.Height,Hash:b.Hash};r,_:=json.Marshal(cur);if err:=tx.Bucket([]byte("meta")).Put([]byte("cursor"),r);err!=nil{return err}};return nil});if err!=nil{return ExplorerIndexerStatus{},err}
 return x.statusUnlocked()
}
func (x *explorerIndexer) statusUnlocked()(ExplorerIndexerStatus,error){db,err:=x.open();if err!=nil{return ExplorerIndexerStatus{},err};defer db.Close();var c explorerIndexCursor;err=db.View(func(tx *bolt.Tx)error{r:=tx.Bucket([]byte("meta")).Get([]byte("cursor"));if r!=nil{return json.Unmarshal(r,&c)};return nil});if err!=nil{return ExplorerIndexerStatus{},err};s,err:=storage.OpenBolt(x.paths.DB);if err!=nil{return ExplorerIndexerStatus{},err};defer s.Close();tip,err:=s.Tip();if err!=nil{return ExplorerIndexerStatus{},err};lag:=uint64(0);if tip.Height>c.Height{lag=tip.Height-c.Height};return ExplorerIndexerStatus{Mode:"persistent",IndexedHeight:c.Height,ChainHeight:tip.Height,Lag:lag,Ready:c.Hash!=""&&c.Height==tip.Height&&c.Hash==tip.Hash,LastIndexedHash:c.Hash,SchemaVersion:explorerIndexerSchemaVersion},nil}
func (x *explorerIndexer) status()(ExplorerIndexerStatus,error){x.mu.Lock();defer x.mu.Unlock();return x.statusUnlocked()}

func (x *explorerIndexer) transaction(id string)(explorerIndexedTx,bool,error){db,err:=x.open();if err!=nil{return explorerIndexedTx{},false,err};defer db.Close();var out explorerIndexedTx;err=db.View(func(tx *bolt.Tx)error{r:=tx.Bucket([]byte("txs")).Get([]byte(id));if r==nil{return nil};return json.Unmarshal(r,&out)});return out,out.ID!=""&&err==nil,err}
func (x *explorerIndexer) addressHistory(address string,limit,offset int)([]explorerIndexedHistory,int,error){db,err:=x.open();if err!=nil{return nil,0,err};defer db.Close();var all []explorerIndexedHistory;err=db.View(func(tx *bolt.Tx)error{return tx.Bucket([]byte("addresses")).ForEach(func(k,v []byte)error{var e explorerIndexedHistory;if err:=json.Unmarshal(v,&e);err!=nil{return err};prefix := []byte(address+"|"); if strings.HasPrefix(string(k), string(prefix)){all=append(all,e)};return nil})});if err!=nil{return nil,0,err};sort.Slice(all,func(i,j int)bool{if all[i].BlockHeight!=all[j].BlockHeight{return all[i].BlockHeight>all[j].BlockHeight};return all[i].TxID>all[j].TxID});total:=len(all);if offset>total{offset=total};end:=offset+limit;if end>total{end=total};return all[offset:end],total,nil}