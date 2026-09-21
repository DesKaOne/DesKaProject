package p2p

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	p2pMessageAuthVersion       uint32 = 1
	p2pMessageAuthDomain               = "IndoChain/p2p-message/v1"
	p2pMessageAuthTimeSkew             = 5 * time.Minute
	p2pMessageAuthBodyLimit            = 8 << 20
	p2pMessageAuthResponseLimit        = 16 << 20

	authHeaderVersion    = "X-IND-Auth-Version"
	authHeaderNodeID     = "X-IND-Node-ID"
	authHeaderNodePubKey = "X-IND-Node-PubKey"
	authHeaderNetworkID  = "X-IND-Network-ID"
	authHeaderChainID    = "X-IND-Chain-ID"
	authHeaderTimestamp  = "X-IND-Auth-Timestamp"
	authHeaderNonce      = "X-IND-Auth-Nonce"
	authHeaderSignature  = "X-IND-Auth-Signature"
)

type p2pRequestAuth struct {
	NodeID    string
	PublicKey ed25519.PublicKey
	NetworkID string
	ChainID   uint64
	Timestamp int64
	Nonce     string
}

var requestReplayCache = struct {
	sync.Mutex
	seen map[string]int64
}{
	seen: make(map[string]int64),
}

func SignP2PRequest(identity NodeIdentity, networkID string, chainID uint64, req *http.Request, body []byte, now time.Time, nonce string) error {
	if identity.IdentityVersion() != NodeIdentityVersion {
		return errors.New("unsupported node identity version")
	}
	if strings.TrimSpace(networkID) == "" {
		return errors.New("network id is required")
	}
	if req == nil {
		return errors.New("request is required")
	}
	if len(nonce) != 64 {
		return errors.New("invalid auth nonce")
	}
	if _, err := hex.DecodeString(nonce); err != nil {
		return errors.New("invalid auth nonce")
	}
	timestamp := now.Unix()
	req.Header.Set(authHeaderVersion, strconv.FormatUint(uint64(p2pMessageAuthVersion), 10))
	req.Header.Set(authHeaderNodeID, identity.NodeID)
	req.Header.Set(authHeaderNodePubKey, hex.EncodeToString(identity.PublicKey))
	req.Header.Set(authHeaderNetworkID, networkID)
	req.Header.Set(authHeaderChainID, strconv.FormatUint(chainID, 10))
	req.Header.Set(authHeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(authHeaderNonce, nonce)
	payload := p2pRequestSigningBytes(identity.NodeID, identity.PublicKey, networkID, chainID, req.Method, req.URL.RequestURI(), timestamp, nonce, body)
	signature := ed25519.Sign(identity.PrivateKey, payload)
	req.Header.Set(authHeaderSignature, hex.EncodeToString(signature))
	return nil
}

func VerifyP2PRequest(localNetworkID string, localChainID uint64, req *http.Request, body []byte, now time.Time) (p2pRequestAuth, error) {
	if req == nil {
		return p2pRequestAuth{}, errors.New("request is required")
	}
	version, err := parseAuthVersion(req.Header.Get(authHeaderVersion))
	if err != nil {
		return p2pRequestAuth{}, err
	}
	if version != p2pMessageAuthVersion {
		return p2pRequestAuth{}, errors.New("unsupported p2p message auth version")
	}
	nodeID := strings.TrimSpace(req.Header.Get(authHeaderNodeID))
	networkID := strings.TrimSpace(req.Header.Get(authHeaderNetworkID))
	publicKeyRaw := strings.TrimSpace(req.Header.Get(authHeaderNodePubKey))
	signatureRaw := strings.TrimSpace(req.Header.Get(authHeaderSignature))
	nonce := strings.TrimSpace(req.Header.Get(authHeaderNonce))
	timestampText := strings.TrimSpace(req.Header.Get(authHeaderTimestamp))
	chainText := strings.TrimSpace(req.Header.Get(authHeaderChainID))
	if nodeID == "" || networkID == "" || publicKeyRaw == "" || signatureRaw == "" || nonce == "" || timestampText == "" || chainText == "" {
		return p2pRequestAuth{}, errors.New("incomplete p2p message authentication headers")
	}
	if networkID != localNetworkID {
		return p2pRequestAuth{}, errors.New("peer rejected: network id mismatch")
	}
	chainID, err := strconv.ParseUint(chainText, 10, 64)
	if err != nil || chainID != localChainID {
		return p2pRequestAuth{}, errors.New("peer rejected: chain id mismatch")
	}
	publicKey, err := hex.DecodeString(publicKeyRaw)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return p2pRequestAuth{}, errors.New("invalid p2p message public key")
	}
	signature, err := hex.DecodeString(signatureRaw)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return p2pRequestAuth{}, errors.New("invalid p2p message signature")
	}
	if len(nonce) != 64 {
		return p2pRequestAuth{}, errors.New("invalid p2p message nonce")
	}
	if _, err := hex.DecodeString(nonce); err != nil {
		return p2pRequestAuth{}, errors.New("invalid p2p message nonce")
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return p2pRequestAuth{}, errors.New("invalid p2p message timestamp")
	}
	if delta := now.Unix() - timestamp; delta > int64(p2pMessageAuthTimeSkew/time.Second) || delta < -int64(p2pMessageAuthTimeSkew/time.Second) {
		return p2pRequestAuth{}, errors.New("p2p message timestamp outside allowed skew")
	}
	payload := p2pRequestSigningBytes(nodeID, publicKey, networkID, chainID, req.Method, req.URL.RequestURI(), timestamp, nonce, body)
	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		return p2pRequestAuth{}, errors.New("invalid p2p message signature")
	}
	cacheKey := hex.EncodeToString(publicKey) + ":" + nonce
	if !consumeRequestNonce(cacheKey, now.Unix()) {
		return p2pRequestAuth{}, errors.New("p2p message nonce replay detected")
	}
	return p2pRequestAuth{
		NodeID:    nodeID,
		PublicKey: append(ed25519.PublicKey(nil), publicKey...),
		NetworkID: networkID,
		ChainID:   chainID,
		Timestamp: timestamp,
		Nonce:     nonce,
	}, nil
}

