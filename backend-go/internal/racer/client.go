package racer

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type RelayTimeouts struct {
	DialMS      int64 `json:"dial_ms"`
	FirstByteMS int64 `json:"first_byte_ms"`
	TotalMS     int64 `json:"total_ms"`
}

type RelayRequest struct {
	Method      string            `json:"method"`
	UpstreamURL string            `json:"upstream_url"`
	Headers     map[string]string `json:"headers"`
	Body        []byte            `json:"body"`
	Timeouts    RelayTimeouts     `json:"timeouts"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: BaseURL(),
		httpClient: &http.Client{
			Timeout: 0, // streaming; upstream timeouts enforced by racer
		},
	}
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("racer health %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) Relay(ctx context.Context, relayReq RelayRequest) (*http.Response, error) {
	payload := struct {
		Method      string            `json:"method"`
		UpstreamURL string            `json:"upstream_url"`
		Headers     map[string]string `json:"headers"`
		Body        string            `json:"body"`
		Timeouts    RelayTimeouts     `json:"timeouts"`
	}{
		Method:      relayReq.Method,
		UpstreamURL: relayReq.UpstreamURL,
		Headers:     relayReq.Headers,
		Body:        base64.StdEncoding.EncodeToString(relayReq.Body),
		Timeouts:    relayReq.Timeouts,
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/relay", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func DefaultTimeouts() RelayTimeouts {
	return RelayTimeouts{
		DialMS:      5_000,
		FirstByteMS: 30_000,
		TotalMS:     600_000,
	}
}

func DefaultClient() *Client {
	return NewClient()
}

func Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return DefaultClient().Health(ctx)
}