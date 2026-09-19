package p2p

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"deskachain/internal/types"
)

type Client struct {
	http      *http.Client
	NodeID    string
	P2PURL    string
	NetworkID string
}

func NewClient() Client {
	return Client{http: &http.Client{Timeout: 5 * time.Second}}
}

func NewClientWithTimeout(timeout time.Duration) Client {
	return Client{http: &http.Client{Timeout: timeout}}
}

func (c Client) WithIdentity(nodeID, p2pURL, networkID string) Client {
	c.NodeID = nodeID
	c.P2PURL = p2pURL
	c.NetworkID = networkID
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
	err := c.getJSON(peer, "/p2p/handshake?challenge="+challenge, &handshake)
	if err != nil {
		return Handshake{}, err
	}
	if handshake.AuthChallenge != challenge {
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
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(peer, "/")+path, nil)
	if err != nil {
		return err
	}
	c.addHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("peer returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func (c Client) postJSON(peer, path string, body, target any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(peer, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	c.addHeaders(req)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return fmt.Errorf("peer returned %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(target)
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