func SignP2PResponse(identity NodeIdentity, networkID string, chainID uint64, requestNonce string, status int, body []byte, now time.Time) (http.Header, error) {
	if identity.IdentityVersion() != NodeIdentityVersion {
		return nil, errors.New("unsupported node identity version")
	}
	if len(requestNonce) != 64 {
		return nil, errors.New("invalid response request nonce")
	}
	if _, err := hex.DecodeString(requestNonce); err != nil {
		return nil, errors.New("invalid response request nonce")
	}
	timestamp := now.Unix()
	payload := p2pResponseSigningBytes(identity.NodeID, identity.PublicKey, networkID, chainID, requestNonce, status, timestamp, body)
	signature := ed25519.Sign(identity.PrivateKey, payload)
	h := make(http.Header)
	h.Set(authHeaderVersion, strconv.FormatUint(uint64(p2pMessageAuthVersion), 10))
	h.Set(authHeaderNodeID, identity.NodeID)
	h.Set(authHeaderNodePubKey, hex.EncodeToString(identity.PublicKey))
	h.Set(authHeaderNetworkID, networkID)
	h.Set(authHeaderChainID, strconv.FormatUint(chainID, 10))
	h.Set(authHeaderTimestamp, strconv.FormatInt(timestamp, 10))
	h.Set(authHeaderNonce, requestNonce)
	h.Set(authHeaderSignature, hex.EncodeToString(signature))
	return h, nil
}

