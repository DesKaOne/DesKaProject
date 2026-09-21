package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/scrypt"
)

const (
	secureStoreVersion = 1
	saltSize            = 16
	nonceSize           = 12
	keySize             = 32
	scryptN             = 1 << 15
	scryptR             = 8
	scryptP             = 1
)

type encryptedWalletFile struct {
	Version    int    `json:"version"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func marshalEncryptedWallets(wallets []Wallet, password string) ([]byte, error) {
	if password == "" {
		return nil, errors.New("wallet storage password is required")
	}
	plain, err := json.Marshal(wallets)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}
	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keySize)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nil, nonce, plain, nil)
	payload := encryptedWalletFile{
		Version: secureStoreVersion,
		Salt: base64.StdEncoding.EncodeToString(salt),
		Nonce: base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
	return json.MarshalIndent(payload, "", "  ")
}

func unmarshalEncryptedWallets(raw []byte, password string) ([]Wallet, error) {
	if password == "" {
		return nil, errors.New("wallet storage password is required")
	}
	var payload encryptedWalletFile
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, errors.New("invalid encrypted wallet file")
	}
	if payload.Version != secureStoreVersion {
		return nil, fmt.Errorf("unsupported encrypted wallet version: %d", payload.Version)
	}
	salt, err := base64.StdEncoding.DecodeString(payload.Salt)
	if err != nil || len(salt) != saltSize {
		return nil, errors.New("invalid wallet salt")
	}
	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil || len(nonce) != nonceSize {
		return nil, errors.New("invalid wallet nonce")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, errors.New("invalid wallet ciphertext")
	}
	key, err := scrypt.Key([]byte(password), salt, scryptN, scryptR, scryptP, keySize)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("wallet password or ciphertext is invalid")
	}
	var wallets []Wallet
	if err := json.Unmarshal(plain, &wallets); err != nil {
		return nil, errors.New("invalid wallet payload")
	}
	return wallets, nil
}

type SecureStore struct {
	path string
	password string
}

func NewSecureStore(path, password string) SecureStore {
	return SecureStore{path: path, password: password}
}

func (s SecureStore) Load() ([]Wallet, error) {
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	return unmarshalEncryptedWallets(raw, s.password)
}

func (s SecureStore) Save(wallets []Wallet) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return err
	}
	raw, err := marshalEncryptedWallets(wallets, s.password)
	if err != nil {
		return err
	}
	return atomicWriteFile(s.path, raw, 0600)
}

func (s SecureStore) Add(wallet Wallet) error {
	wallets, err := s.Load()
	if err != nil {
		return err
	}
	for _, existing := range wallets {
		if existing.Address == wallet.Address {
			return nil
		}
	}
	wallets = append(wallets, wallet)
	return s.Save(wallets)
}

func (s SecureStore) Find(address string) (Wallet, bool, error) {
	wallets, err := s.Load()
	if err != nil {
		return Wallet{}, false, err
	}
	for _, wallet := range wallets {
		if wallet.Address == address {
			return wallet, true, nil
		}
	}
	return Wallet{}, false, nil
}
