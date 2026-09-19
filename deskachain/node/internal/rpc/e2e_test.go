package rpc

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"deskachain/internal/config"
	"deskachain/internal/crypto"
	"deskachain/internal/wallet"
)

const e2eEndpoint = "http://127.0.0.1:9971"

type faucetStakeServiceFixture struct {
	paths        config.Paths
	serverURL    string
	faucetWallet wallet.Wallet
	ownerWallet  wallet.Wallet
}

func TestFaucetToStakeToServiceEligibleE2E(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	before := totalSupplyFixture(t, fixture.paths)

	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	if got := totalSupplyFixture(t, fixture.paths); got != before {
		t.Fatalf("faucet request changed supply: before=%d after=%d", before, got)
	}
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	afterFaucetMine := totalSupplyFixture(t, fixture.paths)
	if afterFaucetMine-before != config.InitialBlockReward {
		t.Fatalf("faucet tx supply delta = %d, want one coinbase reward", afterFaucetMine-before)
	}
	assertRPCBalance(t, fixture.serverURL, fixture.ownerWallet.Address, "1000", "1000", "0")

	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	if got := totalSupplyFixture(t, fixture.paths); got != afterFaucetMine {
		t.Fatalf("stake lock request changed supply: before=%d after=%d", afterFaucetMine, got)
	}
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	afterStakeMine := totalSupplyFixture(t, fixture.paths)
	if afterStakeMine-afterFaucetMine != config.InitialBlockReward {
		t.Fatalf("stake tx supply delta = %d, want one coinbase reward", afterStakeMine-afterFaucetMine)
	}
	assertActiveStake(t, fixture.serverURL, fixture.ownerWallet.Address, "1000", "active")
	assertRPCBalance(t, fixture.serverURL, fixture.ownerWallet.Address, "1000", "0", "1000")

	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)
	score := serviceScore(t, fixture.serverURL, fixture.ownerWallet.Address)
	assertServiceCollateral(t, score, 1000*config.UnitsPerCoin, 1000*config.UnitsPerCoin, true, "eligible", true)
	if !strings.Contains(score["note"].(string), "not spendable IDR") {
		t.Fatalf("score missing simulation-only note: %#v", score)
	}
	assertRPCBalance(t, fixture.serverURL, fixture.ownerWallet.Address, "1000", "0", "1000")
	if got := totalSupplyFixture(t, fixture.paths); got != afterStakeMine {
		t.Fatalf("service flow changed supply: before=%d after=%d", afterStakeMine, got)
	}
}

func TestServiceNotEligibleBeforeStakeE2E(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())

	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)
	score := serviceScore(t, fixture.serverURL, fixture.ownerWallet.Address)
	assertServiceCollateral(t, score, 1000*config.UnitsPerCoin, 0, false, "none", false)
}

func TestServiceNotEligibleWithInsufficientStakeE2E(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "100"}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())

	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)
	score := serviceScore(t, fixture.serverURL, fixture.ownerWallet.Address)
	assertServiceCollateral(t, score, 1000*config.UnitsPerCoin, 100*config.UnitsPerCoin, false, "insufficient", false)
}

func TestServiceEligibilityLostAfterUnlock(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	lock := postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())

	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)
	assertServiceCollateral(t, serviceScore(t, fixture.serverURL, fixture.ownerWallet.Address), 1000*config.UnitsPerCoin, 1000*config.UnitsPerCoin, true, "eligible", true)

	stakeID := lock["stake_id"].(string)
	postRPCMap(t, fixture.serverURL+"/stake/unlock", map[string]string{"address": fixture.ownerWallet.Address, "stake_id": stakeID}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	assertActiveStake(t, fixture.serverURL, fixture.ownerWallet.Address, "1000", "unlocking")
	score := serviceScore(t, fixture.serverURL, fixture.ownerWallet.Address)
	assertServiceCollateral(t, score, 1000*config.UnitsPerCoin, 0, false, "unlocking", false)
}

func TestFaucetStakeDoesNotMutateSupplyDirectly(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	before := totalSupplyFixture(t, fixture.paths)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	if got := totalSupplyFixture(t, fixture.paths); got != before {
		t.Fatalf("faucet request changed supply: before=%d after=%d", before, got)
	}
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	afterFaucetMine := totalSupplyFixture(t, fixture.paths)
	if afterFaucetMine-before != config.InitialBlockReward {
		t.Fatalf("faucet tx minted outside coinbase: delta=%d", afterFaucetMine-before)
	}

	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	if got := totalSupplyFixture(t, fixture.paths); got != afterFaucetMine {
		t.Fatalf("stake request changed supply: before=%d after=%d", afterFaucetMine, got)
	}
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	afterStakeMine := totalSupplyFixture(t, fixture.paths)
	if afterStakeMine-afterFaucetMine != config.InitialBlockReward {
		t.Fatalf("stake tx minted outside coinbase: delta=%d", afterStakeMine-afterFaucetMine)
	}

	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)
	_ = getRPCMap(t, fixture.serverURL+"/service/rewards?address="+url.QueryEscape(fixture.ownerWallet.Address), http.StatusOK)
	if got := totalSupplyFixture(t, fixture.paths); got != afterStakeMine {
		t.Fatalf("service scoring/rewards changed supply: before=%d after=%d", afterStakeMine, got)
	}
}

