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

type RaceRequest struct {
	Method            string            `json:"method"`
	Path              string            `json:"path"`
	Endpoints         []string          `json:"endpoints"`
	Headers           map[string]string `json:"headers"`
	Body              []byte            `json:"body"`
	Timeouts          RelayTimeouts     `json:"timeouts"`
	Stream            bool              `json:"stream"`
	CancelOnFirstWin  bool              `json:"cancel_on_first_win"`
}

type RaceFailureMeta struct {
	Endpoint       string `json:"endpoint"`
	Status         int    `json:"status"`
	QuotaExceeded  bool   `json:"quota_exceeded"`
	RateLimited    bool   `json:"rate_limited"`
	ClientError    bool   `json:"client_error"`
	Message        string `json:"message"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: BaseURL(),
		httpClient: &http.Client{
			Timeout: 0,
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

func (c *Client) postJSON(ctx context.Context, path string, payload any) (*http.Response, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
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
	return c.postJSON(ctx, "/relay", payload)
}

func (c *Client) Race(ctx context.Context, raceReq RaceRequest) (*http.Response, error) {
	payload := struct {
		Method           string            `json:"method"`
		Path             string            `json:"path"`
		Endpoints        []string          `json:"endpoints"`
		Headers          map[string]string `json:"headers"`
		Body             string            `json:"body"`
		Timeouts         RelayTimeouts     `json:"timeouts"`
		Stream           bool              `json:"stream"`
		CancelOnFirstWin bool              `json:"cancel_on_first_win"`
	}{
		Method:           raceReq.Method,
		Path:             raceReq.Path,
		Endpoints:        raceReq.Endpoints,
		Headers:          raceReq.Headers,
		Body:             base64.StdEncoding.EncodeToString(raceReq.Body),
		Timeouts:         raceReq.Timeouts,
		Stream:           raceReq.Stream,
		CancelOnFirstWin: raceReq.CancelOnFirstWin,
	}
	return c.postJSON(ctx, "/race", payload)
}

func ParseRaceFailures(header string) ([]RaceFailureMeta, error) {
	if strings.TrimSpace(header) == "" {
		return nil, nil
	}
	raw, err := base64.StdEncoding.DecodeString(header)
	if err != nil {
		return nil, err
	}
	var failures []RaceFailureMeta
	if err := json.Unmarshal(raw, &failures); err != nil {
		return nil, err
	}
	return failures, nil
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