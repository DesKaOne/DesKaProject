package storage

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"

	"deskachain/internal/ledger"
	"deskachain/internal/staking"
	"deskachain/internal/state"
	"deskachain/internal/types"
)

var (
	blocksBucket        = []byte("blocks")
	metaBucket          = []byte("meta")
	tipKey              = []byte("tip")
	stateBucket         = []byte("state")
	stateMetaBucket     = []byte("meta")
	stateAccountsBucket      = []byte("accounts")
	stateStakesBucket        = []byte("stakes")
	stateStakesByOwnerBucket = []byte("stakes_by_owner")
	stateVersionKey     = []byte("version")
	stateHeightKey      = []byte("height")
	stateRootKey        = []byte("root")
)

var ErrStateNotInitialized = errors.New("state is not initialized")

type BoltStore struct {
	path string
	db   *bolt.DB
}

func OpenBolt(path string) (*BoltStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	store := &BoltStore{path: path, db: db}
	return store, store.Init()
}

func (s *BoltStore) Init() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists(blocksBucket); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists(metaBucket); err != nil {
			return err
		}
		root, err := tx.CreateBucketIfNotExists(stateBucket)
		if err != nil {
			return err
		}
		if _, err := root.CreateBucketIfNotExists(stateMetaBucket); err != nil {
			return err
		}
		if _, err := root.CreateBucketIfNotExists(stateAccountsBucket); err != nil {
			return err
		}
		if _, err := root.CreateBucketIfNotExists(stateStakesBucket); err != nil {
			return err
		}
		_, err = root.CreateBucketIfNotExists(stateStakesByOwnerBucket)
		return err
	})
}

func (s *BoltStore) Close() error {
	return s.db.Close()
}

func (s *BoltStore) HasChain() (bool, error) {
	var ok bool
	err := s.db.View(func(tx *bolt.Tx) error {
		ok = tx.Bucket(metaBucket).Get(tipKey) != nil
		return nil
	})
	return ok, err
}

func (s *BoltStore) SaveBlock(block types.Block) error {
	raw, err := json.Marshal(block)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		key := heightKey(block.Height)
		if err := tx.Bucket(blocksBucket).Put(key, raw); err != nil {
			return err
		}
		return tx.Bucket(metaBucket).Put(tipKey, key)
	})
}

func (s *BoltStore) SaveBlockAndState(block types.Block, snapshot state.Snapshot) error {
	raw, err := json.Marshal(block)
	if err != nil {
		return err
	}
	if err := validateBlockStatePair(block, snapshot); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		key := heightKey(block.Height)
		if err := tx.Bucket(blocksBucket).Put(key, raw); err != nil {
			return err
		}
		if err := tx.Bucket(metaBucket).Put(tipKey, key); err != nil {
			return err
		}
		return saveStateTx(tx, snapshot)
	})
}

func (s *BoltStore) Blocks() ([]types.Block, error) {
	var blocks []types.Block
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(blocksBucket).ForEach(func(_, v []byte) error {
			var block types.Block
			if err := json.Unmarshal(v, &block); err != nil {
				return err
			}
			blocks = append(blocks, block)
			return nil
		})
	})
	return blocks, err
}

func (s *BoltStore) Tip() (types.Block, error) {
	var block types.Block
	err := s.db.View(func(tx *bolt.Tx) error {
		key := tx.Bucket(metaBucket).Get(tipKey)
		if key == nil {
			return errors.New("chain is not initialized")
		}
		raw := tx.Bucket(blocksBucket).Get(key)
		if raw == nil {
			return errors.New("tip block not found")
		}
		return json.Unmarshal(raw, &block)
	})
	return block, err
}

func heightKey(height uint64) []byte {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	return key
}

func (s *BoltStore) GetBlockByHeight(height uint64) (types.Block, error) {
	var block types.Block
	err := s.db.View(func(tx *bolt.Tx) error {
		raw := tx.Bucket(blocksBucket).Get(heightKey(height))
		if raw == nil {
			return errors.New("block not found")
		}
		return json.Unmarshal(raw, &block)
	})
	return block, err
}

