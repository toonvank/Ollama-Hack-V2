package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestUserStruct(t *testing.T) {
	now := time.Now()
	planID := 1
	user := User{
		ID:             1,
		Username:       "testuser",
		HashedPassword: "hashedpassword123",
		IsAdmin:        true,
		PlanID:         &planID,
		CreatedAt:      now,
	}

	if user.ID != 1 {
		t.Errorf("Expected ID 1, got %d", user.ID)
	}
	if user.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", user.Username)
	}
	if !user.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
	if user.PlanID == nil || *user.PlanID != 1 {
		t.Error("Expected PlanID to be 1")
	}
}

func TestUserJSONSerialization(t *testing.T) {
	now := time.Now()
	user := User{
		ID:             1,
		Username:       "testuser",
		HashedPassword: "secrethash",
		IsAdmin:        false,
		CreatedAt:      now,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	// HashedPassword should not appear in JSON (json:"-")
	jsonStr := string(data)
	if contains(jsonStr, "secrethash") {
		t.Error("HashedPassword should not be serialized to JSON")
	}
	if !contains(jsonStr, "testuser") {
		t.Error("Username should be serialized to JSON")
	}
}

func TestUserCreate(t *testing.T) {
	planID := 2
	create := UserCreate{
		Username: "newuser",
		Password: "password123",
		IsAdmin:  true,
		PlanID:   &planID,
	}

	if create.Username != "newuser" {
		t.Errorf("Expected Username 'newuser', got '%s'", create.Username)
	}
	if create.Password != "password123" {
		t.Errorf("Expected Password 'password123', got '%s'", create.Password)
	}
}

func TestUserUpdate(t *testing.T) {
	username := "updateduser"
	password := "newpassword"
	isAdmin := true

	update := UserUpdate{
		Username: &username,
		Password: &password,
		IsAdmin:  &isAdmin,
	}

	if update.Username == nil || *update.Username != "updateduser" {
		t.Error("Expected Username to be 'updateduser'")
	}
	if update.Password == nil || *update.Password != "newpassword" {
		t.Error("Expected Password to be 'newpassword'")
	}
	if update.IsAdmin == nil || !*update.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestUserInfo(t *testing.T) {
	planID := 1
	info := UserInfo{
		ID:       1,
		Username: "testuser",
		IsAdmin:  true,
		PlanID:   &planID,
		PlanName: "Premium",
	}

	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Failed to marshal UserInfo: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "Premium") {
		t.Error("PlanName should be serialized to JSON")
	}
}

func TestLoginRequest(t *testing.T) {
	req := LoginRequest{
		Username: "admin",
		Password: "adminpass",
	}

	if req.Username != "admin" {
		t.Errorf("Expected Username 'admin', got '%s'", req.Username)
	}
	if req.Password != "adminpass" {
		t.Errorf("Expected Password 'adminpass', got '%s'", req.Password)
	}
}

func TestTokenResponse(t *testing.T) {
	resp := TokenResponse{
		AccessToken: "jwt-token-here",
		TokenType:   "bearer",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal TokenResponse: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "jwt-token-here") {
		t.Error("AccessToken should be serialized")
	}
	if !contains(jsonStr, "bearer") {
		t.Error("TokenType should be serialized")
	}
}

func TestChangePasswordRequest(t *testing.T) {
	req := ChangePasswordRequest{
		OldPassword: "oldpass123",
		NewPassword: "newpass456",
	}

	if req.OldPassword != "oldpass123" {
		t.Errorf("Expected OldPassword 'oldpass123', got '%s'", req.OldPassword)
	}
	if req.NewPassword != "newpass456" {
		t.Errorf("Expected NewPassword 'newpass456', got '%s'", req.NewPassword)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
