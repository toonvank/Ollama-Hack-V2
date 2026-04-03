package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAIModelStruct(t *testing.T) {
	now := time.Now()
	model := AIModel{
		ID:        1,
		Name:      "llama2",
		Tag:       "7b",
		Enabled:   true,
		CreatedAt: now,
	}

	if model.ID != 1 {
		t.Errorf("Expected ID 1, got %d", model.ID)
	}
	if model.Name != "llama2" {
		t.Errorf("Expected Name 'llama2', got '%s'", model.Name)
	}
	if model.Tag != "7b" {
		t.Errorf("Expected Tag '7b', got '%s'", model.Tag)
	}
	if !model.Enabled {
		t.Error("Expected Enabled to be true")
	}
}

func TestAIModelInfo(t *testing.T) {
	now := time.Now()
	info := AIModelInfo{
		ID:        1,
		Name:      "codellama",
		Tag:       "13b",
		Enabled:   true,
		CreatedAt: now,
		Endpoints: 5,
	}

	if info.Endpoints != 5 {
		t.Errorf("Expected Endpoints 5, got %d", info.Endpoints)
	}
}

func TestAIModelToggle(t *testing.T) {
	toggle := AIModelToggle{
		Enabled: true,
	}

	if !toggle.Enabled {
		t.Error("Expected Enabled to be true")
	}

	toggle.Enabled = false
	if toggle.Enabled {
		t.Error("Expected Enabled to be false")
	}
}

func TestAIModelPerformance(t *testing.T) {
	tps := 25.5
	connTime := 1.2
	perf := AIModelPerformance{
		EndpointID:        1,
		EndpointName:      "Local Server",
		Status:            "available",
		TokenPerSecond:    &tps,
		MaxConnectionTime: &connTime,
	}

	if perf.EndpointID != 1 {
		t.Errorf("Expected EndpointID 1, got %d", perf.EndpointID)
	}
	if perf.EndpointName != "Local Server" {
		t.Errorf("Expected EndpointName 'Local Server', got '%s'", perf.EndpointName)
	}
	if perf.Status != "available" {
		t.Errorf("Expected Status 'available', got '%s'", perf.Status)
	}
	if perf.TokenPerSecond == nil || *perf.TokenPerSecond != 25.5 {
		t.Error("Expected TokenPerSecond to be 25.5")
	}
	if perf.MaxConnectionTime == nil || *perf.MaxConnectionTime != 1.2 {
		t.Error("Expected MaxConnectionTime to be 1.2")
	}
}

func TestAIModelPerformanceWithNils(t *testing.T) {
	perf := AIModelPerformance{
		EndpointID:        1,
		EndpointName:      "Test Server",
		Status:            "unavailable",
		TokenPerSecond:    nil,
		MaxConnectionTime: nil,
	}

	if perf.TokenPerSecond != nil {
		t.Error("Expected TokenPerSecond to be nil")
	}
	if perf.MaxConnectionTime != nil {
		t.Error("Expected MaxConnectionTime to be nil")
	}
}

func TestAIModelDetail(t *testing.T) {
	now := time.Now()
	tps := 30.0
	detail := AIModelDetail{
		AIModelInfo: AIModelInfo{
			ID:        1,
			Name:      "mistral",
			Tag:       "7b",
			Enabled:   true,
			CreatedAt: now,
			Endpoints: 3,
		},
		Performances: []AIModelPerformance{
			{
				EndpointID:     1,
				EndpointName:   "Server 1",
				Status:         "available",
				TokenPerSecond: &tps,
			},
			{
				EndpointID:     2,
				EndpointName:   "Server 2",
				Status:         "available",
				TokenPerSecond: &tps,
			},
		},
	}

	if detail.Name != "mistral" {
		t.Errorf("Expected Name 'mistral', got '%s'", detail.Name)
	}
	if len(detail.Performances) != 2 {
		t.Errorf("Expected 2 performances, got %d", len(detail.Performances))
	}
}

func TestAIModelJSONSerialization(t *testing.T) {
	now := time.Now()
	model := AIModel{
		ID:        1,
		Name:      "llama2",
		Tag:       "latest",
		Enabled:   true,
		CreatedAt: now,
	}

	data, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("Failed to marshal AIModel: %v", err)
	}

	var parsed AIModel
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal AIModel: %v", err)
	}

	if parsed.Name != model.Name {
		t.Errorf("Expected Name '%s', got '%s'", model.Name, parsed.Name)
	}
	if parsed.Tag != model.Tag {
		t.Errorf("Expected Tag '%s', got '%s'", model.Tag, parsed.Tag)
	}
}

func TestAIModelInfoJSONSerialization(t *testing.T) {
	now := time.Now()
	info := AIModelInfo{
		ID:        1,
		Name:      "phi",
		Tag:       "2",
		Enabled:   false,
		CreatedAt: now,
		Endpoints: 10,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal AIModelInfo: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, `"endpoints":10`) {
		t.Error("Endpoints should be serialized as 10")
	}
	if !contains(jsonStr, `"enabled":false`) {
		t.Error("Enabled should be serialized as false")
	}
}

func TestAIModelDetailJSONSerialization(t *testing.T) {
	now := time.Now()
	detail := AIModelDetail{
		AIModelInfo: AIModelInfo{
			ID:        1,
			Name:      "gemma",
			Tag:       "7b",
			Enabled:   true,
			CreatedAt: now,
			Endpoints: 2,
		},
		Performances: []AIModelPerformance{},
	}

	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("Failed to marshal AIModelDetail: %v", err)
	}

	var parsed AIModelDetail
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal AIModelDetail: %v", err)
	}

	if parsed.Name != "gemma" {
		t.Errorf("Expected Name 'gemma', got '%s'", parsed.Name)
	}
}