func (s *BoltStore) DeleteBlockByHeight(height uint64) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(blocksBucket).Delete(heightKey(height))
	})
}

func (s *BoltStore) ReplaceFromHeight(from uint64, blocks []types.Block) error {
	return s.replaceFromHeight(from, blocks, nil)
}

func (s *BoltStore) ReplaceFromHeightAndState(from uint64, blocks []types.Block, snapshot state.Snapshot) error {
	if err := validateBlockStateBranch(blocks, snapshot); err != nil {
		return err
	}
	return s.replaceFromHeight(from, blocks, &snapshot)
}

func (s *BoltStore) replaceFromHeight(from uint64, blocks []types.Block, snapshot *state.Snapshot) error {
	if len(blocks) == 0 {
		return errors.New("replacement branch is empty")
	}
	for i, block := range blocks {
		expectedHeight, err := checkedReplacementHeight(from, uint64(i))
		if err != nil {
			return err
		}
		if block.Height != expectedHeight {
			return errors.New("replacement branch is not contiguous")
		}
	}

	rawBlocks := make([][]byte, len(blocks))
	for i, block := range blocks {
		raw, err := json.Marshal(block)
		if err != nil {
			return err
		}
		rawBlocks[i] = raw
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)
		if from > 0 && b.Get(heightKey(from-1)) == nil {
			return errors.New("replacement branch predecessor not found")
		}
		cursor := b.Cursor()
		for key, _ := cursor.Seek(heightKey(from)); key != nil; key, _ = cursor.Next() {
			if err := cursor.Delete(); err != nil {
				return err
			}
		}
		for i, block := range blocks {
			height, err := checkedReplacementHeight(from, uint64(i))
			if err != nil {
				return err
			}
			if err := b.Put(heightKey(height), rawBlocks[i]); err != nil {
				return err
			}
		}
		if err := tx.Bucket(metaBucket).Put(tipKey, heightKey(blocks[len(blocks)-1].Height)); err != nil {
			return err
		}
		if snapshot != nil {
			if err := saveStateTx(tx, *snapshot); err != nil {
				return err
			}
		}
		return nil
	})
}

func checkedReplacementHeight(from, offset uint64) (uint64, error) {
	if ^uint64(0)-from < offset {
		return 0, errors.New("replacement branch height overflow")
	}
	return from + offset, nil
}

func (s *BoltStore) SetTip(height uint64, hash string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(blocksBucket)
		raw := b.Get(heightKey(height))
		if raw == nil {
			return errors.New("tip block not found")
		}
		var block types.Block
		if err := json.Unmarshal(raw, &block); err != nil {
			return err
		}
		if block.Hash != hash {
			return errors.New("tip hash mismatch")
		}
		return tx.Bucket(metaBucket).Put(tipKey, heightKey(height))
	})
}

func (s *BoltStore) GetHeight() (uint64, error) {
	block, err := s.Tip()
	if err != nil {
		return 0, err
	}
	return block.Height, nil
}

func (s *BoltStore) GetTip() (uint64, string, error) {
	block, err := s.Tip()
	if err != nil {
		return 0, "", err
	}
	return block.Height, block.Hash, nil
}

func (s *BoltStore) SaveState(snapshot state.Snapshot) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		return saveStateTx(tx, snapshot)
	})
}

func (s *BoltStore) GetStateMetadata() (version uint8, height uint64, stateRoot string, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		if root == nil {
			return ErrStateNotInitialized
		}
		meta := root.Bucket(stateMetaBucket)
		if meta == nil {
			return ErrStateNotInitialized
		}
		rawVersion := meta.Get(stateVersionKey)
		rawHeight := meta.Get(stateHeightKey)
		rawRoot := meta.Get(stateRootKey)
		if len(rawVersion) != 1 || len(rawHeight) != 8 || len(rawRoot) == 0 {
			return ErrStateNotInitialized
		}
		version = rawVersion[0]
		height = binary.BigEndian.Uint64(rawHeight)
		stateRoot = string(rawRoot)
		return nil
	})
	return version, height, stateRoot, err
}

