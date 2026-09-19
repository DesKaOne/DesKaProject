package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"deskachain/internal/config"
	"deskachain/internal/types"
)

type Client struct {
	http                 *http.Client
	NodeID               string
	P2PURL               string
	NetworkID            string
	NodeIdentity         NodeIdentity
	AuthenticateRequests bool
}

func NewClient() Client {
	return Client{http: &http.Client{Timeout: 5 * time.Second}}
}

func NewClientWithTimeout(timeout time.Duration) Client {
	return Client{http: &http.Client{Timeout: timeout}}
}

func NewClientForProfile(paths config.Paths, profile config.NetworkConfig, timeout time.Duration) (Client, error) {
	client := NewClientWithTimeout(timeout)
	if !profile.RequireAuthenticatedNode {
		return client, nil
	}
	identity, err := LoadOrCreateNodeIdentity(paths.NodeID)
	if err != nil {
		return Client{}, err
	}
	return client.WithNodeIdentity(identity), nil
}

func (c Client) WithIdentity(nodeID, p2pURL, networkID string) Client {
	c.NodeID = nodeID
	c.P2PURL = p2pURL
	c.NetworkID = networkID
	return c
}

func (c Client) WithNodeIdentity(identity NodeIdentity) Client {
	c.NodeIdentity = identity
	c.NodeID = identity.NodeID
	c.AuthenticateRequests = true
	return c
}

func (c Client) Status(peer string) (Status, error) {
	var status Status
	err := c.getJSON(peer, "/p2p/status", &status)
	return status, err
}

func (c Client) Handshake(peer string) (Handshake, error) {
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		return Handshake{}, err
	}
	challenge := hex.EncodeToString(challengeBytes)
	var handshake Handshake
	err := c.getJSONMode(peer, "/p2p/handshake?challenge="+challenge, nil, &handshake, false, true)
	if err != nil {
		return Handshake{}, err
	}
	if handshake.AuthChallenge != "" && handshake.AuthChallenge != challenge {
		return Handshake{}, fmt.Errorf("peer handshake challenge mismatch")
	}
	if handshake.IdentityVersion != 0 || handshake.NodePublicKey != "" || handshake.NodeSignature != "" {
		if err := VerifyHandshakeIdentity(handshake); err != nil {
			return Handshake{}, err
		}
	}
	return handshake, nil
}

func (c Client) Peers(peer string) (PeersResponse, error) {
	var response PeersResponse
	err := c.getJSON(peer, "/p2p/peers", &response)
	return response, err
}

func (c Client) Headers(peer string, from uint64, limit int) ([]BlockHeader, error) {
	var response HeadersResponse
	err := c.getJSON(peer, fmt.Sprintf("/p2p/headers?from=%d&limit=%d", from, limit), &response)
	return response.Headers, err
}

func (c Client) Locator(peer string) (LocatorResponse, error) {
	var response LocatorResponse
	err := c.getJSON(peer, "/p2p/locator", &response)
	return response, err
}

func (c Client) CommonAncestor(peer string, locator CommonAncestorRequest) (CommonAncestorResponse, error) {
	var response CommonAncestorResponse
	err := c.postJSON(peer, "/p2p/common-ancestor", locator, &response)
	return response, err
}

func (c Client) Block(peer string, height uint64) (types.Block, error) {
	var block types.Block
	err := c.getJSON(peer, fmt.Sprintf("/p2p/block/%d", height), &block)
	return block, err
}

func (c Client) BroadcastTx(peer string, tx types.Transaction) (TxResponse, error) {
	var response TxResponse
	err := c.postJSON(peer, "/p2p/tx", tx, &response)
	return response, err
}

func (c Client) BroadcastBlock(peer string, block types.Block) (BlockResponse, error) {
	var response BlockResponse
	err := c.postJSON(peer, "/p2p/block", block, &response)
	return response, err
}

func (c Client) IntroducePeer(peer string, intro PeerIntroduction) (map[string]any, error) {
	var response map[string]any
	err := c.postJSON(peer, "/p2p/peer", intro, &response)
	return response, err
}

func (c Client) getJSON(peer, path string, target any) error {
	return c.getJSONMode(peer, path, nil, target, c.AuthenticateRequests, true)
}

func (c Client) getJSONMode(peer, path string, raw []byte, target any, authenticated bool, rejectClientError bool) error {
	return c.doJSON(peer, http.MethodGet, path, raw, target, authenticated, rejectClientError)
}

func (c Client) postJSON(peer, path string, body, target any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.doJSON(peer, http.MethodPost, path, raw, target, c.AuthenticateRequests, false)
}

func (c Client) doJSON(peer, method, path string, raw []byte, target any, authenticated, rejectClientError bool) error {
	var handshake Handshake
	if authenticated {
		handshake, err := c.Handshake(peer)
		if err != nil {
			return err
		}
		if err := c.NodeIdentityValidation(); err != nil {
			return err
		}
		if c.NetworkID != "" && c.NetworkID != handshake.NetworkID {
			return fmt.Errorf("peer handshake network id mismatch")
		}
	}
	var bodyReader io.Reader
	if raw != nil {
		bodyReader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, strings.TrimRight(peer, "/")+path, bodyReader)
	if err != nil {
		return err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	c.addHeaders(req)
	var requestNonce string
	if authenticated {
		requestNonce, err = newP2PAuthNonce()
		if err != nil {
			return err
		}
		if err := SignP2PRequest(c.NodeIdentity, handshake.NetworkID, handshake.ChainID, req, raw, time.Now(), requestNonce); err != nil {
			return err
		}
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if authenticated {
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, p2pMessageAuthResponseLimit+1))
		if readErr != nil {
			return readErr
		}
		if len(responseBody) > p2pMessageAuthResponseLimit {
			return fmt.Errorf("peer authenticated response too large")
		}
		if err := VerifyP2PResponse(handshake, handshake.NetworkID, handshake.ChainID, requestNonce, resp.StatusCode, responseBody, resp.Header, time.Now()); err != nil {
			return err
		}
		if rejectClientError {
			if resp.StatusCode >= 400 {
				return fmt.Errorf("peer returned %s", resp.Status)
			}
		} else if resp.StatusCode >= 500 {
			return fmt.Errorf("peer returned %s", resp.Status)
		}
		return json.Unmarshal(responseBody, target)
	}

	if rejectClientError {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("peer returned %s", resp.Status)
		}
	} else if resp.StatusCode >= 500 {
		return fmt.Errorf("peer returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c Client) NodeIdentityValidation() error {
	if c.NodeIdentity.IdentityVersion() != NodeIdentityVersion {
		return fmt.Errorf("authenticated client requires a valid node identity")
	}
	return nil
}

func (c Client) addHeaders(req *http.Request) {
	if c.NodeID != "" {
		req.Header.Set("X-DKC-Node-ID", c.NodeID)
	}
	if c.P2PURL != "" {
		req.Header.Set("X-DKC-P2P-URL", c.P2PURL)
	}
	if c.NetworkID != "" {
		req.Header.Set("X-DKC-Network-ID", c.NetworkID)
	}
}
