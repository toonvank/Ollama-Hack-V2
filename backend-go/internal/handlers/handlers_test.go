package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// Helper to create test context
func createTestContext(method, url string, body interface{}) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var bodyReader *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonBytes)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	c.Request, _ = http.NewRequest(method, url, bodyReader)
	c.Request.Header.Set("Content-Type", "application/json")

	return w, c
}

// AuthHandler tests
func TestNewAuthHandler(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	if handler == nil {
		t.Error("Expected AuthHandler to be created")
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// UserHandler tests
func TestNewUserHandler(t *testing.T) {
	handler := NewUserHandler(nil)
	if handler == nil {
		t.Error("Expected UserHandler to be created")
	}
}

func TestUserHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUserHandler_Update_InvalidJSON(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/users/1", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestUserHandler_Update_NoFields(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/users/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// APIKeyHandler tests
func TestNewAPIKeyHandler(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	if handler == nil {
		t.Error("Expected APIKeyHandler to be created")
	}
}

func TestAPIKeyHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("user_id", 1)

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestAPIKeyHandler_GetStats(t *testing.T) {
	handler := NewAPIKeyHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/api-keys/1/stats", nil)
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.GetStats(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["id"] != "1" {
		t.Errorf("Expected id '1', got '%v'", response["id"])
	}
}

// PlanHandler tests
func TestNewPlanHandler(t *testing.T) {
	handler := NewPlanHandler(nil)
	if handler == nil {
		t.Error("Expected PlanHandler to be created")
	}
}

func TestPlanHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_Update_InvalidJSON(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/plans/1", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestPlanHandler_Update_NoFields(t *testing.T) {
	handler := NewPlanHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/plans/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// EndpointHandler tests
func TestNewEndpointHandler(t *testing.T) {
	handler := NewEndpointHandler(nil)
	if handler == nil {
		t.Error("Expected EndpointHandler to be created")
	}
}

func TestEndpointHandler_Create_InvalidJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_InvalidJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchCreate_Empty(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoints": []interface{}{},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestEndpointHandler_Update_InvalidJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/endpoints/1", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_Update_NoFields(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/endpoints/1", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Update(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchDelete_InvalidJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/batch", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchTest_InvalidJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/test", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchTest(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_TriggerTest_InvalidID(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/abc/test", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.TriggerTest(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_GetTask_InvalidID(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/endpoints/abc/task", nil)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}

	handler.GetTask(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchGetTasks_InvalidJSON(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/tasks", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchGetTasks(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchGetTasks_Empty(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoint_ids": []int{},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/tasks", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchGetTasks(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// AIModelHandler tests
func TestNewAIModelHandler(t *testing.T) {
	handler := NewAIModelHandler(nil)
	if handler == nil {
		t.Error("Expected AIModelHandler to be created")
	}
}

func TestAIModelHandler_Toggle_InvalidJSON(t *testing.T) {
	handler := NewAIModelHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("PUT", "/models/1/toggle", bytes.NewBuffer([]byte("invalid json")))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "1"}}

	handler.Toggle(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// OllamaHandler tests
func TestNewOllamaHandler(t *testing.T) {
	handler := NewOllamaHandler(nil)
	if handler == nil {
		t.Error("Expected OllamaHandler to be created")
	}
}

// Additional validation tests for handlers

func TestUserHandler_Create_EmptyUsername(t *testing.T) {
	handler := NewUserHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"username": "",
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Create(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for empty username, got %d", w.Code)
	}
}

func TestAuthHandler_Login_MissingUsername(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"password": "password123",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.Login(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing username, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword_MissingOldPassword(t *testing.T) {
	handler := NewAuthHandler(nil, nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", 1)

	body := map[string]interface{}{
		"new_password": "newpass123456789",
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("PUT", "/change-password", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ChangePassword(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing old_password, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchDelete_Empty(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoint_ids": []int{},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty batch delete, got %d", w.Code)
	}
}

func TestEndpointHandler_BatchTest_Empty(t *testing.T) {
	handler := NewEndpointHandler(nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoint_ids": []int{},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch/test", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchTest(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for empty batch test, got %d", w.Code)
	}
}