func VerifyP2PResponse(handshake Handshake, networkID string, chainID uint64, requestNonce string, status int, body []byte, headers http.Header, now time.Time) error {
	version, err := parseAuthVersion(headers.Get(authHeaderVersion))
	if err != nil {
		return err
	}
	if version != p2pMessageAuthVersion {
		return errors.New("unsupported p2p message auth version")
	}
	nodeID := strings.TrimSpace(headers.Get(authHeaderNodeID))
	publicKeyRaw := strings.TrimSpace(headers.Get(authHeaderNodePubKey))
	responseNetworkID := strings.TrimSpace(headers.Get(authHeaderNetworkID))
	chainText := strings.TrimSpace(headers.Get(authHeaderChainID))
	timestampText := strings.TrimSpace(headers.Get(authHeaderTimestamp))
	nonce := strings.TrimSpace(headers.Get(authHeaderNonce))
	signatureRaw := strings.TrimSpace(headers.Get(authHeaderSignature))
	if nodeID == "" || publicKeyRaw == "" || responseNetworkID == "" || chainText == "" || timestampText == "" || nonce == "" || signatureRaw == "" {
		return errors.New("incomplete p2p response authentication headers")
	}
	if nodeID != handshake.NodeID {
		return fmt.Errorf("p2p response node id mismatch: handshake=%s response=%s", handshake.NodeID, nodeID)
	}
	if publicKeyRaw != strings.TrimSpace(handshake.NodePublicKey) {
		return errors.New("p2p response node public key mismatch")
	}
	if responseNetworkID != networkID {
		return errors.New("p2p response network id mismatch")
	}
	responseChainID, err := strconv.ParseUint(chainText, 10, 64)
	if err != nil || responseChainID != chainID {
		return errors.New("p2p response chain id mismatch")
	}
	if nonce != requestNonce {
		return errors.New("p2p response request nonce mismatch")
	}
	publicKey, err := hex.DecodeString(publicKeyRaw)
	if err != nil || len(publicKey) != ed25519.PublicKeySize {
		return errors.New("invalid p2p response public key")
	}
	signature, err := hex.DecodeString(signatureRaw)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return errors.New("invalid p2p response signature")
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return errors.New("invalid p2p response timestamp")
	}
	if delta := now.Unix() - timestamp; delta > int64(p2pMessageAuthTimeSkew/time.Second) || delta < -int64(p2pMessageAuthTimeSkew/time.Second) {
		return errors.New("p2p response timestamp outside allowed skew")
	}
	payload := p2pResponseSigningBytes(nodeID, publicKey, responseNetworkID, responseChainID, requestNonce, status, timestamp, body)
	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		return errors.New("invalid p2p response signature")
	}
	return nil
}

func p2pRequestSigningBytes(nodeID string, publicKey []byte, networkID string, chainID uint64, method, requestURI string, timestamp int64, nonce string, body []byte) []byte {
	var buf bytes.Buffer
	writeP2PAuthString(&buf, p2pMessageAuthDomain)
	writeP2PAuthUint32(&buf, p2pMessageAuthVersion)
	writeP2PAuthString(&buf, "request")
	writeP2PAuthString(&buf, nodeID)
	writeP2PAuthBytes(&buf, publicKey)
	writeP2PAuthString(&buf, networkID)
	writeP2PAuthUint64(&buf, chainID)
	writeP2PAuthString(&buf, method)
	writeP2PAuthString(&buf, requestURI)
	writeP2PAuthInt64(&buf, timestamp)
	writeP2PAuthString(&buf, nonce)
	writeP2PAuthBytes(&buf, bodyDigest(body))
	return buf.Bytes()
}

func p2pResponseSigningBytes(nodeID string, publicKey []byte, networkID string, chainID uint64, requestNonce string, status int, timestamp int64, body []byte) []byte {
	var buf bytes.Buffer
	writeP2PAuthString(&buf, p2pMessageAuthDomain)
	writeP2PAuthUint32(&buf, p2pMessageAuthVersion)
	writeP2PAuthString(&buf, "response")
	writeP2PAuthString(&buf, nodeID)
	writeP2PAuthBytes(&buf, publicKey)
	writeP2PAuthString(&buf, networkID)
	writeP2PAuthUint64(&buf, chainID)
	writeP2PAuthString(&buf, requestNonce)
	writeP2PAuthUint32(&buf, uint32(status))
	writeP2PAuthInt64(&buf, timestamp)
	writeP2PAuthBytes(&buf, bodyDigest(body))
	return buf.Bytes()
}

