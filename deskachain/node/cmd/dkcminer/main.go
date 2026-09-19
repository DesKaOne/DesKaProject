package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"deskachain/internal/cpuminer"
	"deskachain/internal/types"
	"deskachain/internal/version"
)

type templateResponse struct {
	TemplateID string      `json:"template_id"`
	Network    string      `json:"network"`
	Height     uint64      `json:"height"`
	Difficulty uint32      `json:"difficulty"`
	TxCount    int         `json:"tx_count"`
	Block      types.Block `json:"block"`
}

type submitResponse struct {
	Accepted      bool   `json:"accepted"`
	Reason        string `json:"reason"`
	Height        uint64 `json:"height"`
	Hash          string `json:"hash"`
	CurrentHeight uint64 `json:"current_height"`
	CurrentTip    string `json:"current_tip"`
	Duplicate     bool   `json:"duplicate"`
}

type chainInfoResponse struct {
	Height  uint64 `json:"height"`
	TipHash string `json:"tip_hash"`
}

type minerStats struct {
	JobsReceived    uint64
	JobsStale       uint64
	BlocksFound     uint64
	SubmitsAccepted uint64
	SubmitsRejected uint64
	RPCErrors       uint64
	Reconnects      uint64
	StartedAt       time.Time
	TotalHashes     atomic.Uint64
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer, errOut io.Writer) error {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		printMinerHelp(out)
		return nil
	}
	if len(args) > 0 && (args[0] == "--version" || args[0] == "-version" || args[0] == "version") {
		fmt.Fprint(out, version.String("dkcminer"))
		return nil
	}
	fs := flag.NewFlagSet("dkcminer", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "print version and exit")
	showHelp := fs.Bool("help", false, "print help and exit")
	rpcURL := fs.String("rpc-url", "", "node RPC URL")
	address := fs.String("address", "", "reward address")
	threads := fs.Int("threads", runtime.NumCPU(), "CPU mining threads")
	pollInterval := fs.Duration("poll-interval", 5*time.Second, "template poll interval")
	logInterval := fs.Duration("log-interval", 10*time.Second, "hashrate log interval")
	once := fs.Bool("once", false, "mine one block then exit")
	maxBlocks := fs.Int("max-blocks", 0, "maximum accepted blocks, 0 means unlimited")
	retryInterval := fs.Duration("retry-interval", 3*time.Second, "retry interval")
	retry := fs.Bool("retry", true, "retry RPC failures in continuous mode")
	retryDelay := fs.Duration("retry-delay", 3*time.Second, "initial retry backoff")
	maxRetryDelay := fs.Duration("max-retry-delay", 30*time.Second, "maximum retry backoff")
	submitTimeout := fs.Duration("submit-timeout", 20*time.Second, "submit request timeout")
	jobRefreshInterval := fs.Duration("job-refresh-interval", 5*time.Second, "stale job refresh interval")
	duration := fs.Duration("duration", 0, "maximum miner runtime, 0 means unlimited")
	userAgent := fs.String("user-agent", version.UserAgent("dkcminer"), "HTTP user agent")
	_ = fs.String("network", "", "optional network hint")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprint(out, version.String("dkcminer"))
		return nil
	}
	if *showHelp {
		printMinerHelp(out)
		return nil
	}
	if *rpcURL == "" || *address == "" {
		return errors.New("usage: dkcminer --rpc-url <url> --address <DKC_ADDR> [--threads N]")
	}
	if *threads < 1 {
		*threads = 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if *duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *duration)
		defer cancel()
	}
	if *retryDelay <= 0 {
		*retryDelay = *retryInterval
	}
	if *retryDelay <= 0 {
		*retryDelay = 3 * time.Second
	}
	if *maxRetryDelay < *retryDelay {
		*maxRetryDelay = *retryDelay
	}
	if *jobRefreshInterval <= 0 {
		*jobRefreshInterval = *pollInterval
	}
	if *jobRefreshInterval <= 0 {
		*jobRefreshInterval = 5 * time.Second
	}
	templateClient := &http.Client{Timeout: 10 * time.Second}
	submitClient := &http.Client{Timeout: *submitTimeout}
	base := strings.TrimRight(*rpcURL, "/")
	fmt.Fprintln(out, "DesKaChain CPU Miner")
	fmt.Fprintf(out, "rpc: %s\n", base)
	fmt.Fprintf(out, "reward address: %s\n", *address)
	fmt.Fprintf(out, "threads: %d\n\n", *threads)
	accepted := 0
	stats := minerStats{StartedAt: time.Now()}
	backoff := *retryDelay
	for {
		if ctx.Err() != nil {
			fmt.Fprintln(out, "miner stopped cleanly")
			printMinerStats(out, stats)
			return nil
		}
		if *maxBlocks > 0 && accepted >= *maxBlocks {
			fmt.Fprintln(out, "max blocks reached; miner stopped cleanly")
			printMinerStats(out, stats)
			return nil
		}
		tpl, err := fetchTemplate(ctx, templateClient, base, *address, *userAgent)
		if err != nil {
			stats.RPCErrors++
			if *once || !*retry {
				return fmt.Errorf("rpc unreachable: %w", err)
			}
			fmt.Fprintf(errOut, "rpc unreachable: %v; retrying in %s\n", err, backoff.String())
			if !sleepContext(ctx, backoff) {
				fmt.Fprintln(out, "miner stopped cleanly")
				printMinerStats(out, stats)
				return nil
			}
			backoff = nextBackoff(backoff, *maxRetryDelay)
			stats.Reconnects++
			continue
		}
		backoff = *retryDelay
		stats.JobsReceived++
		fmt.Fprintf(out, "new job height=%d difficulty=%d txs=%d prev=%s\n", tpl.Height, tpl.Difficulty, tpl.TxCount, first(tpl.Block.PreviousHash, 16))
		miningCtx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		staleCh := make(chan string, 1)
		started := time.Now()
		go logHashrate(miningCtx, out, &stats.TotalHashes, started, *logInterval, done)
		go monitorStaleJob(miningCtx, templateClient, base, tpl, *jobRefreshInterval, *userAgent, staleCh, cancel)
		result, err := cpuminer.Mine(miningCtx, tpl.Block, *threads, &stats.TotalHashes)
		cancel()
		<-done
		if err != nil {
			if ctx.Err() != nil {
				fmt.Fprintln(out, "miner stopped cleanly")
				printMinerStats(out, stats)
				return nil
			}
			if reason := readStaleReason(staleCh); reason != "" {
				stats.JobsStale++
				fmt.Fprintf(out, "stale job height=%d reason=%s; refreshing job\n", tpl.Height, reason)
				continue
			}
			fmt.Fprintf(errOut, "mining failed: %v; refreshing job\n", err)
			if !sleepContext(ctx, *pollInterval) {
				fmt.Fprintln(out, "miner stopped cleanly")
				printMinerStats(out, stats)
				return nil
			}
			continue
		}
		stats.BlocksFound++
		fmt.Fprintf(out, "block found height=%d hash=%s nonce=%d thread=%d\n", result.Block.Height, result.Block.Hash, result.Block.Nonce, result.Thread)
		resp, err := submitBlock(ctx, submitClient, base, tpl.TemplateID, result.Block, *userAgent)
		if err != nil {
			stats.RPCErrors++
			fmt.Fprintf(errOut, "submit failed: %v; checking latest chain before refreshing job\n", err)
			if latest, latestErr := fetchChainInfo(ctx, templateClient, base, *userAgent); latestErr == nil {
				fmt.Fprintf(errOut, "latest chain after submit error height=%d tip=%s\n", latest.Height, first(latest.TipHash, 16))
			} else {
				fmt.Fprintf(errOut, "latest chain check failed after submit error: %v\n", latestErr)
			}
			continue
		}
		if !resp.Accepted {
			stats.SubmitsRejected++
			if isStaleSubmit(resp.Reason) {
				stats.JobsStale++
				fmt.Fprintf(out, "stale job height=%d reason=%s current_height=%d current_tip=%s; refreshing job\n", tpl.Height, resp.Reason, resp.CurrentHeight, first(resp.CurrentTip, 16))
			} else if resp.Duplicate {
				fmt.Fprintf(out, "duplicate/already known block height=%d hash=%s; refreshing job\n", resp.Height, first(resp.Hash, 16))
			} else {
				fmt.Fprintf(errOut, "submit rejected: %s\n", resp.Reason)
			}
			continue
		}
		accepted++
		stats.SubmitsAccepted++
		fmt.Fprintf(out, "submit accepted height=%d hash=%s\n", resp.Height, resp.Hash)
		if *once {
			printMinerStats(out, stats)
			return nil
		}
	}
}

