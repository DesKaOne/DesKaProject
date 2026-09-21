package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RPCClient struct {
	BaseURL string
	Client  *http.Client
}

func New(baseURL string) *RPCClient {
	return &RPCClient{BaseURL: strings.TrimRight(baseURL, "/"), Client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *RPCClient) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil { return err }
	resp, err := c.Client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 400 { return fmt.Errorf("rpc GET %s: %s", path, resp.Status) }
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *RPCClient) postJSON(ctx context.Context, path string, in any, out any) error {
	body, err := json.Marshal(in)
	if err != nil { return err }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, strings.NewReader(string(body)))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 400 { return fmt.Errorf("rpc POST %s: %s", path, resp.Status) }
	if out == nil { return nil }
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *RPCClient) NetworkInfo(ctx context.Context) (map[string]any, error) {
	var out map[string]any
	return out, c.get(ctx, "/network/info", &out)
}

func (c *RPCClient) Balance(ctx context.Context, address string) (map[string]any, error) {
	var out map[string]any
	path := "/balance/" + url.PathEscape(address)
	return out, c.get(ctx, path, &out)
}

func (c *RPCClient) Send(ctx context.Context, tx any) (map[string]any, error) {
	var out map[string]any
	return out, c.postJSON(ctx, "/send", tx, &out)
}

func (c *RPCClient) Tx(ctx context.Context, id string) (map[string]any, error) {
	var out map[string]any
	return out, c.get(ctx, "/tx/"+url.PathEscape(id), &out)
}
