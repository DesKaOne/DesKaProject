package servicenode

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"deskachain/internal/config"
	"deskachain/internal/ledger"
	"deskachain/internal/storage"
	"deskachain/internal/state"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func testAddress(t *testing.T) string {
	t.Helper()
	w, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	return w.Address
}

func testStore(t *testing.T, now time.Time) Store {
	t.Helper()
	return NewStore(config.NewPaths(t.TempDir()), config.Localnet()).WithNow(func() time.Time { return now })
}

func TestRegisterValidInvalidAndDuplicate(t *testing.T) {
	store := testStore(t, time.Unix(1000, 0))
	address := testAddress(t)
	node, err := store.Register(address, "http://127.0.0.1:9501", Metadata{ClientVersion: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if node.ServiceNodeID == "" || node.Status != StatusRegistered || node.OwnerAddress != address {
		t.Fatalf("unexpected node: %+v", node)
	}
	if _, err := store.Register("bad", "", Metadata{}); err == nil {
		t.Fatal("expected invalid address rejection")
	}
	node, err = store.Register(address, "http://127.0.0.1:9502", Metadata{Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	nodes, err := store.LoadNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("duplicate register should update one record, got %d", len(nodes))
	}
	if node.AdvertisedEndpoint != "http://127.0.0.1:9502" {
		t.Fatalf("endpoint not updated: %s", node.AdvertisedEndpoint)
	}
}

func TestHeartbeatChallengeScoreAndRewards(t *testing.T) {
	now := time.Unix(2000, 0)
	store := testStore(t, now)
	address := testAddress(t)
	if _, err := store.Register(address, "http://127.0.0.1:9501", Metadata{}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(10 * time.Second)
	store = store.WithNow(func() time.Time { return now })
	score, node, err := store.Heartbeat(address, "http://127.0.0.1:9501", Metadata{Platform: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if node.Status != StatusActive || node.LastSeenAt != now.Unix() || score.UptimeScore == 0 {
		t.Fatalf("unexpected heartbeat node=%+v score=%+v", node, score)
	}
	challenge, err := store.CreateChallenge(address)
	if err != nil {
		t.Fatal(err)
	}
	if challenge.ChallengeID == "" || challenge.Status != ChallengePending || challenge.ExpiresAt <= challenge.IssuedAt {
		t.Fatalf("unexpected challenge: %+v", challenge)
	}
	score, challenge, err = store.SubmitChallenge(challenge.ChallengeID, 50, 10_000_000, 50_000_000, true)
	if err != nil {
		t.Fatal(err)
	}
	if challenge.Status != ChallengePassed {
		t.Fatalf("expected passed challenge: %+v", challenge)
	}
	if score.LatencyScore != 100 || score.BandwidthScore != 80 || score.ReliabilityScore != 100 || score.ServiceScore != 94 {
		t.Fatalf("unexpected score: %+v", score)
	}
	rewards, err := store.Rewards(address)
	if err != nil {
		t.Fatal(err)
	}
	if len(rewards) != 1 || rewards[0].SimulatedPoints != score.ServiceScore*10 {
		t.Fatalf("unexpected rewards: %+v", rewards)
	}
	if _, _, err := store.SubmitChallenge(challenge.ChallengeID, 50, 1, 1, true); err == nil {
		t.Fatal("expected duplicate challenge submit rejection")
	}
}

func TestChallengeValidationAndAbuseFlags(t *testing.T) {
	now := time.Unix(3000, 0)
	store := testStore(t, now)
	address := testAddress(t)
	if _, err := store.Register(address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	challenge, err := store.CreateChallenge(address)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.SubmitChallenge(challenge.ChallengeID, -1, 0, 0, true); err == nil {
		t.Fatal("expected negative values rejection")
	}
	score, _, err := store.SubmitChallenge(challenge.ChallengeID, 70, 900_000_000, 200_000_000, true)
	if err != nil {
		t.Fatal(err)
	}
	if score.AbusePenalty == 0 || !hasFlag(score.Flags, FlagImpossibleBandwidth) {
		t.Fatalf("expected impossible bandwidth flag: %+v", score)
	}

	expired, err := store.CreateChallenge(address)
	if err != nil {
		t.Fatal(err)
	}
	store = store.WithNow(func() time.Time { return now.Add(10 * time.Minute) })
	if _, _, err := store.SubmitChallenge(expired.ChallengeID, 1, 1, 1, true); err == nil {
		t.Fatal("expected expired challenge rejection")
	}
	node, _, err := store.FindNode(address)
	if err != nil {
		t.Fatal(err)
	}
	if !hasFlag(node.Flags, FlagChallengeExpired) {
		t.Fatalf("expected expired flag: %+v", node.Flags)
	}
}

func TestStoragePersistenceAndAtomicJSON(t *testing.T) {
	store := testStore(t, time.Unix(4000, 0))
	address := testAddress(t)
	if _, err := store.Register(address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	reloaded := NewStore(store.paths, config.Localnet())
	node, ok, err := reloaded.FindNode(address)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || node.OwnerAddress != address {
		t.Fatalf("missing persisted node: %+v", node)
	}
	raw, err := os.ReadFile(store.paths.ServiceNodes)
	if err != nil {
		t.Fatal(err)
	}
	var nodes []Node
	if err := json.Unmarshal(raw, &nodes); err != nil {
		t.Fatalf("service_nodes.json is invalid: %v", err)
	}
	empty := NewStore(config.NewPaths(t.TempDir()), config.Localnet())
	nodes, err = empty.LoadNodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatalf("missing service store should load empty, got %d", len(nodes))
	}
}

func TestScoreCollateralEligibility(t *testing.T) {
	now := time.Unix(5000, 0)
	paths := config.NewPaths(t.TempDir())
	store := NewStore(paths, config.Localnet()).WithNow(func() time.Time { return now })
	w := mustWallet(t)
	if _, err := store.Register(w.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Heartbeat(w.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	score, err := store.Score(w.Address)
	if err != nil {
		t.Fatal(err)
	}
	if score.StakeEligible || score.CollateralStatus != "none" || score.EligibleSimulatedPoints != 0 {
		t.Fatalf("expected ineligible without stake: %+v", score)
	}
	funding := []types.Block{
		{Height: 0},
		{Height: 1, Transactions: []types.Transaction{types.NewCoinbaseTransaction(w.Address, config.InitialBlockReward, 1)}},
	}
	for height := uint64(2); height <= config.Localnet().Consensus.CoinbaseMaturity+1; height++ {
		funding = append(funding, types.Block{Height: height})
	}
	saveServiceBlocks(t, paths, funding)
	lock := types.NewStakeLockTransaction(w.Address, config.Localnet().Consensus.Staking.MinServiceStake, config.Localnet().Consensus.CoinbaseMaturity+2)
	if err := w.SignTransaction(&lock); err != nil {
		t.Fatal(err)
	}
	lock.StakeID = lock.ID
	saveServiceBlocks(t, paths, []types.Block{{Height: 2, Transactions: []types.Transaction{lock}}})
	score, err = store.Score(w.Address)
	if err != nil {
		t.Fatal(err)
	}
	if !score.StakeEligible || score.CollateralStatus != "eligible" || score.ActiveStake != config.Localnet().Consensus.Staking.MinServiceStake {
		t.Fatalf("expected eligible with service stake: %+v", score)
	}
	if score.EligibleSimulatedPoints != score.SimulatedPoints {
		t.Fatalf("eligible points should count when stake eligible: %+v", score)
	}
}

func TestServiceCollateralUnlockingAndReleasedNotEligible(t *testing.T) {
	now := time.Unix(6000, 0)
	paths := config.NewPaths(t.TempDir())
	store := NewStore(paths, config.Localnet()).WithNow(func() time.Time { return now })
	w := mustWallet(t)
	if _, err := store.Register(w.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Heartbeat(w.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	funding := []types.Block{
		{Height: 0},
		{Height: 1, Transactions: []types.Transaction{types.NewCoinbaseTransaction(w.Address, config.InitialBlockReward, 1)}},
	}
	for height := uint64(2); height <= config.Localnet().Consensus.CoinbaseMaturity+1; height++ {
		funding = append(funding, types.Block{Height: height})
	}
	saveServiceBlocks(t, paths, funding)
	lock, unlock := serviceStakeTxs(t, w, config.Localnet().Consensus.Staking.MinServiceStake)
	lockHeight := config.Localnet().Consensus.CoinbaseMaturity + 2
	unlockHeight := lockHeight + 1
	lock.Nonce = 1
	unlock.Nonce = 2
	lockBlock := types.Block{Height: lockHeight, Transactions: []types.Transaction{lock}}
	unlockBlock := types.Block{Height: unlockHeight, Transactions: []types.Transaction{unlock}}
	saveServiceBlocks(t, paths, []types.Block{lockBlock, unlockBlock})
	score, err := store.Score(w.Address)
	if err != nil {
		t.Fatal(err)
	}
	if score.StakeEligible || score.CollateralStatus != "unlocking" || score.ActiveStake != 0 || score.EligibleSimulatedPoints != 0 {
		t.Fatalf("expected unlocking stake to be ineligible: %+v", score)
	}
	releaseHeight := unlockHeight + config.Localnet().Consensus.Staking.UnbondingPeriodBlocks
	for height := unlockHeight + 1; height <= releaseHeight; height++ {
		saveServiceBlocks(t, paths, []types.Block{{Height: height}})
	}
	score, err = store.Score(w.Address)
	if err != nil {
		t.Fatal(err)
	}
	if score.StakeEligible || score.CollateralStatus != "released" || score.ActiveStake != 0 || score.EligibleSimulatedPoints != 0 {
		t.Fatalf("expected released stake to be ineligible: %+v", score)
	}
}

func TestServicePointsStillNotIDR(t *testing.T) {
	now := time.Unix(7000, 0)
	paths := config.NewPaths(t.TempDir())
	store := NewStore(paths, config.Localnet()).WithNow(func() time.Time { return now })
	w := mustWallet(t)
	saveServiceBlocks(t, paths, []types.Block{
		{Height: 0},
		{Height: 1, Transactions: []types.Transaction{types.NewCoinbaseTransaction(w.Address, config.InitialBlockReward, 1)}},
	})
	before := serviceBlocks(t, paths)
	beforeSupply := ledger.TotalSupply(before)
	if _, err := store.Register(w.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Heartbeat(w.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Rewards(w.Address); err != nil {
		t.Fatal(err)
	}
	after := serviceBlocks(t, paths)
	if afterSupply := ledger.TotalSupply(after); afterSupply != beforeSupply {
		t.Fatalf("service rewards changed IDR supply: before=%d after=%d", beforeSupply, afterSupply)
	}
}

func TestServiceCollateralUsesActiveProfile(t *testing.T) {
	now := time.Unix(8000, 0)
	localStore := NewStore(config.NewPaths(t.TempDir()), config.Localnet()).WithNow(func() time.Time { return now })
	testnetStore := NewStore(config.NewPaths(t.TempDir()), config.Testnet()).WithNow(func() time.Time { return now })
	localAddress := mustWallet(t).Address
	testnetWallet, err := wallet.NewWithProfile(config.Testnet())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := localStore.Register(localAddress, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	if _, err := testnetStore.Register(testnetWallet.Address, "", Metadata{}); err != nil {
		t.Fatal(err)
	}
	localScore, err := localStore.Score(localAddress)
	if err != nil {
		t.Fatal(err)
	}
	testnetScore, err := testnetStore.Score(testnetWallet.Address)
	if err != nil {
		t.Fatal(err)
	}
	if localScore.RequiredStake != config.Localnet().Consensus.Staking.MinServiceStake {
		t.Fatalf("local required stake = %d", localScore.RequiredStake)
	}
	if testnetScore.RequiredStake != config.Testnet().Consensus.Staking.MinServiceStake {
		t.Fatalf("testnet required stake = %d", testnetScore.RequiredStake)
	}
	if localScore.RequiredStake == testnetScore.RequiredStake {
		t.Fatalf("required stake should differ by profile")
	}
}

func serviceStakeTxs(t *testing.T, w wallet.Wallet, value uint64) (types.Transaction, types.Transaction) {
	t.Helper()
	lock := types.NewStakeLockTransaction(w.Address, value, 1)
	if err := w.SignTransaction(&lock); err != nil {
		t.Fatal(err)
	}
	lock.StakeID = lock.ID
	unlock := types.NewStakeUnlockTransaction(w.Address, lock.StakeID, 2)
	if err := w.SignTransaction(&unlock); err != nil {
		t.Fatal(err)
	}
	return lock, unlock
}

func prepareServiceBlocks(blocks []types.Block) {
	for i := range blocks {
		blocks[i].Hash = fmt.Sprintf("service-test-%d", blocks[i].Height)
		if i > 0 { blocks[i].PreviousHash = blocks[i-1].Hash }
	}
}

func saveServiceBlocks(t *testing.T, paths config.Paths, blocks []types.Block) {
	t.Helper()
	bolt, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer bolt.Close()

	canonical, err := bolt.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	var previous string
	if len(canonical) > 0 {
		previous = canonical[len(canonical)-1].Hash
	}
	for i := range blocks {
		blocks[i].Hash = fmt.Sprintf("service-test-%d", blocks[i].Height)
		if previous != "" {
			blocks[i].PreviousHash = previous
		}
		if err := bolt.SaveBlock(blocks[i]); err != nil {
			t.Fatal(err)
		}
		previous = blocks[i].Hash
	}
	canonical, err = bolt.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	profile := config.Localnet()
	snapshot, err := state.SnapshotForBlocks(canonical, profile.Consensus, profile)
	if err != nil {
		t.Fatal(err)
	}
	if err := bolt.SaveState(snapshot); err != nil {
		t.Fatal(err)
	}
}

func serviceBlocks(t *testing.T, paths config.Paths) []types.Block {
	t.Helper()
	bolt, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer bolt.Close()
	blocks, err := bolt.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	return blocks
}

func hasFlag(flags []string, want string) bool {
	for _, flag := range flags {
		if flag == want {
			return true
		}
	}
	return false
}

func mustWallet(t *testing.T) wallet.Wallet {
	t.Helper()
	w, err := wallet.NewWithProfile(config.Localnet())
	if err != nil {
		t.Fatal(err)
	}
	return w
}
