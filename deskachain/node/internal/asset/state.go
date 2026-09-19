package asset

import (
	"errors"
	"fmt"

	"deskachain/internal/arith"
)

const FeeCollectorAddress = "__DESKACHAIN_FEE_POOL__"

var (
	ErrAssetNotFound       = errors.New("asset not found")
	ErrAssetExists         = errors.New("asset already exists")
	ErrUnauthorized        = errors.New("asset operation unauthorized")
	ErrInsufficientBalance = errors.New("insufficient asset balance")
	ErrAssetFrozen         = errors.New("asset is frozen")
)

type State struct {
	assets   map[string]Definition
	balances map[string]map[string]uint64
}

func NewState() *State {
	return &State{
		assets:   make(map[string]Definition),
		balances: make(map[string]map[string]uint64),
	}
}

func (s *State) Clone() *State {
	clone := NewState()
	for id, def := range s.assets {
		clone.assets[id] = def
	}
	for address, assets := range s.balances {
		clone.balances[address] = make(map[string]uint64, len(assets))
		for id, balance := range assets {
			clone.balances[address][id] = balance
		}
	}
	return clone
}

func (s *State) Definition(assetID string) (Definition, bool) {
	def, ok := s.assets[assetID]
	return def, ok
}

func (s *State) Balance(address, assetID string) uint64 {
	assets := s.balances[address]
	if assets == nil {
		return 0
	}
	return assets[assetID]
}

func (s *State) Create(def Definition) error {
	if def.ID == "" {
		return errors.New("asset id is required")
	}
	if IsNative(def.ID) {
		return errors.New("native IDR cannot be recreated as an issued asset")
	}
	if def.Kind == "" {
		def.Kind = KindFungible
	}
	if def.Status == "" {
		def.Status = StatusActive
	}
	if err := ValidateDefinition(def); err != nil {
		return err
	}
	if _, ok := s.assets[def.ID]; ok {
		return ErrAssetExists
	}
	s.assets[def.ID] = def
	return nil
}

func (s *State) CreateFromTransactionID(txID, name, symbol string, decimals uint8, maxSupply uint64, mintable, burnable, pausable, permissioned bool, issuer string) (Definition, error) {
	def := Definition{
		ID:           DerivedID(txID),
		Name:         name,
		Symbol:       symbol,
		Decimals:     decimals,
		Kind:         KindFungible,
		Issuer:       issuer,
		MaxSupply:    maxSupply,
		Mintable:     mintable,
		Burnable:     burnable,
		Pausable:     pausable,
		Permissioned: permissioned,
		Status:       StatusActive,
	}
	if err := s.Create(def); err != nil {
		return Definition{}, err
	}
	return def, nil
}

func (s *State) Mint(assetID, issuer, to string, amount uint64) error {
	if amount == 0 {
		return errors.New("mint amount must be greater than zero")
	}
	def, ok := s.assets[assetID]
	if !ok {
		return ErrAssetNotFound
	}
	if !def.Mintable {
		return errors.New("asset is not mintable")
	}
	if def.Issuer != issuer {
		return ErrUnauthorized
	}
	if def.Status == StatusFrozen {
		return ErrAssetFrozen
	}
	newSupply, err := arith.Add(def.TotalSupply, amount)
	if err != nil {
		return fmt.Errorf("asset supply overflow: %w", err)
	}
	if def.MaxSupply > 0 && newSupply > def.MaxSupply {
		return errors.New("asset max supply exceeded")
	}
	if err := s.addBalance(to, assetID, amount); err != nil {
		return err
	}
	def.TotalSupply = newSupply
	s.assets[assetID] = def
	return nil
}

func (s *State) Burn(assetID, owner string, amount uint64) error {
	if amount == 0 {
		return errors.New("burn amount must be greater than zero")
	}
	def, ok := s.assets[assetID]
	if !ok {
		return ErrAssetNotFound
	}
	if !def.Burnable {
		return errors.New("asset is not burnable")
	}
	if def.Status == StatusFrozen {
		return ErrAssetFrozen
	}
	if err := s.subtractBalance(owner, assetID, amount); err != nil {
		return err
	}
	newSupply, err := arith.Sub(def.TotalSupply, amount)
	if err != nil {
		return fmt.Errorf("asset supply underflow: %w", err)
	}
	def.TotalSupply = newSupply
	s.assets[assetID] = def
	return nil
}

func (s *State) Transfer(assetID, from, to string, amount uint64) error {
	if amount == 0 {
		return errors.New("transfer amount must be greater than zero")
	}
	if from == "" || to == "" || from == to {
		return errors.New("invalid asset transfer addresses")
	}
	def, ok := s.assets[assetID]
	if !ok {
		return ErrAssetNotFound
	}
	if def.Status == StatusFrozen {
		return ErrAssetFrozen
	}
	if err := s.subtractBalance(from, assetID, amount); err != nil {
		return err
	}
	if err := s.addBalance(to, assetID, amount); err != nil {
		// Restore the sender on a failed recipient update.
		_ = s.addBalance(from, assetID, amount)
		return err
	}
	return nil
}

func (s *State) ChargeFee(payer string, fee uint64) error {
	if fee == 0 {
		return nil
	}
	if payer == "" {
		return errors.New("fee payer is required")
	}
	if err := s.subtractBalance(payer, NativeAssetID, fee); err != nil {
		return fmt.Errorf("native IDR fee: %w", err)
	}
	if err := s.addBalance(FeeCollectorAddress, NativeAssetID, fee); err != nil {
		_ = s.addBalance(payer, NativeAssetID, fee)
		return fmt.Errorf("native IDR fee collector: %w", err)
	}
	return nil
}

func (s *State) ExecuteTokenTransfer(assetID, from, to, feePayer string, amount, fee uint64) error {
	if err := s.Transfer(assetID, from, to, amount); err != nil {
		return err
	}
	if err := s.ChargeFee(feePayer, fee); err != nil {
		_ = s.Transfer(assetID, to, from, amount)
		return err
	}
	return nil
}

func (s *State) addBalance(address, assetID string, amount uint64) error {
	if amount == 0 {
		return nil
	}
	assets := s.balances[address]
	if assets == nil {
		assets = make(map[string]uint64)
		s.balances[address] = assets
	}
	next, err := arith.Add(assets[assetID], amount)
	if err != nil {
		return fmt.Errorf("asset balance overflow: %w", err)
	}
	assets[assetID] = next
	return nil
}

func (s *State) subtractBalance(address, assetID string, amount uint64) error {
	if amount == 0 {
		return nil
	}
	assets := s.balances[address]
	if assets == nil {
		return ErrInsufficientBalance
	}
	current := assets[assetID]
	next, err := arith.Sub(current, amount)
	if err != nil {
		return ErrInsufficientBalance
	}
	assets[assetID] = next
	return nil
}
