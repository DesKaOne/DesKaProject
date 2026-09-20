package rpc

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"deskachain/internal/amount"
	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/mempool"
	"deskachain/internal/storage"
	"deskachain/internal/state"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestFaucetInfoDisabledByDefault(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())
	info := getRPCMap(t, server.URL+"/faucet/info", http.StatusOK)
	if info["enabled"].(bool) {
		t.Fatalf("faucet should be disabled by default: %#v", info)
	}
}

func TestFaucetInfoTestnetEnabled(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	_ = paths
	info := getRPCMap(t, server.URL+"/faucet/info", http.StatusOK)
	if !info["enabled"].(bool) || info["network"] != "testnet" || info["network_id"] != config.Testnet().NetworkID || uint64(info["chain_id"].(float64)) != config.Testnet().ChainID {
		t.Fatalf("unexpected faucet info: %#v", info)
	}
	if info["faucet_address"] != faucetWallet.Address || info["amount"] != "100" {
		t.Fatalf("unexpected faucet config: %#v", info)
	}
}

func TestFaucetRejectsLocalnetByDefault(t *testing.T) {
	_, _, server := newFaucetRPCServer(t, config.Localnet(), faucetInfo())
	recipient, _ := wallet.NewWithProfile(config.Localnet())
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "testnet-only") {
		t.Fatalf("expected localnet rejection, got %#v", info)
	}
}

func TestFaucetRejectsPublicRPCWithoutExplicitEnable(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{PublicRPC: true, Profile: config.Testnet()})
	server := httptest.NewServer(mux)
	defer server.Close()
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "faucet disabled") {
		t.Fatalf("expected faucet disabled, got %#v", info)
	}
}

func TestFaucetGuardRejectsIsolatedTestnetWrites(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	faucetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{
		RPCListen:          ":0",
		P2PListen:          ":0",
		Profile:            config.Testnet(),
		EnableFaucetRPC:    true,
		EnableFaucetRPCSet: true,
		FaucetAddress:      faucetWallet.Address,
		FaucetAmount:       100 * config.UnitsPerCoin,
		MinWritePeers:      1,
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "isolated writes are disabled") {
		t.Fatalf("expected isolated write guard, got %#v", info)
	}
}

func TestFaucetRequestInvalidAddress(t *testing.T) {
	_, _, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": "bad"}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "invalid address") {
		t.Fatalf("expected invalid address, got %#v", info)
	}
}

func TestFaucetRejectsWrongNetworkFaucetAddress(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, faucetInfo().withAddress(localWallet.Address))
	server := httptest.NewServer(mux)
	defer server.Close()
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "invalid faucet address") || !strings.Contains(info["error"].(string), "wrong network version") {
		t.Fatalf("expected wrong-network faucet address, got %#v", info)
	}
}

func TestFaucetRejectsWrongNetworkRecipient(t *testing.T) {
	_, _, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": localWallet.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "invalid address") {
		t.Fatalf("expected invalid recipient, got %#v", info)
	}
}

func TestFaucetAcceptsTestnetFaucetAndRecipient(t *testing.T) {
	_, _, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	recipient, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if strings.Contains(info["error"].(string), "wrong network version") {
		t.Fatalf("valid testnet addresses hit wrong network error: %#v", info)
	}
}

func TestFaucetRequestInsufficientMatureBalanceAfterValidAddress(t *testing.T) {
	_, _, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	recipient, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "insufficient mature faucet balance") {
		t.Fatalf("expected mature balance error after valid address, got %#v", info)
	}
}

