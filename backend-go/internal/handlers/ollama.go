package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type OllamaHandler struct {
	db          *database.DB
	fallbacks   map[string]string
	smartRouter *services.SmartRouter
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
	err := h.db.Select(&rows, `
		SELECT e.url
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		JOIN ai_models m ON m.id = eam.ai_model_id
		WHERE m.name = $1 AND m.tag = $2
		  AND m.enabled = TRUE
		  AND eam.status = 'available'
		  AND e.status = 'available'
		ORDER BY eam.token_per_second DESC NULLS LAST
		LIMIT 10
	`, modelName, modelTag)
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(rows))
	for _, r := range rows {
		urls = append(urls, r.URL)
	}
	return urls, nil
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
	data := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		data = append(data, gin.H{
			"id":       fmt.Sprintf("%s:%s", r.Name, r.Tag),
			"object":   "model",
			"owned_by": "user",
			"created":  timestamp,
		})
	}
	c.JSON(200, gin.H{"object": "list", "data": data})
}

// ChatCompletions proxies POST /v1/chat/completions to the best Ollama endpoint
func (h *OllamaHandler) ChatCompletions(c *gin.Context) {
	h.proxyRequest(c, "POST", "/v1/chat/completions")
}

// Completions proxies POST /v1/completions
func (h *OllamaHandler) Completions(c *gin.Context) {
	h.proxyRequest(c, "POST", "/v1/completions")
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
					originalModel := modelRaw
					modelRaw = result.PreferModel
					smartRouteHeader = services.FormatRouteHeader(result.Category, result.PreferModel)
					
					log.Printf("[smart-router] Routing '%s' → '%s' (category: %s, confidence: %.2f)",
						originalModel, result.PreferModel, result.Category, result.Confidence)
					
					// Update the body with the new model
					bodyMap["model"] = modelRaw
					rawBody, _ = json.Marshal(bodyMap)
				} else {
					log.Printf("[smart-router] Preferred model '%s' not available, keeping original '%s'",
						result.PreferModel, modelRaw)
				}
			}
		}
	}

	name, tag := parseModel(modelRaw)
	endpoints, err := h.bestEndpointsForModel(name, tag)

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
		resp        *http.Response
		err         error
		endpointURL string
		index       int
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
				resp.Body.Close()
				resultCh <- raceResult{err: fmt.Errorf("status %d", resp.StatusCode), endpointURL: url, index: index}
				return
			}

			resultCh <- raceResult{resp: resp, endpointURL: url, index: index}
		}(i, endpointURL, ctx)
	}

	var winningResp *http.Response
	var winningEndpoint string
	failures := 0

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
			// Record failure for health tracking
			healthTracker.RecordFailure(res.endpointURL)
		}
	}

	// Always ensure the winning context eventually cancels when the entire proxy request finishes
	defer func() {
		for _, cancelFunc := range cancels {
			cancelFunc()
		}
	}()

	if winningResp == nil {
		c.JSON(502, gin.H{"error": "All endpoints failed the race or didn't respond"})
		return
	}

	resp := winningResp

	// Copy response headers
	for k, vs := range resp.Header {
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
		// Streaming: flush chunks as they arrive
		flusher, ok := c.Writer.(http.Flusher)
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				c.Writer.Write(buf[:n])
				if ok {
					flusher.Flush()
				}
			}
			if readErr != nil {
				break
			}
		}
		resp.Body.Close()
	} else {
		// Non-streaming: copy full body and cache it
		respBytes, bodyErr := io.ReadAll(resp.Body)
		resp.Body.Close()
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
