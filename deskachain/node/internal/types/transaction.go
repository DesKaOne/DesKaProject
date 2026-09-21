package types

import (
	"encoding/json"
	"fmt"
	"time"

	"indochain/internal/asset"
	"indochain/internal/config"
	"indochain/internal/crypto"
)

const (
	TxVersionLegacy    uint32 = 1
	TxVersionCanonical uint32 = 2
	TxVersionAsset     uint32 = 3
)

const (
	CoinbaseSender               = "COINBASE"
	MaxSupportedTxVersion uint32 = TxVersionAsset
)

const (
	TxTypeTransfer    = "transfer"
	TxTypeCoinbase    = "coinbase"
	TxTypeStakeLock   = "stake_lock"
	TxTypeStakeUnlock = "stake_unlock"
	TxTypeAssetCreate = "asset_create"
	TxTypeAssetMint   = "asset_mint"
	TxTypeAssetBurn   = "asset_burn"
)

type Transaction struct {
	Version           uint32 `json:"version,omitempty"`
	ID                string `json:"id"`
	From              string `json:"from"`
	To                string `json:"to"`
	Amount            uint64 `json:"amount"`
	Fee               uint64 `json:"fee"`
	Nonce             uint64 `json:"nonce"`
	Timestamp         int64  `json:"timestamp"`
	Signature         string `json:"signature"`
	PublicKey         string `json:"public_key"`
	Coinbase          bool   `json:"coinbase"`
	Type              string `json:"type,omitempty"`
	StakeID           string `json:"stake_id,omitempty"`
	AssetID           string `json:"asset_id,omitempty"`
	FeePayer          string `json:"fee_payer,omitempty"`
	FeePayerPublicKey string `json:"fee_payer_public_key,omitempty"`
	FeePayerSignature string `json:"fee_payer_signature,omitempty"`
	AssetName         string `json:"asset_name,omitempty"`
	AssetSymbol       string `json:"asset_symbol,omitempty"`
	AssetDecimals     uint8  `json:"asset_decimals,omitempty"`
	AssetMaxSupply    uint64 `json:"asset_max_supply,omitempty"`
	AssetMintable     bool   `json:"asset_mintable,omitempty"`
	AssetBurnable     bool   `json:"asset_burnable,omitempty"`
	AssetPausable     bool   `json:"asset_pausable,omitempty"`
	AssetPermissioned bool   `json:"asset_permissioned,omitempty"`
}

func NewUnsignedTransaction(from, to string, amount, fee, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionLegacy,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
	}
	tx.ID = tx.CalculateID()
	return tx
}

func NewStakeLockTransaction(address string, amount, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionLegacy,
		From:      address,
		To:        address,
		Amount:    amount,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		Type:      TxTypeStakeLock,
	}
	tx.ID = tx.CalculateID()
	tx.StakeID = tx.ID
	return tx
}

func NewStakeUnlockTransaction(address, stakeID string, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionLegacy,
		From:      address,
		To:        address,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		Type:      TxTypeStakeUnlock,
		StakeID:   stakeID,
	}
	tx.ID = tx.CalculateID()
	return tx
}

// NewAssetTransferTransaction creates a v3 transfer whose amount is denominated
// in AssetID while Fee is always denominated in native dIDR.
func NewAssetTransferTransaction(from, to, assetID string, amount, fee, nonce uint64) Transaction {
	tx := newAssetTransaction(TxTypeTransfer, from, to, assetID, amount, fee, nonce)
	return tx
}

// NewAssetCreateTransaction creates a v3 fungible token definition.
// The resulting AssetID is derived deterministically from the transaction ID.
func NewAssetCreateTransaction(from, name, symbol string, decimals uint8, maxSupply uint64, mintable, burnable, pausable, permissioned bool, fee, nonce uint64) Transaction {
	tx := newAssetTransaction(TxTypeAssetCreate, from, "", "", 0, fee, nonce)
	tx.AssetName = name
	tx.AssetSymbol = symbol
	tx.AssetDecimals = decimals
	tx.AssetMaxSupply = maxSupply
	tx.AssetMintable = mintable
	tx.AssetBurnable = burnable
	tx.AssetPausable = pausable
	tx.AssetPermissioned = permissioned
	tx.ID = tx.CalculateID()
	tx.AssetID = asset.DerivedID(tx.ID)
	return tx
}

