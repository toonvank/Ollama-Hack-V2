package services

import (
	"os"
	"testing"
)

func TestSmartRouterDisabledByDefault(t *testing.T) {
	// Clear env
	os.Unsetenv("SMART_ROUTING_ENABLED")
	os.Unsetenv("SMART_ROUTING_RULES")

	sr := NewSmartRouter()
	if sr.IsEnabled() {
		t.Error("Expected smart router to be disabled by default")
	}
}

func TestSmartRouterEnabled(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()
	if !sr.IsEnabled() {
		t.Error("Expected smart router to be enabled when SMART_ROUTING_ENABLED=true")
	}
}

func TestSmartRouterClassifyCoding(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	tests := []struct {
		name     string
		prompt   string
		wantCat  string
		wantNil  bool
	}{
		{
			name:    "code keyword",
			prompt:  "Write a function to sort an array",
			wantCat: "coding",
		},
		{
			name:    "python keyword",
			prompt:  "Help me with this python script",
			wantCat: "coding",
		},
		{
			name:    "code block",
			prompt:  "What does this code do? ```print('hello')```",
			wantCat: "coding",
		},
		{
			name:    "debug keyword",
			prompt:  "Debug this error message",
			wantCat: "coding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sr.Classify(tt.prompt)
			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil result, got %v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("Expected classification result, got nil")
			}
			if result.Category != tt.wantCat {
				t.Errorf("Expected category %s, got %s", tt.wantCat, result.Category)
			}
		})
	}
}

func TestSmartRouterClassifyCreative(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	tests := []struct {
		name    string
		prompt  string
		wantCat string
	}{
		{
			name:    "story keyword",
			prompt:  "Write me a short story about a dragon",
			wantCat: "creative",
		},
		{
			name:    "poem keyword",
			prompt:  "Write a poem about love",
			wantCat: "creative",
		},
		{
			name:    "imagine keyword",
			prompt:  "Imagine a world where cats rule",
			wantCat: "creative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sr.Classify(tt.prompt)
			if result == nil {
				t.Fatal("Expected classification result, got nil")
			}
			if result.Category != tt.wantCat {
				t.Errorf("Expected category %s, got %s", tt.wantCat, result.Category)
			}
		})
	}
}

func TestSmartRouterClassifyAnalysis(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	tests := []struct {
		name    string
		prompt  string
		wantCat string
	}{
		{
			name:    "analyze keyword",
			prompt:  "Analyze this data for patterns",
			wantCat: "analysis",
		},
		{
			name:    "summarize keyword",
			prompt:  "Summarize the main points of this article",
			wantCat: "analysis",
		},
		{
			name:    "compare keyword",
			prompt:  "Compare and contrast these two approaches",
			wantCat: "analysis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sr.Classify(tt.prompt)
			if result == nil {
				t.Fatal("Expected classification result, got nil")
			}
			if result.Category != tt.wantCat {
				t.Errorf("Expected category %s, got %s", tt.wantCat, result.Category)
			}
		})
	}
}

func TestSmartRouterClassifyMessages(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	messages := []interface{}{
		map[string]interface{}{
			"role":    "system",
			"content": "You are a helpful assistant.",
		},
		map[string]interface{}{
			"role":    "user",
			"content": "Write a python function to calculate fibonacci",
		},
	}

	result := sr.ClassifyMessages(messages)
	if result == nil {
		t.Fatal("Expected classification result, got nil")
	}
	if result.Category != "coding" {
		t.Errorf("Expected category 'coding', got %s", result.Category)
	}
	if result.PreferModel != "codellama" {
		t.Errorf("Expected prefer model 'codellama', got %s", result.PreferModel)
	}
}

func TestSmartRouterDisabledReturnsNil(t *testing.T) {
	os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	result := sr.Classify("Write a python function")
	if result != nil {
		t.Errorf("Expected nil when disabled, got %v", result)
	}
}

func TestSmartRouterCustomRules(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	os.Setenv("SMART_ROUTING_RULES", `[{"category":"math","keywords":["calculate","equation","math"],"prefer_model":"mathstral"}]`)
	defer func() {
		os.Unsetenv("SMART_ROUTING_ENABLED")
		os.Unsetenv("SMART_ROUTING_RULES")
	}()

	sr := NewSmartRouter()

	result := sr.Classify("Calculate the square root of 144")
	if result == nil {
		t.Fatal("Expected classification result, got nil")
	}
	if result.Category != "math" {
		t.Errorf("Expected category 'math', got %s", result.Category)
	}
	if result.PreferModel != "mathstral" {
		t.Errorf("Expected prefer model 'mathstral', got %s", result.PreferModel)
	}
}

func TestFormatRouteHeader(t *testing.T) {
	header := FormatRouteHeader("coding", "codellama")
	expected := "coding→codellama"
	if header != expected {
		t.Errorf("Expected %s, got %s", expected, header)
	}
}

func TestSmartRouterSetEnabled(t *testing.T) {
	os.Unsetenv("SMART_ROUTING_ENABLED")
	sr := NewSmartRouter()

	if sr.IsEnabled() {
		t.Error("Should start disabled")
	}

	sr.SetEnabled(true)
	if !sr.IsEnabled() {
		t.Error("Should be enabled after SetEnabled(true)")
	}

	sr.SetEnabled(false)
	if sr.IsEnabled() {
		t.Error("Should be disabled after SetEnabled(false)")
	}
}

func TestSmartRouterConfidenceScoring(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	// Single keyword match
	result1 := sr.Classify("Write some code")

	// Multiple keyword matches should have higher confidence
	result2 := sr.Classify("Write a python function to debug and fix this programming error")

	if result1 == nil || result2 == nil {
		t.Fatal("Expected both results to be non-nil")
	}

	if result2.Confidence <= result1.Confidence {
		t.Errorf("Expected result2 confidence (%.2f) > result1 confidence (%.2f)",
			result2.Confidence, result1.Confidence)
	}
}

func TestSmartRouterEmptyPrompt(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	result := sr.Classify("")
	if result != nil {
		t.Error("Expected nil for empty prompt")
	}
}

func TestSmartRouterGetRules(t *testing.T) {
	os.Unsetenv("SMART_ROUTING_RULES")
	sr := NewSmartRouter()

	rules := sr.GetRules()
	if len(rules) != len(DefaultRoutingRules) {
		t.Errorf("Expected %d default rules, got %d", len(DefaultRoutingRules), len(rules))
	}
}

func TestSmartRouterSetRules(t *testing.T) {
	os.Setenv("SMART_ROUTING_ENABLED", "true")
	defer os.Unsetenv("SMART_ROUTING_ENABLED")

	sr := NewSmartRouter()

	customRules := []RoutingRule{
		{
			Category:    "custom",
			Keywords:    []string{"customword"},
			PreferModel: "custommodel",
		},
	}

	sr.SetRules(customRules)

	result := sr.Classify("test customword here")
	if result == nil {
		t.Fatal("Expected result with custom rules")
	}
	if result.Category != "custom" {
		t.Errorf("Expected category 'custom', got %s", result.Category)
	}
}
