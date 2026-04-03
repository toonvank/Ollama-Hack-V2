package services

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// SemanticCacheEntry stores an embedding vector along with its cached response
type SemanticCacheEntry struct {
	Embedding []float64
	Data      []byte
	Headers   map[string][]string
	ExpiresAt time.Time
	Prompt    string // For debugging/logging
}

// SemanticCache provides similarity-based caching using embeddings
type SemanticCache struct {
	sync.RWMutex
	entries         []SemanticCacheEntry
	enabled         bool
	threshold       float64
	embeddingModel  string
	ollamaURL       string
	maxEntries      int
}

// GlobalSemanticCache is the singleton semantic cache instance
var GlobalSemanticCache *SemanticCache

func init() {
	GlobalSemanticCache = NewSemanticCache()
}

// NewSemanticCache creates a new semantic cache with configuration from environment
func NewSemanticCache() *SemanticCache {
	enabled := os.Getenv("SEMANTIC_CACHE_ENABLED") == "true"
	
	threshold := 0.95
	if t := os.Getenv("SEMANTIC_CACHE_THRESHOLD"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil && parsed > 0 && parsed <= 1 {
			threshold = parsed
		}
	}
	
	embeddingModel := os.Getenv("SEMANTIC_CACHE_MODEL")
	if embeddingModel == "" {
		embeddingModel = "nomic-embed-text"
	}
	
	ollamaURL := os.Getenv("SEMANTIC_CACHE_OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	
	maxEntries := 500
	if m := os.Getenv("SEMANTIC_CACHE_MAX_ENTRIES"); m != "" {
		if parsed, err := strconv.Atoi(m); err == nil && parsed > 0 {
			maxEntries = parsed
		}
	}
	
	if enabled {
		log.Printf("[semantic-cache] Enabled with threshold=%.3f, model=%s, url=%s, max_entries=%d",
			threshold, embeddingModel, ollamaURL, maxEntries)
	}
	
	return &SemanticCache{
		entries:        make([]SemanticCacheEntry, 0),
		enabled:        enabled,
		threshold:      threshold,
		embeddingModel: embeddingModel,
		ollamaURL:      ollamaURL,
		maxEntries:     maxEntries,
	}
}

// IsEnabled returns whether semantic caching is enabled
func (c *SemanticCache) IsEnabled() bool {
	return c.enabled
}

// GetEmbedding calls the Ollama embeddings API to get a vector for the given text
func (c *SemanticCache) GetEmbedding(text string) ([]float64, error) {
	reqBody := map[string]interface{}{
		"model":  c.embeddingModel,
		"prompt": text,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(c.ollamaURL+"/api/embeddings", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	
	return result.Embedding, nil
}

// cosineSimilarity computes the cosine similarity between two vectors
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// SearchResult contains a cache hit with its similarity score
type SearchResult struct {
	Data       []byte
	Headers    map[string][]string
	Similarity float64
	Prompt     string
}

// Search finds the most similar cached entry above the threshold
func (c *SemanticCache) Search(embedding []float64) (*SearchResult, bool) {
	c.RLock()
	defer c.RUnlock()
	
	now := time.Now()
	var bestResult *SearchResult
	bestSimilarity := 0.0
	
	for _, entry := range c.entries {
		// Skip expired entries
		if now.After(entry.ExpiresAt) {
			continue
		}
		
		similarity := cosineSimilarity(embedding, entry.Embedding)
		if similarity >= c.threshold && similarity > bestSimilarity {
			bestSimilarity = similarity
			bestResult = &SearchResult{
				Data:       entry.Data,
				Headers:    entry.Headers,
				Similarity: similarity,
				Prompt:     entry.Prompt,
			}
		}
	}
	
	return bestResult, bestResult != nil
}

// Store adds a new entry to the semantic cache
func (c *SemanticCache) Store(embedding []float64, data []byte, headers map[string][]string, prompt string, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()
	
	// Clean up expired entries and enforce max size
	now := time.Now()
	validEntries := make([]SemanticCacheEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		if now.Before(entry.ExpiresAt) {
			validEntries = append(validEntries, entry)
		}
	}
	
	// If still at max capacity, remove oldest entries
	if len(validEntries) >= c.maxEntries {
		// Remove the first 10% to make room
		removeCount := c.maxEntries / 10
		if removeCount < 1 {
			removeCount = 1
		}
		validEntries = validEntries[removeCount:]
	}
	
	// Add new entry
	validEntries = append(validEntries, SemanticCacheEntry{
		Embedding: embedding,
		Data:      data,
		Headers:   headers,
		ExpiresAt: now.Add(ttl),
		Prompt:    prompt,
	})
	
	c.entries = validEntries
}

// ExtractPromptFromRequest extracts the prompt text from a completion request body
func ExtractPromptFromRequest(bodyMap map[string]interface{}) string {
	// Check for chat completion format (messages array)
	if messages, ok := bodyMap["messages"].([]interface{}); ok && len(messages) > 0 {
		var parts []string
		for _, msg := range messages {
			if m, ok := msg.(map[string]interface{}); ok {
				if content, ok := m["content"].(string); ok {
					parts = append(parts, content)
				}
			}
		}
		if len(parts) > 0 {
			return joinStrings(parts, " ")
		}
	}
	
	// Check for completion format (prompt string)
	if prompt, ok := bodyMap["prompt"].(string); ok {
		return prompt
	}
	
	return ""
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// Size returns the current number of entries in the cache
func (c *SemanticCache) Size() int {
	c.RLock()
	defer c.RUnlock()
	return len(c.entries)
}

// Clear removes all entries from the cache
func (c *SemanticCache) Clear() {
	c.Lock()
	defer c.Unlock()
	c.entries = make([]SemanticCacheEntry, 0)
}
