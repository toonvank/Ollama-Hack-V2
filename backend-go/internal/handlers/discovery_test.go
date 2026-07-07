package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/services"
)

func TestDiscoveryHandler_TriggerManualScan_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	scanner := services.NewDiscoveryScanner(&database.DB{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/discovery/scan", bytes.NewBufferString("invalid json"))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewDiscoveryHandler(&database.DB{}, scanner)
	handler.TriggerManualScan(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDiscoveryHandler_TriggerManualScan_MissingField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	scanner := services.NewDiscoveryScanner(&database.DB{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/discovery/scan", bytes.NewBufferString(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewDiscoveryHandler(&database.DB{}, scanner)
	handler.TriggerManualScan(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestDiscoveryHandler_TriggerManualScan_ValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	scanner := services.NewDiscoveryScanner(&database.DB{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := map[string]string{"ip_range": "192.168.1.0/24"}
	jsonBody, _ := json.Marshal(body)
	c.Request = httptest.NewRequest("POST", "/discovery/scan", bytes.NewBuffer(jsonBody))
	c.Request.Header.Set("Content-Type", "application/json")

	handler := NewDiscoveryHandler(&database.DB{}, scanner)
	handler.TriggerManualScan(c)

	// Should return 200 OK (scan is triggered asynchronously)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "Manual scan triggered" {
		t.Errorf("Expected success message, got %v", response["message"])
	}
	if response["ip_range"] != "192.168.1.0/24" {
		t.Errorf("Expected ip_range to match, got %v", response["ip_range"])
	}
}

func TestDiscoveryHandler_GetScanStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	scanner := services.NewDiscoveryScanner(&database.DB{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/discovery/status", nil)

	handler := NewDiscoveryHandler(&database.DB{}, scanner)
	handler.GetScanStatus(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["status"] != "running" {
		t.Errorf("Expected status 'running', got %v", response["status"])
	}
	if response["message"] != "Discovery scanner is active" {
		t.Errorf("Expected message, got %v", response["message"])
	}
}

func TestNewDiscoveryHandler(t *testing.T) {
	db := &database.DB{}
	scanner := services.NewDiscoveryScanner(db)

	handler := NewDiscoveryHandler(db, scanner)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}
	if handler.db != db {
		t.Error("Expected handler to have reference to db")
	}
	if handler.scanner != scanner {
		t.Error("Expected handler to have reference to scanner")
	}
}
