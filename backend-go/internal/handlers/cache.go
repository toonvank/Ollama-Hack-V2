package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	data      []byte
	headers   map[string][]string
	expiresAt time.Time
}

type ResponseCache struct {
	sync.RWMutex
	m map[string]cacheEntry
}

// GlobalCache stores exact match responses for a limited time.
var GlobalCache = &ResponseCache{
	m: make(map[string]cacheEntry),
}

// Get attempts to retrieve a cache entry by key.
func (c *ResponseCache) Get(key string) ([]byte, map[string][]string, bool) {
	c.RLock()
	defer c.RUnlock()

	entry, ok := c.m[key]
	if !ok {
		return nil, nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, nil, false
	}
	return entry.data, entry.headers, true
}

// Set stores a response in the cache with a Time-To-Live.
func (c *ResponseCache) Set(key string, data []byte, headers map[string][]string, ttl time.Duration) {
	c.Lock()
	defer c.Unlock()

	// Simple cleanup to prevent unbounded memory growth
	if len(c.m) > 1000 {
		c.m = make(map[string]cacheEntry)
	}

	c.m[key] = cacheEntry{
		data:      data,
		headers:   headers,
		expiresAt: time.Now().Add(ttl),
	}
}

// GenerateCacheKey computes a SHA256 hash for the given completion request.
func GenerateCacheKey(bodyMap map[string]interface{}) string {
	model, _ := bodyMap["model"]
	msgs, _ := json.Marshal(bodyMap["messages"])
	prompt, _ := json.Marshal(bodyMap["prompt"])
	temp, _ := bodyMap["temperature"]
	topP, _ := bodyMap["top_p"]

	str := fmt.Sprintf("%v|%s|%s|%v|%v", model, msgs, prompt, temp, topP)
	hash := sha256.Sum256([]byte(str))
	return hex.EncodeToString(hash[:])
}