func TestChainInfoCirculatingSupplyWithMaturity(t *testing.T) {
	paths, server := newProfileRPCServer(t, config.Testnet())
	miner, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	fundFaucetFixture(t, paths, miner.Address, config.Testnet(), 122)

	info := getRPCMap(t, server.URL+"/chain/info", http.StatusOK)
	if info["height"].(float64) != 122 {
		t.Fatalf("height = %v want 122: %#v", info["height"], info)
	}
	if info["total_supply"] != "6100 IDR" {
		t.Fatalf("total supply = %v want 6100 IDR", info["total_supply"])
	}
	if info["circulating_supply"] != "1100 IDR" {
		t.Fatalf("circulating supply = %v want mature coinbase supply 1100 IDR", info["circulating_supply"])
	}
}

func TestChainInfoCirculatingSupplyIncludesActiveStake(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())

	info := getRPCMap(t, fixture.serverURL+"/chain/info", http.StatusOK)
	if info["circulating_supply"] != "1350 IDR" {
		t.Fatalf("circulating supply = %v want 1350 IDR with active stake still counted", info["circulating_supply"])
	}
	if info["total_active_stake"] != "1000 IDR" {
		t.Fatalf("total active stake = %v want 1000 IDR", info["total_active_stake"])
	}
}

func TestChainInfoSpendableNotEqualCirculating(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())

	chainInfo := getRPCMap(t, fixture.serverURL+"/chain/info", http.StatusOK)
	balance := getRPCMap(t, fixture.serverURL+"/balance/"+fixture.ownerWallet.Address, http.StatusOK)
	if chainInfo["circulating_supply"] != "1350 IDR" || balance["spendable_balance"] != "0" || balance["active_stake"] != "1000" {
		t.Fatalf("expected active stake to reduce spendable but not circulating: chain=%#v balance=%#v", chainInfo, balance)
	}
}

func TestChainInfoSupplyUnaffectedByFaucetStakeService(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	before := totalSupplyFixture(t, fixture.paths)
	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	afterFaucetMine := totalSupplyFixture(t, fixture.paths)
	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	afterStakeMine := totalSupplyFixture(t, fixture.paths)
	runServiceAgentEquivalent(t, fixture.serverURL, fixture.ownerWallet.Address)

	info := getRPCMap(t, fixture.serverURL+"/chain/info", http.StatusOK)
	if info["total_supply"] != "6350 IDR" || afterFaucetMine-before != config.InitialBlockReward || afterStakeMine-afterFaucetMine != config.InitialBlockReward || totalSupplyFixture(t, fixture.paths) != afterStakeMine {
		t.Fatalf("unexpected supply after faucet/stake/service: before=%d afterFaucet=%d afterStake=%d info=%#v", before, afterFaucetMine, afterStakeMine, info)
	}
}

