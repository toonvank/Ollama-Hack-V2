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

func TestAuthHandler_ChangePassword_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/change-password", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.ChangePassword(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_InitializeAdmin_InvalidJSON(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestEndpointHandler_BatchDelete_EmptyList(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.EndpointBatchOperation{
		EndpointIDs: []int{},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchTest_EmptyList(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := models.EndpointBatchOperation{
		EndpointIDs: []int{},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/test", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchTest(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUserHandler_Create_MissingRequiredFields(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing required username and password
	body := map[string]interface{}{
		"is_admin": true,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_Create_MissingRequiredFields(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing required name
	body := map[string]interface{}{
		"rpm": 100,
		"rpd": 1000,
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_Create_MissingRequiredFields(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing required URL
	body := map[string]interface{}{
		"name": "Test Endpoint",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAPIKeyHandler_Create_MissingRequiredFields(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", 1)

	// Missing required name
	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_MissingRequiredFields(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing required endpoints array (though empty is valid)
	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	// Should fail because endpoints is required
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingCredentials(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing username and password
	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword_MissingFields(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", 1)

	// Missing old_password and new_password
	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/change-password", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChangePassword(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_List_QueryParams(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestEndpointHandler_List_InvalidQueryParams(t *testing.T) {
	// This test is skipped as it requires DB access
	t.Skip("Requires database connection")
}

func TestUserHandler_PasswordValidation(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Password too short (min 8 chars)
	body := map[string]interface{}{
		"username": "testuser",
		"password": "short",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for short password, got %d", w.Code)
	}
}
