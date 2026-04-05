package handlers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type smartModelCandidate struct {
	urls []string
	name string
	tag  string
}

type smartModelCacheEntry struct {
	candidates []smartModelCandidate // ranked best→worst
	exp        time.Time
}

type OllamaHandler struct {
	db          *database.DB
	fallbacks   map[string]string
	smartRouter *services.SmartRouter

	smartCache sync.Map
}

func NewOllamaHandler(db *database.DB) *OllamaHandler {
	fallbacks := make(map[string]string)
	fallbackStr := os.Getenv("APP_FALLBACK_MODELS")
	if fallbackStr != "" {
		pairs := strings.Split(fallbackStr, ",")
		for _, pair := range pairs {
			kv := strings.Split(pair, "=")
			if len(kv) == 2 {
				fallbacks[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}
	return &OllamaHandler{
		db:          db,
		fallbacks:   fallbacks,
		smartRouter: services.NewSmartRouter(),
	}
}

// bestEndpointForModel returns the URL of the top-ranked endpoint for a model
// (by token_per_second desc), respecting the model's enabled flag.
func (h *OllamaHandler) bestEndpointsForModel(modelName, modelTag string) ([]string, error) {
	type row struct {
		URL string `db:"url"`
	}
	var rows []row
	minTPS := 0.0
	if val := os.Getenv("MIN_TPS_THRESHOLD"); val != "" {
		fmt.Sscanf(val, "%f", &minTPS)
	}

	err := h.db.Select(&rows, `
		SELECT e.url
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		JOIN ai_models m ON m.id = eam.ai_model_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
		WHERE m.name = $1 AND m.tag = $2
		  AND m.enabled = TRUE
		  AND eam.status = 'available'
		  AND e.status = 'available'
		  AND (eh.disabled IS NULL OR eh.disabled = FALSE)
		  AND (eam.token_per_second >= $3 OR eam.token_per_second IS NULL)
		ORDER BY eam.token_per_second DESC NULLS LAST
		LIMIT 5
	`, modelName, modelTag, minTPS)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(rows))
	for _, r := range rows {
		urls = append(urls, r.URL)
	}
	return urls, nil
}

// resolveSmartModel dynamically calculates the best real models for a pseudo-model tag.
// Returns a ranked slice of candidates (best→worst) to enable cascade fallback.
func (h *OllamaHandler) resolveSmartModel(smartTag string) ([]smartModelCandidate, error) {
	// ⚡ Fast path: Check 60-second TTL cache for smart model resolutions
	if val, ok := h.smartCache.Load(smartTag); ok {
		entry := val.(smartModelCacheEntry)
		if time.Now().Before(entry.exp) {
			return entry.candidates, nil
		}
	}

	var heuristic string
	switch smartTag {
	case "fastest":
		heuristic = "1=1"
	case "large":
		heuristic = "(m.name ILIKE '%70b%' OR m.name ILIKE '%104b%' OR m.name ILIKE '%72b%')"
	case "small":
		heuristic = "(m.name ILIKE '%8b%' OR m.name ILIKE '%7b%' OR m.name ILIKE '%3b%' OR m.name ILIKE '%1.5b%')"
	case "coding":
		heuristic = "(m.name ILIKE '%code%' OR m.name ILIKE '%coder%')"
	default:
		heuristic = "1=1"
	}

	// Fetch top 3 distinct (name, tag) candidates ranked by speed
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (m.name, m.tag) m.name, m.tag
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		JOIN ai_models m ON m.id = eam.ai_model_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
		WHERE %s
		  AND m.enabled = TRUE
		  AND eam.status = 'available'
		  AND e.status = 'available'
		  AND (eh.disabled IS NULL OR eh.disabled = FALSE)
		ORDER BY m.name, m.tag, eam.token_per_second DESC NULLS LAST
		LIMIT 3
	`, heuristic)

	type modelRow struct {
		Name string `db:"name"`
		Tag  string `db:"tag"`
	}
	var mRows []modelRow
	err := h.db.Select(&mRows, query)
	if err != nil || len(mRows) == 0 {
		return nil, fmt.Errorf("no models available for smart tag '%s'", smartTag)
	}

	candidates := make([]smartModelCandidate, 0, len(mRows))
	for _, mRow := range mRows {
		urls, err := h.bestEndpointsForModel(mRow.Name, mRow.Tag)
		if err == nil && len(urls) > 0 {
			candidates = append(candidates, smartModelCandidate{
				urls: urls,
				name: mRow.Name,
				tag:  mRow.Tag,
			})
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no reachable endpoints for smart tag '%s'", smartTag)
	}

	// Cache the result for 60 seconds to relieve database load
	h.smartCache.Store(smartTag, smartModelCacheEntry{
		candidates: candidates,
		exp:        time.Now().Add(60 * time.Second),
	})

	return candidates, nil
}

// parseModel splits "name:tag" into name, tag. If no ":" present, tag = "latest".
func parseModel(model string) (string, string) {
	parts := strings.SplitN(model, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], "latest"
}

// Models returns the list of available (enabled) models — OpenAI /v1/models format
func (h *OllamaHandler) Models(c *gin.Context) {
	log.Println("[Models] Handler called!")
	type row struct {
		Name string `db:"name"`
		Tag  string `db:"tag"`
	}
	var rows []row
	err := h.db.Select(&rows, `
		SELECT DISTINCT m.name, m.tag
		FROM ai_models m
		JOIN endpoint_ai_models eam ON eam.ai_model_id = m.id
		WHERE m.enabled = TRUE AND eam.status = 'available'
		ORDER BY m.name, m.tag
	`)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch models")
		return
	}
	timestamp := time.Now().Unix()
	
	// Inject pseudo-models FIRST so they appear at the top
	pseudoModels := []string{"smart:fastest", "smart:large", "smart:small", "smart:coding"}
	data := make([]gin.H, 0, len(rows)+len(pseudoModels))
	for _, pm := range pseudoModels {
		data = append(data, gin.H{
			"id":       pm,
			"object":   "model",
			"owned_by": "system",
			"created":  timestamp,
		})
	}
	
	// Add real models
	for _, r := range rows {
		data = append(data, gin.H{
			"id":       fmt.Sprintf("%s:%s", r.Name, r.Tag),
			"object":   "model",
			"owned_by": "user",
			"created":  timestamp,
		})
	}
	
	log.Printf("[Models] Returning %d models (%d real + %d smart)", len(data), len(rows), len(pseudoModels))

	c.JSON(200, gin.H{"object": "list", "data": data})
}

// Tags returns the list of available models in Ollama /api/tags format
func (h *OllamaHandler) Tags(c *gin.Context) {
	type row struct {
		Name string `db:"name"`
		Tag  string `db:"tag"`
	}
	var rows []row
	err := h.db.Select(&rows, `
		SELECT DISTINCT m.name, m.tag
		FROM ai_models m
		JOIN endpoint_ai_models eam ON eam.ai_model_id = m.id
		WHERE m.enabled = TRUE AND eam.status = 'available'
		ORDER BY m.name, m.tag
	`)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch models")
		return
	}
	
	now := time.Now().Format(time.RFC3339)
	models := make([]gin.H, 0, len(rows)+4)
	for _, r := range rows {
		modelName := fmt.Sprintf("%s:%s", r.Name, r.Tag)
		models = append(models, gin.H{
			"name":        modelName,
			"model":       modelName,
			"modified_at": now,
		})
	}

	// Inject pseudo-models
	pseudoModels := []string{"smart:fastest", "smart:large", "smart:small", "smart:coding"}
	for _, pm := range pseudoModels {
		models = append(models, gin.H{
			"name":        pm,
			"model":       pm,
			"modified_at": now,
		})
	}

	c.JSON(200, gin.H{"models": models})
}

// ChatCompletions proxies POST /v1/chat/completions to the best node
func (h *OllamaHandler) ChatCompletions(c *gin.Context) {
	h.proxyRequest(c, "POST", "/v1/chat/completions")
}

// Completions proxies POST /v1/completions
func (h *OllamaHandler) Completions(c *gin.Context) {
	h.proxyRequest(c, "POST", "/v1/completions")
}

// Generate proxies native Ollama POST /api/generate
func (h *OllamaHandler) Generate(c *gin.Context) {
	h.proxyRequest(c, "POST", "/api/generate")
}

// Chat proxies native Ollama POST /api/chat
func (h *OllamaHandler) Chat(c *gin.Context) {
	h.proxyRequest(c, "POST", "/api/chat")
}

// proxyRequest reads the model from the request body, finds the best endpoint,
// and streams or forwards the response.
func (h *OllamaHandler) proxyRequest(c *gin.Context, method, path string) {
	utils.TotalRequests.Add(1)
	utils.ActiveRequests.Add(1)
	defer utils.ActiveRequests.Add(-1)

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.BadRequest(c, "Failed to read request body")
		return
	}

	// Parse model from body
	var bodyMap map[string]interface{}
	if err := json.Unmarshal(rawBody, &bodyMap); err != nil {
		utils.BadRequest(c, "Invalid JSON body")
		return
	}

	modelRaw, _ := bodyMap["model"].(string)
	if modelRaw == "" {
		utils.BadRequest(c, "Field 'model' is required")
		return
	}
	
	originalModelRequested := modelRaw

	// 🧠 NEVER-SLEEP INJECTOR: Eliminate Cold-Starts
	// Inject infinite keep_alive if the user hasn't explicitly set one. This securely
	// locks the model into VRAM immediately after use.
	if _, ok := bodyMap["keep_alive"]; !ok {
		bodyMap["keep_alive"] = -1
		rawBody, _ = json.Marshal(bodyMap)
	}

	// 🧠 SMART ROUTING: Classify prompt and route to optimal model
	var smartRouteHeader string
	if h.smartRouter.IsEnabled() {
		// Extract messages for classification
		if messages, ok := bodyMap["messages"].([]interface{}); ok && len(messages) > 0 {
			if result := h.smartRouter.ClassifyMessages(messages); result != nil && result.PreferModel != "" {
				// Check if the preferred model is available
				preferName, preferTag := parseModel(result.PreferModel)
				preferEndpoints, preferErr := h.bestEndpointsForModel(preferName, preferTag)
				
				if preferErr == nil && len(preferEndpoints) > 0 {
					modelRaw = result.PreferModel
					smartRouteHeader = services.FormatRouteHeader(result.Category, result.PreferModel)
					
					log.Printf("[smart-router] Routing '%s' → '%s' (category: %s, confidence: %.2f)",
						originalModelRequested, result.PreferModel, result.Category, result.Confidence)
					
					// Update the body with the new model
					bodyMap["model"] = modelRaw
					rawBody, _ = json.Marshal(bodyMap)
				} else {
					log.Printf("[smart-router] Preferred model '%s' not available, keeping original '%s'",
						result.PreferModel, originalModelRequested)
				}
			}
		}
	}

	name, tag := parseModel(modelRaw)
	var endpoints []string
	// For smart models, we keep the full candidate list for cascade fallback
	var smartCandidates []smartModelCandidate

	if name == "smart" {
		smartCandidates, err = h.resolveSmartModel(tag)
		if err == nil && len(smartCandidates) > 0 {
			// Start with the best candidate
			best := smartCandidates[0]
			name, tag = best.name, best.tag
			endpoints = best.urls
			log.Printf("[smart-model] Resolved '%s' → '%s:%s' (%d fallback candidates)",
				originalModelRequested, name, tag, len(smartCandidates)-1)
			modelRaw = fmt.Sprintf("%s:%s", name, tag)
			bodyMap["model"] = modelRaw
			rawBody, _ = json.Marshal(bodyMap)
			smartRouteHeader = services.FormatRouteHeader("smart", modelRaw)
		}
	} else {
		endpoints, err = h.bestEndpointsForModel(name, tag)
	}

	// Attempt blazing fast in-memory fallback route if unavailable
	if err != nil || len(endpoints) == 0 {
		lookupKey := fmt.Sprintf("%s:%s", name, tag)
		if fallbackRaw, ok := h.fallbacks[lookupKey]; ok {
			log.Printf("[proxy] Model %s unavailable, applying fallback to %s", lookupKey, fallbackRaw)

			name, tag = parseModel(fallbackRaw)
			endpoints, err = h.bestEndpointsForModel(name, tag)
			if err == nil && len(endpoints) > 0 {
				c.Header("X-Model-Fallback", fallbackRaw)

				// Rewrite the model name in the payload to match the fallback
				bodyMap["model"] = fallbackRaw
				rawBody, _ = json.Marshal(bodyMap)
			}
		}
	}

	if err != nil || len(endpoints) == 0 {
		c.JSON(404, gin.H{"error": fmt.Sprintf("No available endpoint found for model %s:%s", name, tag)})
		return
	}

	// 🏥 HEALTH FILTER: Remove temporarily disabled endpoints
	healthTracker := services.GetHealthTracker()
	endpoints = healthTracker.FilterHealthyEndpoints(endpoints)
	if len(endpoints) == 0 {
		c.JSON(503, gin.H{"error": fmt.Sprintf("All endpoints for model %s:%s are temporarily disabled due to failures", name, tag)})
		return
	}

	// 🔪 MAP-REDUCE INTERCEPTOR: The Document Cracker
	// Chop context into simultaneous parallel multi-node chunks unconditionally
	if mr, ok := bodyMap["x_map_reduce"].(bool); ok && mr {
		h.mapReduceProxy(c, method, path, bodyMap, endpoints)
		return
	}

	stream, _ := bodyMap["stream"].(bool)
	var cacheKey string
	var promptEmbedding []float64 // For semantic cache

	// Attempt Cache Hit for EXACT non-streaming requests
	// We skip caching for massive payloads (e.g. >500KB base64 images) to preserve memory
	if !stream && len(rawBody) < 500*1024 {
		cacheKey = GenerateCacheKey(bodyMap)
		if cachedData, cachedHeaders, ok := GlobalCache.Get(cacheKey); ok {
			utils.CacheHits.Add(1)
			log.Printf("[proxy] Cache HIT for key %s", cacheKey)
			for k, vs := range cachedHeaders {
				for _, v := range vs {
					c.Header(k, v)
				}
			}
			c.Header("X-Cache-Hit", "true")
			c.Data(200, "application/json", cachedData)
			return
		}

		// 🧠 SEMANTIC CACHE: Try similarity-based cache lookup
		if services.GlobalSemanticCache.IsEnabled() {
			promptText := services.ExtractPromptFromRequest(bodyMap)
			if promptText != "" {
				embedding, err := services.GlobalSemanticCache.GetEmbedding(promptText)
				if err != nil {
					log.Printf("[semantic-cache] Failed to get embedding: %v", err)
				} else {
					promptEmbedding = embedding // Save for potential storing later
					if result, found := services.GlobalSemanticCache.Search(embedding); found {
						utils.CacheHits.Add(1)
						log.Printf("[semantic-cache] HIT with similarity %.4f", result.Similarity)
						for k, vs := range result.Headers {
							for _, v := range vs {
								c.Header(k, v)
							}
						}
						c.Header("X-Semantic-Cache-Hit", "true")
						c.Header("X-Semantic-Cache-Similarity", fmt.Sprintf("%.4f", result.Similarity))
						c.Data(200, "application/json", result.Data)
						return
					}
				}
			}
		}
	}

	// 🚀 ZERO-LATENCY RACER MODE 🚀
	// Launch simultaneous requests to all available endpoints. The first one to answer wins.

	type raceResult struct {
		resp          *http.Response
		err           error
		endpointURL   string
		index         int
		rateLimited   bool   // true if the error was a 429 — don't health-penalize
		isClientError bool   // true if 400 <= status < 500 — bad prompts shouldn't penalize node health
		failStatus    int    // Forward upstream error status back to client if race fails
		failBody      []byte // Forward upstream error body back to client if race fails
	}

	resultCh := make(chan raceResult, len(endpoints))
	cancels := make([]context.CancelFunc, len(endpoints))

	for i, endpointURL := range endpoints {
		ctx, reqCancel := context.WithCancel(c.Request.Context())
		cancels[i] = reqCancel

		go func(index int, url string, reqCtx context.Context) {
			target := url + path
			req, err := http.NewRequestWithContext(reqCtx, method, target, bytes.NewReader(rawBody))
			if err != nil {
				resultCh <- raceResult{err: err, endpointURL: url, index: index}
				return
			}
			req.Header.Set("Content-Type", "application/json")

			// Forward relevant headers (skip auth/host)
			for k, vs := range c.Request.Header {
				k = strings.ToLower(k)
				if k == "authorization" || k == "host" || k == "content-length" {
					continue
				}
				for _, v := range vs {
					req.Header.Add(k, v)
				}
			}

			client := &http.Client{Timeout: 120 * time.Second}
			resp, err := client.Do(req)

			if err != nil {
				resultCh <- raceResult{err: err, endpointURL: url, index: index}
				return
			}

			// Non-200 responses are failures in the race
			if resp.StatusCode >= 400 {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				if resp.StatusCode == 429 {
					// Rate-limited: record separately so the endpoint isn't health-penalized
					resultCh <- raceResult{err: fmt.Errorf("rate-limited (429)"), endpointURL: url, index: index, rateLimited: true, failStatus: resp.StatusCode, failBody: bodyBytes}
				} else if resp.StatusCode < 500 {
					// Client error (400, 401, etc) - payload is bad, context too long, etc.
					resultCh <- raceResult{err: fmt.Errorf("status %d", resp.StatusCode), endpointURL: url, index: index, isClientError: true, failStatus: resp.StatusCode, failBody: bodyBytes}
				} else {
					// Server error (500, 502, 504)
					resultCh <- raceResult{err: fmt.Errorf("status %d", resp.StatusCode), endpointURL: url, index: index, failStatus: resp.StatusCode, failBody: bodyBytes}
				}
				return
			}

			// Validate Content-Type
			contentType := strings.ToLower(resp.Header.Get("Content-Type"))
			if strings.Contains(contentType, "text/html") {
				resp.Body.Close()
				resultCh <- raceResult{err: fmt.Errorf("rejected honeypot: html response"), endpointURL: url, index: index}
				return
			}

			// Enforce streaming response if the client requested it to prevent Open WebUI parser crashes
			streamReq, _ := bodyMap["stream"].(bool)
			if streamReq && !strings.Contains(contentType, "event-stream") && !strings.Contains(contentType, "ndjson") {
				resp.Body.Close()
				resultCh <- raceResult{err: fmt.Errorf("rejected node: expected stream but got %s", contentType), endpointURL: url, index: index}
				return
			}

			// Verify actual data arrives (Time-To-First-Byte) to filter out fake 200 OK honeypots
			firstChunk := make([]byte, 512)
			n, readErr := resp.Body.Read(firstChunk)
			if n == 0 && readErr != nil {
				resp.Body.Close()
				resultCh <- raceResult{err: fmt.Errorf("empty response body or immediate EOF"), endpointURL: url, index: index}
				return
			}

			// Sniff the payload to verify it's actual AI JSON/SSE and not an HTML captive portal / honeypot
			sniffStr := strings.TrimSpace(string(firstChunk[:n]))
			if len(sniffStr) > 0 {
				firstChar := sniffStr[0]
				if firstChar != '{' && firstChar != '[' && firstChar != 'd' && firstChar != '"' {
					resp.Body.Close()
					resultCh <- raceResult{err: fmt.Errorf("rejected honeypot: invalid payload start %q", firstChar), endpointURL: url, index: index}
					return
				}
				
				// Aggressively reject JSON error payloads wrapped in 200 OK
				if strings.HasPrefix(sniffStr, `{"error"`) || strings.HasPrefix(sniffStr, `{"message"`) {
					resp.Body.Close()
					resultCh <- raceResult{err: fmt.Errorf("rejected node: returned 200 OK error JSON payload"), endpointURL: url, index: index}
					return
				}
				
				// Validate streaming integrity: If stream requested, it MUST start with "data:"
				// Also catch upstream API errors that are embedded inside the initial SSE chunk (very common in LiteLLM/Ollama proxies)
				if streamReq {
					if !strings.HasPrefix(sniffStr, "data:") {
						resp.Body.Close()
						resultCh <- raceResult{err: fmt.Errorf("rejected node: node ignored stream parameter and returned non-chunked response"), endpointURL: url, index: index}
						return
					}
					if strings.Contains(sniffStr, `"error"`) {
						resp.Body.Close()
						resultCh <- raceResult{err: fmt.Errorf("rejected node: upstream model threw an API error hidden in the SSE stream"), endpointURL: url, index: index}
						return
					}
				}
			}

			// Reconstruct body with the read chunk
			resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(firstChunk[:n]), resp.Body))

			resultCh <- raceResult{resp: resp, endpointURL: url, index: index}
		}(i, endpointURL, ctx)
	}

	var winningResp *http.Response
	var winningEndpoint string
	failures := 0
	
	// Keep track of the most interesting upstream error to return if the race fails entirely
	var lastFailStatus int
	var lastFailBody []byte

	for i := 0; i < len(endpoints); i++ {
		res := <-resultCh

		if res.err == nil && winningResp == nil {
			// WE HAVE A WINNER!
			winningResp = res.resp
			winningEndpoint = res.endpointURL
			log.Printf("[proxy-race] 🏁 WINNER: %s", res.endpointURL)

			// Record success for the winning endpoint
			healthTracker.RecordSuccess(res.endpointURL)

			// INSTANTLY send Cancellation Signals dropped to all slower GPU nodes to free their VRAM
			for j, cancelFunc := range cancels {
				if j != res.index {
					cancelFunc()
				}
			}
		} else if res.resp != nil {
			// This node finished processing, but it's a loser (or we already have a winner). Clean it up.
			res.resp.Body.Close()
		}

		if res.err != nil {
			failures++
			utils.FailedRequests.Add(1)
			log.Printf("[proxy-race] endpoint %s failed: %v", res.endpointURL, res.err)
			
			// Save the most recent upstream error to bubble back if the proxy fails
			if res.failStatus > 0 {
				lastFailStatus = res.failStatus
				lastFailBody = res.failBody
			}

			// 429 rate-limits don't penalize health score — endpoint is fine, just throttled
			if res.rateLimited {
				healthTracker.RecordRateLimit(res.endpointURL)
			} else if !res.isClientError {
				// Only penalize 5xx server errors or hard connection timeouts
				healthTracker.RecordFailure(res.endpointURL)
			}
		}
	}

	// Always ensure the winning context eventually cancels when the entire proxy request finishes
	defer func() {
		for _, cancelFunc := range cancels {
			cancelFunc()
		}
	}()

	if winningResp == nil {
		// 🔄 CASCADE FALLBACK: If this was a smart model and the best candidate failed,
		// try the next ranked candidates one by one before giving up.
		if len(smartCandidates) > 1 {
			for _, fallback := range smartCandidates[1:] {
				fallbackURLs := healthTracker.FilterHealthyEndpoints(fallback.urls)
				if len(fallbackURLs) == 0 {
					continue
				}
				log.Printf("[smart-cascade] Primary failed, trying fallback model '%s:%s'",
					fallback.name, fallback.tag)

				fallbackModel := fmt.Sprintf("%s:%s", fallback.name, fallback.tag)
				bodyMap["model"] = fallbackModel
				cascadeBody, _ := json.Marshal(bodyMap)

				// Simple single-endpoint attempt for cascade (no nested racer)
				for _, cascadeURL := range fallbackURLs {
					creq, cerr := http.NewRequestWithContext(c.Request.Context(), method,
						cascadeURL+path, bytes.NewReader(cascadeBody))
					if cerr != nil {
						continue
					}
					creq.Header.Set("Content-Type", "application/json")
					client := &http.Client{Timeout: 120 * time.Second}
					cresp, cerr := client.Do(creq)
					if cerr != nil || cresp.StatusCode >= 400 {
						if cresp != nil {
							var b []byte
							if cresp.Body != nil {
								b, _ = io.ReadAll(cresp.Body)
								cresp.Body.Close()
							}
							lastFailStatus = cresp.StatusCode
							lastFailBody = b
						}
						continue
					}
					// Cascade winner found!
					log.Printf("[smart-cascade] 🏁 Cascade winner: %s via %s", fallbackModel, cascadeURL)
					c.Header("X-Smart-Cascade", fmt.Sprintf("%s→%s", modelRaw, fallbackModel))
					// Rewrite body for model masking (show original requested model to client)
					winningResp = cresp
					modelRaw = fallbackModel
					break
				}
				if winningResp != nil {
					break
				}
			}
		}
		if winningResp == nil {
			if lastFailStatus >= 400 {
				// We have a direct upstream error (like 400 Bad Request) that we should bubble back
				var errContentType = "application/json"
				c.Data(lastFailStatus, errContentType, lastFailBody)
			} else {
				c.JSON(502, gin.H{"error": "All endpoints failed the race or didn't respond"})
			}
			return
		}
	}

	resp := winningResp

	// Copy response headers but filter out hop-by-hop protocols
	for k, vs := range resp.Header {
		kLower := strings.ToLower(k)
		if kLower == "content-length" || kLower == "transfer-encoding" || kLower == "connection" || kLower == "keep-alive" {
			continue
		}
		for _, v := range vs {
			c.Header(k, v)
		}
	}
	// Add smart routing header if model was rerouted
	if smartRouteHeader != "" {
		c.Header("X-Smart-Route", smartRouteHeader)
	}
	// Add endpoint health info header
	if health := healthTracker.GetHealth(winningEndpoint); health != nil {
		c.Header("X-Endpoint-Health", fmt.Sprintf("%d", health.Score))
	}
	c.Status(resp.StatusCode)

	if stream {
		// Anti-buffering headers to guarantee Nginx/aiohttp stream it live instead of waiting for EOF
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")

		// Streaming: flush chunks as they arrive safely line-by-line
		flusher, ok := c.Writer.(http.Flusher)
		reader := bufio.NewReader(resp.Body)
		
		targetModelStr := []byte(fmt.Sprintf(`"model":"%s"`, modelRaw))
		replModelStr := []byte(fmt.Sprintf(`"model":"%s"`, originalModelRequested))
		
		for {
			line, readErr := reader.ReadBytes('\n')
			if len(line) > 0 {
				// Rewrite the output model JSON cleanly at logical data boundaries
				if originalModelRequested != modelRaw {
					line = bytes.ReplaceAll(line, targetModelStr, replModelStr)
				}
				
				c.Writer.Write(line)
				if ok {
					flusher.Flush()
				}
			}
			if readErr != nil {
				// Inject a terminal Usage object chunk (OpenAI standard) to protect Litellm from crashing if 'stream_options.include_usage: true' was passed
				usageChunk := fmt.Sprintf("\ndata: {\"id\":\"chatcmpl-end\",\"object\":\"chat.completion.chunk\",\"created\":%d,\"model\":\"%s\",\"choices\":[],\"usage\":{\"prompt_tokens\":0,\"completion_tokens\":0,\"total_tokens\":0}}\n\n", time.Now().Unix(), originalModelRequested)
				c.Writer.Write([]byte(usageChunk))
				
				// Inject a guaranteed DONE frame if the stream ends or is aborted abruptly,
				// which prevents python/aiohttp ClientPayloadError crashes in Open WebUI
				c.Writer.Write([]byte("data: [DONE]\n\n"))
				if ok {
					flusher.Flush()
				}
				if readErr != io.EOF {
					log.Printf("[proxy] Stream aborted early: %v", readErr)
				}
				break
			}
		}
		resp.Body.Close()
	} else {
		// Non-streaming: copy full body and cache it
		respBytes, bodyErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if originalModelRequested != modelRaw {
			targetModelStr := []byte(fmt.Sprintf(`"model":"%s"`, modelRaw))
			replModelStr := []byte(fmt.Sprintf(`"model":"%s"`, originalModelRequested))
			respBytes = bytes.ReplaceAll(respBytes, targetModelStr, replModelStr)
		}
		
		if bodyErr == nil {
			// Cache successful responses for 10 minutes
			if resp.StatusCode == 200 && cacheKey != "" {
				GlobalCache.Set(cacheKey, respBytes, resp.Header, 10*time.Minute)
			}
			// 🧠 SEMANTIC CACHE: Store with embedding for similarity lookups
			if resp.StatusCode == 200 && promptEmbedding != nil && services.GlobalSemanticCache.IsEnabled() {
				promptText := services.ExtractPromptFromRequest(bodyMap)
				services.GlobalSemanticCache.Store(promptEmbedding, respBytes, resp.Header, promptText, 10*time.Minute)
				log.Printf("[semantic-cache] Stored new entry, cache size: %d", services.GlobalSemanticCache.Size())
			}
			c.Writer.Write(respBytes)
		}
	}
	// Finished executing winner
}

// 🔪 THE DOCUMENT CRACKER: Auto-splits massive documents and blasts them to multiple GPUs 
func (h *OllamaHandler) mapReduceProxy(c *gin.Context, method, path string, bodyMap map[string]interface{}, endpoints []string) {
	startTime := time.Now()
	
	messages, ok := bodyMap["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		c.JSON(400, gin.H{"error": "messages array required for map-reduce"})
		return
	}

	lastMsg, ok := messages[len(messages)-1].(map[string]interface{})
	if !ok {
		c.JSON(400, gin.H{"error": "invalid messages format"})
		return
	}
	content, ok := lastMsg["content"].(string)
	if !ok {
		c.JSON(400, gin.H{"error": "last message must have string content"})
		return
	}

	// Calculate splits based on available physical nodes
	chunkCount := len(endpoints)
	if chunkCount > 4 {
		chunkCount = 4
	}
	
	chunkSize := len(content) / chunkCount
	var chunks []string
	if chunkSize < 50 {
		chunks = []string{content}
	} else {
		for i := 0; i < chunkCount; i++ {
			start := i * chunkSize
			end := start + chunkSize
			if i == chunkCount-1 {
				end = len(content)
			}
			chunks = append(chunks, content[start:end])
		}
	}

	type mrResult struct {
		index int
		text  string
		err   error
	}
	
	resultCh := make(chan mrResult, len(chunks))
	streamRaw, _ := bodyMap["stream"].(bool)
	bodyMap["stream"] = false // Map-Reduce runs blocking-sync natively

	for i, chunkText := range chunks {
		// Launch N simultaneous GPU Map jobs!
		go func(idx int, text string, endpointURL string) {
			bodyClone := make(map[string]interface{})
			for k, v := range bodyMap { bodyClone[k] = v }
			
			msgsClone := make([]interface{}, len(messages)-1)
			copy(msgsClone, messages[:len(messages)-1])
			
			newLastMsg := map[string]interface{}{}
			for k, v := range lastMsg { newLastMsg[k] = v }
			newLastMsg["content"] = "[MAP-REDUCE SUB-CHUNK, SUMMARIZE THIS PORTION EXACTLY]:\n\n" + text
			
			msgsClone = append(msgsClone, newLastMsg)
			bodyClone["messages"] = msgsClone
			
			reqBytes, _ := json.Marshal(bodyClone)
			target := endpointURL + path

			req, err := http.NewRequest("POST", target, bytes.NewReader(reqBytes))
			if err != nil {
				resultCh <- mrResult{index: idx, err: err}
				return
			}
			req.Header.Set("Content-Type", "application/json")
			for k, vs := range c.Request.Header {
				k = strings.ToLower(k)
				if k == "authorization" || k == "host" || k == "content-length" {
					continue
				}
				for _, v := range vs { req.Header.Add(k, v) }
			}

			client := &http.Client{Timeout: 300 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				resultCh <- mrResult{index: idx, err: err}
				return
			}
			defer resp.Body.Close()
			
			respBytes, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 200 {
				resultCh <- mrResult{index: idx, err: fmt.Errorf("node failed with status %d", resp.StatusCode)}
				return
			}
			
			// Unmarshal Ollama's OpenAI-compatible JSON representation
			var oaiResp struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
					} `json:"message"`
				} `json:"choices"`
			}
			json.Unmarshal(respBytes, &oaiResp)
			
			var outText string
			if len(oaiResp.Choices) > 0 {
				outText = oaiResp.Choices[0].Message.Content
			} else {
				outText = string(respBytes) // fallback to raw
			}
			
			resultCh <- mrResult{index: idx, text: outText}
		}(i, chunkText, endpoints[i % len(endpoints)])
	}
	
	results := make([]string, len(chunks))
	errs := 0
	
	// Wait for all GPUs to finish and Reduce the output
	for i := 0; i < len(chunks); i++ {
		res := <-resultCh
		if res.err != nil {
			errs++
			log.Printf("[map-reduce] chunk %d crashed: %v", res.index, res.err)
		} else {
			results[res.index] = res.text
		}
	}
	
	if errs == len(chunks) {
		c.JSON(500, gin.H{"error": "Map-Reduce failed completely across all cluster nodes."})
		return
	}
	
	finalText := strings.Join(results, "\n\n---\n\n")
	log.Printf("[map-reduce] Crushed %d chunks in %v", len(chunks), time.Since(startTime))
	
	if streamRaw {
		// Emit fake SSE stream to satisfy streaming clients seamlessly!
		c.Header("Content-Type", "text/event-stream")
		chunk := gin.H{
			"id": "chatcmpl-mapreduce", "object": "chat.completion.chunk",
			"created": time.Now().Unix(), "model": bodyMap["model"],
			"choices": []gin.H{ { "index": 0, "delta": gin.H{"content": finalText} } },
		}
		b, _ := json.Marshal(chunk)
		c.Writer.Write([]byte("data: " + string(b) + "\n\n"))
		c.Writer.Write([]byte("data: [DONE]\n\n"))
	} else {
		ans := gin.H{
			"id": "chatcmpl-mapreduce", "object": "chat.completion",
			"created": time.Now().Unix(), "model": bodyMap["model"],
			"choices": []gin.H{ { "index": 0, "message": gin.H{"role": "assistant", "content": finalText} } },
		}
		c.JSON(200, ans)
	}
}
