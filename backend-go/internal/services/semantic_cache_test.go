package services

import (
	"math"
	"testing"
	"time"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        []float64
		b        []float64
		expected float64
	}{
		{
			name:     "identical vectors",
			a:        []float64{1, 2, 3},
			b:        []float64{1, 2, 3},
			expected: 1.0,
		},
		{
			name:     "orthogonal vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{0, 1, 0},
			expected: 0.0,
		},
		{
			name:     "opposite vectors",
			a:        []float64{1, 0, 0},
			b:        []float64{-1, 0, 0},
			expected: -1.0,
		},
		{
			name:     "empty vectors",
			a:        []float64{},
			b:        []float64{},
			expected: 0.0,
		},
		{
			name:     "different lengths",
			a:        []float64{1, 2},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
		{
			name:     "zero vector",
			a:        []float64{0, 0, 0},
			b:        []float64{1, 2, 3},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cosineSimilarity(tt.a, tt.b)
			if math.Abs(result-tt.expected) > 0.0001 {
				t.Errorf("cosineSimilarity(%v, %v) = %f, want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSemanticCache_NewSemanticCache(t *testing.T) {
	cache := NewSemanticCache()

	if cache == nil {
		t.Fatal("Expected non-nil cache")
	}
	if cache.threshold != 0.95 {
		t.Errorf("Expected threshold=0.95, got %f", cache.threshold)
	}
	if cache.embeddingModel != "nomic-embed-text" {
		t.Errorf("Expected embeddingModel=nomic-embed-text, got %s", cache.embeddingModel)
	}
	if cache.ollamaURL != "http://localhost:11434" {
		t.Errorf("Expected ollamaURL=http://localhost:11434, got %s", cache.ollamaURL)
	}
	if cache.maxEntries != 500 {
		t.Errorf("Expected maxEntries=500, got %d", cache.maxEntries)
	}
}

func TestSemanticCache_IsEnabled(t *testing.T) {
	cache := NewSemanticCache()

	// By default, enabled is false (unless env var is set)
	_ = cache.IsEnabled()
}

func TestSemanticCache_StoreAndSearch(t *testing.T) {
	cache := &SemanticCache{
		entries:   make([]SemanticCacheEntry, 0),
		enabled:   true,
		threshold: 0.95,
		maxEntries: 500,
	}

	embedding := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	data := []byte(`{"result": "success"}`)
	headers := map[string][]string{"Content-Type": {"application/json"}}
	prompt := "Test prompt"

	// Store entry
	cache.Store(embedding, data, headers, prompt, 10*time.Second)

	if cache.Size() != 1 {
		t.Errorf("Expected size=1, got %d", cache.Size())
	}

	// Search with same embedding (should find exact match)
	result, found := cache.Search(embedding)
	if !found {
		t.Fatal("Expected to find entry")
	}
	if string(result.Data) != string(data) {
		t.Errorf("Expected data=%s, got %s", string(data), string(result.Data))
	}
	if result.Similarity != 1.0 {
		t.Errorf("Expected similarity=1.0, got %f", result.Similarity)
	}
	if result.Prompt != prompt {
		t.Errorf("Expected prompt=%s, got %s", prompt, result.Prompt)
	}
}

func TestSemanticCache_SearchExpired(t *testing.T) {
	cache := &SemanticCache{
		entries:   make([]SemanticCacheEntry, 0),
		enabled:   true,
		threshold: 0.95,
		maxEntries: 500,
	}

	embedding := []float64{0.1, 0.2, 0.3}
	data := []byte(`{"result": "success"}`)

	// Store with very short TTL
	cache.Store(embedding, data, nil, "Test", 1*time.Nanosecond)
	time.Sleep(10 * time.Millisecond)

	// Should not find expired entry
	_, found := cache.Search(embedding)
	if found {
		t.Error("Expected not to find expired entry")
	}
}

func TestSemanticCache_SearchBelowThreshold(t *testing.T) {
	cache := &SemanticCache{
		entries:   make([]SemanticCacheEntry, 0),
		enabled:   true,
		threshold: 0.95,
		maxEntries: 500,
	}

	embedding1 := []float64{1.0, 0.0, 0.0}
	embedding2 := []float64{0.0, 1.0, 0.0} // Orthogonal = 0 similarity
	data := []byte(`{"result": "success"}`)

	cache.Store(embedding1, data, nil, "Test", 10*time.Second)

	// Search with very different embedding
	_, found := cache.Search(embedding2)
	if found {
		t.Error("Expected not to find entry below threshold")
	}
}

func TestSemanticCache_Clear(t *testing.T) {
	cache := &SemanticCache{
		entries:   make([]SemanticCacheEntry, 0),
		enabled:   true,
		threshold: 0.95,
		maxEntries: 500,
	}

	// Add some entries
	for i := 0; i < 10; i++ {
		cache.Store([]float64{0.1, 0.2, 0.3}, []byte("data"), nil, "test", 10*time.Second)
	}

	if cache.Size() != 10 {
		t.Errorf("Expected size=10, got %d", cache.Size())
	}

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Expected size=0 after clear, got %d", cache.Size())
	}
}

func TestSemanticCache_MaxEntries(t *testing.T) {
	cache := &SemanticCache{
		entries:    make([]SemanticCacheEntry, 0),
		enabled:    true,
		threshold:  0.95,
		maxEntries: 5,
	}

	// Add more entries than max
	for i := 0; i < 10; i++ {
		cache.Store([]float64{0.1, 0.2, 0.3}, []byte("data"), nil, "test", 10*time.Second)
	}

	// Should have cleaned up old entries
	if cache.Size() > cache.maxEntries {
		t.Errorf("Expected size <= %d, got %d", cache.maxEntries, cache.Size())
	}
}

func TestExtractPromptFromRequest_ChatCompletion(t *testing.T) {
	bodyMap := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user", "content": "Hello"},
			map[string]interface{}{"role": "assistant", "content": "Hi there!"},
			map[string]interface{}{"role": "user", "content": "How are you?"},
		},
	}

	prompt := ExtractPromptFromRequest(bodyMap)
	expected := "Hello Hi there! How are you?"
	if prompt != expected {
		t.Errorf("ExtractPromptFromRequest() = %q, want %q", prompt, expected)
	}
}

func TestExtractPromptFromRequest_Completion(t *testing.T) {
	bodyMap := map[string]interface{}{
		"prompt": "Complete this sentence: The sky is",
	}

	prompt := ExtractPromptFromRequest(bodyMap)
	expected := "Complete this sentence: The sky is"
	if prompt != expected {
		t.Errorf("ExtractPromptFromRequest() = %q, want %q", prompt, expected)
	}
}

func TestExtractPromptFromRequest_Empty(t *testing.T) {
	bodyMap := map[string]interface{}{}

	prompt := ExtractPromptFromRequest(bodyMap)
	if prompt != "" {
		t.Errorf("ExtractPromptFromRequest() = %q, want empty string", prompt)
	}
}

func TestExtractPromptFromRequest_InvalidMessages(t *testing.T) {
	// Messages with non-string content
	bodyMap := map[string]interface{}{
		"messages": []interface{}{
			map[string]interface{}{"role": "user"}, // No content field
		},
	}

	prompt := ExtractPromptFromRequest(bodyMap)
	if prompt != "" {
		t.Errorf("ExtractPromptFromRequest() = %q, want empty string", prompt)
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		strs     []string
		sep      string
		expected string
	}{
		{[]string{"a", "b", "c"}, " ", "a b c"},
		{[]string{"a", "b", "c"}, ",", "a,b,c"},
		{[]string{"single"}, " ", "single"},
		{[]string{}, " ", ""},
	}

	for _, tt := range tests {
		result := joinStrings(tt.strs, tt.sep)
		if result != tt.expected {
			t.Errorf("joinStrings(%v, %q) = %q, want %q", tt.strs, tt.sep, result, tt.expected)
		}
	}
}
