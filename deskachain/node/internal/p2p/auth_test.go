package p2p

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"indochain/internal/config"
)

func testNodeIdentity(t *testing.T, nodeID string) NodeIdentity {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate identity: %v", err)
	}
	return NodeIdentity{NodeID: nodeID, PublicKey: publicKey, PrivateKey: privateKey}
}

func signedTestRequest(t *testing.T, identity NodeIdentity, networkID string, chainID uint64, method, requestURI string, body []byte, now time.Time) (*http.Request, string) {
	t.Helper()
	req := httptest.NewRequest(method, "http://peer.example"+requestURI, bytes.NewReader(body))
	nonce, err := newP2PAuthNonce()
	if err != nil {
		t.Fatalf("new nonce: %v", err)
	}
	if err := SignP2PRequest(identity, networkID, chainID, req, body, now, nonce); err != nil {
		t.Fatalf("sign request: %v", err)
	}
	return req, nonce
}

func TestP2PMessageAuthRequestRoundTripAndReplay(t *testing.T) {
	identity := testNodeIdentity(t, "node-a")
	now := time.Unix(1_800_000_000, 0)
	body := []byte("{\"hello\":\"world\"}")
	req, _ := signedTestRequest(t, identity, "ind-testnet-1", 777101, http.MethodPost, "/p2p/tx?x=1", body, now)

	got, err := VerifyP2PRequest("ind-testnet-1", 777101, req, body, now)
	if err != nil {
		t.Fatalf("verify request: %v", err)
	}
	if got.NodeID != identity.NodeID || hex.EncodeToString(got.PublicKey) != hex.EncodeToString(identity.PublicKey) {
		t.Fatalf("verified identity mismatch")
	}
	if _, err := VerifyP2PRequest("ind-testnet-1", 777101, req, body, now); err == nil || !strings.Contains(err.Error(), "replay") {
		t.Fatalf("expected replay rejection, got %v", err)
	}
}

func TestP2PMessageAuthRequestRejectsTampering(t *testing.T) {
	identity := testNodeIdentity(t, "node-b")
	now := time.Unix(1_800_000_100, 0)
	body := []byte("payload")

	tests := []struct {
		name   string
		mutate func(*http.Request)
	}{
		{
			name: "body",
			mutate: func(req *http.Request) {
				req.Body = ioNopCloser([]byte("tampered"))
			},
		},
		{
			name: "path",
			mutate: func(req *http.Request) {
				req.URL.Path = "/p2p/block"
			},
		},
		{
			name: "node id",
			mutate: func(req *http.Request) {
				req.Header.Set(authHeaderNodeID, "node-evil")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := signedTestRequest(t, identity, "ind-testnet-1", 777101, http.MethodPost, "/p2p/tx", body, now)
			tt.mutate(req)
			verifyBody := body
			if tt.name == "body" {
				verifyBody = []byte("tampered")
			}
			if _, err := VerifyP2PRequest("ind-testnet-1", 777101, req, verifyBody, now); err == nil {
				t.Fatalf("expected tampering rejection")
			}
		})
	}
}

func TestP2PMessageAuthResponseRoundTripAndTampering(t *testing.T) {
	serverIdentity := testNodeIdentity(t, "server-a")
	handshake := Handshake{
		NodeID:          serverIdentity.NodeID,
		NodePublicKey:   hex.EncodeToString(serverIdentity.PublicKey),
		IdentityVersion: NodeIdentityVersion,
	}
	requestNonce, err := newP2PAuthNonce()
	if err != nil {
		t.Fatalf("new nonce: %v", err)
	}
	now := time.Unix(1_800_000_200, 0)
	body := []byte("{\"ok\":true}")
	headers, err := SignP2PResponse(serverIdentity, "ind-testnet-1", 777101, requestNonce, http.StatusOK, body, now)
	if err != nil {
		t.Fatalf("sign response: %v", err)
	}
	if err := VerifyP2PResponse(handshake, "ind-testnet-1", 777101, requestNonce, http.StatusOK, body, headers, now); err != nil {
		t.Fatalf("verify response: %v", err)
	}
	tampered := append([]byte(nil), body...)
	tampered[0] ^= 0x01
	if err := VerifyP2PResponse(handshake, "ind-testnet-1", 777101, requestNonce, http.StatusOK, tampered, headers, now); err == nil {
		t.Fatalf("expected response body tampering rejection")
	}
}

