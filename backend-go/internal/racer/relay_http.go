package racer

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// NewRelayHTTPClient returns an http.Client that proxies requests through POST /relay.
// Used for background tester I/O when BACKGROUND_ENDPOINT_OUTBOUND=rust.
func NewRelayHTTPClient(defaultTimeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   0,
		Transport: &relayTransport{defaultTimeout: defaultTimeout},
	}
}

type relayTransport struct {
	defaultTimeout time.Duration
}

func (t *relayTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	if req.Body != nil {
		raw, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		body = raw
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	timeout := t.defaultTimeout
	if deadline, ok := req.Context().Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 {
			timeout = remaining
		}
	}

	headers := map[string]string{}
	for k, vs := range req.Header {
		kl := strings.ToLower(k)
		if kl == "host" || kl == "content-length" {
			continue
		}
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}

	timeouts := DefaultTimeouts()
	timeouts.TotalMS = timeout.Milliseconds()
	if timeouts.TotalMS <= 0 {
		timeouts.TotalMS = DefaultTimeouts().TotalMS
	}

	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	return DefaultClient().Relay(ctx, RelayRequest{
		Method:      req.Method,
		UpstreamURL: req.URL.String(),
		Headers:     headers,
		Body:        body,
		Timeouts:    timeouts,
	})
}