func TestE2EProfileConsistency(t *testing.T) {
	fixture := newFaucetStakeServiceFixture(t, 1000)
	if err := crypto.ValidateAddressForNetwork(fixture.faucetWallet.Address, config.Testnet()); err != nil {
		t.Fatalf("faucet wallet not testnet-valid: %v", err)
	}
	if err := crypto.ValidateAddressForNetwork(fixture.ownerWallet.Address, config.Testnet()); err != nil {
		t.Fatalf("owner wallet not testnet-valid: %v", err)
	}

	localWallet, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	if err := crypto.ValidateAddressForNetwork(localWallet.Address, config.Testnet()); err == nil {
		t.Fatalf("localnet wallet unexpectedly validates as testnet: %s", localWallet.Address)
	}
	assertWrongNetworkRejected(t, postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": localWallet.Address}, http.StatusBadRequest))
	assertWrongNetworkRejected(t, postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": localWallet.Address, "amount": "100"}, http.StatusBadRequest))
	assertWrongNetworkRejected(t, postRPCMap(t, fixture.serverURL+"/service/register", map[string]string{"address": localWallet.Address, "endpoint": e2eEndpoint}, http.StatusBadRequest))

	postRPCMap(t, fixture.serverURL+"/faucet/request", map[string]string{"address": fixture.ownerWallet.Address}, http.StatusOK)
	minePendingFixture(t, fixture.paths, fixture.faucetWallet.Address, config.Testnet())
	postRPCMap(t, fixture.serverURL+"/stake/lock", map[string]string{"address": fixture.ownerWallet.Address, "amount": "1000"}, http.StatusOK)
	postRPCMap(t, fixture.serverURL+"/service/register", map[string]string{"address": fixture.ownerWallet.Address, "endpoint": e2eEndpoint}, http.StatusOK)
}

func newFaucetStakeServiceFixture(t *testing.T, faucetAmountIDR uint64) faucetStakeServiceFixture {
	t.Helper()
	info := faucetInfo()
	info.FaucetAmount = faucetAmountIDR * config.UnitsPerCoin
	info.FaucetMaxPerAddress = 2000 * config.UnitsPerCoin
	info.FaucetMinInterval = time.Second
	paths, faucetWallet, server := newFaucetRPCServer(t, config.Testnet(), info)
	ownerWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if err := wallet.NewStore(paths.Wallets).Add(ownerWallet); err != nil {
		t.Fatal(err)
	}
	fundFaucetFixture(t, paths, faucetWallet.Address, config.Testnet(), 125)
	return faucetStakeServiceFixture{paths: paths, serverURL: server.URL, faucetWallet: faucetWallet, ownerWallet: ownerWallet}
}

func runServiceAgentEquivalent(t *testing.T, serverURL, address string) {
	t.Helper()
	postRPCMap(t, serverURL+"/service/register", map[string]any{"address": address, "endpoint": e2eEndpoint, "client_version": "e2e", "platform": "test"}, http.StatusOK)
	postRPCMap(t, serverURL+"/service/heartbeat", map[string]any{"address": address, "endpoint": e2eEndpoint, "client_version": "e2e", "platform": "test"}, http.StatusOK)
	challenge := postRPCMap(t, serverURL+"/service/challenge/create", map[string]string{"address": address}, http.StatusOK)
	postRPCMap(t, serverURL+"/service/challenge/submit", map[string]any{
		"challenge_id": challenge["challenge_id"],
		"latency_ms":   50,
		"bytes_up":     100000000,
		"bytes_down":   100000000,
		"success":      true,
	}, http.StatusOK)
}

func serviceScore(t *testing.T, serverURL, address string) map[string]any {
	t.Helper()
	resp := getRPCMap(t, serverURL+"/service/score?address="+url.QueryEscape(address), http.StatusOK)
	score, ok := resp["score"].(map[string]any)
	if !ok {
		t.Fatalf("invalid score response: %#v", resp)
	}
	return score
}

func assertServiceCollateral(t *testing.T, score map[string]any, required, active uint64, eligible bool, status string, wantEligiblePoints bool) {
	t.Helper()
	if got := uint64(scoreNumber(score["required_stake"])); got != required {
		t.Fatalf("required stake = %d want %d score=%#v", got, required, score)
	}
	if got := uint64(scoreNumber(score["active_stake"])); got != active {
		t.Fatalf("active stake = %d want %d score=%#v", got, active, score)
	}
	if got := score["stake_eligible"].(bool); got != eligible {
		t.Fatalf("stake eligible = %t want %t score=%#v", got, eligible, score)
	}
	if got := score["collateral_status"].(string); got != status {
		t.Fatalf("collateral status = %q want %q score=%#v", got, status, score)
	}
	points := int(scoreNumber(score["eligible_simulated_points"]))
	if wantEligiblePoints && points <= 0 {
		t.Fatalf("eligible simulated points = %d, want > 0 score=%#v", points, score)
	}
	if !wantEligiblePoints && points != 0 {
		t.Fatalf("eligible simulated points = %d, want 0 score=%#v", points, score)
	}
}

func assertRPCBalance(t *testing.T, serverURL, address, confirmed, spendable, activeStake string) {
	t.Helper()
	info := getRPCMap(t, serverURL+"/balance/"+address, http.StatusOK)
	if info["confirmed_balance"] != confirmed || info["spendable_balance"] != spendable || info["active_stake"] != activeStake {
		t.Fatalf("unexpected balance: %#v", info)
	}
}

func assertActiveStake(t *testing.T, serverURL, address, amountText, status string) {
	t.Helper()
	list := getRPCMap(t, serverURL+"/stake/list?address="+url.QueryEscape(address), http.StatusOK)
	if list["count"].(float64) != 1 {
		t.Fatalf("unexpected stake count: %#v", list)
	}
	stakes := list["stakes"].([]any)
	record := stakes[0].(map[string]any)
	if record["amount"] != amountText || record["status"] != status {
		t.Fatalf("unexpected stake record: %#v", record)
	}
}

func assertWrongNetworkRejected(t *testing.T, resp map[string]any) {
	t.Helper()
	if !strings.Contains(resp["error"].(string), "wrong network version") && !strings.Contains(resp["error"].(string), "invalid address") {
		t.Fatalf("expected wrong-network rejection, got %#v", resp)
	}
}

func scoreNumber(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case uint64:
		return float64(n)
	default:
		return 0
	}
}
