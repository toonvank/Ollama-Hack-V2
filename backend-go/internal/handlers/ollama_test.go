package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

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
