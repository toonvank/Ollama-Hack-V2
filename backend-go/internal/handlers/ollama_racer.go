package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/racer"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type racerDeliverContext struct {
	stream                 bool
	rawBody                []byte
	bodyMap                map[string]interface{}
	modelRaw               string
	originalModelRequested string
	stripThinking          bool
	smartRouteHeader       string
	cacheKey               string
	promptEmbedding        []float64
}

func racerProxyHeaders(c *gin.Context) map[string]string {
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if accept := c.GetHeader("Accept"); accept != "" {
		headers["Accept"] = accept
	}
	for k, vs := range c.Request.Header {
		kl := strings.ToLower(k)
		if kl == "authorization" || kl == "host" || kl == "content-length" {
			continue
		}
		if len(vs) > 0 {
			headers[k] = vs[0]
		}
	}
	return headers
}

func applyRaceFailures(healthTracker *services.HealthTracker, resp *http.Response) {
	failures, err := racer.ParseRaceFailures(resp.Header.Get("X-Race-Failures-B64"))
	if err != nil || len(failures) == 0 {
		return
	}
	for _, f := range failures {
		if f.QuotaExceeded {
			healthTracker.RecordQuotaExceeded(f.Endpoint)
		} else if f.RateLimited {
			healthTracker.RecordRateLimit(f.Endpoint)
		} else if f.ClientError {
			continue
		} else if isModelNotFoundError(f.Status, nil) {  // best effort; sidecar may not send body
			log.Printf("[health] skipping racer sidecar penalty for model-not-found on %s", f.Endpoint)
			continue
		} else if f.Status >= 500 || f.Status == 0 {
			healthTracker.RecordFailure(f.Endpoint)
		}
	}
}

