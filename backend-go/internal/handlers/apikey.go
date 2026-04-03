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
	utils.SuccessPage(c, keyInfos, len(keyInfos), 1, 50, 1)
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	userIDRaw, _ := c.Get("user_id")
	userID := userIDRaw.(int)
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
	apiKey := "sk-" + hex.EncodeToString(bytes)

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

func (h *APIKeyHandler) GetStats(c *gin.Context) {
	// Dummy stats implementation for now
	keyID := c.Param("id")
	utils.Success(c, gin.H{
		"id":                 keyID,
		"usage_last_30_days": 1500,
		"last_used_at":       "2024-03-20T15:30:00Z",
	})
}
