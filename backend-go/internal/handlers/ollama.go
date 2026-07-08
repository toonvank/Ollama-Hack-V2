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
	"regexp"
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
				fallbacks[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
			}
		}
	}
	return &OllamaHandler{
		db:          db,
		fallbacks:   fallbacks,
		smartRouter: services.NewSmartRouter(),
	}
}

// EndpointRankMode controls how endpoints are ordered for a fixed model.
type EndpointRankMode string

const (
	RankByTPS       EndpointRankMode = "tps"
	RankByReplyTime EndpointRankMode = "reply"
	RankByComposite EndpointRankMode = "composite"
)

func endpointOrderClause(mode EndpointRankMode) string {
	switch mode {
	case RankByReplyTime:
		return "eam.max_connection_time ASC NULLS LAST, eam.token_per_second DESC NULLS LAST"
	case RankByTPS:
		return "eam.token_per_second DESC NULLS LAST"
	default:
		return utils.EndpointCompositeScoreSQL() + " DESC NULLS LAST, eam.token_per_second DESC NULLS LAST"
	}
}

func defaultEndpointRankMode() EndpointRankMode {
	switch strings.ToLower(os.Getenv("ROUTING_RANK_MODE")) {
	case "tps":
		return RankByTPS
	case "reply":
		return RankByReplyTime
	default:
		return RankByComposite
	}
}

type resolvedModelRoute struct {
	Name string
	Tag  string
	URLs []string
}

// bestEndpointsForModel returns the top-ranked endpoint URLs for a model.
// Name/tag matching is case-insensitive so clients like Open WebUI can send
// "Llama3.2:3b" while the catalog stores "llama3.2:3b".
func (h *OllamaHandler) bestEndpointsForModel(modelName, modelTag string, rankMode EndpointRankMode) (*resolvedModelRoute, error) {
	type row struct {
		URL  string `db:"url"`
		Name string `db:"name"`
		Tag  string `db:"tag"`
	}
	var rows []row
	minTPS := 0.0
	if val := os.Getenv("MIN_TPS_THRESHOLD"); val != "" {
		fmt.Sscanf(val, "%f", &minTPS)
	}

	orderBy := endpointOrderClause(rankMode)

	err := h.db.Select(&rows, fmt.Sprintf(`
		SELECT e.url, m.name, m.tag
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		JOIN ai_models m ON m.id = eam.ai_model_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
		WHERE LOWER(m.name) = LOWER($1) AND LOWER(m.tag) = LOWER($2)
		  AND m.enabled = TRUE
		  AND eam.status = 'available'
		  AND e.status = 'available'
		  AND (eh.disabled IS NULL OR eh.disabled = FALSE)
		  AND (eam.token_per_second >= $3 OR eam.token_per_second IS NULL)
		ORDER BY %s
		LIMIT 5
	`, orderBy), modelName, modelTag, minTPS)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	urls := make([]string, 0, len(rows))
	for _, r := range rows {
		urls = append(urls, r.URL)
	}
	return &resolvedModelRoute{
		Name: rows[0].Name,
		Tag:  rows[0].Tag,
		URLs: urls,
	}, nil
}