func (s *BoltStore) GetStateAccount(address string) (ledger.StateAccount, bool, error) {
	var account ledger.StateAccount
	var found bool
	err := s.db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		if root == nil {
			return ErrStateNotInitialized
		}
		bucket := root.Bucket(stateAccountsBucket)
		if bucket == nil {
			return ErrStateNotInitialized
		}
		raw := bucket.Get([]byte(address))
		if raw == nil {
			return nil
		}
		if err := json.Unmarshal(raw, &account); err != nil {
			return err
		}
		if account.Address != address {
			return errors.New("state account key mismatch")
		}
		found = true
		return nil
	})
	return account, found, err
}

func (s *BoltStore) GetStateStake(stakeID string) (staking.Record, bool, error) {
	var record staking.Record
	var found bool
	err := s.db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		if root == nil {
			return ErrStateNotInitialized
		}
		bucket := root.Bucket(stateStakesBucket)
		if bucket == nil {
			return ErrStateNotInitialized
		}
		raw := bucket.Get([]byte(stakeID))
		if raw == nil {
			return nil
		}
		if err := json.Unmarshal(raw, &record); err != nil {
			return err
		}
		if record.StakeID != stakeID {
			return errors.New("state stake key mismatch")
		}
		found = true
		return nil
	})
	return record, found, err
}

func stateStakeOwnerPrefix(address string) []byte {
	return append([]byte(address), 0)
}

func stateStakeOwnerKey(address, stakeID string) []byte {
	return append(stateStakeOwnerPrefix(address), []byte(stakeID)...)
}

func (s *BoltStore) GetStateStakesForAddress(address string) ([]staking.Record, error) {
	var records []staking.Record
	err := s.db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		if root == nil {
			return ErrStateNotInitialized
		}
		bucket := root.Bucket(stateStakesByOwnerBucket)
		if bucket == nil {
			return ErrStateNotInitialized
		}
		prefix := stateStakeOwnerPrefix(address)
		cursor := bucket.Cursor()
		for key, value := cursor.Seek(prefix); key != nil && bytes.HasPrefix(key, prefix); key, value = cursor.Next() {
			var record staking.Record
			if err := json.Unmarshal(value, &record); err != nil {
				return err
			}
			if record.OwnerAddress != address {
				return errors.New("state owner stake index mismatch")
			}
			records = append(records, record)
		}
		return nil
	})
	return records, err
}

func (s *BoltStore) LoadState() (state.Snapshot, error) {
	var snapshot state.Snapshot
	err := s.db.View(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		if root == nil {
			return ErrStateNotInitialized
		}
		meta := root.Bucket(stateMetaBucket)
		accountsBucket := root.Bucket(stateAccountsBucket)
		stakesBucket := root.Bucket(stateStakesBucket)
		if meta == nil || accountsBucket == nil || stakesBucket == nil {
			return ErrStateNotInitialized
		}
		version := meta.Get(stateVersionKey)
		height := meta.Get(stateHeightKey)
		stateRoot := meta.Get(stateRootKey)
		if len(version) != 1 || len(height) != 8 || len(stateRoot) == 0 {
			return ErrStateNotInitialized
		}
		snapshot.Version = version[0]
		snapshot.Height = binary.BigEndian.Uint64(height)
		snapshot.StateRoot = string(stateRoot)
		if err := accountsBucket.ForEach(func(k, v []byte) error {
			var account ledger.StateAccount
			if err := json.Unmarshal(v, &account); err != nil {
				return err
			}
			if account.Address != string(k) {
				return errors.New("state account key mismatch")
			}
			snapshot.Accounts = append(snapshot.Accounts, account)
			return nil
		}); err != nil {
			return err
		}
		if err := stakesBucket.ForEach(func(k, v []byte) error {
			var record staking.Record
			if err := json.Unmarshal(v, &record); err != nil {
				return err
			}
			if record.StakeID != string(k) {
				return errors.New("state stake key mismatch")
			}
			snapshot.Stakes = append(snapshot.Stakes, record)
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return state.Snapshot{}, err
	}
	if err := snapshot.Validate(); err != nil {
		return state.Snapshot{}, err
	}
	return snapshot, nil
}

func (s *BoltStore) DeleteState() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		root := tx.Bucket(stateBucket)
		if root == nil {
			return ErrStateNotInitialized
		}
		meta := root.Bucket(stateMetaBucket)
		accounts := root.Bucket(stateAccountsBucket)
		stakes := root.Bucket(stateStakesBucket)
		stakesByOwner := root.Bucket(stateStakesByOwnerBucket)
		if meta == nil || accounts == nil || stakes == nil || stakesByOwner == nil {
			return ErrStateNotInitialized
		}
		if err := clearBucket(meta); err != nil {
			return err
		}
		if err := clearBucket(accounts); err != nil {
			return err
		}
		return clearBucket(stakes)
	})
}

