package p2p

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"deskachain/internal/chain"
	"deskachain/internal/config"
)

func TestP2PBlockAndHeaderResponsesHonorNetworkLimits(t *testing.T) {
	profile := config.Localnet()
	profile.NetworkLimits.MaxHeaderBatch = 1
	profile.NetworkLimits.MaxSyncBlocks = 1
	paths := config.NewPaths(t.TempDir())
	server := NewServerWithProfile(paths, profile)

	req := httptest.NewRequest(http.MethodGet, "/p2p/headers?from=0&limit=500", nil)
	rec := httptest.NewRecorder()
	server.headers(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("headers status = %d", rec.Code)
	}
	var headers HeadersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &headers); err != nil {
		t.Fatal(err)
	}
	if len(headers.Headers) > 1 {
		t.Fatalf("headers response exceeded configured limit: %d", len(headers.Headers))
	}

	req = httptest.NewRequest(http.MethodGet, "/p2p/blocks?from=0&limit=500", nil)
	rec = httptest.NewRecorder()
	server.blocks(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("blocks status = %d", rec.Code)
	}
	var blocks BlocksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &blocks); err != nil {
		t.Fatal(err)
	}
	if len(blocks.Blocks) > 1 {
		t.Fatalf("blocks response exceeded configured limit: %d", len(blocks.Blocks))
	}
}

func TestSyncRejectsRemoteRangeBeforeHeaderFetch(t *testing.T) {
	profile := config.Testnet()
	profile.NetworkLimits.MaxSyncBlocks = 2
	profile.NetworkLimits.MaxHeaderBatch = 2
	paths := config.NewPaths(t.TempDir())
	genesis := chain.GenesisBlockForNetwork(profile)
	statusRequests := 0
	headerRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/p2p/handshake":
			writeJSON(w, http.StatusOK, Handshake{
				NetworkName: profile.NetworkName,
				NetworkID: profile.NetworkID,
				ChainID: profile.ChainID,
				ProtocolVersion: profile.ProtocolVersion,
				MinProtocolVersion: profile.MinProtocolVersion,
				P2PProtocolVersion: profile.P2PProtocolVersion,
				GenesisHash: genesis.Hash,
			})
		case "/p2p/status":
			statusRequests++
			writeJSON(w, http.StatusOK, Status{
				Network: profile.Name,
				NetworkID: profile.NetworkID,
				ChainID: profile.ChainID,
				GenesisHash: genesis.Hash,
				ProtocolVersion: profile.ProtocolVersion,
				P2PProtocolVersion: profile.P2PProtocolVersion,
				Height: 3,
				TipHash: "peer-tip",
			})
		case "/p2p/headers":
			headerRequests++
			writeJSON(w, http.StatusOK, HeadersResponse{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := SyncFromPeerWithProfile(paths, server.URL, nil, profile); err == nil {
		t.Fatal("expected oversized sync range rejection")
	}
	if statusRequests != 1 {
		t.Fatalf("status requests = %d, want 1", statusRequests)
	}
	if headerRequests != 0 {
		t.Fatalf("headers requests = %d, want 0", headerRequests)
	}
}
