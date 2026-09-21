package ledger

import (
	"errors"
	"fmt"

	"indochain/internal/arith"
	"indochain/internal/config"
	"indochain/internal/crypto"
	"indochain/internal/types"
)

type Account struct {
	Balance uint64 `json:"balance"`
	Nonce   uint64 `json:"nonce"`
}

type Ledger struct {
	accounts map[string]Account
}

func New() *Ledger {
	return &Ledger{accounts: make(map[string]Account)}
}

func Replay(blocks []types.Block) (*Ledger, error) {
	l := New()
	for _, block := range blocks {
		if err := l.ApplyBlock(block); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func (l *Ledger) Balance(address string) uint64 {
	return l.accounts[address].Balance
}

func (l *Ledger) Nonce(address string) uint64 {
	return l.accounts[address].Nonce
}

func (l *Ledger) Snapshot() map[string]Account {
	out := make(map[string]Account, len(l.accounts))
	for k, v := range l.accounts {
		out[k] = v
	}
	return out
}

func (l *Ledger) ApplyBlock(block types.Block) error {
	coinbaseCount := 0
	for _, tx := range block.Transactions {
		if tx.Coinbase {
			coinbaseCount++
			if coinbaseCount > 1 {
				return errors.New("multiple coinbase transactions")
			}
			if err := l.ApplyCoinbase(tx); err != nil {
				return err
			}
			continue
		}
		if err := l.ApplyTransaction(tx); err != nil {
			return err
		}
	}
	return nil
}

func (l *Ledger) ApplyCoinbase(tx types.Transaction) error {
	if !tx.Coinbase {
		return errors.New("transaction is not coinbase")
	}
	if err := crypto.ValidateAddress(tx.To); err != nil {
		return fmt.Errorf("invalid coinbase recipient: %w", err)
	}
	acct := l.accounts[tx.To]
	balance, err := arith.Add(acct.Balance, tx.Amount)
	if err != nil {
		return fmt.Errorf("coinbase balance overflow: %w", err)
	}
	acct.Balance = balance
	l.accounts[tx.To] = acct
	return nil
}

func (l *Ledger) ApplyTransaction(tx types.Transaction) error {
	if err := l.ValidateTransaction(tx); err != nil {
		return err
	}
	from := l.accounts[tx.From]
	if tx.TxType() == types.TxTypeTransfer {
		to := l.accounts[tx.To]
		cost, err := arith.Add(tx.Amount, tx.Fee)
		if err != nil {
			return fmt.Errorf("transaction cost overflow: %w", err)
		}
		balance, err := arith.Sub(from.Balance, cost)
		if err != nil {
			return errors.New("insufficient balance")
		}
		toBalance, err := arith.Add(to.Balance, tx.Amount)
		if err != nil {
			return fmt.Errorf("recipient balance overflow: %w", err)
		}
		from.Balance = balance
		from.Nonce = tx.Nonce
		to.Balance = toBalance
		l.accounts[tx.From] = from
		l.accounts[tx.To] = to
		return nil
	}
	from.Nonce = tx.Nonce
	l.accounts[tx.From] = from
	return nil
}

func (l *Ledger) ValidateTransaction(tx types.Transaction) error {
	profile := config.Localnet()
	if err := types.ValidateTransactionVersion(tx.ProtocolVersion(), profile.TxVersion); err != nil {
		return err
	}
	if tx.Coinbase {
		return nil
	}
	if tx.TxType() == types.TxTypeTransfer && tx.Amount == 0 {
		return errors.New("amount must be greater than zero")
	}
	if tx.TxType() == types.TxTypeStakeLock && tx.Amount == 0 {
		return errors.New("amount must be greater than zero")
	}
	if tx.TxType() == types.TxTypeTransfer && tx.From == tx.To {
		return errors.New("sender and recipient must differ")
	}
	if err := crypto.ValidateAddress(tx.From); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if tx.TxType() == types.TxTypeTransfer {
		if err := crypto.ValidateAddress(tx.To); err != nil {
			return fmt.Errorf("invalid recipient address: %w", err)
		}
	}
	if tx.TxType() == types.TxTypeStakeLock && tx.To != "" && tx.To != tx.From {
		return errors.New("invalid stake lock: recipient must match owner")
	}
	if crypto.AddressFromPublicKey(tx.PublicKey) != tx.From {
		return errors.New("public key does not match sender address")
	}
	expectedID, idErr := tx.CalculateIDForChainID(profile.ChainID)
	if idErr != nil {
		return idErr
	}
	if tx.ID != expectedID {
		return errors.New("transaction id mismatch")
	}
	if tx.TxType() == types.TxTypeStakeLock && tx.StakeID != "" && tx.StakeID != tx.ID {
		return errors.New("invalid stake lock: stake id mismatch")
	}
	signingBytes, signingErr := tx.SigningBytesWithChainID(profile.ChainID)
	if signingErr != nil {
		return signingErr
	}
	if !crypto.VerifyHex(tx.PublicKey, tx.Signature, signingBytes) {
		return errors.New("invalid transaction signature")
	}
	account := l.accounts[tx.From]
	expectedNonce, nonceErr := arith.Add(account.Nonce, 1)
	if nonceErr != nil {
		return errors.New("account nonce overflow")
	}
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("invalid account nonce: got %d want %d", tx.Nonce, expectedNonce)
	}
	if tx.TxType() == types.TxTypeTransfer {
		cost, err := arith.Add(tx.Amount, tx.Fee)
		if err != nil {
			return fmt.Errorf("transaction cost overflow: %w", err)
		}
		if account.Balance < cost {
			return errors.New("insufficient balance")
		}
	}
	if tx.TxType() == types.TxTypeStakeLock && account.Balance < tx.Amount {
		return errors.New("insufficient balance")
	}
	return nil
}

func (l *Ledger) Clone() *Ledger {
	clone := New()
	for address, account := range l.accounts {
		clone.accounts[address] = account
	}
	return clone
}

func TotalSupply(blocks []types.Block) uint64 {
	return TotalSupplyWithProfile(blocks, config.Localnet())
}

func TotalSupplyWithProfile(blocks []types.Block, profile config.NetworkConfig) uint64 {
	var total uint64
	for _, block := range blocks {
		if block.Height == 0 {
			continue
		}
		for _, tx := range block.Transactions {
			if !tx.Coinbase {
				continue
			}
			if profile.TxVersion >= types.TxVersionAsset {
				total = arith.AddCap(total, tx.Amount)
			} else {
				// Legacy coinbase amounts bundle subsidy + collected fees.
				total = arith.AddCap(total, config.InitialBlockReward)
			}
		}
	}
	return total
}