func setCanonicalModelInBody(bodyMap map[string]interface{}, name, tag string) string {
	canonical := fmt.Sprintf("%s:%s", name, tag)
	bodyMap["model"] = canonical
	return canonical
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

	heuristic, _, rankingClause := smartProfileConfig(smartTag)

	// Fetch top 3 models ranked by reply time (best endpoint per model, then global sort)
	query := fmt.Sprintf(`
		SELECT name, tag FROM (
			SELECT DISTINCT ON (m.name, m.tag)
				m.name, m.tag,
				eam.max_connection_time,
				eam.token_per_second
			FROM endpoint_ai_models eam
			JOIN endpoints e ON e.id = eam.endpoint_id
			JOIN ai_models m ON m.id = eam.ai_model_id
			LEFT JOIN endpoint_health eh ON eh.url = e.url
			WHERE %s
			  AND m.enabled = TRUE
			  AND eam.status = 'available'
			  AND e.status = 'available'
			  AND (eh.disabled IS NULL OR eh.disabled = FALSE)
			ORDER BY m.name, m.tag, %s
		) ranked_models
		ORDER BY max_connection_time ASC NULLS LAST, token_per_second DESC NULLS LAST
		LIMIT 3
	`, heuristic, rankingClause)

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
		resolved, err := h.bestEndpointsForModel(mRow.Name, mRow.Tag, RankByReplyTime)
		if err == nil && resolved != nil && len(resolved.URLs) > 0 {
			candidates = append(candidates, smartModelCandidate{
				urls: resolved.URLs,
				name: resolved.Name,
				tag:  resolved.Tag,
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

// parseThinkValue normalizes Ollama's think flag from JSON bodies or headers.
func parseThinkValue(v interface{}) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		switch s {
		case "true", "on", "1", "yes":
			return true, true
		case "false", "off", "0", "no":
			return false, true
		}
	}
	return false, false
}

// resolveThinkPreference returns an explicit think preference when the client sets one.
// If nothing is set, thinking is left at the upstream model default.
func resolveThinkPreference(body map[string]interface{}, thinkHeader string) (value bool, explicit bool) {
	if thinkHeader != "" {
		if v, ok := parseThinkValue(thinkHeader); ok {
			return v, true
		}
	}
	if v, ok := body["think"]; ok {
		return parseThinkValue(v)
	}
	if opts, ok := body["options"].(map[string]interface{}); ok {
		if v, ok := opts["think"]; ok {
			return parseThinkValue(v)
		}
	}
	if env := os.Getenv("DEFAULT_THINK"); env != "" {
		if v, ok := parseThinkValue(env); ok {
			return v, true
		}
	}
	return false, false
}

var thinkingTagPattern = regexp.MustCompile(`(?is)<think>.*?</think>|<thinking>.*?</thinking>|<reasoning>.*?</reasoning>`)

func stripThinkingText(content string) string {
	return strings.TrimSpace(thinkingTagPattern.ReplaceAllString(content, ""))
}

func stripThinkingTextStream(content string) string {
	return thinkingTagPattern.ReplaceAllString(content, "")
}

func stripThinkingFieldsFromMap(msg map[string]interface{}, isStream bool) {
	for _, key := range []string{"thinking", "reasoning", "reasoning_content"} {
		delete(msg, key)
	}
	if content, ok := msg["content"].(string); ok {
		if isStream {
			msg["content"] = stripThinkingTextStream(content)
		} else {
			msg["content"] = stripThinkingText(content)
		}
	}
}

func stripThinkingFromChatResponse(respBytes []byte) []byte {
	var payload map[string]interface{}
	if err := json.Unmarshal(respBytes, &payload); err != nil {
		return respBytes
	}
	choices, ok := payload["choices"].([]interface{})
	if !ok {
		return respBytes
	}
	for _, choiceRaw := range choices {
		choice, ok := choiceRaw.(map[string]interface{})
		if !ok {
			continue
		}
		if msg, ok := choice["message"].(map[string]interface{}); ok {
			stripThinkingFieldsFromMap(msg, false)
		}
		if delta, ok := choice["delta"].(map[string]interface{}); ok {
			stripThinkingFieldsFromMap(delta, true)
		}
	}
	out, err := json.Marshal(payload)
	if err != nil {
		return respBytes
	}
	return out
}

func stripThinkingFromStreamLine(line []byte) []byte {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("data: [DONE]")) {
		return line
	}
	if !bytes.HasPrefix(trimmed, []byte("data:")) {
		return line
	}
	jsonPart := bytes.TrimSpace(bytes.TrimPrefix(trimmed, []byte("data:")))
	if len(jsonPart) == 0 || bytes.Equal(jsonPart, []byte("[DONE]")) {
		return line
	}
	cleaned := stripThinkingFromChatResponse(jsonPart)
	return append([]byte("data: "), cleaned...)
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
	models := make([]gin.H, 0, len(rows)+len(pseudoModels))

	// Inject pseudo-models first (OpenWebUI-friendly names at the top)
	for _, pm := range pseudoModels {
		models = append(models, gin.H{
			"name":        pm,
			"model":       pm,
			"modified_at": now,
		})
	}

	for _, r := range rows {
		modelName := fmt.Sprintf("%s:%s", r.Name, r.Tag)
		models = append(models, gin.H{
			"name":        modelName,
			"model":       modelName,
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

// Embeddings proxies OpenAI POST /v1/embeddings
func (h *OllamaHandler) Embeddings(c *gin.Context) {
	h.proxyRequest(c, "POST", "/v1/embeddings")
}

// EmbeddingsNative proxies native Ollama POST /api/embeddings
func (h *OllamaHandler) EmbeddingsNative(c *gin.Context) {
	h.proxyRequest(c, "POST", "/api/embeddings")
}

// EmbedNative proxies native Ollama POST /api/embed
func (h *OllamaHandler) EmbedNative(c *gin.Context) {
	h.proxyRequest(c, "POST", "/api/embed")
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
	thinkVal, thinkExplicit := resolveThinkPreference(bodyMap, c.GetHeader("X-Ollama-Think"))
	stripThinking := thinkExplicit && !thinkVal
	if thinkExplicit {
		bodyMap["think"] = thinkVal
		rawBody, _ = json.Marshal(bodyMap)
		log.Printf("[proxy] Forwarding think=%v for model %s", thinkVal, modelRaw)
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
	skipSmartRoute := strings.EqualFold(c.GetHeader("X-Skip-Smart-Route"), "true")
	if messages, ok := bodyMap["messages"].([]interface{}); ok && messagesContainImages(messages) {
		skipSmartRoute = true
	}
	if h.smartRouter.IsEnabled() && !skipSmartRoute {
		// Extract messages for classification
		if messages, ok := bodyMap["messages"].([]interface{}); ok && len(messages) > 0 {
			if result := h.smartRouter.ClassifyMessages(messages); result != nil && result.PreferModel != "" {
				// Check if the preferred model is available (real or pseudo-model)
				preferName, preferTag := parseModel(result.PreferModel)
				var available bool
				if preferName == "smart" {
					candidates, err := h.resolveSmartModel(preferTag)
					available = err == nil && len(candidates) > 0
				} else {
					preferResolved, preferErr := h.bestEndpointsForModel(preferName, preferTag, defaultEndpointRankMode())
					available = preferErr == nil && preferResolved != nil && len(preferResolved.URLs) > 0
				}

				if available {
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

	if name == "smart" || name == "best-abliterated" {
		profile := tag
		if name == "best-abliterated" {
			profile = "abliterated"
		}
		smartCandidates, err = h.resolveSmartModel(profile)
		if err == nil && len(smartCandidates) > 0 {
			healthTracker := services.GetHealthTracker()
			var best smartModelCandidate
			foundHealthy := false
			for i, candidate := range smartCandidates {
				healthyURLs := healthTracker.FilterHealthyEndpoints(candidate.urls)
				if len(healthyURLs) > 0 {
					best = candidate
					endpoints = healthyURLs
					// Rearrange smartCandidates so that the chosen candidate is at index 0,
					// and the rest are available as fallback candidates.
					smartCandidates = append([]smartModelCandidate{candidate}, append(smartCandidates[:i], smartCandidates[i+1:]...)...)
					foundHealthy = true
					break
				}
			}
			if foundHealthy {
				name, tag = best.name, best.tag
				log.Printf("[smart-model] Resolved '%s' → '%s:%s' (%d fallback candidates)",
					originalModelRequested, name, tag, len(smartCandidates)-1)
				modelRaw = fmt.Sprintf("%s:%s", name, tag)
				bodyMap["model"] = modelRaw
				rawBody, _ = json.Marshal(bodyMap)
				smartRouteHeader = services.FormatRouteHeader("smart", modelRaw)
			} else {
				endpoints = nil
				err = fmt.Errorf("no healthy endpoints for smart candidates")
			}
		}
	} else {
		var resolved *resolvedModelRoute
		resolved, err = h.bestEndpointsForModel(name, tag, defaultEndpointRankMode())
		if resolved != nil && len(resolved.URLs) > 0 {
			name, tag = resolved.Name, resolved.Tag
			endpoints = resolved.URLs
			modelRaw = setCanonicalModelInBody(bodyMap, name, tag)
			rawBody, _ = json.Marshal(bodyMap)
		}
	}

	// Attempt blazing fast in-memory fallback route if unavailable
	if err != nil || len(endpoints) == 0 {
		lookupKey := strings.ToLower(fmt.Sprintf("%s:%s", name, tag))
		if fallbackRaw, ok := h.fallbacks[lookupKey]; ok {
			log.Printf("[proxy] Model %s unavailable, applying fallback to %s", lookupKey, fallbackRaw)

			name, tag = parseModel(fallbackRaw)
			var resolved *resolvedModelRoute
			resolved, err = h.bestEndpointsForModel(name, tag, defaultEndpointRankMode())
			if resolved != nil && len(resolved.URLs) > 0 {
				name, tag = resolved.Name, resolved.Tag
				endpoints = resolved.URLs
				c.Header("X-Model-Fallback", fallbackRaw)

				modelRaw = setCanonicalModelInBody(bodyMap, name, tag)
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
		quotaExceeded bool   // true if the error was a quota/balance limit error
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

			client := utils.SharedRaceClient()
			resp, err := client.Do(req)

			if err != nil {
				resultCh <- raceResult{err: err, endpointURL: url, index: index}
				return
			}

			// Non-200 responses are failures in the race
			if resp.StatusCode >= 400 {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				isQuota := isQuotaExceededError(resp.StatusCode, bodyBytes)
				if isQuota {
					resultCh <- raceResult{err: fmt.Errorf("quota exceeded"), endpointURL: url, index: index, quotaExceeded: true, failStatus: resp.StatusCode, failBody: bodyBytes}
				} else if resp.StatusCode == 429 {
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
					isQuota := isQuotaExceededError(200, []byte(sniffStr))
					resultCh <- raceResult{err: fmt.Errorf("rejected node: returned 200 OK error JSON payload"), endpointURL: url, index: index, quotaExceeded: isQuota}
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
						isQuota := isQuotaExceededError(200, []byte(sniffStr))
						resultCh <- raceResult{err: fmt.Errorf("rejected node: upstream model threw an API error hidden in the SSE stream"), endpointURL: url, index: index, quotaExceeded: isQuota}
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

			// Handle different error categories for health scoring
			if res.quotaExceeded {
				healthTracker.RecordQuotaExceeded(res.endpointURL)
			} else if res.rateLimited {
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
					client := utils.SharedProxyClient()
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

							// Record health failure/quota/rate limit for fallback endpoint
							if isQuotaExceededError(cresp.StatusCode, b) {
								healthTracker.RecordQuotaExceeded(cascadeURL)
							} else if cresp.StatusCode == 429 {
								healthTracker.RecordRateLimit(cascadeURL)
							} else if cresp.StatusCode >= 500 {
								healthTracker.RecordFailure(cascadeURL)
							}
						} else {
							healthTracker.RecordFailure(cascadeURL)
						}
						continue
					}
					// Cascade winner found!
					log.Printf("[smart-cascade] 🏁 Cascade winner: %s via %s", fallbackModel, cascadeURL)
					c.Header("X-Smart-Cascade", fmt.Sprintf("%s->%s", modelRaw, fallbackModel))
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
		if winningResp == nil && isCloudTaggedModel(name, tag, originalModelRequested) {
			triedCloudModels := collectTriedCloudModels(name, tag, smartCandidates)
			fbResp, fbModel, fbFailStatus, fbFailBody := h.attemptCloudProviderCascade(
				c.Request.Context(), method, path, bodyMap, healthTracker, triedCloudModels)
			if fbResp != nil {
				winningResp = fbResp
				c.Header("X-Cloud-Provider-Fallback", fmt.Sprintf("%s->%s", originalModelRequested, fbModel))
				modelRaw = fbModel
			} else if fbFailStatus > 0 {
				lastFailStatus = fbFailStatus
				lastFailBody = fbFailBody
			}
		}

		if winningResp == nil && isCloudTaggedModel(name, tag, originalModelRequested) {
			if fallbackRoute, fbErr := h.resolveConfiguredFallbackRoute(originalModelRequested); fbErr == nil && fallbackRoute != nil {
				log.Printf("[cloud-fallback] No cloud providers left for '%s', using configured fallback '%s:%s'",
					originalModelRequested, fallbackRoute.Name, fallbackRoute.Tag)

				fbResp, fbModel, fbFailStatus, fbFailBody := h.attemptModelOnEndpoints(
					c.Request.Context(), method, path, fallbackRoute, bodyMap, healthTracker)
				if fbResp != nil {
					winningResp = fbResp
					c.Header("X-Model-Fallback", fmt.Sprintf("%s->%s", originalModelRequested, fbModel))
					modelRaw = fbModel
				} else if fbFailStatus > 0 {
					lastFailStatus = fbFailStatus
					lastFailBody = fbFailBody
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
				if stripThinking {
					line = stripThinkingFromStreamLine(line)
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
		if stripThinking {
			respBytes = stripThinkingFromChatResponse(respBytes)
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
			for k, v := range bodyMap {
				bodyClone[k] = v
			}

			msgsClone := make([]interface{}, len(messages)-1)
			copy(msgsClone, messages[:len(messages)-1])

			newLastMsg := map[string]interface{}{}
			for k, v := range lastMsg {
				newLastMsg[k] = v
			}
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
				for _, v := range vs {
					req.Header.Add(k, v)
				}
			}

			client := utils.NewVPNHTTPClient(300 * time.Second)
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
		}(i, chunkText, endpoints[i%len(endpoints)])
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
			"choices": []gin.H{{"index": 0, "delta": gin.H{"content": finalText}}},
		}
		b, _ := json.Marshal(chunk)
		c.Writer.Write([]byte("data: " + string(b) + "\n\n"))
		c.Writer.Write([]byte("data: [DONE]\n\n"))
	} else {
		ans := gin.H{
			"id": "chatcmpl-mapreduce", "object": "chat.completion",
			"created": time.Now().Unix(), "model": bodyMap["model"],
			"choices": []gin.H{{"index": 0, "message": gin.H{"role": "assistant", "content": finalText}}},
		}
		c.JSON(200, ans)
	}
}

// messagesContainImages reports whether any chat message still carries image input.
func messagesContainImages(messages []interface{}) bool {
	for _, raw := range messages {
		msg, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		if images, ok := msg["images"].([]interface{}); ok && len(images) > 0 {
			return true
		}

		switch content := msg["content"].(type) {
		case string:
			if strings.Contains(content, "data:image") {
				return true
			}
		case []interface{}:
			for _, partRaw := range content {
				part, ok := partRaw.(map[string]interface{})
				if !ok {
					continue
				}
				partType, _ := part["type"].(string)
				switch partType {
				case "image_url", "image", "input_image":
					return true
				case "text":
					if text, ok := part["text"].(string); ok && strings.Contains(text, "data:image") {
						return true
					}
				}
			}
		}
	}
	return false
}

// isCloudTaggedModel reports whether the request targets an Ollama cloud model.
func isCloudTaggedModel(name, tag, originalRequested string) bool {
	if strings.EqualFold(tag, "cloud") {
		return true
	}
	return strings.HasSuffix(strings.ToLower(originalRequested), ":cloud")
}

func collectTriedCloudModels(name, tag string, smartCandidates []smartModelCandidate) map[string]bool {
	tried := map[string]bool{
		strings.ToLower(fmt.Sprintf("%s:%s", name, tag)): true,
	}
	for _, candidate := range smartCandidates {
		tried[strings.ToLower(fmt.Sprintf("%s:%s", candidate.name, candidate.tag))] = true
	}
	return tried
}

// attemptCloudProviderCascade tries alternate cloud models/providers before giving up.
func (h *OllamaHandler) attemptCloudProviderCascade(
	ctx context.Context,
	method, path string,
	bodyMap map[string]interface{},
	healthTracker *services.HealthTracker,
	triedModels map[string]bool,
) (*http.Response, string, int, []byte) {
	cloudCandidates, err := h.resolveSmartModel("cloud")
	if err != nil {
		return nil, "", 0, nil
	}

	var lastStatus int
	var lastBody []byte

	for _, candidate := range cloudCandidates {
		candidateModel := strings.ToLower(fmt.Sprintf("%s:%s", candidate.name, candidate.tag))
		if triedModels[candidateModel] {
			continue
		}

		urls := healthTracker.FilterHealthyEndpoints(candidate.urls)
		if len(urls) == 0 {
			continue
		}

		log.Printf("[cloud-provider-cascade] Trying alternate cloud provider '%s:%s'", candidate.name, candidate.tag)
		route := &resolvedModelRoute{
			Name: candidate.name,
			Tag:  candidate.tag,
			URLs: urls,
		}
		resp, model, failStatus, failBody := h.attemptModelOnEndpoints(ctx, method, path, route, bodyMap, healthTracker)
		if resp != nil {
			return resp, model, 0, nil
		}
		if failStatus > 0 {
			lastStatus = failStatus
			lastBody = failBody
		}
		triedModels[candidateModel] = true
	}

	return nil, "", lastStatus, lastBody
}

// resolveConfiguredFallbackRoute returns an explicit APP_FALLBACK_MODELS mapping only.
func (h *OllamaHandler) resolveConfiguredFallbackRoute(lookupModel string) (*resolvedModelRoute, error) {
	lookupKey := strings.ToLower(lookupModel)
	fallbackRaw, ok := h.fallbacks[lookupKey]
	if !ok {
		return nil, fmt.Errorf("no configured fallback for %s", lookupModel)
	}
	fbName, fbTag := parseModel(fallbackRaw)
	return h.bestEndpointsForModel(fbName, fbTag, defaultEndpointRankMode())
}

// attemptModelOnEndpoints tries ranked endpoints for a fallback model.
func (h *OllamaHandler) attemptModelOnEndpoints(
	ctx context.Context,
	method, path string,
	fallbackRoute *resolvedModelRoute,
	bodyMap map[string]interface{},
	healthTracker *services.HealthTracker,
) (*http.Response, string, int, []byte) {
	fallbackModel := fmt.Sprintf("%s:%s", fallbackRoute.Name, fallbackRoute.Tag)
	bodyMap["model"] = fallbackModel
	fallbackBody, _ := json.Marshal(bodyMap)

	var lastStatus int
	var lastBody []byte

	for _, fallbackURL := range fallbackRoute.URLs {
		req, err := http.NewRequestWithContext(ctx, method, fallbackURL+path, bytes.NewReader(fallbackBody))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		client := utils.SharedProxyClient()
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode >= 400 {
			if resp != nil {
				var b []byte
				if resp.Body != nil {
					b, _ = io.ReadAll(resp.Body)
					resp.Body.Close()
				}
				lastStatus = resp.StatusCode
				lastBody = b

				if isQuotaExceededError(resp.StatusCode, b) {
					healthTracker.RecordQuotaExceeded(fallbackURL)
				} else if resp.StatusCode == 429 {
					healthTracker.RecordRateLimit(fallbackURL)
				} else if resp.StatusCode >= 500 {
					healthTracker.RecordFailure(fallbackURL)
				}
			} else {
				healthTracker.RecordFailure(fallbackURL)
			}
			continue
		}
		return resp, fallbackModel, 0, nil
	}
	return nil, "", lastStatus, lastBody
}

// isQuotaExceededError checks if an upstream response indicates cloud billing, quota,
// or subscription access is unavailable.
func isQuotaExceededError(statusCode int, body []byte) bool {
	quotaKeywords := []string{
		"insufficient_quota",
		"insufficient_balance",
		"insufficient balance",
		"quota_exceeded",
		"quota exceeded",
		"exceeded your current quota",
		"billing_not_active",
		"out of balance",
		"run out of balance",
		"credit limit reached",
		"free_trial_quota",
		"requires a subscription",
		"requires subscription",
		"subscription required",
		"upgrade for access",
		"ollama.com/upgrade",
		"ollama.com/settings",
		"session usage limit",
		"reached your session usage limit",
		"upgrade for higher limits",
	}

	bodyStr := strings.ToLower(string(body))
	for _, kw := range quotaKeywords {
		if strings.Contains(bodyStr, kw) {
			return true
		}
	}
	return false
}