func TestP2PMessageAuthServerMiddleware(t *testing.T) {
	profile := config.Localnet()
	profile.RequireAuthenticatedNode = true
	paths := config.NewPaths(t.TempDir())
	server := NewServerWithProfile(paths, profile)
	remote := testNodeIdentity(t, "remote-node")
	now := time.Now().Truncate(time.Second)
	body := []byte("{\"message\":\"hello\"}")

	var next http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "ok")
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})
	handler := server.authenticated(next)

	req, nonce := signedTestRequest(t, remote, profile.NetworkID, profile.ChainID, http.MethodPost, "/p2p/tx", body, now)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("authenticated request status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get(authHeaderNonce) != nonce {
		t.Fatalf("response nonce mismatch")
	}
	serverIdentity, err := LoadOrCreateNodeIdentity(paths.NodeID)
	if err != nil {
		t.Fatalf("load server identity: %v", err)
	}
	hs := Handshake{NodeID: serverIdentity.NodeID, NodePublicKey: hex.EncodeToString(serverIdentity.PublicKey), IdentityVersion: NodeIdentityVersion}
	if err := VerifyP2PResponse(hs, profile.NetworkID, profile.ChainID, nonce, rec.Code, rec.Body.Bytes(), rec.Header(), time.Now()); err != nil {
		t.Fatalf("verify middleware response: %v", err)
	}

	unsigned := httptest.NewRequest(http.MethodPost, "http://peer.example/p2p/tx", bytes.NewReader(body))
	unsignedRec := httptest.NewRecorder()
	handler.ServeHTTP(unsignedRec, unsigned)
	if unsignedRec.Code != http.StatusUnauthorized {
		t.Fatalf("unsigned status=%d want=%d", unsignedRec.Code, http.StatusUnauthorized)
	}
}

func TestP2PMessageAuthOptionalUnsignedCompatibility(t *testing.T) {
	profile := config.Localnet()
	profile.RequireAuthenticatedNode = false
	server := NewServerWithProfile(config.NewPaths(t.TempDir()), profile)
	handler := server.authenticated(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}))
	req := httptest.NewRequest(http.MethodGet, "http://peer.example/p2p/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("optional unsigned status=%d want=%d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get(authHeaderSignature) != "" {
		t.Fatalf("unsigned compatibility response should not claim authentication")
	}
}

func TestNewClientForProfileBindsNetworkIdentity(t *testing.T) {
	profile := config.Testnet()
	paths := config.NewPaths(t.TempDir())

	client, err := NewClientForProfile(paths, profile, time.Second)
	if err != nil {
		t.Fatalf("create authenticated client: %v", err)
	}
	if !client.AuthenticateRequests {
		t.Fatalf("testnet client must authenticate requests")
	}
	if client.NetworkID != profile.NetworkID {
		t.Fatalf("client network id=%q want=%q", client.NetworkID, profile.NetworkID)
	}
	if client.NodeIdentity.IdentityVersion() != NodeIdentityVersion {
		t.Fatalf("client node identity is not initialized")
	}

	req := httptest.NewRequest(http.MethodGet, "http://peer.example/p2p/status", nil)
	client.addHeaders(req)
	if got := req.Header.Get(authHeaderNetworkID); got != profile.NetworkID {
		t.Fatalf("auth network header=%q want=%q", got, profile.NetworkID)
	}
}

func TestCheckPeerWithProfileUsesAuthenticatedClient(t *testing.T) {
	profile := config.Testnet()
	paths := config.NewPaths(t.TempDir())
	client, err := NewClientForProfile(paths, profile, time.Second)
	if err != nil {
		t.Fatalf("create authenticated client: %v", err)
	}
	if !client.AuthenticateRequests {
		t.Fatalf("testnet peer checks must use authenticated client")
	}
	if client.NetworkID != profile.NetworkID {
		t.Fatalf("client network id=%q want=%q", client.NetworkID, profile.NetworkID)
	}
}

func TestP2PClientSurfacesUnsignedAuthRejection(t *testing.T) {
	serverIdentity := testNodeIdentity(t, "server-auth-error")
	profile := config.Testnet()
	paths := config.NewPaths(t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/p2p/handshake" {
			hs := Handshake{NetworkID: profile.NetworkID, ChainID: profile.ChainID, NodeID: serverIdentity.NodeID, IdentityVersion: NodeIdentityVersion, NodePublicKey: hex.EncodeToString(serverIdentity.PublicKey), AuthChallenge: r.URL.Query().Get("challenge")}
			writeJSON(w, http.StatusOK, hs)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid p2p message signature"})
	}))
	defer server.Close()
	client, err := NewClientForProfile(paths, profile, time.Second)
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	_, err = client.Status(server.URL)
	if err == nil || !strings.Contains(err.Error(), "invalid p2p message signature") {
		t.Fatalf("expected server auth rejection, got %v", err)
	}
}

func TestP2PMessageAuthRejectsInvalidRequestBeforeSigning(t *testing.T) {
	profile := config.Testnet()
	paths := config.NewPaths(t.TempDir())
	server := NewServerWithProfile(paths, profile)
	remote := testNodeIdentity(t, "remote-node-invalid-request")
	req, _ := signedTestRequest(t, remote, profile.NetworkID, profile.ChainID, http.MethodGet, "/p2p/status", nil, time.Now().Truncate(time.Second))
	req.Header.Set(authHeaderVersion, "not-a-number")
	rec := httptest.NewRecorder()
	server.authenticated(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})).ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want=%d", rec.Code, http.StatusUnauthorized)
	}
	if rec.Header().Get(authHeaderVersion) != "" {
		t.Fatalf("pre-auth rejection must not claim an authenticated response")
	}
}

type nopCloser struct {
	*bytes.Reader
}

func (nopCloser) Close() error { return nil }

func ioNopCloser(data []byte) *nopCloser {
	return &nopCloser{Reader: bytes.NewReader(data)}
}