func TestFaucetRequestCreatesPendingTx(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 105)
	recipient, _ := wallet.NewWithProfile(config.Testnet())

	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusOK)
	if info["status"] != "pending" || info["from"] != faucetWallet.Address || info["to"] != recipient.Address || info["amount"] != "100" {
		t.Fatalf("unexpected faucet response: %#v", info)
	}
	pending, err := mempool.New(paths.Mempool).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].ID != info["tx_id"] {
		t.Fatalf("unexpected pending txs: %+v", pending)
	}
	beforeSupply := totalSupplyFixture(t, paths)
	minePendingFixture(t, paths, faucetWallet.Address, config.Testnet())
	afterSupply := totalSupplyFixture(t, paths)
	if afterSupply-beforeSupply != config.InitialBlockReward {
		t.Fatalf("supply delta = %d, want coinbase reward only", afterSupply-beforeSupply)
	}
	blocks := faucetBlocks(t, paths)
	details, err := ledger.BalanceDetailsForWithProfile(recipient.Address, blocks, nil, config.Testnet().Consensus, config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if details.Confirmed != 100*config.UnitsPerCoin {
		t.Fatalf("recipient confirmed = %d", details.Confirmed)
	}
}

func TestFaucetRequiresMatureBalance(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 1)
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "insufficient mature faucet balance") {
		t.Fatalf("expected mature balance error, got %#v", info)
	}
}

func TestFaucetCannotOverspend(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 101)
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "insufficient mature faucet balance") {
		t.Fatalf("expected overspend error, got %#v", info)
	}
}

func TestFaucetRateLimitByAddress(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 105)
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	if got := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusOK); got["tx_id"] == "" {
		t.Fatalf("unexpected first faucet response: %#v", got)
	}
	_ = mempool.New(paths.Mempool).Clear()
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "rate limited") {
		t.Fatalf("expected rate limit, got %#v", info)
	}
}

func TestFaucetPendingDuplicateRejected(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 105)
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusOK)
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "rate limited") && !strings.Contains(info["error"].(string), "pending faucet tx") {
		t.Fatalf("expected duplicate rejection, got %#v", info)
	}
}

func TestFaucetStatePersists(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 105)
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusOK)
	server.Close()
	_ = mempool.New(paths.Mempool).Clear()

	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, faucetInfo().withAddress(faucetWallet.Address))
	reloaded := httptest.NewServer(mux)
	defer reloaded.Close()
	info := postRPCMap(t, reloaded.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "rate limited") {
		t.Fatalf("expected persisted rate limit, got %#v", info)
	}
}

func TestFaucetStateCorruptHandled(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 105)
	if err := os.WriteFile(paths.FaucetState, []byte("{bad"), 0644); err != nil {
		t.Fatal(err)
	}
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	info := postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusBadRequest)
	if !strings.Contains(info["error"].(string), "failed to load faucet state") {
		t.Fatalf("expected corrupt state error, got %#v", info)
	}
}

func TestFaucetDoesNotChangeTotalSupplyDirectly(t *testing.T) {
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), faucetInfo())
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 105)
	before := totalSupplyFixture(t, paths)
	recipient, _ := wallet.NewWithProfile(config.Testnet())
	postRPCMap(t, server.URL+"/faucet/request", map[string]string{"address": recipient.Address}, http.StatusOK)
	afterRequest := totalSupplyFixture(t, paths)
	if afterRequest != before {
		t.Fatalf("faucet request changed supply: before=%d after=%d", before, afterRequest)
	}
	minePendingFixture(t, paths, faucetWallet.Address, config.Testnet())
	afterMine := totalSupplyFixture(t, paths)
	if afterMine-before != config.InitialBlockReward {
		t.Fatalf("faucet transfer minted supply: before=%d after=%d", before, afterMine)
	}
}

type faucetNodeInfo NodeInfo

func (f faucetNodeInfo) withAddress(address string) NodeInfo {
	info := NodeInfo(f)
	info.FaucetAddress = address
	return info
}

func faucetInfo() faucetNodeInfo {
	return faucetNodeInfo{
		RPCListen:           ":0",
		P2PListen:           ":0",
		Profile:             config.Testnet(),
		EnableFaucetRPC:     true,
		EnableFaucetRPCSet:  true,
		FaucetAmount:        100 * config.UnitsPerCoin,
		FaucetMaxPerAddress: 1000 * config.UnitsPerCoin,
		FaucetMinInterval:   time.Minute,
		AllowIsolatedWrites: true,
	}
}