func validateBlockStatePair(block types.Block, snapshot state.Snapshot) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	if snapshot.Height != block.Height {
		return errors.New("state snapshot height does not match block height")
	}
	if block.ProtocolVersion() == types.BlockVersionCanonical {
		if block.StateRoot == "" || block.StateRoot != snapshot.StateRoot {
			return errors.New("block state root does not match persisted state")
		}
	}
	return nil
}

func validateBlockStateBranch(blocks []types.Block, snapshot state.Snapshot) error {
	if len(blocks) == 0 {
		return errors.New("replacement branch is empty")
	}
	if err := snapshot.Validate(); err != nil {
		return err
	}
	tip := blocks[len(blocks)-1]
	if snapshot.Height != tip.Height {
		return errors.New("state snapshot height does not match replacement tip")
	}
	for _, block := range blocks {
		if block.ProtocolVersion() == types.BlockVersionCanonical && block.StateRoot == "" {
			return errors.New("canonical replacement block has empty state root")
		}
	}
	if tip.ProtocolVersion() == types.BlockVersionCanonical && tip.StateRoot != snapshot.StateRoot {
		return errors.New("replacement tip state root does not match persisted state")
	}
	return nil
}

func saveStateTx(tx *bolt.Tx, snapshot state.Snapshot) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	root, err := tx.CreateBucketIfNotExists(stateBucket)
	if err != nil {
		return err
	}
	meta, err := root.CreateBucketIfNotExists(stateMetaBucket)
	if err != nil {
		return err
	}
	accountsBucket, err := root.CreateBucketIfNotExists(stateAccountsBucket)
	if err != nil {
		return err
	}
	stakesBucket, err := root.CreateBucketIfNotExists(stateStakesBucket)
	if err != nil {
		return err
	}
	stakesByOwnerBucket, err := root.CreateBucketIfNotExists(stateStakesByOwnerBucket)
	if err != nil {
		return err
	}

	if err := clearBucket(meta); err != nil {
		return err
	}
	if err := clearBucket(accountsBucket); err != nil {
		return err
	}
	if err := clearBucket(stakesBucket); err != nil {
		return err
	}
	if err := clearBucket(stakesByOwnerBucket); err != nil {
		return err
	}

	if err := meta.Put(stateVersionKey, []byte{snapshot.Version}); err != nil {
		return err
	}
	if err := meta.Put(stateHeightKey, heightKey(snapshot.Height)); err != nil {
		return err
	}
	if err := meta.Put(stateRootKey, []byte(snapshot.StateRoot)); err != nil {
		return err
	}

	for _, account := range snapshot.Accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return err
		}
		if err := accountsBucket.Put([]byte(account.Address), raw); err != nil {
			return err
		}
	}
	for _, record := range snapshot.Stakes {
		raw, err := json.Marshal(record)
		if err != nil {
			return err
		}
		if err := stakesBucket.Put([]byte(record.StakeID), raw); err != nil {
			return err
		}
		if err := stakesByOwnerBucket.Put(stateStakeOwnerKey(record.OwnerAddress, record.StakeID), raw); err != nil {
			return err
		}
	}
	return nil
}

func clearBucket(bucket *bolt.Bucket) error {
	if bucket == nil {
		return nil
	}
	var keys [][]byte
	if err := bucket.ForEach(func(k, _ []byte) error {
		keys = append(keys, append([]byte(nil), k...))
		return nil
	}); err != nil {
		return err
	}
	for _, key := range keys {
		if err := bucket.Delete(key); err != nil {
			return err
		}
	}
	return nil
}