// proxyViaRacerRace uses the Rust sidecar for parallel endpoint racing (Phase 2).
func (h *OllamaHandler) proxyViaRacerRace(
	c *gin.Context,
	method, path string,
	endpoints []string,
	deliver racerDeliverContext,
	healthTracker *services.HealthTracker,
) bool {
	client := racer.DefaultClient()
	resp, err := client.Race(c.Request.Context(), racer.RaceRequest{
		Method:           method,
		Path:             path,
		Endpoints:        endpoints,
		Headers:          racerProxyHeaders(c),
		Body:             deliver.rawBody,
		Timeouts:         racer.DefaultTimeouts(),
		Stream:           deliver.stream,
		CancelOnFirstWin: true,
	})
	if err != nil {
		log.Printf("[racer-race] request failed: %v", err)
		return false
	}
	defer resp.Body.Close()

	applyRaceFailures(healthTracker, resp)

	if resp.StatusCode == http.StatusBadGateway {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		var payload map[string]interface{}
		if json.Unmarshal(body, &payload) == nil {
			c.JSON(502, payload)
		} else {
			c.JSON(502, gin.H{"error": "All endpoints failed the race or didn't respond"})
		}
		utils.FailedRequests.Add(1)
		return true
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		utils.FailedRequests.Add(1)
		winner := resp.Header.Get("X-Race-Winner")
		if winner != "" {
			if resp.StatusCode == 429 {
				healthTracker.RecordRateLimit(winner)
			} else if isQuotaExceededError(resp.StatusCode, body) {
				healthTracker.RecordQuotaExceeded(winner)
			} else if resp.StatusCode >= 500 {
				healthTracker.RecordFailure(winner)
			}
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return true
	}

	winner := resp.Header.Get("X-Race-Winner")
	if winner == "" {
		log.Printf("[racer-race] missing X-Race-Winner header")
		return false
	}

	healthTracker.RecordSuccess(winner)
	log.Printf("[racer-race] 🏁 WINNER: %s (ttfb %sms, cancelled %s)",
		winner, resp.Header.Get("X-Race-Ttfb-Ms"), resp.Header.Get("X-Race-Losers-Cancelled"))

	h.deliverUpstreamToClient(c, resp, winner, deliver, healthTracker, true)
	return true
}

// proxyViaRacerRelay uses the Rust sidecar for single-endpoint egress (Phase 1).
// Returns true when the response was fully delivered to the client.
func (h *OllamaHandler) proxyViaRacerRelay(
	c *gin.Context,
	method, path, endpointURL string,
	deliver racerDeliverContext,
	healthTracker *services.HealthTracker,
) bool {
	upstreamURL := utils.JoinEndpointURL(endpointURL, path)

	client := racer.DefaultClient()
	resp, err := client.Relay(c.Request.Context(), racer.RelayRequest{
		Method:      method,
		UpstreamURL: upstreamURL,
		Headers:     racerProxyHeaders(c),
		Body:        deliver.rawBody,
		Timeouts:    racer.DefaultTimeouts(),
	})
	if err != nil {
		log.Printf("[racer-relay] request failed for %s: %v", upstreamURL, err)
		utils.FailedRequests.Add(1)
		healthTracker.RecordFailure(endpointURL)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
		log.Printf("[racer-relay] upstream %s returned %d", upstreamURL, resp.StatusCode)
		utils.FailedRequests.Add(1)
		if resp.StatusCode == 429 {
			healthTracker.RecordRateLimit(endpointURL)
		} else if isQuotaExceededError(resp.StatusCode, body) {
			healthTracker.RecordQuotaExceeded(endpointURL)
		} else if isModelNotFoundError(resp.StatusCode, body) {
			log.Printf("[health] skipping relay penalty for model-not-found on %s", endpointURL)
		} else if resp.StatusCode >= 500 {
			healthTracker.RecordFailure(endpointURL)
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return true
	}

	healthTracker.RecordSuccess(endpointURL)
	log.Printf("[racer-relay] streaming via sidecar → %s (ttfb header %s)",
		upstreamURL, resp.Header.Get("X-Relay-Ttfb-Ms"))

	h.deliverUpstreamToClient(c, resp, endpointURL, deliver, healthTracker, false)
	return true
}

func (h *OllamaHandler) deliverUpstreamToClient(
	c *gin.Context,
	resp *http.Response,
	winningEndpoint string,
	deliver racerDeliverContext,
	healthTracker *services.HealthTracker,
	fromRace bool,
) {
	for k, vs := range resp.Header {
		kLower := strings.ToLower(k)
		if kLower == "content-length" || kLower == "transfer-encoding" ||
			kLower == "connection" || kLower == "keep-alive" ||
			strings.HasPrefix(kLower, "x-relay-") ||
			strings.HasPrefix(kLower, "x-race-") {
			continue
		}
		for _, v := range vs {
			c.Header(k, v)
		}
	}

	if deliver.smartRouteHeader != "" {
		c.Header("X-Smart-Route", deliver.smartRouteHeader)
	}
	if fromRace {
		c.Header("X-Race", "true")
		if ttfb := resp.Header.Get("X-Race-Ttfb-Ms"); ttfb != "" {
			c.Header("X-Race-Ttfb-Ms", ttfb)
		}
	} else {
		c.Header("X-Relay", "true")
		if ttfb := resp.Header.Get("X-Relay-Ttfb-Ms"); ttfb != "" {
			c.Header("X-Relay-Ttfb-Ms", ttfb)
		}
	}
	if health := healthTracker.GetHealth(winningEndpoint); health != nil {
		c.Header("X-Endpoint-Health", fmt.Sprintf("%d", health.Score))
	}
	c.Status(resp.StatusCode)

	if deliver.stream {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		writeNormalizedSSE(c, resp.Body, deliver.modelRaw, deliver.originalModelRequested, deliver.stripThinking)
		return
	}

	respBytes, bodyErr := io.ReadAll(resp.Body)
	if deliver.originalModelRequested != deliver.modelRaw {
		targetModelStr := []byte(fmt.Sprintf(`"model":"%s"`, deliver.modelRaw))
		replModelStr := []byte(fmt.Sprintf(`"model":"%s"`, deliver.originalModelRequested))
		respBytes = bytes.ReplaceAll(respBytes, targetModelStr, replModelStr)
	}
	if deliver.stripThinking {
		respBytes = stripThinkingFromChatResponse(respBytes)
	} else {
		respBytes = normalizeChatResponseForClients(respBytes)
		var payload map[string]interface{}
		if err := json.Unmarshal(respBytes, &payload); err == nil {
			if normalizeReasoningInPayload(payload) {
				if b, err := json.Marshal(payload); err == nil {
					respBytes = b
				}
			}
		}
	}

	if bodyErr == nil {
		if resp.StatusCode == 200 && deliver.cacheKey != "" {
			GlobalCache.Set(deliver.cacheKey, respBytes, resp.Header, 10*time.Minute)
		}
		if resp.StatusCode == 200 && deliver.promptEmbedding != nil && services.GlobalSemanticCache.IsEnabled() {
			promptText := services.ExtractPromptFromRequest(deliver.bodyMap)
			services.GlobalSemanticCache.Store(deliver.promptEmbedding, respBytes, resp.Header, promptText, 10*time.Minute)
		}
		c.Writer.Write(respBytes)
	}
}