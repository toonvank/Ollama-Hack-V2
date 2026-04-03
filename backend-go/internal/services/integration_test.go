//go:build integration

package services

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timlzh/ollama-hack/internal/config"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/utils"
)

func getTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Port:      8080,
			JWTSecret: "test-secret-key",
		},
	}
}

func setupTestDB(t *testing.T) (*database.DB, func()) {
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	cfg := &config.DatabaseConfig{
		Engine:   "postgresql",
		Host:     host,
		Port:     port.Int(),
		Username: "testuser",
		Password: "testpass",
		DB:       "testdb",
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	err = db.CreateTables()
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}

	cleanup := func() {
		db.Close()
		pgContainer.Terminate(ctx)
	}

	return db, cleanup
}

func TestIntegration_AuthService_Login(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	authService := NewAuthService(db, getTestConfig())

	// Create a test user
	hashedPassword, _ := utils.HashPassword("testpass123")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"testuser", hashedPassword, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("ValidCredentials", func(t *testing.T) {
		tokenResp, err := authService.Login("testuser", "testpass123")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if tokenResp == nil || tokenResp.Token == "" {
			t.Error("Expected token, got empty")
		}
	})

	t.Run("InvalidPassword", func(t *testing.T) {
		_, err := authService.Login("testuser", "wrongpassword")
		if err == nil {
			t.Error("Expected error for wrong password")
		}
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		_, err := authService.Login("nonexistent", "anypassword")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})
}

func TestIntegration_AuthService_GetUserByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	authService := NewAuthService(db, getTestConfig())

	// Create a test user
	hashedPassword, _ := utils.HashPassword("testpass123")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"testuser", hashedPassword, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("ExistingUser", func(t *testing.T) {
		user, err := authService.GetUserByID(1)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if user == nil {
			t.Error("Expected user, got nil")
		}
		if user != nil && user.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", user.Username)
		}
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		user, err := authService.GetUserByID(999)
		if err == nil && user != nil {
			t.Error("Expected error or nil user for non-existent user")
		}
	})
}

func TestIntegration_AuthService_GetUserByAPIKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	authService := NewAuthService(db, getTestConfig())

	// Create a test user and API key
	hashedPassword, _ := utils.HashPassword("testpass123")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"testuser", hashedPassword, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	testAPIKey := "sk-test-api-key-12345678901234567890"
	_, err = db.Exec(`INSERT INTO api_keys (key, name, user_id) VALUES ($1, $2, $3)`,
		testAPIKey, "Test Key", 1)
	if err != nil {
		t.Fatalf("Failed to create test API key: %v", err)
	}

	t.Run("ValidAPIKey", func(t *testing.T) {
		user, err := authService.GetUserByAPIKey(testAPIKey)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if user == nil {
			t.Error("Expected user, got nil")
		}
		if user != nil && user.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", user.Username)
		}
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		user, err := authService.GetUserByAPIKey("sk-invalid-key")
		if err == nil && user != nil {
			t.Error("Expected error or nil user for invalid API key")
		}
	})
}

func TestIntegration_AuthService_TokenValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	authService := NewAuthService(db, getTestConfig())

	// Create a test user
	hashedPassword, _ := utils.HashPassword("testpass123")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"testuser", hashedPassword, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Login to get a token
	tokenResp, err := authService.Login("testuser", "testpass123")
	if err != nil {
		t.Fatalf("Failed to login: %v", err)
	}

	// Validate the token
	claims, err := authService.ValidateToken(tokenResp.Token)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if claims == nil {
		t.Error("Expected claims, got nil")
	}
	if claims != nil && claims.UserID != 1 {
		t.Errorf("Expected user ID 1, got %d", claims.UserID)
	}
}
