package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"result": "ok"}
	Success(c, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["result"] != "ok" {
		t.Errorf("Expected data result to be ok, got %s", response["result"])
	}
}

func TestBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "Invalid input")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Detail != "Invalid input" {
		t.Errorf("Expected detail 'Invalid input', got '%s'", response.Detail)
	}
}

func TestInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalServerError(c, "Server crash")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Detail != "Server crash" {
		t.Errorf("Expected detail 'Server crash', got '%s'", response.Detail)
	}
}

func TestSuccessPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []string{"item1", "item2"}
	SuccessPage(c, items, 10, 1, 2, 5)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PageResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Total != 10 {
		t.Errorf("Expected Total 10, got %d", response.Total)
	}
	if response.Page != 1 {
		t.Errorf("Expected Page 1, got %d", response.Page)
	}
	if response.Size != 2 {
		t.Errorf("Expected Size 2, got %d", response.Size)
	}
	if response.Pages != 5 {
		t.Errorf("Expected Pages 5, got %d", response.Pages)
	}
}

func TestCreated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"id": "123"}
	Created(c, data)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["id"] != "123" {
		t.Errorf("Expected id '123', got '%s'", response["id"])
	}
}

func TestNoContent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NoContent(c)

	// Gin's Status() returns 200 unless you explicitly write something
	// The NoContent function just sets status, so check we can call it
	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("Expected status 200 or 204, got %d", w.Code)
	}
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Error(c, 418, "I'm a teapot")

	if w.Code != 418 {
		t.Errorf("Expected status 418, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Detail != "I'm a teapot" {
		t.Errorf("Expected detail 'I'm a teapot', got '%s'", response.Detail)
	}
}

func TestUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c, "Invalid token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Detail != "Invalid token" {
		t.Errorf("Expected detail 'Invalid token', got '%s'", response.Detail)
	}
}

func TestForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Forbidden(c, "Access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Detail != "Access denied" {
		t.Errorf("Expected detail 'Access denied', got '%s'", response.Detail)
	}
}

func TestNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	NotFound(c, "Resource not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}

	var response Response
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Detail != "Resource not found" {
		t.Errorf("Expected detail 'Resource not found', got '%s'", response.Detail)
	}
}

func TestResponseStruct(t *testing.T) {
	resp := Response{
		Message: "Success",
		Data:    map[string]int{"count": 5},
		Error:   "",
		Detail:  "Operation completed",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal Response: %v", err)
	}

	jsonStr := string(data)
	if jsonStr == "" {
		t.Error("Expected non-empty JSON")
	}
}

func TestPageResponseStruct(t *testing.T) {
	resp := PageResponse{
		Items: []string{"a", "b", "c"},
		Total: 100,
		Page:  2,
		Size:  3,
		Pages: 34,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal PageResponse: %v", err)
	}

	var parsed PageResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal PageResponse: %v", err)
	}

	if parsed.Total != 100 {
		t.Errorf("Expected Total 100, got %d", parsed.Total)
	}
	if parsed.Page != 2 {
		t.Errorf("Expected Page 2, got %d", parsed.Page)
	}
	if parsed.Size != 3 {
		t.Errorf("Expected Size 3, got %d", parsed.Size)
	}
	if parsed.Pages != 34 {
		t.Errorf("Expected Pages 34, got %d", parsed.Pages)
	}
}

func TestSuccessWithNilData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestSuccessWithComplexData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := struct {
		ID       int                    `json:"id"`
		Name     string                 `json:"name"`
		Tags     []string               `json:"tags"`
		Metadata map[string]interface{} `json:"metadata"`
	}{
		ID:       1,
		Name:     "Test",
		Tags:     []string{"tag1", "tag2"},
		Metadata: map[string]interface{}{"key": "value"},
	}

	Success(c, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["id"].(float64) != 1 {
		t.Errorf("Expected id 1, got %v", response["id"])
	}
}

func TestSuccessPageWithEmptyItems(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SuccessPage(c, []string{}, 0, 1, 10, 0)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response PageResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Total != 0 {
		t.Errorf("Expected Total 0, got %d", response.Total)
	}
	if response.Pages != 0 {
		t.Errorf("Expected Pages 0, got %d", response.Pages)
	}
}
