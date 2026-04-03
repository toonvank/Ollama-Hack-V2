package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAPIKeyStruct(t *testing.T) {
	now := time.Now()
	lastUsed := now.Add(-time.Hour)
	apiKey := APIKey{
		ID:         1,
		Key:        "sk-testkey123",
		Name:       "Test API Key",
		UserID:     1,
		LastUsedAt: &lastUsed,
		CreatedAt:  now,
	}

	if apiKey.ID != 1 {
		t.Errorf("Expected ID 1, got %d", apiKey.ID)
	}
	if apiKey.Key != "sk-testkey123" {
		t.Errorf("Expected Key 'sk-testkey123', got '%s'", apiKey.Key)
	}
	if apiKey.Name != "Test API Key" {
		t.Errorf("Expected Name 'Test API Key', got '%s'", apiKey.Name)
	}
	if apiKey.UserID != 1 {
		t.Errorf("Expected UserID 1, got %d", apiKey.UserID)
	}
	if apiKey.LastUsedAt == nil {
		t.Error("Expected LastUsedAt to be set")
	}
}

func TestAPIKeyJSONSerialization(t *testing.T) {
	now := time.Now()
	apiKey := APIKey{
		ID:        1,
		Key:       "sk-testkey123",
		Name:      "Test Key",
		UserID:    1,
		CreatedAt: now,
	}

	data, err := json.Marshal(apiKey)
	if err != nil {
		t.Fatalf("Failed to marshal APIKey: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "sk-testkey123") {
		t.Error("Key should be serialized to JSON")
	}
	if !contains(jsonStr, "Test Key") {
		t.Error("Name should be serialized to JSON")
	}
}

func TestAPIKeyCreate(t *testing.T) {
	create := APIKeyCreate{
		Name: "Production Key",
	}

	if create.Name != "Production Key" {
		t.Errorf("Expected Name 'Production Key', got '%s'", create.Name)
	}
}

func TestAPIKeyResponse(t *testing.T) {
	now := time.Now()
	resp := APIKeyResponse{
		ID:        1,
		Key:       "sk-newkey456",
		Name:      "New Key",
		UserID:    2,
		CreatedAt: now,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal APIKeyResponse: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "sk-newkey456") {
		t.Error("Key should be serialized in response")
	}
}

func TestAPIKeyInfo(t *testing.T) {
	now := time.Now()
	info := APIKeyInfo{
		ID:        1,
		Name:      "Info Key",
		UserID:    1,
		CreatedAt: now,
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal APIKeyInfo: %v", err)
	}

	jsonStr := string(data)
	// APIKeyInfo should NOT include the key
	if contains(jsonStr, "sk-") {
		t.Error("APIKeyInfo should not contain the actual key")
	}
	if !contains(jsonStr, "Info Key") {
		t.Error("Name should be serialized")
	}
}

func TestAPIKeyWithNilLastUsedAt(t *testing.T) {
	now := time.Now()
	apiKey := APIKey{
		ID:         1,
		Key:        "sk-test",
		Name:       "Test",
		UserID:     1,
		LastUsedAt: nil,
		CreatedAt:  now,
	}

	data, err := json.Marshal(apiKey)
	if err != nil {
		t.Fatalf("Failed to marshal APIKey with nil LastUsedAt: %v", err)
	}

	var parsed APIKey
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal APIKey: %v", err)
	}

	if parsed.LastUsedAt != nil {
		t.Error("Expected LastUsedAt to be nil after round-trip")
	}
}
