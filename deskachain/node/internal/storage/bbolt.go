package storage

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"

	"deskachain/internal/types"
)

var (
	blocksBucket = []byte("blocks")
	metaBucket   = []byte("meta")
	tipKey       = []byte("tip")
)

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
		_, err := tx.CreateBucketIfNotExists(metaBucket)
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
	return s.db.Update(func(tx *bolt.Tx) error { return tx.Bucket(blocksBucket).Delete(heightKey(height)) })
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
