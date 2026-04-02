package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type AuthHandler struct {
	authService *services.AuthService
	db          *database.DB
}

func NewAuthHandler(authService *services.AuthService, db *database.DB) *AuthHandler {
	return &AuthHandler{authService: authService, db: db}
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	token, err := h.authService.Login(req.Username, req.Password)
	if err != nil {
		utils.Unauthorized(c, err.Error())
		return
	}

	utils.Success(c, token)
}

// GetCurrentUser returns the current logged-in user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	user, err := h.authService.GetUserByID(userID.(int))
	if err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	// Get plan name if plan_id is set
	var planName string
	if user.PlanID != nil {
		var plan models.Plan
		err := h.db.Get(&plan, "SELECT * FROM plans WHERE id = $1", *user.PlanID)
		if err == nil {
			planName = plan.Name
		}
	}

	userInfo := models.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		PlanID:   user.PlanID,
		PlanName: planName,
	}

	utils.Success(c, userInfo)
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, _ := c.Get("user_id")
	
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	user, err := h.authService.GetUserByID(userID.(int))
	if err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	if !utils.CheckPassword(req.OldPassword, user.HashedPassword) {
		utils.BadRequest(c, "Current password is incorrect")
		return
	}

	hashedPassword, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		utils.InternalServerError(c, "Failed to hash password")
		return
	}

	_, err = h.db.Exec("UPDATE users SET hashed_password = $1 WHERE id = $2", hashedPassword, user.ID)
	if err != nil {
		utils.InternalServerError(c, "Failed to update password")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// InitializeAdmin creates the initial admin user
func (h *AuthHandler) InitializeAdmin(c *gin.Context) {
	// Check if any users exist
	var count int
	err := h.db.Get(&count, "SELECT COUNT(*) FROM users")
	if err != nil {
		utils.InternalServerError(c, "Database error")
		return
	}

	if count > 0 {
		utils.BadRequest(c, "System already initialized")
		return
	}

	var req models.UserCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalServerError(c, "Failed to hash password")
		return
	}

	_, err = h.db.Exec(
		"INSERT INTO users (username, hashed_password, is_admin) VALUES ($1, $2, true)",
		req.Username, hashedPassword,
	)
	if err != nil {
		utils.InternalServerError(c, "Failed to create admin user")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Admin user created successfully",
	})
}
