package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestStakeInfoRPC(t *testing.T) {
	_, server := newMinerRPCServer(t)
	resp := getRPCMap(t, server.URL+"/stake/info", http.StatusOK)
	if resp["staking_enabled"] != true {
		t.Fatalf("staking should be enabled: %#v", resp)
	}
	if resp["min_stake_amount"] != "10" || resp["min_service_stake"] != "100" || resp["unbonding_period"].(float64) != 10 {
		t.Fatalf("unexpected stake info: %#v", resp)
	}
}

func fundedRPCProfile() config.NetworkConfig {
	profile := config.Localnet()
	profile.Economic.BlockSubsidy = config.InitialBlockReward
	profile.Economic.FeeOnlyBlocks = false
	return profile
}

func TestStakeLockRPCAdminMode(t *testing.T) {
	paths, server := newMinerRPCServerWithProfile(t, fundedRPCProfile())
	miner := newRPCWallet(t)
	if err := wallet.NewStore(paths.Wallets).Add(miner); err != nil {
		t.Fatal(err)
	}
	fundMatureRPCMiner(t, server.URL, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+1)
	resp := postRPCMap(t, server.URL+"/stake/lock", map[string]string{"address": miner.Address, "amount": "10"}, http.StatusOK)
	if resp["tx_id"] == "" || resp["stake_id"] == "" || resp["status"] != "pending" {
		t.Fatalf("unexpected stake lock response: %#v", resp)
	}
}

func TestStakeUnlockRPCAdminMode(t *testing.T) {
	paths, server := newMinerRPCServerWithProfile(t, fundedRPCProfile())
	miner := newRPCWallet(t)
	if err := wallet.NewStore(paths.Wallets).Add(miner); err != nil {
		t.Fatal(err)
	}
	fundMatureRPCMiner(t, server.URL, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+1)
	lock := postRPCMap(t, server.URL+"/stake/lock", map[string]string{"address": miner.Address, "amount": "10"}, http.StatusOK)
	stakeID, _ := lock["stake_id"].(string)
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	if submit := submitMinerBlock(t, server.URL, tpl.TemplateID, chainMine(tpl.Block)); !submit.Accepted {
		t.Fatalf("stake lock mining rejected: %#v", submit)
	}
	resp := postRPCMap(t, server.URL+"/stake/unlock", map[string]string{"address": miner.Address, "stake_id": stakeID}, http.StatusOK)
	if resp["tx_id"] == "" || resp["stake_id"] != stakeID || resp["status"] != "pending" || resp["release_height"].(float64) == 0 {
		t.Fatalf("unexpected stake unlock response: %#v", resp)
	}
}

func TestStakeListAndStatusRPC(t *testing.T) {
	paths, server := newMinerRPCServer(t)
	miner := newRPCWallet(t)
	if err := wallet.NewStore(paths.Wallets).Add(miner); err != nil {
		t.Fatal(err)
	}
	fundMatureRPCMiner(t, server.URL, miner.Address, int(config.Localnet().Consensus.CoinbaseMaturity)+1)
	lock := postRPCMap(t, server.URL+"/stake/lock", map[string]string{"address": miner.Address, "amount": "10"}, http.StatusOK)
	stakeID, _ := lock["stake_id"].(string)
	tpl := fetchMinerTemplate(t, server.URL, miner.Address)
	if submit := submitMinerBlock(t, server.URL, tpl.TemplateID, chainMine(tpl.Block)); !submit.Accepted {
		t.Fatalf("stake lock mining rejected: %#v", submit)
	}
	list := getRPCMap(t, server.URL+"/stake/list", http.StatusOK)
	if list["count"].(float64) != 1 {
		t.Fatalf("unexpected stake list: %#v", list)
	}
	filtered := getRPCMap(t, server.URL+"/stake/list?address="+miner.Address, http.StatusOK)
	if filtered["count"].(float64) != 1 {
		t.Fatalf("unexpected filtered stake list: %#v", filtered)
	}
	status := getRPCMap(t, server.URL+"/stake/status?id="+stakeID, http.StatusOK)
	if status["stake_id"] != stakeID || status["status"] != "active" {
		t.Fatalf("unexpected stake status: %#v", status)
	}
}

func TestPublicRPCStakeInfoAllowedAndWritesDisabled(t *testing.T) {
	address := newRPCWallet(t).Address
	_, server := newHardeningRPCServer(t, NodeInfo{PublicRPC: true})
	_ = getRPCMap(t, server.URL+"/stake/info", http.StatusOK)
	lock := postRPCMap(t, server.URL+"/stake/lock", map[string]string{"address": address, "amount": "10"}, http.StatusForbidden)
	if !strings.Contains(lock["error"].(string), "endpoint disabled in public RPC mode") {
		t.Fatalf("unexpected public stake lock error: %#v", lock)
	}
	unlock := postRPCMap(t, server.URL+"/stake/unlock", map[string]string{"address": address, "stake_id": "missing"}, http.StatusForbidden)
	if !strings.Contains(unlock["error"].(string), "endpoint disabled in public RPC mode") {
		t.Fatalf("unexpected public stake unlock error: %#v", unlock)
	}
}

func TestStakeRPCBodyLimit(t *testing.T) {
	_, server := newHardeningRPCServer(t, NodeInfo{})
	body := `{"address":"` + strings.Repeat("x", 1024*1024+1) + `","amount":"10"}`
	resp, err := http.Post(server.URL+"/stake/lock", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized stake body status = %d want 413", resp.StatusCode)
	}
}

func postRPCMap(t *testing.T, url string, body any, wantStatus int) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("POST %s status = %d want %d", url, resp.StatusCode, wantStatus)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func getRPCMap(t *testing.T, url string, wantStatus int) map[string]any {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s status = %d want %d", url, resp.StatusCode, wantStatus)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func chainMine(block types.Block) types.Block {
	return chain.Mine(block)
}
