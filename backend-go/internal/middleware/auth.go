package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.Unauthorized(c, "Authorization header required")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			utils.Unauthorized(c, "Invalid authorization header format")
			c.Abort()
			return
		}

		authType := strings.ToLower(parts[0])
		token := parts[1]

		var userID int
		var isAdmin bool

		var allowedModels []string

		switch authType {
		case "bearer":
			// First try JWT token
			claims, err := authService.ValidateToken(token)
			if err == nil {
				userID = claims.UserID
				isAdmin = claims.IsAdmin
			} else {
				// JWT failed, try as API key (OpenAI-compatible clients send API keys as Bearer tokens)
				user, apiKeyRow, apiKeyErr := authService.GetUserAndKeyByAPIKey(token)
				if apiKeyErr != nil {
					utils.Unauthorized(c, "Invalid or expired token")
					c.Abort()
					return
				}
				userID = user.ID
				isAdmin = user.IsAdmin
				allowedModels = apiKeyRow.ParseAllowedModels()
			}

		default:
			// Try API key
			user, apiKeyRow, err := authService.GetUserAndKeyByAPIKey(token)
			if err != nil {
				utils.Unauthorized(c, "Invalid API key")
				c.Abort()
				return
			}
			userID = user.ID
			isAdmin = user.IsAdmin
			allowedModels = apiKeyRow.ParseAllowedModels()
		}

		c.Set("user_id", userID)
		c.Set("is_admin", isAdmin)
		c.Set("allowed_models", allowedModels)
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, exists := c.Get("is_admin")
		if !exists || !isAdmin.(bool) {
			c.JSON(http.StatusForbidden, gin.H{
				"detail": "Admin privileges required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
