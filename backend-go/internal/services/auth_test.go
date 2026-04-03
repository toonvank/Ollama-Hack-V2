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

func TestNewAuthService(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "test-secret",
			AccessTokenExpireMinutes: 30,
		},
	}

	service := NewAuthService(nil, cfg)
	if service == nil {
		t.Error("Expected AuthService to be created")
	}
	if service.cfg != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	cfg1 := &config.Config{
		App: config.AppConfig{
			SecretKey:                "secret-1",
			AccessTokenExpireMinutes: 30,
		},
	}

	cfg2 := &config.Config{
		App: config.AppConfig{
			SecretKey:                "secret-2",
			AccessTokenExpireMinutes: 30,
		},
	}

	service1 := NewAuthService(nil, cfg1)
	service2 := NewAuthService(nil, cfg2)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		IsAdmin:  true,
	}

	token, err := service1.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	_, err = service2.ValidateToken(token)
	if err == nil {
		t.Error("Expected error when validating with wrong secret")
	}
}

func TestClaimsStructure(t *testing.T) {
	claims := &Claims{
		UserID:   123,
		Username: "admin",
		IsAdmin:  true,
	}

	if claims.UserID != 123 {
		t.Errorf("Expected UserID 123, got %d", claims.UserID)
	}
	if claims.Username != "admin" {
		t.Errorf("Expected Username 'admin', got '%s'", claims.Username)
	}
	if !claims.IsAdmin {
		t.Error("Expected IsAdmin to be true")
	}
}

func TestTokenExpiration(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "test-secret-key",
			AccessTokenExpireMinutes: 30,
		},
	}

	service := NewAuthService(nil, cfg)

	user := &models.User{
		ID:       1,
		Username: "testuser",
		IsAdmin:  false,
	}

	token, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Check that expiration is set correctly (approximately 30 minutes from now)
	expectedExpiry := time.Now().Add(30 * time.Minute)
	actualExpiry := claims.ExpiresAt.Time

	// Allow 1 minute tolerance
	diff := actualExpiry.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("Token expiration is not within expected range")
	}
}

func TestGenerateTokenForAdminUser(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "admin-secret",
			AccessTokenExpireMinutes: 60,
		},
	}

	service := NewAuthService(nil, cfg)

	adminUser := &models.User{
		ID:       999,
		Username: "superadmin",
		IsAdmin:  true,
	}

	token, err := service.GenerateToken(adminUser)
	if err != nil {
		t.Fatalf("Failed to generate token for admin: %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate admin token: %v", err)
	}

	if !claims.IsAdmin {
		t.Error("Expected claims.IsAdmin to be true for admin user")
	}
	if claims.UserID != 999 {
		t.Errorf("Expected UserID 999, got %d", claims.UserID)
	}
}

func TestMultipleTokenGeneration(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "multi-secret",
			AccessTokenExpireMinutes: 30,
		},
	}

	service := NewAuthService(nil, cfg)

	user := &models.User{
		ID:       1,
		Username: "user1",
		IsAdmin:  false,
	}

	// Generate multiple tokens
	token1, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token1: %v", err)
	}

	token2, err := service.GenerateToken(user)
	if err != nil {
		t.Fatalf("Failed to generate token2: %v", err)
	}

	// Both tokens should be valid
	claims1, err := service.ValidateToken(token1)
	if err != nil {
		t.Fatalf("Failed to validate token1: %v", err)
	}

	claims2, err := service.ValidateToken(token2)
	if err != nil {
		t.Fatalf("Failed to validate token2: %v", err)
	}

	// Both should have the same user ID
	if claims1.UserID != claims2.UserID {
		t.Error("Expected both tokens to have same UserID")
	}
}

func TestGenerateTokenForNonAdminUser(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey:                "user-secret",
			AccessTokenExpireMinutes: 15,
		},
	}

	service := NewAuthService(nil, cfg)

	regularUser := &models.User{
		ID:       42,
		Username: "regularuser",
		IsAdmin:  false,
	}

	token, err := service.GenerateToken(regularUser)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.IsAdmin {
		t.Error("Expected claims.IsAdmin to be false for regular user")
	}
	if claims.UserID != 42 {
		t.Errorf("Expected UserID 42, got %d", claims.UserID)
	}
	if claims.Username != "regularuser" {
		t.Errorf("Expected Username 'regularuser', got '%s'", claims.Username)
	}
}

func TestValidateToken_EmptyToken(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test_secret_key",
		},
	}

	authService := NewAuthService(nil, cfg)

	_, err := authService.ValidateToken("")
	if err == nil {
		t.Fatalf("Expected error for empty token, got none")
	}
}

func TestValidateToken_MalformedToken(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test_secret_key",
		},
	}

	authService := NewAuthService(nil, cfg)

	testCases := []string{
		"notajwt",
		"a.b",
		"a.b.c.d",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
		"random.garbage.here",
	}

	for _, tc := range testCases {
		_, err := authService.ValidateToken(tc)
		if err == nil {
			t.Errorf("Expected error for malformed token '%s', got none", tc)
		}
	}
}
