package ledger

import (
	"errors"
	"fmt"

	"deskachain/internal/arith"
	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/staking"
	"deskachain/internal/types"
)

var ErrImmatureBalance = errors.New("spends immature coinbase")

type BalanceDetails struct {
	Address          string
	Confirmed        uint64
	Mature           uint64
	Immature         uint64
	Spendable        uint64
	PendingOutgoing  uint64
	PendingIncoming  uint64
	ActiveStake      uint64
	UnlockingStake   uint64
	ReleasedStake    uint64
	PendingStakeLock uint64
	CoinbaseMaturity uint64
	CurrentHeight    uint64
}

type MatureAccount struct {
	Confirmed uint64
	Mature    uint64
	Nonce     uint64
}

type MatureLedger struct {
	accounts      map[string]MatureAccount
	coinbases     []coinbaseCredit
	maturity      uint64
	currentHeight uint64
	stakes        *staking.State
	params        config.ConsensusParams
	profile       config.NetworkConfig
}

type coinbaseCredit struct {
	Address string
	Amount  uint64
	Height  uint64
	Matured bool
}

func NewMature(params config.ConsensusParams) *MatureLedger {
	return NewMatureWithProfile(params, config.Localnet())
}

func NewMatureWithProfile(params config.ConsensusParams, profile config.NetworkConfig) *MatureLedger {
	if profile.Name == "" {
		profile = config.Localnet()
	}
	return &MatureLedger{
		accounts: make(map[string]MatureAccount),
		maturity: params.CoinbaseMaturity,
		stakes:   staking.NewState(params.Staking),
		params:   params,
		profile:  profile,
	}
}

func NewMatureFromState(
	params config.ConsensusParams,
	profile config.NetworkConfig,
	height uint64,
	accounts []StateAccount,
	stakes []staking.Record,
	coinbases []StateCoinbase,
) *MatureLedger {
	l := NewMatureWithProfile(params, profile)
	l.currentHeight = height
	for _, account := range accounts {
		l.accounts[account.Address] = MatureAccount{
			Confirmed: account.Confirmed,
			Mature:    account.Mature,
			Nonce:     account.Nonce,
		}
	}
	for _, record := range stakes {
		l.stakes.ApplyRecord(record)
	}
	for _, credit := range coinbases {
		l.coinbases = append(l.coinbases, coinbaseCredit{
			Address: credit.Address,
			Amount:  credit.Amount,
			Height:  credit.Height,
			Matured: false,
		})
	}
	return l
}

func ReplayMature(blocks []types.Block, params config.ConsensusParams) (*MatureLedger, error) {
	return ReplayMatureWithProfile(blocks, params, config.Localnet())
}

func ReplayMatureWithProfile(blocks []types.Block, params config.ConsensusParams, profile config.NetworkConfig) (*MatureLedger, error) {
	l := NewMatureWithProfile(params, profile)
	for _, block := range blocks {
		if err := l.ApplyBlock(block); err != nil {
			return nil, err
		}
	}
	if len(blocks) > 0 {
		l.matureCoinbases(blocks[len(blocks)-1].Height)
	}
	return l, nil
}

func BalanceDetailsFor(address string, blocks []types.Block, pending []types.Transaction, params config.ConsensusParams) (BalanceDetails, error) {
	return BalanceDetailsForWithProfile(address, blocks, pending, params, config.Localnet())
}

func BalanceDetailsForWithProfile(address string, blocks []types.Block, pending []types.Transaction, params config.ConsensusParams, profile config.NetworkConfig) (BalanceDetails, error) {
	l, err := ReplayMatureWithProfile(blocks, params, profile)
	if err != nil {
		return BalanceDetails{}, err
	}
	currentHeight := uint64(0)
	if len(blocks) > 0 {
		currentHeight = blocks[len(blocks)-1].Height
	}
	return l.BalanceDetails(address, pending, currentHeight), nil
}

func CirculatingSupply(blocks []types.Block, params config.ConsensusParams) uint64 {
	return CirculatingSupplyWithProfile(blocks, params, config.Localnet())
}

func CirculatingSupplyWithProfile(blocks []types.Block, params config.ConsensusParams, profile config.NetworkConfig) uint64 {
	l, err := ReplayMatureWithProfile(blocks, params, profile)
	if err != nil {
		return 0
	}
	return l.TotalMature()
}

type StateAccount struct {
	Address   string `json:"address"`
	Confirmed uint64 `json:"confirmed"`
	Mature    uint64 `json:"mature"`
	Nonce     uint64 `json:"nonce"`
}

