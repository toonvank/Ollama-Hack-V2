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