func bodyDigest(body []byte) []byte {
	sum := sha256.Sum256(body)
	return sum[:]
}

func hasP2PAuthHeaders(header http.Header) bool {
	for _, key := range []string{authHeaderVersion, authHeaderNodePubKey, authHeaderTimestamp, authHeaderNonce, authHeaderSignature} {
		if strings.TrimSpace(header.Get(key)) != "" {
			return true
		}
	}
	return false
}

func parseAuthVersion(value string) (uint32, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 32)
	if err != nil {
		return 0, errors.New("invalid p2p message auth version")
	}
	return uint32(parsed), nil
}

func consumeRequestNonce(key string, nowUnix int64) bool {
	requestReplayCache.Lock()
	defer requestReplayCache.Unlock()
	for existing, expiresAt := range requestReplayCache.seen {
		if expiresAt <= nowUnix {
			delete(requestReplayCache.seen, existing)
		}
	}
	if _, exists := requestReplayCache.seen[key]; exists {
		return false
	}
	requestReplayCache.seen[key] = nowUnix + int64(p2pMessageAuthTimeSkew/time.Second)
	if len(requestReplayCache.seen) > 4096 {
		for existing, expiresAt := range requestReplayCache.seen {
			if expiresAt <= nowUnix {
				delete(requestReplayCache.seen, existing)
			}
		}
	}
	return true
}

func newP2PAuthNonce() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

func writeP2PAuthString(buf *bytes.Buffer, value string) {
	writeP2PAuthUint32(buf, uint32(len(value)))
	buf.WriteString(value)
}

func writeP2PAuthBytes(buf *bytes.Buffer, value []byte) {
	writeP2PAuthUint32(buf, uint32(len(value)))
	buf.Write(value)
}

func writeP2PAuthUint32(buf *bytes.Buffer, value uint32) {
	var raw [4]byte
	binary.BigEndian.PutUint32(raw[:], value)
	buf.Write(raw[:])
}

func writeP2PAuthUint64(buf *bytes.Buffer, value uint64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], value)
	buf.Write(raw[:])
}

func writeP2PAuthInt64(buf *bytes.Buffer, value int64) {
	var raw [8]byte
	binary.BigEndian.PutUint64(raw[:], uint64(value))
	buf.Write(raw[:])
}

type authenticatedResponseWriter struct {
	original http.ResponseWriter
	header   http.Header
	body     bytes.Buffer
	status   int
}

func newAuthenticatedResponseWriter(original http.ResponseWriter) *authenticatedResponseWriter {
	return &authenticatedResponseWriter{original: original, header: make(http.Header)}
}

func (w *authenticatedResponseWriter) Header() http.Header {
	return w.header
}

func (w *authenticatedResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
}

func (w *authenticatedResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

func (w *authenticatedResponseWriter) commit(responseHeaders http.Header, body []byte, status int) error {
	dst := w.original.Header()
	for key, values := range responseHeaders {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	for key, values := range w.header {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	for key, values := range responseHeaders {
		dst.Set(key, strings.Join(values, ","))
	}
	if status == 0 {
		status = http.StatusOK
	}
	for key, values := range w.header {
		if key == authHeaderVersion || key == authHeaderNodeID || key == authHeaderNodePubKey || key == authHeaderNetworkID || key == authHeaderChainID || key == authHeaderTimestamp || key == authHeaderNonce || key == authHeaderSignature {
			continue
		}
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
	w.original.WriteHeader(status)
	_, err := w.original.Write(body)
	return err
}

func (w *authenticatedResponseWriter) Unwrap() http.ResponseWriter {
	return w.original
}

func readAuthenticatedRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, p2pMessageAuthBodyLimit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > p2pMessageAuthBodyLimit {
		return nil, fmt.Errorf("authenticated request body too large")
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return raw, nil
}
