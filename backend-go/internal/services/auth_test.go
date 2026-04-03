package services

import (
	"testing"
	"time"

	"github.com/timlzh/ollama-hack/internal/config"
	"github.com/timlzh/ollama-hack/internal/models"
)

func TestGenerateAndValidateToken(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "test_secret_key",
			AccessTokenExpireMinutes: 1,
		},
	}

	authService := NewAuthService(nil, cfg)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		IsAdmin:  true,
	}

	token, err := authService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if token == "" {
		t.Fatalf("Expected token to not be empty")
	}

	claims, err := authService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Expected no error on validation, got %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("Expected user ID %d, got %d", user.ID, claims.UserID)
	}
	if claims.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, claims.Username)
	}
	if claims.IsAdmin != user.IsAdmin {
		t.Errorf("Expected IsAdmin %v, got %v", user.IsAdmin, claims.IsAdmin)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test_secret_key",
		},
	}

	authService := NewAuthService(nil, cfg)

	_, err := authService.ValidateToken("invalid.token.string")
	if err == nil {
		t.Fatalf("Expected error for invalid token, got none")
	}
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	// Let's create an expired token
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "test_secret_key",
			AccessTokenExpireMinutes: -1, // Expired immediately
		},
	}

	authService := NewAuthService(nil, cfg)

	user := &models.User{
		ID:       1,
		Username: "testuser",
	}

	token, err := authService.GenerateToken(user)
	if err != nil {
		t.Fatalf("Expected no error generating token, got %v", err)
	}

	time.Sleep(1 * time.Second) // Ensure it's definitely in the past

	_, err = authService.ValidateToken(token)
	if err == nil {
		t.Fatalf("Expected error for expired token, got none")
	}
}
