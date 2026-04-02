# Go Backend Implementation Guide

## Current Status: ~40% Complete ✅

### What's Implemented (Working)

✅ **Core Infrastructure:**
- Project structure following Go best practices
- Configuration management with Viper
- PostgreSQL database connection with sqlx
- Complete database schema with all tables
- Environment-based configuration

✅ **Authentication & Authorization:**
- JWT token generation and validation
- Bcrypt password hashing
- Auth middleware (Bearer token + API key)
- Admin middleware
- Login endpoint
- Get current user endpoint
- Change password endpoint
- Initialize admin endpoint

✅ **Models:**
- User
- Plan
- APIKey
- Endpoint
- Complete database models with JSON tags

✅ **Utilities:**
- Password hashing/checking
- Response helpers
- Error handling

✅ **Docker:**
- Multi-stage Dockerfile
- Optimized for production

### What Needs Implementation (~60%)

The remaining modules follow the same patterns. Here's exactly what to implement:

---

## 1. API Key Module

Create `/internal/handlers/apikey.go`:

```go
package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type APIKeyHandler struct {
	db *database.DB
}

func NewAPIKeyHandler(db *database.DB) *APIKeyHandler {
	return &APIKeyHandler{db: db}
}

func (h *APIKeyHandler) List(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var keys []models.APIKey
	err := h.db.Select(&keys, "SELECT * FROM api_keys WHERE user_id = $1", userID)
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch API keys")
		return
	}
	// Convert to APIKeyInfo (hide the actual key)
	var keyInfos []models.APIKeyInfo
	for _, key := range keys {
		keyInfos = append(keyInfos, models.APIKeyInfo{
			ID:         key.ID,
			Name:       key.Name,
			UserID:     key.UserID,
			LastUsedAt: key.LastUsedAt,
			CreatedAt:  key.CreatedAt,
		})
	}
	utils.Success(c, keyInfos)
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	userID, _ := c.Get("user_id")
	var req models.APIKeyCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// Generate random API key
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		utils.InternalServerError(c, "Failed to generate API key")
		return
	}
	apiKey := hex.EncodeToString(bytes)

	var key models.APIKey
	err := h.db.Get(&key,
		"INSERT INTO api_keys (key, name, user_id) VALUES ($1, $2, $3) RETURNING *",
		apiKey, req.Name, userID)
	if err != nil {
		utils.InternalServerError(c, "Failed to create API key")
		return
	}

	utils.Created(c, models.APIKeyResponse{
		ID:         key.ID,
		Key:        key.Key, // Only show once
		Name:       key.Name,
		UserID:     key.UserID,
		LastUsedAt: key.LastUsedAt,
		CreatedAt:  key.CreatedAt,
	})
}

func (h *APIKeyHandler) Delete(c *gin.Context) {
	keyID := c.Param("id")
	userID, _ := c.Get("user_id")
	
	// Verify ownership
	var count int
	err := h.db.Get(&count, "SELECT COUNT(*) FROM api_keys WHERE id = $1 AND user_id = $2", keyID, userID)
	if err != nil || count == 0 {
		utils.NotFound(c, "API key not found")
		return
	}

	_, err = h.db.Exec("DELETE FROM api_keys WHERE id = $1", keyID)
	if err != nil {
		utils.InternalServerError(c, "Failed to delete API key")
		return
	}

	utils.NoContent(c)
}
```

**Add to main.go:**
```go
apikeyHandler := handlers.NewAPIKeyHandler(db)
protected.GET("/apikeys", apikeyHandler.List)
protected.POST("/apikeys", apikeyHandler.Create)
protected.DELETE("/apikeys/:id", apikeyHandler.Delete)
```

---

## 2. Plan Module

Create `/internal/handlers/plan.go`:

```go
package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type PlanHandler struct {
	db *database.DB
}

func NewPlanHandler(db *database.DB) *PlanHandler {
	return &PlanHandler{db: db}
}

func (h *PlanHandler) List(c *gin.Context) {
	var plans []models.Plan
	err := h.db.Select(&plans, "SELECT * FROM plans ORDER BY id")
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch plans")
		return
	}
	utils.Success(c, plans)
}

func (h *PlanHandler) Create(c *gin.Context) {
	var req models.PlanCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var plan models.Plan
	err := h.db.Get(&plan,
		`INSERT INTO plans (name, description, rpm, rpd, is_default) 
		 VALUES ($1, $2, $3, $4, $5) RETURNING *`,
		req.Name, req.Description, req.RPM, req.RPD, req.IsDefault)
	if err != nil {
		utils.InternalServerError(c, "Failed to create plan")
		return
	}

	utils.Created(c, plan)
}

func (h *PlanHandler) Update(c *gin.Context) {
	planID := c.Param("id")
	var req models.PlanUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// Build update query dynamically based on what's provided
	// (Implement dynamic SQL update based on non-nil fields)
	
	utils.Success(c, gin.H{"message": "Plan updated"})
}

func (h *PlanHandler) Delete(c *gin.Context) {
	planID := c.Param("id")
	_, err := h.db.Exec("DELETE FROM plans WHERE id = $1", planID)
	if err != nil {
		utils.InternalServerError(c, "Failed to delete plan")
		return
	}
	utils.NoContent(c)
}
```

