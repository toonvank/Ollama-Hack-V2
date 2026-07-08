package handlers

import (
	"bufio"
	"bytes"
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

// proxyViaRacerRelay uses the Rust sidecar for single-endpoint egress (Phase 1).
// Returns true when the response was fully delivered to the client.
func (h *OllamaHandler) proxyViaRacerRelay(
	c *gin.Context,
	method, path, endpointURL string,
	deliver racerDeliverContext,
	healthTracker *services.HealthTracker,
) bool {
	upstreamURL := utils.JoinEndpointURL(endpointURL, path)
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if accept := c.GetHeader("Accept"); accept != "" {
		headers["Accept"] = accept
	}

	client := racer.DefaultClient()
	resp, err := client.Relay(c.Request.Context(), racer.RelayRequest{
		Method:      method,
		UpstreamURL: upstreamURL,
		Headers:     headers,
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
		} else if resp.StatusCode >= 500 {
			healthTracker.RecordFailure(endpointURL)
		}
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return true
	}

	healthTracker.RecordSuccess(endpointURL)
	log.Printf("[racer-relay] streaming via sidecar → %s (ttfb header %s)",
		upstreamURL, resp.Header.Get("X-Relay-Ttfb-Ms"))

	h.deliverUpstreamToClient(c, resp, endpointURL, deliver, healthTracker)
	return true
}

func (h *OllamaHandler) deliverUpstreamToClient(
	c *gin.Context,
	resp *http.Response,
	winningEndpoint string,
	deliver racerDeliverContext,
	healthTracker *services.HealthTracker,
) {
	for k, vs := range resp.Header {
		kLower := strings.ToLower(k)
		if kLower == "content-length" || kLower == "transfer-encoding" ||
			kLower == "connection" || kLower == "keep-alive" ||
			strings.HasPrefix(kLower, "x-relay-") {
			continue
		}
		for _, v := range vs {
			c.Header(k, v)
		}
	}

	if deliver.smartRouteHeader != "" {
		c.Header("X-Smart-Route", deliver.smartRouteHeader)
	}
	c.Header("X-Relay", "true")
	if ttfb := resp.Header.Get("X-Relay-Ttfb-Ms"); ttfb != "" {
		c.Header("X-Relay-Ttfb-Ms", ttfb)
	}
	if health := healthTracker.GetHealth(winningEndpoint); health != nil {
		c.Header("X-Endpoint-Health", fmt.Sprintf("%d", health.Score))
	}
	c.Status(resp.StatusCode)

	if deliver.stream {
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		flusher, ok := c.Writer.(http.Flusher)
		reader := bufio.NewReader(resp.Body)

		targetModelStr := []byte(fmt.Sprintf(`"model":"%s"`, deliver.modelRaw))
		replModelStr := []byte(fmt.Sprintf(`"model":"%s"`, deliver.originalModelRequested))

		for {
			line, readErr := reader.ReadBytes('\n')
			if len(line) > 0 {
				if deliver.originalModelRequested != deliver.modelRaw {
					line = bytes.ReplaceAll(line, targetModelStr, replModelStr)
				}
				if deliver.stripThinking {
					line = stripThinkingFromStreamLine(line)
				}
				c.Writer.Write(line)
				if ok {
					flusher.Flush()
				}
			}
			if readErr != nil {
				usageChunk := fmt.Sprintf("\ndata: {\"id\":\"chatcmpl-end\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[],\"usage\":{\"prompt_tokens\":0,\"completion_tokens\":0,\"total_tokens\":0}}\n\n", time.Now().Unix(), deliver.originalModelRequested)
				c.Writer.Write([]byte(usageChunk))
				c.Writer.Write([]byte("data: [DONE]\n\n"))
				if ok {
					flusher.Flush()
				}
				if readErr != io.EOF {
					log.Printf("[racer-relay] stream aborted: %v", readErr)
				}
				break
			}
		}
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