type StateCoinbase struct {
	Address string `json:"address"`
	Amount  uint64 `json:"amount"`
	Height  uint64 `json:"height"`
}

func (l *MatureLedger) StateAccounts() []StateAccount {
	out := make([]StateAccount, 0, len(l.accounts))
	for address, account := range l.accounts {
		out = append(out, StateAccount{
			Address:   address,
			Confirmed: account.Confirmed,
			Mature:    account.Mature,
			Nonce:     account.Nonce,
		})
	}
	return out
}

func (l *MatureLedger) StateStakes() []staking.Record {
	return l.stakes.Records(l.currentHeight)
}

func (l *MatureLedger) StateCoinbases() []StateCoinbase {
	out := make([]StateCoinbase, 0, len(l.coinbases))
	for _, credit := range l.coinbases {
		if credit.Matured {
			continue
		}
		out = append(out, StateCoinbase{
			Address: credit.Address,
			Amount:  credit.Amount,
			Height:  credit.Height,
		})
	}
	return out
}

func (l *MatureLedger) Height() uint64 {
	return l.currentHeight
}

func (l *MatureLedger) Balance(address string) uint64 {
	return l.accounts[address].Confirmed
}

func (l *MatureLedger) MatureBalance(address string) uint64 {
	return l.accounts[address].Mature
}

func (l *MatureLedger) Nonce(address string) uint64 {
	return l.accounts[address].Nonce
}

func (l *MatureLedger) TotalMature() uint64 {
	var total uint64
	for _, account := range l.accounts {
		total = arith.AddCap(total, account.Mature)
	}
	return total
}

func (l *MatureLedger) Clone() *MatureLedger {
	clone := &MatureLedger{
		accounts:      make(map[string]MatureAccount, len(l.accounts)),
		coinbases:     make([]coinbaseCredit, len(l.coinbases)),
		maturity:      l.maturity,
		currentHeight: l.currentHeight,
		stakes:        staking.NewState(l.params.Staking),
		params:        l.params,
		profile:       l.profile,
	}
	for address, account := range l.accounts {
		clone.accounts[address] = account
	}
	copy(clone.coinbases, l.coinbases)
	for _, record := range l.stakes.Records(l.currentHeight) {
		clone.stakes.ApplyRecord(record)
	}
	return clone
}

func (l *MatureLedger) ApplyBlock(block types.Block) error {
	if block.Height > 0 {
		l.matureCoinbases(block.Height - 1)
	}
	coinbaseCount := 0
	for _, tx := range block.Transactions {
		if tx.Coinbase {
			coinbaseCount++
			if coinbaseCount > 1 {
				return errors.New("multiple coinbase transactions")
			}
			if err := l.ApplyCoinbaseAtHeight(tx, block.Height); err != nil {
				return err
			}
			continue
		}
		if err := l.ApplyTransactionAtHeight(tx, block.Height); err != nil {
			return err
		}
	}
	l.currentHeight = block.Height
	l.matureCoinbases(block.Height)
	return nil
}

func (l *MatureLedger) ApplyCoinbaseAtHeight(tx types.Transaction, height uint64) error {
	if !tx.Coinbase {
		return errors.New("transaction is not coinbase")
	}
	if err := crypto.ValidateAddressForNetwork(tx.To, l.profile); err != nil {
		return fmt.Errorf("invalid coinbase recipient: %w", err)
	}
	acct := l.accounts[tx.To]
	confirmed, err := arith.Add(acct.Confirmed, tx.Amount)
	if err != nil { return fmt.Errorf("coinbase balance overflow: %w", err) }
	acct.Confirmed = confirmed
	l.accounts[tx.To] = acct
	l.coinbases = append(l.coinbases, coinbaseCredit{Address: tx.To, Amount: tx.Amount, Height: height})
	return nil
}

func (l *MatureLedger) ApplyTransaction(tx types.Transaction) error {
	return l.ApplyTransactionAtHeight(tx, l.currentHeight)
}

