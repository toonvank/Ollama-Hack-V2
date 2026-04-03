package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEndpointStruct(t *testing.T) {
	now := time.Now()
	endpoint := Endpoint{
		ID:        1,
		URL:       "http://localhost:11434",
		Name:      "Local Ollama",
		Status:    "available",
		CreatedAt: now,
	}

	if endpoint.ID != 1 {
		t.Errorf("Expected ID 1, got %d", endpoint.ID)
	}
	if endpoint.URL != "http://localhost:11434" {
		t.Errorf("Expected URL 'http://localhost:11434', got '%s'", endpoint.URL)
	}
	if endpoint.Name != "Local Ollama" {
		t.Errorf("Expected Name 'Local Ollama', got '%s'", endpoint.Name)
	}
	if endpoint.Status != "available" {
		t.Errorf("Expected Status 'available', got '%s'", endpoint.Status)
	}
}

func TestEndpointPerformance(t *testing.T) {
	now := time.Now()
	version := "0.1.30"
	perf := EndpointPerformance{
		ID:            1,
		Status:        "available",
		OllamaVersion: &version,
		CreatedAt:     now,
	}

	if perf.ID != 1 {
		t.Errorf("Expected ID 1, got %d", perf.ID)
	}
	if perf.Status != "available" {
		t.Errorf("Expected Status 'available', got '%s'", perf.Status)
	}
	if perf.OllamaVersion == nil || *perf.OllamaVersion != "0.1.30" {
		t.Error("Expected OllamaVersion to be '0.1.30'")
	}
}

func TestEndpointWithAIModelCount(t *testing.T) {
	now := time.Now()
	taskStatus := "done"
	ep := EndpointWithAIModelCount{
		ID:                    1,
		URL:                   "http://example.com:11434",
		Name:                  "Example",
		Status:                "available",
		CreatedAt:             now,
		RecentPerformances:    []EndpointPerformance{},
		TotalAIModelCount:     10,
		AvailableAIModelCount: 8,
		TaskStatus:            &taskStatus,
	}

	if ep.TotalAIModelCount != 10 {
		t.Errorf("Expected TotalAIModelCount 10, got %d", ep.TotalAIModelCount)
	}
	if ep.AvailableAIModelCount != 8 {
		t.Errorf("Expected AvailableAIModelCount 8, got %d", ep.AvailableAIModelCount)
	}
	if ep.TaskStatus == nil || *ep.TaskStatus != "done" {
		t.Error("Expected TaskStatus to be 'done'")
	}
}

func TestEndpointCreate(t *testing.T) {
	create := EndpointCreate{
		URL:  "http://newhost:11434",
		Name: "New Endpoint",
	}

	if create.URL != "http://newhost:11434" {
		t.Errorf("Expected URL 'http://newhost:11434', got '%s'", create.URL)
	}
	if create.Name != "New Endpoint" {
		t.Errorf("Expected Name 'New Endpoint', got '%s'", create.Name)
	}
}

func TestEndpointUpdate(t *testing.T) {
	name := "Updated Name"
	url := "http://updated:11434"
	update := EndpointUpdate{
		Name: &name,
		URL:  &url,
	}

	if update.Name == nil || *update.Name != "Updated Name" {
		t.Error("Expected Name to be 'Updated Name'")
	}
	if update.URL == nil || *update.URL != "http://updated:11434" {
		t.Error("Expected URL to be 'http://updated:11434'")
	}
}

func TestEndpointBatchCreate(t *testing.T) {
	batch := EndpointBatchCreate{
		Endpoints: []EndpointCreate{
			{URL: "http://host1:11434", Name: "Host 1"},
			{URL: "http://host2:11434", Name: "Host 2"},
			{URL: "http://host3:11434", Name: "Host 3"},
		},
	}

	if len(batch.Endpoints) != 3 {
		t.Errorf("Expected 3 endpoints, got %d", len(batch.Endpoints))
	}
}

func TestBatchOperationResult(t *testing.T) {
	result := BatchOperationResult{
		SuccessCount: 5,
		FailedCount:  2,
		FailedIDs: map[string]string{
			"1": "Not found",
			"3": "Server error",
		},
	}

	if result.SuccessCount != 5 {
		t.Errorf("Expected SuccessCount 5, got %d", result.SuccessCount)
	}
	if result.FailedCount != 2 {
		t.Errorf("Expected FailedCount 2, got %d", result.FailedCount)
	}
	if len(result.FailedIDs) != 2 {
		t.Errorf("Expected 2 failed IDs, got %d", len(result.FailedIDs))
	}
}

func TestEndpointBatchOperation(t *testing.T) {
	op := EndpointBatchOperation{
		EndpointIDs: []int{1, 2, 3, 4, 5},
	}

	if len(op.EndpointIDs) != 5 {
		t.Errorf("Expected 5 endpoint IDs, got %d", len(op.EndpointIDs))
	}
}

func TestEndpointTestTask(t *testing.T) {
	now := time.Now()
	lastTried := now.Add(-time.Minute)
	task := EndpointTestTask{
		ID:          1,
		EndpointID:  2,
		Status:      "pending",
		ScheduledAt: now,
		LastTried:   &lastTried,
		CreatedAt:   now,
	}

	if task.ID != 1 {
		t.Errorf("Expected ID 1, got %d", task.ID)
	}
	if task.EndpointID != 2 {
		t.Errorf("Expected EndpointID 2, got %d", task.EndpointID)
	}
	if task.Status != "pending" {
		t.Errorf("Expected Status 'pending', got '%s'", task.Status)
	}
	if task.LastTried == nil {
		t.Error("Expected LastTried to be set")
	}
}

func TestEndpointJSONSerialization(t *testing.T) {
	now := time.Now()
	endpoint := Endpoint{
		ID:        1,
		URL:       "http://test:11434",
		Name:      "Test",
		Status:    "pending",
		CreatedAt: now,
	}

	data, err := json.Marshal(endpoint)
	if err != nil {
		t.Fatalf("Failed to marshal Endpoint: %v", err)
	}

	var parsed Endpoint
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal Endpoint: %v", err)
	}

	if parsed.URL != endpoint.URL {
		t.Errorf("Expected URL '%s', got '%s'", endpoint.URL, parsed.URL)
	}
}

func TestEndpointPerformanceWithNilVersion(t *testing.T) {
	now := time.Now()
	perf := EndpointPerformance{
		ID:            1,
		Status:        "unavailable",
		OllamaVersion: nil,
		CreatedAt:     now,
	}

	if perf.OllamaVersion != nil {
		t.Error("Expected OllamaVersion to be nil")
	}
}
