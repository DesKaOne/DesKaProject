package rpc

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"indochain/internal/config"
)

func TestExplorerUIStaticRoutes(t *testing.T) {
	_, server := newProfileRPCServer(t, config.Testnet())

	index := getText(t, server.URL+"/explorer-ui", http.StatusOK)
	if !strings.Contains(index, "IndoChain Explorer") || !strings.Contains(index, `id="app"`) {
		t.Fatalf("index missing app shell: %s", index)
	}
	indexSlash := getText(t, server.URL+"/explorer-ui/", http.StatusOK)
	if !strings.Contains(indexSlash, "/explorer-ui/assets/app.js") || !strings.Contains(indexSlash, "/explorer-ui/assets/styles.css") {
		t.Fatalf("index missing static asset references: %s", indexSlash)
	}

	js := getText(t, server.URL+"/explorer-ui/assets/app.js", http.StatusOK)
	if !strings.Contains(js, "/explorer/status") || !strings.Contains(js, "/explorer/blocks") || !strings.Contains(js, "/explorer/indexed/search") || !strings.Contains(js, "/explorer/indexer/stats") || !strings.Contains(js, "/explorer/indexed/asset/") {
		t.Fatalf("app js missing explorer API references")
	}
	for _, item := range []string{"dashboardSection", "dashboard-summary", "data-refresh=\"dashboard\"", "Last sync", "data-block-nav", "Previous block", "Next block", "tx-badge", "Status", "Type", "pagerHTML(\"asset\"", "selectLimit(\"asset\"", "Asset events", "address-header", "address-value"} {
		if !strings.Contains(js, item) {
			t.Fatalf("app js missing dashboard presentation feature %q", item)
		}
	}
	if !strings.Contains(js, "data-copy") || !strings.Contains(js, "Copied") {
		t.Fatalf("app js missing copy helper feedback")
	}
	css := getText(t, server.URL+"/explorer-ui/assets/styles.css", http.StatusOK)
	if !strings.Contains(css, ".topbar") || !strings.Contains(css, "@media") {
		t.Fatalf("css missing expected responsive styles")
	}

	resp, err := http.Get(server.URL + "/explorer-ui/assets/missing.js")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing asset status = %d want 404", resp.StatusCode)
	}

	resp, err = http.Post(server.URL+"/explorer-ui/", "text/plain", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("POST explorer UI status = %d want 405", resp.StatusCode)
	}
}

func TestExplorerUIPublicRPCSafety(t *testing.T) {
	paths := newProfileRPCTestNode(t, config.Testnet())
	mux := http.NewServeMux()
	RegisterHandlers(mux, paths, NodeInfo{PublicRPC: true, Profile: config.Testnet()})
	server := httptest.NewServer(mux)
	defer server.Close()

	_ = getText(t, server.URL+"/explorer-ui/", http.StatusOK)
	_ = getRPCMap(t, server.URL+"/explorer/status", http.StatusOK)

	walletResp, err := http.Post(server.URL+"/wallet/new", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_ = walletResp.Body.Close()
	if walletResp.StatusCode != http.StatusForbidden {
		t.Fatalf("wallet endpoint status = %d want 403", walletResp.StatusCode)
	}

	adminResp, err := http.Get(server.URL + "/debug/p2p")
	if err != nil {
		t.Fatal(err)
	}
	_ = adminResp.Body.Close()
	if adminResp.StatusCode != http.StatusForbidden {
		t.Fatalf("admin endpoint status = %d want 403", adminResp.StatusCode)
	}
}

func TestExplorerUIEmbeddedFilesAndNoWriteEndpointStrings(t *testing.T) {
	index, err := explorerUIFiles.ReadFile("web/explorer/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "assets/app.js") || !strings.Contains(string(index), "assets/styles.css") {
		t.Fatalf("index asset references missing")
	}

	appJS, err := explorerUIFiles.ReadFile("web/explorer/assets/app.js")
	if err != nil {
		t.Fatal(err)
	}
	source := string(appJS)
	forbidden := []string{
		"/wallet/new",
		"/wallet/export",
		"/send",
		"/stake/lock",
		"/stake/unlock",
		"/faucet/request",
		"/service/register",
		"/admin",
		"/dev/reset",
	}
	for _, item := range forbidden {
		if strings.Contains(source, item) {
			t.Fatalf("app js contains write endpoint string %q", item)
		}
	}
}

func getText(t *testing.T, url string, want int) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != want {
		t.Fatalf("%s status = %d want %d body=%s", url, resp.StatusCode, want, string(data))
	}
	return string(data)
}
