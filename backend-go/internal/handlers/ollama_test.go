package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/services"
)

func TestParseThinkValue(t *testing.T) {
	if v, ok := parseThinkValue(true); !ok || !v {
		t.Fatal("expected true")
	}
	if v, ok := parseThinkValue("off"); !ok || v {
		t.Fatal("expected false for off")
	}
	if _, ok := parseThinkValue("maybe"); ok {
		t.Fatal("expected invalid value to be ignored")
	}
}

func TestResolveThinkPreference(t *testing.T) {
	body := map[string]interface{}{"model": "kimi-k2.5:cloud"}
	if _, explicit := resolveThinkPreference(body, ""); explicit {
		t.Fatal("expected no default think override")
	}

	body["think"] = false
	if v, explicit := resolveThinkPreference(body, ""); !explicit || v {
		t.Fatal("expected explicit think=false")
	}

	if v, explicit := resolveThinkPreference(map[string]interface{}{}, "true"); !explicit || !v {
		t.Fatal("expected header override")
	}
}

func TestStripThinkingFromChatResponse(t *testing.T) {
	raw := []byte(`{"choices":[{"message":{"role":"assistant","content":"hello","reasoning":"internal thoughts"}}]}`)
	cleaned := stripThinkingFromChatResponse(raw)
	var payload map[string]interface{}
	if err := json.Unmarshal(cleaned, &payload); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	msg := payload["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})
	if _, ok := msg["reasoning"]; ok {
		t.Fatal("expected reasoning field to be stripped")
	}
	if msg["content"] != "hello" {
		t.Fatalf("unexpected content: %v", msg["content"])
	}
}

func TestRecordRaceFailure_SkipsCanceledLosers(t *testing.T) {
	handler := NewOllamaHandler(nil)
	tracker := services.NewHealthTracker(services.HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    time.Hour,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)

	url := "http://example.test:11434"
	var failures, lastStatus int
	var lastBody []byte
	handler.recordRaceFailure(raceResult{
		err:          context.Canceled,
		endpointURL:  url,
		raceCanceled: true,
	}, tracker, &failures, &lastStatus, &lastBody)

	if tracker.IsDisabled(url) {
		t.Fatal("canceled race loser should not disable endpoint")
	}
}

func TestGlmFamilyAlternates(t *testing.T) {
	alts := glmFamilyAlternates("glm-5.1")
	if len(alts) == 0 || alts[0] != "glm-5.2" {
		t.Fatalf("unexpected glm-5.1 alternates: %v", alts)
	}
	if got := glmFamilyAlternates("llama3"); got != nil {
		t.Fatalf("expected nil alternates for non-glm model, got %v", got)
	}
}

func TestStripThinkingText(t *testing.T) {
	in := "<think>secret</think>\n\nFinal answer"
	if got := stripThinkingText(in); got != "Final answer" {
		t.Fatalf("got %q", got)
	}
}

func TestOllamaHandler_ChatCompletions_InvalidJSON(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChatCompletions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_ChatCompletions_MissingModel(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChatCompletions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_Completions_InvalidJSON(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/completions", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_Completions_MissingModel(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"prompt": "Complete this",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/completions", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_Models_WithNilDB(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestOllamaHandler_ChatCompletions_EmptyBody(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer([]byte("")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChatCompletions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_Completions_EmptyBody(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/v1/completions", bytes.NewBuffer([]byte("")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_ChatCompletions_WithModel_NoEndpoints(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestOllamaHandler_Completions_WithModel_NoEndpoints(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestOllamaHandler_ChatCompletions_ModelWithoutTag(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestOllamaHandler_ChatCompletions_StreamingMode(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestParseModel_WithTag(t *testing.T) {
	name, tag := parseModel("llama2:7b")
	if name != "llama2" {
		t.Errorf("Expected name 'llama2', got '%s'", name)
	}
	if tag != "7b" {
		t.Errorf("Expected tag '7b', got '%s'", tag)
	}
}

func TestParseModel_WithoutTag(t *testing.T) {
	name, tag := parseModel("llama2")
	if name != "llama2" {
		t.Errorf("Expected name 'llama2', got '%s'", name)
	}
	if tag != "latest" {
		t.Errorf("Expected tag 'latest', got '%s'", tag)
	}
}

func TestParseModel_EmptyString(t *testing.T) {
	name, tag := parseModel("")
	if name != "" {
		t.Errorf("Expected empty name, got '%s'", name)
	}
	if tag != "latest" {
		t.Errorf("Expected tag 'latest', got '%s'", tag)
	}
}

func TestParseModel_MultipleColons(t *testing.T) {
	name, tag := parseModel("name:tag:extra")
	if name != "name" {
		t.Errorf("Expected name 'name', got '%s'", name)
	}
	if tag != "tag:extra" {
		t.Errorf("Expected tag 'tag:extra', got '%s'", tag)
	}
}

func TestParseModel_OnlyColon(t *testing.T) {
	name, tag := parseModel(":")
	if name != "" {
		t.Errorf("Expected empty name, got '%s'", name)
	}
	if tag != "" {
		t.Errorf("Expected empty tag, got '%s'", tag)
	}
}

func TestParseModel_WithSpecialChars(t *testing.T) {
	name, tag := parseModel("mistral-7b:q4_0")
	if name != "mistral-7b" {
		t.Errorf("Expected name 'mistral-7b', got '%s'", name)
	}
	if tag != "q4_0" {
		t.Errorf("Expected tag 'q4_0', got '%s'", tag)
	}
}

func TestOllamaHandler_ProxyRequest_EmptyModel(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"model": "",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChatCompletions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestOllamaHandler_Completions_EmptyModel(t *testing.T) {
	handler := NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"model":  "",
		"prompt": "Hello",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/completions", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Completions(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestMessagesContainImages(t *testing.T) {
	messages := []interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": "What is this?",
				},
				map[string]interface{}{
					"type": "image_url",
					"image_url": map[string]interface{}{
						"url": "data:image/png;base64,abc",
					},
				},
			},
		},
	}
	if !messagesContainImages(messages) {
		t.Fatal("expected image content to be detected")
	}

	textOnly := []interface{}{
		map[string]interface{}{
			"role":    "user",
			"content": "follow up question",
		},
	}
	if messagesContainImages(textOnly) {
		t.Fatal("expected text-only history to have no images")
	}
}

func TestIsQuotaExceededError(t *testing.T) {
	testCases := []struct {
		statusCode int
		body       string
		expected   bool
	}{
		{429, `{"error":{"message":"You exceeded your current quota, please check your plan and billing details.","type":"insufficient_quota","param":null,"code":"insufficient_quota"}}`, true},
		{400, `{"error":{"message":"Insufficient Balance","type":"billing_not_active"}}`, true},
		{403, `{"error":{"message":"free_trial_quota_exceeded"}}`, true},
		{403, `Forbidden: this model requires a subscription, upgrade for access: https://ollama.com/upgrade`, true},
		{429, `{"error":"you (novelantig7) have reached your session usage limit, upgrade for higher limits: https://ollama.com/upgrade or add extra usage: https://ollama.com/settings"}`, true},
		{403, `{"error":{"message":"this model requires a subscription, upgrade for access: https://ollama.com/upgrade"}}`, true},
		{429, `rate limit exceeded, try again in 5s`, false},
		{200, `{"choices":[]}`, false},
		{500, `internal server error`, false},
	}

	for _, tc := range testCases {
		res := isQuotaExceededError(tc.statusCode, []byte(tc.body))
		if res != tc.expected {
			t.Errorf("Expected isQuotaExceededError(%d, %q) = %v, got %v", tc.statusCode, tc.body, tc.expected, res)
		}
	}
}
