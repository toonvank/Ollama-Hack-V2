//go:build integration

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/timlzh/ollama-hack/internal/config"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

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

func TestIntegration_UserHandler_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewUserHandler(db)
	gin.SetMode(gin.TestMode)

	// Create a user
	t.Run("Create", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "testuser",
			"password": "testpass123",
			"is_admin": false,
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Create(c)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	// List users
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/users", nil)

		handler.List(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Get user by ID
	t.Run("Get", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/users/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Get(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Update user
	t.Run("Update", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "updateduser",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("PUT", "/users/1", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Update(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Delete user
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("DELETE", "/users/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Delete(c)

		// Should succeed
		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200/204, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_UserHandler_CreateDuplicateUsername(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewUserHandler(db)
	gin.SetMode(gin.TestMode)

	body := map[string]interface{}{
		"username": "duplicateuser",
		"password": "testpass123",
	}

	// Create first user
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.Create(c)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create first user: %d", w.Code)
	}

	// Try to create duplicate
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	jsonBytes2, _ := json.Marshal(body)
	c2.Request, _ = http.NewRequest("POST", "/users", bytes.NewBuffer(jsonBytes2))
	c2.Request.Header.Set("Content-Type", "application/json")
	handler.Create(c2)

	if w2.Code != http.StatusInternalServerError && w2.Code != http.StatusConflict && w2.Code != http.StatusBadRequest {
		t.Errorf("Expected error status for duplicate username, got %d", w2.Code)
	}
}

func TestIntegration_UserHandler_GetNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewUserHandler(db)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/users/999", nil)
	c.Params = gin.Params{{Key: "id", Value: "999"}}

	handler.Get(c)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestIntegration_PlanHandler_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewPlanHandler(db)
	gin.SetMode(gin.TestMode)

	// Create a plan
	t.Run("Create", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"name":        "Basic Plan",
			"description": "A basic plan",
			"rpm":         60,
			"rpd":         1000,
			"is_default":  true,
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/plans", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Create(c)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	// List plans
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/plans", nil)

		handler.List(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Get plan by ID
	t.Run("Get", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/plans/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Get(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Update plan
	t.Run("Update", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"name": "Updated Plan",
			"rpm":  120,
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("PUT", "/plans/1", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Update(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Delete plan
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("DELETE", "/plans/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Delete(c)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200/204, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_EndpointHandler_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewEndpointHandler(db)
	gin.SetMode(gin.TestMode)

	// Create an endpoint
	t.Run("Create", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"url":  "http://localhost:11434",
			"name": "Local Ollama",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/endpoints", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Create(c)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	// List endpoints
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/endpoints", nil)

		handler.List(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Get endpoint by ID
	t.Run("Get", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/endpoints/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Get(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Update endpoint
	t.Run("Update", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"name": "Updated Endpoint",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("PUT", "/endpoints/1", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Update(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Delete endpoint
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("DELETE", "/endpoints/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Delete(c)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200/204, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_EndpointHandler_BatchCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewEndpointHandler(db)
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"endpoints": []map[string]string{
			{"url": "http://localhost:11434", "name": "Endpoint 1"},
			{"url": "http://localhost:11435", "name": "Endpoint 2"},
			{"url": "http://localhost:11436", "name": "Endpoint 3"},
		},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("POST", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchCreate(c)

	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Errorf("Expected status 201/200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegration_EndpointHandler_BatchDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewEndpointHandler(db)
	gin.SetMode(gin.TestMode)

	// First create some endpoints
	for i := 1; i <= 3; i++ {
		_, err := db.Exec(`INSERT INTO endpoints (url, name) VALUES ($1, $2)`,
			"http://localhost:"+string(rune('0'+i)), "Endpoint")
		if err != nil {
			t.Fatalf("Failed to insert test endpoint: %v", err)
		}
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	body := map[string]interface{}{
		"ids": []int{1, 2, 3},
	}
	jsonBytes, _ := json.Marshal(body)
	c.Request, _ = http.NewRequest("DELETE", "/endpoints/batch", bytes.NewBuffer(jsonBytes))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.BatchDelete(c)

	if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
		t.Errorf("Expected status 200/204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegration_APIKeyHandler_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewAPIKeyHandler(db)
	gin.SetMode(gin.TestMode)

	// First create a user
	hashedPassword, _ := utils.HashPassword("testpass")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password) VALUES ($1, $2)`, "testuser", hashedPassword)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create API key
	t.Run("Create", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)

		body := map[string]interface{}{
			"name": "Test API Key",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/api-keys", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Create(c)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	// List API keys
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)
		c.Request, _ = http.NewRequest("GET", "/api-keys", nil)

		handler.List(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Delete API key
	t.Run("Delete", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)
		c.Request, _ = http.NewRequest("DELETE", "/api-keys/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Delete(c)

		if w.Code != http.StatusOK && w.Code != http.StatusNoContent {
			t.Errorf("Expected status 200/204, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_AIModelHandler_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	handler := NewAIModelHandler(db)
	gin.SetMode(gin.TestMode)

	// First insert a model directly
	_, err := db.Exec(`INSERT INTO ai_models (name, tag, enabled) VALUES ($1, $2, $3)`, "llama2", "7b", true)
	if err != nil {
		t.Fatalf("Failed to insert test model: %v", err)
	}

	// List models
	t.Run("List", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/models", nil)

		handler.List(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	// Get model
	t.Run("Get", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/models/1", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Get(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Toggle model
	t.Run("Toggle", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("POST", "/models/1/toggle", nil)
		c.Params = gin.Params{{Key: "id", Value: "1"}}

		handler.Toggle(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_AuthHandler_Login(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test-secret-key",
		},
	}
	authService := services.NewAuthService(db, cfg)
	handler := NewAuthHandler(authService, db)
	gin.SetMode(gin.TestMode)

	// Create a user first
	hashedPassword, _ := utils.HashPassword("testpass123")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"admin", hashedPassword, true)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Test successful login
	t.Run("SuccessfulLogin", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "admin",
			"password": "testpass123",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["data"] == nil {
			t.Error("Expected data in response")
		}
	})

	// Test failed login with wrong password
	t.Run("WrongPassword", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "admin",
			"password": "wrongpass",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	// Test failed login with non-existent user
	t.Run("NonExistentUser", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "nonexistent",
			"password": "testpass",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.Login(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_AuthHandler_GetCurrentUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test-secret-key",
		},
	}
	authService := services.NewAuthService(db, cfg)
	handler := NewAuthHandler(authService, db)
	gin.SetMode(gin.TestMode)

	// Create a user
	hashedPassword, _ := utils.HashPassword("testpass123")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"testuser", hashedPassword, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("user_id", 1)
	c.Request, _ = http.NewRequest("GET", "/auth/me", nil)

	handler.GetCurrentUser(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegration_AuthHandler_ChangePassword(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test-secret-key",
		},
	}
	authService := services.NewAuthService(db, cfg)
	handler := NewAuthHandler(authService, db)
	gin.SetMode(gin.TestMode)

	// Create a user
	hashedPassword, _ := utils.HashPassword("oldpassword")
	_, err := db.Exec(`INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, $3)`,
		"testuser", hashedPassword, false)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	t.Run("SuccessfulChange", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)

		body := map[string]interface{}{
			"old_password": "oldpassword",
			"new_password": "newpassword123",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.ChangePassword(c)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("WrongOldPassword", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Set("user_id", 1)

		body := map[string]interface{}{
			"old_password": "wrongoldpassword",
			"new_password": "newpassword123",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("PUT", "/auth/password", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.ChangePassword(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestIntegration_AuthHandler_InitializeAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	cfg := &config.Config{
		App: config.AppConfig{
			SecretKey: "test-secret-key",
		},
	}
	authService := services.NewAuthService(db, cfg)
	handler := NewAuthHandler(authService, db)
	gin.SetMode(gin.TestMode)

	t.Run("FirstAdmin", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "admin",
			"password": "adminpass123",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/auth/init", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.InitializeAdmin(c)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK {
			t.Errorf("Expected status 201/200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("AdminAlreadyExists", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		body := map[string]interface{}{
			"username": "admin2",
			"password": "adminpass123",
		}
		jsonBytes, _ := json.Marshal(body)
		c.Request, _ = http.NewRequest("POST", "/auth/init", bytes.NewBuffer(jsonBytes))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.InitializeAdmin(c)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusConflict {
			t.Errorf("Expected error status, got %d: %s", w.Code, w.Body.String())
		}
	})
}