// NewAssetMintTransaction mints an issued token to To. From is the issuer authority.
func NewAssetMintTransaction(from, assetID, to string, amount, fee, nonce uint64) Transaction {
	return newAssetTransaction(TxTypeAssetMint, from, to, assetID, amount, fee, nonce)
}

// NewAssetBurnTransaction burns issued tokens from From.
func NewAssetBurnTransaction(from, assetID string, amount, fee, nonce uint64) Transaction {
	return newAssetTransaction(TxTypeAssetBurn, from, from, assetID, amount, fee, nonce)
}

func newAssetTransaction(txType, from, to, assetID string, amount, fee, nonce uint64) Transaction {
	tx := Transaction{
		Version:   TxVersionAsset,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		Type:      txType,
		AssetID:   assetID,
		FeePayer:  from,
	}
	if tx.AssetID == "" && txType != TxTypeAssetCreate {
		tx.AssetID = asset.NativeAssetID
	}
	return tx
}

func NewCoinbaseTransaction(to string, amount uint64, height uint64) Transaction {
	return NewCoinbaseTransactionWithVersion(to, amount, height, TxVersionLegacy)
}

func NewCoinbaseTransactionWithVersion(to string, amount uint64, height uint64, version uint32) Transaction {
	tx := Transaction{
		Version:   version,
		From:      CoinbaseSender,
		To:        to,
		Amount:    amount,
		Timestamp: int64(height),
		Coinbase:  true,
	}
	tx.ID = tx.CalculateID()
	return tx
}

// RefreshDerivedAssetID binds an asset-create transaction to its final
// chain-bound transaction ID. AssetID is deliberately absent from the create
// signing preimage because the ID is what deterministically derives the asset.
func (tx *Transaction) RefreshDerivedAssetID() error {
	if tx == nil {
		return fmt.Errorf("transaction is required")
	}
	if tx.ProtocolVersion() != TxVersionAsset || tx.TxType() != TxTypeAssetCreate {
		return fmt.Errorf("derived asset id requires asset-create transaction version 3")
	}
	if tx.ID == "" {
		return fmt.Errorf("transaction id is required")
	}
	tx.AssetID = asset.DerivedID(tx.ID)
	return nil
}