func newFaucetRPCServer(t *testing.T, profile config.NetworkConfig, base faucetNodeInfo) (config.Paths, wallet.Wallet, *httptest.Server) {
	t.Helper()
	paths := newProfileRPCTestNode(t, profile)
	faucetWallet, err := wallet.NewWithProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := wallet.NewStore(paths.Wallets).Add(faucetWallet); err != nil {
		t.Fatal(err)
	}
	info := base.withAddress(faucetWallet.Address)
	info.Profile = profile
	info.AllowIsolatedWrites = true
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, info)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return paths, faucetWallet, server
}

func fundFaucetFixture(t *testing.T, paths config.Paths, address string, profile config.NetworkConfig, blocksCount int) {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	blocks, err := chain.New(store).Blocks()
	if err != nil {
		t.Fatal(err)
	}
	prior := append([]types.Block(nil), blocks...)
	for i := 0; i < blocksCount; i++ {
		height := prior[len(prior)-1].Height + 1
		cb := types.NewCoinbaseTransaction(address, config.InitialBlockReward, height)
		difficulty := chain.CalculateNextDifficultyWithParams(prior, profile.Difficulty)
		block := types.NewBlock(height, prior[len(prior)-1].Hash, address, difficulty, []types.Transaction{cb})
		block.Timestamp = prior[len(prior)-1].Timestamp + profile.Difficulty.TargetBlockTimeSeconds
		block.Hash = block.CalculateHash()
		if err := store.SaveBlock(block); err != nil {
			t.Fatal(err)
		}
		prior = append(prior, block)
	}
	snapshot, err := state.SnapshotForBlocks(prior, profile.Consensus, profile)
	if err != nil { t.Fatal(err) }
	if err := store.SaveState(snapshot); err != nil { t.Fatal(err) }
}

func minePendingFixture(t *testing.T, paths config.Paths, miner string, profile config.NetworkConfig) {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	blocks, err := chain.New(store).Blocks()
	if err != nil {
		t.Fatal(err)
	}
	pending, err := mempool.New(paths.Mempool).Load()
	if err != nil {
		t.Fatal(err)
	}
	height := blocks[len(blocks)-1].Height + 1
	txs := append([]types.Transaction{types.NewCoinbaseTransaction(miner, config.InitialBlockReward, height)}, pending...)
	block := types.NewBlock(height, blocks[len(blocks)-1].Hash, miner, chain.CalculateNextDifficultyWithParams(blocks, profile.Difficulty), txs)
	block.Timestamp = blocks[len(blocks)-1].Timestamp + profile.Difficulty.TargetBlockTimeSeconds
	block.Hash = block.CalculateHash()
	if err := store.SaveBlock(block); err != nil {
		t.Fatal(err)
	}
	prior := append([]types.Block(nil), blocks...)
	prior = append(prior, block)
	snapshot, err := state.SnapshotForBlocks(prior, profile.Consensus, profile)
	if err != nil { t.Fatal(err) }
	if err := store.SaveState(snapshot); err != nil { t.Fatal(err) }
	ids := map[string]struct{}{}
	for _, tx := range pending {
		ids[tx.ID] = struct{}{}
	}
	if err := mempool.New(paths.Mempool).RemoveIDs(ids); err != nil {
		t.Fatal(err)
	}
}

func totalSupplyFixture(t *testing.T, paths config.Paths) uint64 {
	t.Helper()
	return ledger.TotalSupply(faucetBlocks(t, paths))
}

func faucetBlocks(t *testing.T, paths config.Paths) []types.Block {
	t.Helper()
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	blocks, err := chain.New(store).Blocks()
	if err != nil {
		t.Fatal(err)
	}
	return blocks
}

func TestFaucetAmountFormatting(t *testing.T) {
	if got := amount.Format(100 * config.UnitsPerCoin); got != "100" {
		t.Fatalf("amount format = %s", got)
	}
}
