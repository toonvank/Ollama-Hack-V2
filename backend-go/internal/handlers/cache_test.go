package handlers

import (
	"testing"
	"time"
)

func TestResponseCache_Get_Miss(t *testing.T) {
	cache := &ResponseCache{m: make(map[string]cacheEntry)}

	_, _, ok := cache.Get("nonexistent")
	if ok {
		t.Error("Expected cache miss for nonexistent key")
	}
}

func TestResponseCache_Get_Hit(t *testing.T) {
	cache := &ResponseCache{m: make(map[string]cacheEntry)}
	data := []byte(`{"result": "success"}`)
	headers := map[string][]string{"Content-Type": {"application/json"}}

	cache.Set("test-key", data, headers, 10*time.Second)

	gotData, gotHeaders, ok := cache.Get("test-key")
	if !ok {
		t.Fatal("Expected cache hit")
	}
	if string(gotData) != string(data) {
		t.Errorf("Expected %s, got %s", string(data), string(gotData))
	}
	if gotHeaders["Content-Type"][0] != "application/json" {
		t.Errorf("Expected Content-Type header, got %v", gotHeaders)
	}
}

func TestResponseCache_Get_Expired(t *testing.T) {
	cache := &ResponseCache{m: make(map[string]cacheEntry)}
	data := []byte(`{"result": "success"}`)

	cache.Set("test-key", data, nil, 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond) // Wait for expiration

	_, _, ok := cache.Get("test-key")
	if ok {
		t.Error("Expected cache miss for expired entry")
	}
}

func TestResponseCache_Set_Cleanup(t *testing.T) {
	cache := &ResponseCache{m: make(map[string]cacheEntry)}

	// Fill cache beyond cleanup threshold (>1000 triggers cleanup)
	// The cleanup happens when len > 1000, so we need 1002 to trigger it
	for i := 0; i < 1002; i++ {
		cache.Set(string(rune(i)), []byte("data"), nil, 1*time.Hour)
	}

	// After cleanup, only the last entry should remain
	if len(cache.m) != 1 {
		t.Errorf("Expected cache to be cleaned after 1000 entries, got %d", len(cache.m))
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World!", "hello world"},
		{"  Trimmed  ", "trimmed"},
		{"Punctuation, here.", "punctuation here"},
		{"MiXeD CaSe", "mixed case"},
		{"", ""},
	}

	for _, tt := range tests {
		result := normalizeText(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeText(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGenerateCacheKey(t *testing.T) {
	bodyMap1 := map[string]interface{}{
		"model":       "llama2",
		"messages":    []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
		"prompt":      "Test prompt",
		"temperature": 0.7,
		"top_p":       0.9,
	}

	bodyMap2 := map[string]interface{}{
		"model":       "llama2",
		"messages":    []interface{}{map[string]interface{}{"role": "user", "content": "Hello"}},
		"prompt":      "Test prompt",
		"temperature": 0.7,
		"top_p":       0.9,
	}

	key1 := GenerateCacheKey(bodyMap1)
	key2 := GenerateCacheKey(bodyMap2)

	if key1 != key2 {
		t.Error("Expected same cache key for identical inputs")
	}

	if len(key1) != 64 {
		t.Errorf("Expected 64-char SHA256 hex, got %d", len(key1))
	}

	bodyMap3 := map[string]interface{}{
		"model":       "llama2",
		"messages":    []interface{}{map[string]interface{}{"role": "user", "content": "Different"}},
		"prompt":      "Test prompt",
		"temperature": 0.7,
		"top_p":       0.9,
	}

	key3 := GenerateCacheKey(bodyMap3)
	if key1 == key3 {
		t.Error("Expected different cache keys for different inputs")
	}
}