// ValidateAssetEnvelope validates v3 asset transaction structure without
// applying balances or issuer state. Consensus execution performs those checks.
func (tx Transaction) ValidateAssetEnvelope() error {
	if tx.ProtocolVersion() != TxVersionAsset {
		return fmt.Errorf("asset envelope requires transaction version 3")
	}
	if tx.Coinbase {
		return fmt.Errorf("asset transactions cannot be coinbase")
	}
	if tx.From == "" {
		return fmt.Errorf("asset transaction sender is required")
	}
	if tx.EffectiveFeePayer() == "" {
		return fmt.Errorf("asset transaction fee payer is required")
	}
	switch tx.TxType() {
	case TxTypeTransfer:
		if tx.To == "" || tx.To == tx.From {
			return fmt.Errorf("asset transfer recipient is invalid")
		}
		if tx.EffectiveAssetID() == "" {
			return fmt.Errorf("asset transfer asset id is required")
		}
		if tx.Amount == 0 {
			return fmt.Errorf("asset transfer amount must be greater than zero")
		}
	case TxTypeAssetCreate:
		if tx.To != "" {
			return fmt.Errorf("asset create recipient must be empty")
		}
		if tx.AssetID != "" && tx.ID == "" {
			return fmt.Errorf("asset create asset id requires transaction id")
		}
		if tx.AssetName == "" || tx.AssetSymbol == "" {
			return fmt.Errorf("asset create metadata is incomplete")
		}
		if tx.AssetDecimals > 18 {
			return fmt.Errorf("asset create decimals too large")
		}
	case TxTypeAssetMint:
		if tx.To == "" || asset.IsNative(tx.AssetID) || tx.AssetID == "" {
			return fmt.Errorf("asset mint requires an issued asset and recipient")
		}
		if tx.Amount == 0 {
			return fmt.Errorf("asset mint amount must be greater than zero")
		}
	case TxTypeAssetBurn:
		if tx.To != tx.From || tx.AssetID == "" || asset.IsNative(tx.AssetID) {
			return fmt.Errorf("asset burn requires an issued asset owned by the sender")
		}
		if tx.Amount == 0 {
			return fmt.Errorf("asset burn amount must be greater than zero")
		}
	default:
		return fmt.Errorf("unsupported asset transaction type: %s", tx.TxType())
	}
	if tx.FeePayer != "" && tx.FeePayer != tx.From {
		if tx.FeePayerPublicKey == "" || tx.FeePayerSignature == "" {
			return fmt.Errorf("paymaster authorization is incomplete")
		}
	}
	return nil
}

func (tx Transaction) EffectiveAssetID() string {
	if tx.AssetID == "" {
		return asset.NativeAssetID
	}
	return tx.AssetID
}

func (tx Transaction) EffectiveFeePayer() string {
	if tx.FeePayer == "" {
		return tx.From
	}
	return tx.FeePayer
}

func (tx Transaction) IsIssuedAssetTransaction() bool {
	return tx.ProtocolVersion() >= TxVersionAsset && !asset.IsNative(tx.EffectiveAssetID())
}

func (tx Transaction) ProtocolVersion() uint32 {
	if tx.Version == 0 {
		return TxVersionLegacy
	}
	return tx.Version
}

func ValidateTransactionVersion(version, activeVersion uint32) error {
	if version == 0 {
		version = TxVersionLegacy
	}
	if activeVersion == 0 {
		activeVersion = TxVersionLegacy
	}
	if version > MaxSupportedTxVersion {
		return fmt.Errorf("unsupported transaction version: %d", version)
	}
	// Protocol activation is monotonic: once a network activates a newer
	// transaction version, older transaction versions remain valid for
	// backwards-compatible legacy/staking traffic. Newer versions than the
	// active version remain invalid until explicitly activated.
	if version > activeVersion {
		return fmt.Errorf("transaction version %d is not active on this network (active version %d)", version, activeVersion)
	}
	return nil
}

func (tx Transaction) SigningBytesWithChainID(chainID uint64) ([]byte, error) {
	switch tx.ProtocolVersion() {
	case TxVersionLegacy:
		return tx.SigningBytes(), nil
	case TxVersionCanonical, TxVersionAsset:
		return tx.CanonicalSigningBytesWithChainID(chainID)
	default:
		return nil, fmt.Errorf("unsupported transaction version: %d", tx.ProtocolVersion())
	}
}

func (tx Transaction) CalculateIDForChainID(chainID uint64) (string, error) {
	raw, err := tx.SigningBytesWithChainID(chainID)
	if err != nil {
		return "", err
	}
	return crypto.DoubleSHA256Hex(raw), nil
}

func (tx *Transaction) RefreshIDForChainID(chainID uint64) error {
	id, err := tx.CalculateIDForChainID(chainID)
	if err != nil {
		return err
	}
	tx.ID = id
	return nil
}

func (tx Transaction) TxType() string {
	if tx.Type != "" {
		return tx.Type
	}
	if tx.Coinbase {
		return TxTypeCoinbase
	}
	return TxTypeTransfer
}

