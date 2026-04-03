package handlers

import (
	"bytes"
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
	"github.com/timlzh/ollama-hack/internal/utils"
)

type OllamaHandler struct {
	db        *database.DB
	fallbacks map[string]string
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
	return &OllamaHandler{db: db, fallbacks: fallbacks}
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

	stream, _ := bodyMap["stream"].(bool)
	var cacheKey string

	// Attempt Cache Hit for EXACT non-streaming requests
	// We skip caching for massive payloads (e.g. >500KB base64 images) to preserve memory
	if !stream && len(rawBody) < 500*1024 {
		cacheKey = GenerateCacheKey(bodyMap)
		if cachedData, cachedHeaders, ok := GlobalCache.Get(cacheKey); ok {
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
	}

	// Try each endpoint in priority order
	for _, endpointURL := range endpoints {
		target := endpointURL + path

		req, err := http.NewRequest(method, target, bytes.NewReader(rawBody))
		if err != nil {
			continue
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
			log.Printf("[proxy] endpoint %s failed: %v", endpointURL, err)
			continue
		}

		// Copy response headers
		for k, vs := range resp.Header {
			for _, v := range vs {
				c.Header(k, v)
			}
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
				c.Writer.Write(respBytes)
			}
		}
		return
	}

	c.JSON(502, gin.H{"error": "All available endpoints failed to respond"})
}
