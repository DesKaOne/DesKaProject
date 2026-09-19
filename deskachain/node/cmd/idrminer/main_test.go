package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"deskachain/internal/chain"
	"deskachain/internal/config"
	"deskachain/internal/rpc"
	"deskachain/internal/storage"
	"deskachain/internal/types"
	"deskachain/internal/wallet"
)

func TestRunOnceMinesOneBlockAndExits(t *testing.T) {
	paths := config.NewPaths(t.TempDir())
	store, err := storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	if err := chain.New(store).Init(); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	miner, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	rpc.RegisterHandlers(mux, paths, rpc.NodeInfo{RPCListen: ":0", P2PListen: ":0"})
	server := httptest.NewServer(mux)
	defer server.Close()
	var out, errOut bytes.Buffer
	if err := run([]string{"--rpc-url", server.URL, "--address", miner.Address, "--threads", "2", "--once", "--log-interval", "100ms"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	output := out.String()
	if !strings.Contains(output, "submit accepted height=1") {
		t.Fatalf("unexpected output:\n%s\nstderr:\n%s", output, errOut.String())
	}
	store, err = storage.OpenBolt(paths.DB)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tip, err := chain.New(store).Tip()
	if err != nil {
		t.Fatal(err)
	}
	if tip.Height != 1 {
		t.Fatalf("height = %d want 1", tip.Height)
	}
}

func TestVersionAndHelpDoNotRequireRPC(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"--version"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"idrminer", "version:", "commit:", "built:", "networks: localnet,testnet", "mainnet: not available"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("version output missing %q:\n%s", want, out.String())
		}
	}
	out.Reset()
	if err := run([]string{"--help"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "usage: idrminer") {
		t.Fatalf("help output missing usage:\n%s", out.String())
	}
}

func TestRetryWhenRPCInitiallyDownAndOnceFails(t *testing.T) {
	miner, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	err = run([]string{"--rpc-url", "http://127.0.0.1:1", "--address", miner.Address, "--once", "--retry-delay", "10ms"}, &out, &errOut)
	if err == nil || !strings.Contains(err.Error(), "rpc unreachable") {
		t.Fatalf("expected once rpc failure, got %v", err)
	}
	out.Reset()
	errOut.Reset()
	if err := run([]string{"--rpc-url", "http://127.0.0.1:1", "--address", miner.Address, "--duration", "80ms", "--retry-delay", "10ms", "--max-retry-delay", "20ms"}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "rpc unreachable") || !strings.Contains(out.String(), "miner stopped cleanly") || !strings.Contains(out.String(), "rpc_errors=") {
		t.Fatalf("retry output missing expected status:\nout=%s\nerr=%s", out.String(), errOut.String())
	}
}

func TestStaleJobDetectionCancelsMining(t *testing.T) {
	miner, err := wallet.New()
	if err != nil {
		t.Fatal(err)
	}
	genesis := chain.GenesisBlock()
	blockA := types.NewBlock(1, genesis.Hash, miner.Address, 64, []types.Transaction{types.NewCoinbaseTransaction(miner.Address, config.InitialBlockReward, 1)})
	blockB := types.NewBlock(2, strings.Repeat("a", 64), miner.Address, 64, []types.Transaction{types.NewCoinbaseTransaction(miner.Address, config.InitialBlockReward, 2)})
	var height atomic.Uint64
	var tip atomic.Value
	tip.Store(genesis.Hash)
	jobFetched := make(chan struct{})
	var signalOnce sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/miner/template":
			if height.Load() == 0 {
				signalOnce.Do(func() { close(jobFetched) })
				_ = json.NewEncoder(w).Encode(templateResponse{TemplateID: "job-a", Network: "localnet", Height: 1, Difficulty: 64, TxCount: 1, Block: blockA})
				return
			}
			_ = json.NewEncoder(w).Encode(templateResponse{TemplateID: "job-b", Network: "localnet", Height: 2, Difficulty: 64, TxCount: 1, Block: blockB})
		case "/chain/info":
			_ = json.NewEncoder(w).Encode(chainInfoResponse{Height: height.Load(), TipHash: tip.Load().(string)})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var out, errOut lockedBuffer
	done := make(chan error, 1)
	go func() {
		done <- run([]string{"--rpc-url", server.URL, "--address", miner.Address, "--duration", "2s", "--job-refresh-interval", "100ms", "--log-interval", "0"}, &out, &errOut)
	}()
	select {
	case <-jobFetched:
	case err := <-done:
		t.Fatalf("miner exited before fetching initial job: %v\nout=%s\nerr=%s", err, out.String(), errOut.String())
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for initial job fetch\nout=%s\nerr=%s", out.String(), errOut.String())
	}
	height.Store(1)
	tip.Store(strings.Repeat("a", 64))
	if !waitForOutput(&out, "stale job", 1500*time.Millisecond) && !waitForOutput(&out, "jobs_stale=1", 1500*time.Millisecond) {
		t.Fatalf("stale output missing while miner was running:\nout=%s\nerr=%s", out.String(), errOut.String())
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "stale job") || !strings.Contains(out.String(), "jobs_stale=1") || !strings.Contains(out.String(), "miner stopped cleanly") {
		t.Fatalf("stale output missing expected status:\nout=%s\nerr=%s", out.String(), errOut.String())
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func waitForOutput(out *lockedBuffer, needle string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), needle) {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return strings.Contains(out.String(), needle)
}