func printMinerHelp(out io.Writer) {
	fmt.Fprintln(out, "DesKaChain CPU Miner")
	fmt.Fprintln(out, "usage: dkcminer --rpc-url <url> --address <DKC_ADDR> [--threads N] [--once]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "options:")
	fmt.Fprintln(out, "  --version")
	fmt.Fprintln(out, "  --help")
	fmt.Fprintln(out, "  --rpc-url <url>")
	fmt.Fprintln(out, "  --address <DKC_ADDR>")
	fmt.Fprintln(out, "  --threads <n>")
	fmt.Fprintln(out, "  --retry")
	fmt.Fprintln(out, "  --retry-delay <duration>")
	fmt.Fprintln(out, "  --max-retry-delay <duration>")
	fmt.Fprintln(out, "  --submit-timeout <duration>")
	fmt.Fprintln(out, "  --job-refresh-interval <duration>")
	fmt.Fprintln(out, "  --log-interval <duration>")
	fmt.Fprintln(out, "  --duration <duration>")
	fmt.Fprintln(out, "  --max-blocks <n>")
	fmt.Fprintln(out, "mainnet: not available")
}

func fetchTemplate(ctx context.Context, client *http.Client, base, address, userAgent string) (templateResponse, error) {
	endpoint := base + "/miner/template?address=" + url.QueryEscape(address)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return templateResponse{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return templateResponse{}, err
	}
	defer resp.Body.Close()
	var out templateResponse
	if err := decodeRPC(resp, &out); err != nil {
		return templateResponse{}, err
	}
	return out, nil
}

func submitBlock(ctx context.Context, client *http.Client, base, templateID string, block types.Block, userAgent string) (submitResponse, error) {
	body, _ := json.Marshal(map[string]any{"template_id": templateID, "block": block})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/miner/submit", bytes.NewReader(body))
	if err != nil {
		return submitResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return submitResponse{}, err
	}
	defer resp.Body.Close()
	var out submitResponse
	if err := decodeRPC(resp, &out); err != nil {
		return submitResponse{}, err
	}
	return out, nil
}

func fetchChainInfo(ctx context.Context, client *http.Client, base, userAgent string) (chainInfoResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/chain/info", nil)
	if err != nil {
		return chainInfoResponse{}, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return chainInfoResponse{}, err
	}
	defer resp.Body.Close()
	var out chainInfoResponse
	if err := decodeRPC(resp, &out); err != nil {
		return chainInfoResponse{}, err
	}
	return out, nil
}

func decodeRPC(resp *http.Response, out any) error {
	if resp.StatusCode >= 400 {
		var errBody map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		if msg, ok := errBody["error"].(string); ok && msg != "" {
			return errors.New(msg)
		}
		return fmt.Errorf("rpc returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func logHashrate(ctx context.Context, out io.Writer, counter *atomic.Uint64, started time.Time, interval time.Duration, done chan<- struct{}) {
	defer close(done)
	if interval <= 0 {
		<-ctx.Done()
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(started)
			hashes := counter.Load()
			rate := float64(hashes) / elapsed.Seconds()
			fmt.Fprintf(out, "hashrate: %s total_hashes=%d elapsed=%s\n", formatHashrate(rate), hashes, elapsed.Round(time.Second))
		}
	}
}

func monitorStaleJob(ctx context.Context, client *http.Client, base string, tpl templateResponse, interval time.Duration, userAgent string, staleCh chan<- string, cancel context.CancelFunc) {
	if interval <= 0 {
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := fetchChainInfo(ctx, client, base, userAgent)
			if err != nil {
				continue
			}
			reason := staleJobReason(info, tpl)
			if reason != "" {
				select {
				case staleCh <- reason:
				default:
				}
				cancel()
				return
			}
		}
	}
}

func staleJobReason(info chainInfoResponse, tpl templateResponse) string {
	if info.Height >= tpl.Height {
		return "chain height already advanced"
	}
	if info.Height+1 != tpl.Height {
		return "job height is no longer next height"
	}
	if info.TipHash != "" && info.TipHash != tpl.Block.PreviousHash {
		return "node tip changed"
	}
	return ""
}

func readStaleReason(staleCh <-chan string) string {
	select {
	case reason := <-staleCh:
		return reason
	default:
		return ""
	}
}

func isStaleSubmit(reason string) bool {
	reason = strings.ToLower(reason)
	return strings.Contains(reason, "stale") || strings.Contains(reason, "parent mismatch") || strings.Contains(reason, "does not extend") || strings.Contains(reason, "height")
}

func nextBackoff(current, maxDelay time.Duration) time.Duration {
	if current <= 0 {
		current = time.Second
	}
	next := current * 2
	if next > maxDelay {
		return maxDelay
	}
	return next
}

func printMinerStats(out io.Writer, stats minerStats) {
	elapsed := time.Since(stats.StartedAt)
	hashes := stats.TotalHashes.Load()
	rate := 0.0
	if elapsed > 0 {
		rate = float64(hashes) / elapsed.Seconds()
	}
	fmt.Fprintf(out, "miner stats: jobs_received=%d jobs_stale=%d blocks_found=%d submits_accepted=%d submits_rejected=%d rpc_errors=%d reconnects=%d elapsed=%s approximate_hashrate=%s total_hashes=%d\n",
		stats.JobsReceived,
		stats.JobsStale,
		stats.BlocksFound,
		stats.SubmitsAccepted,
		stats.SubmitsRejected,
		stats.RPCErrors,
		stats.Reconnects,
		elapsed.Round(time.Second),
		formatHashrate(rate),
		hashes,
	)
}

func formatHashrate(rate float64) string {
	switch {
	case rate >= 1_000_000:
		return fmt.Sprintf("%.2f MH/s", rate/1_000_000)
	case rate >= 1_000:
		return fmt.Sprintf("%.2f KH/s", rate/1_000)
	default:
		return fmt.Sprintf("%.2f H/s", rate)
	}
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func first(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[:n]
}