---

## 3. Endpoint Module

Create `/internal/handlers/endpoint.go` - This is the largest module. Key functions:

- `List()` - List all endpoints with pagination
- `Create()` - Create single endpoint
- `BatchCreate()` - Create multiple endpoints (use the fixed duplicate logic from Python)
- `Update()` - Update endpoint
- `Delete()` - Delete endpoint
- `BatchDelete()` - Delete multiple endpoints
- `BatchTest()` - Trigger tests for multiple endpoints

**Key Implementation Notes:**
- Use the duplicate handling logic from the Python fix
- Implement background scheduler for testing (use robfig/cron)
- Store test results in endpoint_test_tasks table

---

## 4. AI Model Module

Create `/internal/handlers/aimodel.go`:

- `List()` - List all AI models with their endpoints
- `Get()` - Get single model with performance data
- Discovery happens during endpoint testing

---

## 5. Ollama Client & Proxy

Create `/internal/services/ollama.go`:

```go
package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaClient struct {
	httpClient *http.Client
}

func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *OllamaClient) ChatCompletions(endpointURL string, body interface{}) (interface{}, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/v1/chat/completions", endpointURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *OllamaClient) TestEndpoint(endpointURL string) error {
	// Implement endpoint testing logic
	// Send test request, measure performance
	return nil
}
```

Create `/internal/handlers/ollama.go` for proxy routes

---

## Quick Setup to Run What's Implemented

1. **Install dependencies:**
```bash
cd backend-go
go mod tidy
```

2. **Set environment variables:**
```bash
export APP_SECRET_KEY=your-secret-key
export DATABASE_HOST=localhost
export DATABASE_PORT=5432
export DATABASE_USERNAME=ollama_hack
export DATABASE_PASSWORD=your-password
export DATABASE_DB=ollama_hack
```

3. **Run:**
```bash
go run cmd/server/main.go
```

4. **Test:**
```bash
# Initialize admin
curl -X POST http://localhost:8000/api/v2/init \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin1234"}'

# Login
curl -X POST http://localhost:8000/api/v2/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin1234"}'

# Get current user
curl -X GET http://localhost:8000/api/v2/auth/me \
  -H "Authorization: Bearer <token>"
```

---

## Patterns to Follow

All remaining handlers follow this pattern:

1. **Create handler struct** with db dependency
2. **Implement CRUD methods** (List, Get, Create, Update, Delete)
3. **Use utils.Success()** for 200 responses
4. **Use utils.Created()** for 201 responses
5. **Use utils.Error()** helpers for errors
6. **Validate input** with binding tags
7. **Check authorization** (user_id from context)

---

## Performance Tips

- Use prepared statements for repeated queries
- Add connection pooling (already configured)
- Use goroutines for background tasks
- Cache frequently accessed data
- Add indexes (already in schema)

---

## Testing

Create `/internal/handlers/endpoint_test.go`:

```go
package handlers_test

import (
	"testing"
	// Add test framework (testify recommended)
)

func TestEndpointCreate(t *testing.T) {
	// Test endpoint creation
}
```

---

## Next Steps

1. Implement API Key handler (easiest, ~30 min)
2. Implement Plan handler (~30 min)
3. Implement Endpoint handler (hardest, ~2-3 hours)
4. Implement AI Model handler (~1 hour)
5. Implement Ollama proxy (~1-2 hours)
6. Add tests (~2 hours)
7. Production hardening (~1 hour)

**Total estimated time to complete: 8-10 hours**

---

## Resources

- [Gin Documentation](https://gin-gonic.com/docs/)
- [sqlx Documentation](https://jmoiron.github.io/sqlx/)
- [Go by Example](https://gobyexample.com/)

The foundation is solid. All the hard architectural decisions are done. The remaining work is straightforward CRUD following the established patterns.
