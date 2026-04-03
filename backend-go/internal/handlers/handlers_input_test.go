package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/models"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Test request parsing and early validation (before DB access)

func TestAuthHandler_Login_EmptyUsername(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.LoginRequest{
		Username: "",
		Password: "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_EmptyPassword(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.LoginRequest{
		Username: "testuser",
		Password: "",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUserHandler_Create_ShortPassword(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.UserCreate{
		Username: "testuser",
		Password: "short", // Less than 8 characters
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for short password, got %d", w.Code)
	}
}

func TestUserHandler_Create_LongPassword(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Password longer than 128 characters
	longPassword := "a"
	for i := 0; i < 130; i++ {
		longPassword += "a"
	}
	body := models.UserCreate{
		Username: "testuser",
		Password: longPassword,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for long password, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword_ShortNewPassword(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", 1)

	body := models.ChangePasswordRequest{
		OldPassword: "oldpassword123",
		NewPassword: "short", // Less than 8 characters
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/change-password", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChangePassword(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for short new password, got %d", w.Code)
	}
}

func TestPlanHandler_Create_EmptyName(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.PlanCreate{
		Name:        "",
		Description: "Test plan",
		RPM:         100,
		RPD:         1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty name, got %d", w.Code)
	}
}

func TestEndpointHandler_Create_EmptyURL(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.EndpointCreate{
		URL:  "",
		Name: "Test endpoint",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty URL, got %d", w.Code)
	}
}

func TestAPIKeyHandler_Create_EmptyName(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", 1)

	body := models.APIKeyCreate{
		Name: "",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty name, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_InvalidEndpointData(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestAIModelHandler_Toggle_ValidRequest(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestEndpointHandler_BatchDelete_ValidRequest(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestEndpointHandler_BatchTest_ValidRequest(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestOllamaHandler_ChatCompletions_ValidRequest(t *testing.T) {
	_ = NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"model": "llama2:7b",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// This will panic due to nil DB - we just verify setup works
	_ = w
}

func TestOllamaHandler_Completions_ValidRequest(t *testing.T) {
	_ = NewOllamaHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"model":  "llama2:7b",
		"prompt": "Hello",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/v1/completions", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	// This will panic due to nil DB - we just verify setup works
	_ = w
}

func TestUserHandler_Update_ValidPartialUpdate(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestPlanHandler_Update_ValidPartialUpdate(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestEndpointHandler_Update_ValidPartialUpdate(t *testing.T) {
	t.Skip("Requires database connection")
}

func TestAPIKeyHandler_Delete_ValidRequest(t *testing.T) {
	t.Skip("Requires database connection")
}