func (l *MatureLedger) ApplyTransactionAtHeight(tx types.Transaction, height uint64) error {
	if err := l.ValidateTransaction(tx); err != nil {
		return err
	}
	from := l.accounts[tx.From]
	switch tx.TxType() {
	case types.TxTypeTransfer:
		to := l.accounts[tx.To]
		cost, err := arith.Add(tx.Amount, tx.Fee)
		if err != nil {
			return fmt.Errorf("transaction cost overflow: %w", err)
		}
		confirmed, err := arith.Sub(from.Confirmed, cost)
		if err != nil {
			return errors.New("insufficient balance")
		}
		mature, err := arith.Sub(from.Mature, cost)
		if err != nil {
			return ErrImmatureBalance
		}
		toConfirmed, err := arith.Add(to.Confirmed, tx.Amount)
		if err != nil {
			return fmt.Errorf("recipient balance overflow: %w", err)
		}
		toMature, err := arith.Add(to.Mature, tx.Amount)
		if err != nil {
			return fmt.Errorf("recipient mature balance overflow: %w", err)
		}
		from.Confirmed = confirmed
		from.Mature = mature
		from.Nonce = tx.Nonce
		to.Confirmed = toConfirmed
		to.Mature = toMature
		l.accounts[tx.From] = from
		l.accounts[tx.To] = to
	case types.TxTypeStakeLock:
		if err := l.stakes.ApplyLock(tx, height); err != nil {
			return err
		}
		from.Nonce = tx.Nonce
		l.accounts[tx.From] = from
	case types.TxTypeStakeUnlock:
		if err := l.stakes.ApplyUnlock(tx, height); err != nil {
			return err
		}
		from.Nonce = tx.Nonce
		l.accounts[tx.From] = from
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.TxType())
	}
	return nil
}

func (l *MatureLedger) ValidateTransaction(tx types.Transaction) error {
	if err := types.ValidateTransactionVersion(tx.ProtocolVersion(), l.profile.TxVersion); err != nil {
		return err
	}
	if tx.Coinbase {
		return nil
	}
	switch tx.TxType() {
	case types.TxTypeTransfer:
		if tx.Amount == 0 {
			return errors.New("amount must be greater than zero")
		}
		if tx.From == tx.To {
			return errors.New("sender and recipient must differ")
		}
	case types.TxTypeStakeLock:
		if tx.Amount == 0 {
			return errors.New("amount must be greater than zero")
		}
		if tx.Amount < l.params.Staking.MinStakeAmount {
			return errors.New("invalid stake lock: amount below minimum")
		}
	case types.TxTypeStakeUnlock:
		if tx.StakeID == "" {
			return errors.New("invalid stake unlock: stake id is required")
		}
	default:
		return fmt.Errorf("unknown transaction type: %s", tx.TxType())
	}
	if tx.TxType() == types.TxTypeTransfer && tx.From == tx.To {
		return errors.New("sender and recipient must differ")
	}
	if err := crypto.ValidateAddressForNetwork(tx.From, l.profile); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if tx.TxType() == types.TxTypeTransfer {
		if err := crypto.ValidateAddressForNetwork(tx.To, l.profile); err != nil {
			return fmt.Errorf("invalid recipient address: %w", err)
		}
	}
	if tx.TxType() == types.TxTypeStakeLock && tx.To != "" && tx.To != tx.From {
		return errors.New("invalid stake lock: recipient must match owner")
	}
	address, err := crypto.AddressFromPublicKeyForNetwork(tx.PublicKey, l.profile)
	if err != nil || address != tx.From {
		return errors.New("public key does not match sender address")
	}
	expectedID, idErr := tx.CalculateIDForChainID(l.profile.ChainID)
	if idErr != nil {
		return idErr
	}
	if tx.ID != expectedID {
		return errors.New("transaction id mismatch")
	}
	if tx.TxType() == types.TxTypeStakeLock && tx.StakeID != "" && tx.StakeID != tx.ID {
		return errors.New("invalid stake lock: stake id mismatch")
	}
	signingBytes, signingErr := tx.SigningBytesWithChainID(l.profile.ChainID)
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
	switch tx.TxType() {
	case types.TxTypeTransfer:
		cost, err := arith.Add(tx.Amount, tx.Fee)
		if err != nil {
			return fmt.Errorf("transaction cost overflow: %w", err)
		}
		if account.Confirmed < cost {
			return errors.New("insufficient balance")
		}
		if account.Mature < cost {
			return ErrImmatureBalance
		}
		active, unlocking, _ := l.stakes.AddressSummary(tx.From, l.currentHeight)
		locked, lockErr := arith.Add(active, unlocking)
		if lockErr != nil {
			return errors.New("locked stake total overflow")
		}
		if account.Mature < locked || account.Mature-locked < cost {
			return errors.New("invalid transaction: spends locked stake")
		}
	case types.TxTypeStakeLock:
		active, unlocking, _ := l.stakes.AddressSummary(tx.From, l.currentHeight)
		locked, lockErr := arith.Add(active, unlocking)
		if lockErr != nil {
			return errors.New("locked stake total overflow")
		}
		if account.Mature < locked || account.Mature-locked < tx.Amount {
			return errors.New("invalid stake lock: insufficient mature spendable balance")
		}
		if l.stakes.ActiveCount(tx.From) >= l.params.Staking.MaxActiveStakesPerAddress && l.params.Staking.MaxActiveStakesPerAddress > 0 {
			return errors.New("invalid stake lock: max active stakes reached")
		}
	case types.TxTypeStakeUnlock:
		record, ok := l.stakes.Find(tx.StakeID, l.currentHeight)
		if !ok {
			return errors.New("invalid stake unlock: stake not found")
		}
		if record.OwnerAddress != tx.From {
			return errors.New("invalid stake unlock: owner mismatch")
		}
		if record.Status != staking.StatusActive {
			return fmt.Errorf("invalid stake unlock: stake %s", record.Status)
		}
	}
	return nil
}

