package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/services"
)

func TestAdminHealthStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Initialize health tracker
	services.InitHealthTracker(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/admin/health", nil)

	AdminHealthStatus(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Check response structure
	if _, ok := response["enabled"]; !ok {
		t.Error("Expected 'enabled' field in response")
	}
	if _, ok := response["config"]; !ok {
		t.Error("Expected 'config' field in response")
	}
	if _, ok := response["summary"]; !ok {
		t.Error("Expected 'summary' field in response")
	}
	if _, ok := response["endpoints"]; !ok {
		t.Error("Expected 'endpoints' field in response")
	}

	// Check config structure
	config, ok := response["config"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected config to be an object")
	}
	if _, ok := config["disable_threshold"]; !ok {
		t.Error("Expected 'disable_threshold' in config")
	}
	if _, ok := config["disable_duration"]; !ok {
		t.Error("Expected 'disable_duration' in config")
	}
	if _, ok := config["probe_interval"]; !ok {
		t.Error("Expected 'probe_interval' in config")
	}

	// Check summary structure
	summary, ok := response["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected summary to be an object")
	}
	if _, ok := summary["total_endpoints"]; !ok {
		t.Error("Expected 'total_endpoints' in summary")
	}
	if _, ok := summary["healthy_endpoints"]; !ok {
		t.Error("Expected 'healthy_endpoints' in summary")
	}
	if _, ok := summary["disabled_endpoints"]; !ok {
		t.Error("Expected 'disabled_endpoints' in summary")
	}
}

func TestAdminResetEndpointHealth_MissingURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	services.InitHealthTracker(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/health/reset", nil)

	AdminResetEndpointHealth(c)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Expected 'error' field in response")
	}
}

func TestAdminResetEndpointHealth_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tracker := services.NewHealthTracker(services.HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	services.SetGlobalHealthTracker(tracker)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/admin/health/reset?url=http://test:8080", nil)

	// First, make the endpoint unhealthy
	for i := 0; i < 8; i++ {
		tracker.RecordFailure("http://test:8080")
	}

	AdminResetEndpointHealth(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["message"] != "Endpoint health reset" {
		t.Errorf("Expected success message, got %v", response["message"])
	}
	if response["url"] != "http://test:8080" {
		t.Errorf("Expected url to match, got %v", response["url"])
	}

	// Verify the endpoint was actually reset
	health := tracker.GetHealth("http://test:8080")
	if health == nil {
		t.Fatal("Expected health entry to exist")
	}
	if health.Disabled {
		t.Error("Expected endpoint to be re-enabled after reset")
	}
}