func (tx Transaction) SigningBytes() []byte {
	switch tx.ProtocolVersion() {
	case TxVersionCanonical:
		return tx.CanonicalSigningBytes()
	case TxVersionAsset:
		return tx.CanonicalSigningBytesV3()
	}
	copy := tx
	copy.ID = ""
	copy.Signature = ""
	copy.Version = 0
	if copy.TxType() == TxTypeStakeLock {
		copy.StakeID = ""
	}
	raw, _ := json.Marshal(copy)
	return raw
}

func (tx Transaction) CalculateID() string {
	return crypto.DoubleSHA256Hex(tx.SigningBytes())
}

func (tx *Transaction) RefreshID() {
	tx.ID = tx.CalculateID()
}

// FeePayerSigningBytesWithChainID returns the payload a paymaster signs.
// Sponsor signature material is intentionally excluded so it can be added
// after the sender has signed without changing tx.ID.
// ValidateFeePayerAuthorization verifies a v3 paymaster authorization.
// The sender transaction signature and transaction ID are validated separately.
func (tx Transaction) ValidateFeePayerAuthorization(profile config.NetworkConfig) error {
	if tx.ProtocolVersion() != TxVersionAsset {
		return fmt.Errorf("fee payer authorization requires transaction version 3")
	}
	if !profile.Asset.PaymasterEnabled && tx.EffectiveFeePayer() != tx.From {
		return fmt.Errorf("paymaster is disabled")
	}
	if tx.EffectiveFeePayer() == tx.From {
		if tx.FeePayerPublicKey != "" || tx.FeePayerSignature != "" {
			return fmt.Errorf("self-paid transaction cannot carry paymaster authorization")
		}
		return nil
	}
	if err := crypto.ValidateAddressForNetwork(tx.EffectiveFeePayer(), profile); err != nil {
		return fmt.Errorf("invalid fee payer address: %w", err)
	}
	if tx.FeePayerPublicKey == "" || tx.FeePayerSignature == "" {
		return fmt.Errorf("paymaster authorization is incomplete")
	}
	address, err := crypto.AddressFromPublicKeyForNetwork(tx.FeePayerPublicKey, profile)
	if err != nil || address != tx.EffectiveFeePayer() {
		return fmt.Errorf("paymaster public key does not match fee payer")
	}
	signingBytes, err := tx.FeePayerSigningBytesWithChainID(profile.ChainID)
	if err != nil {
		return err
	}
	if !crypto.VerifyHex(tx.FeePayerPublicKey, tx.FeePayerSignature, signingBytes) {
		return fmt.Errorf("invalid paymaster signature")
	}
	return nil
}

func (tx Transaction) FeePayerSigningBytesWithChainID(chainID uint64) ([]byte, error) {
	if tx.ProtocolVersion() != TxVersionAsset {
		return nil, fmt.Errorf("fee payer authorization requires transaction version 3")
	}
	// Bind the sponsor authorization to the transaction fee and all fields
	// relevant to the fee obligation. Sponsor material stays outside the
	// payload so it can be attached after the sender signs.
	payload := struct {
		Domain    string `json:"domain"`
		ChainID   uint64 `json:"chain_id"`
		TxID      string `json:"tx_id"`
		From      string `json:"from"`
		To        string `json:"to"`
		Amount    uint64 `json:"amount"`
		Fee       uint64 `json:"fee"`
		Nonce     uint64 `json:"nonce"`
		Timestamp int64  `json:"timestamp"`
		Type      string `json:"type"`
		AssetID   string `json:"asset_id"`
		FeePayer  string `json:"fee_payer"`
	}{
		Domain: "deska-paymaster-v1", ChainID: chainID, TxID: tx.ID,
		From: tx.From, To: tx.To, Amount: tx.Amount, Fee: tx.Fee,
		Nonce: tx.Nonce, Timestamp: tx.Timestamp, Type: tx.TxType(),
		AssetID: tx.EffectiveAssetID(), FeePayer: tx.EffectiveFeePayer(),
	}
	return json.Marshal(payload)
}