func (l *MatureLedger) BalanceDetails(address string, pending []types.Transaction, currentHeight uint64) BalanceDetails {
	account := l.accounts[address]
	return BalanceDetailsFromState(
		address,
		StateAccount{
			Address:   address,
			Confirmed: account.Confirmed,
			Mature:    account.Mature,
			Nonce:     account.Nonce,
		},
		l.stakes.Records(currentHeight),
		pending,
		l.maturity,
		currentHeight,
	)
}

// BalanceDetailsFromState calculates address balances from persisted state
// indexes plus pending transactions, without replaying the blockchain.
func BalanceDetailsFromState(
	address string,
	account StateAccount,
	stakes []staking.Record,
	pending []types.Transaction,
	coinbaseMaturity uint64,
	currentHeight uint64,
) BalanceDetails {
	stakeState := staking.NewState(config.StakingParams{})
	for _, record := range stakes {
		stakeState.ApplyRecord(record)
	}

	pendingOutgoing := uint64(0)
	pendingIncoming := uint64(0)
	for _, tx := range pending {
		if tx.Coinbase {
			continue
		}
		if tx.From == address && tx.TxType() == types.TxTypeTransfer {
			pendingOutgoing = arith.AddCap(pendingOutgoing, arith.AddCap(tx.Amount, tx.Fee))
		}
		if tx.To == address && tx.TxType() == types.TxTypeTransfer {
			pendingIncoming = arith.AddCap(pendingIncoming, tx.Amount)
		}
	}

	activeStake, unlockingStake, releasedStake := stakeState.AddressSummary(address, currentHeight)
	pendingStakeLock := staking.PendingStakeLock(pending, address)
	locked := arith.AddCap(
		arith.AddCap(activeStake, unlockingStake),
		arith.AddCap(pendingStakeLock, pendingOutgoing),
	)

	spendable := uint64(0)
	if account.Mature > locked {
		spendable = account.Mature - locked
	}
	immature := uint64(0)
	if account.Confirmed > account.Mature {
		immature = account.Confirmed - account.Mature
	}

	return BalanceDetails{
		Address:          address,
		Confirmed:        account.Confirmed,
		Mature:           account.Mature,
		Immature:         immature,
		Spendable:        spendable,
		PendingOutgoing:  pendingOutgoing,
		PendingIncoming:  pendingIncoming,
		ActiveStake:      activeStake,
		UnlockingStake:   unlockingStake,
		ReleasedStake:    releasedStake,
		PendingStakeLock: pendingStakeLock,
		CoinbaseMaturity: coinbaseMaturity,
		CurrentHeight:    currentHeight,
	}
}

func (l *MatureLedger) matureCoinbases(currentHeight uint64) {
	pending := l.coinbases[:0]
	for _, credit := range l.coinbases {
		if credit.Matured {
			continue
		}
		maturityHeight, err := arith.Add(credit.Height, l.maturity)
		if err != nil || currentHeight < maturityHeight {
			pending = append(pending, credit)
			continue
		}
		acct := l.accounts[credit.Address]
		mature, err := arith.Add(acct.Mature, credit.Amount)
		if err != nil {
			pending = append(pending, credit)
			continue
		}
		acct.Mature = mature
		l.accounts[credit.Address] = acct
	}
	l.coinbases = pending
}
