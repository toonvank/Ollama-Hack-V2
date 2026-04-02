package handlers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type UserHandler struct {
	db *database.DB
}

func NewUserHandler(db *database.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) List(c *gin.Context) {
	var users []models.User
	if err := h.db.Select(&users, "SELECT * FROM users ORDER BY id"); err != nil {
		utils.InternalServerError(c, "Failed to fetch users")
		return
	}
	var infos []models.UserInfo
	for _, u := range users {
		infos = append(infos, models.UserInfo{
			ID:       u.ID,
			Username: u.Username,
			IsAdmin:  u.IsAdmin,
			PlanID:   u.PlanID,
		})
	}
	utils.SuccessPage(c, infos, len(infos), 1, 50, 1)
}

func (h *UserHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var u models.User
	if err := h.db.Get(&u, "SELECT * FROM users WHERE id = $1", id); err != nil {
		utils.NotFound(c, "User not found")
		return
	}
	utils.Success(c, models.UserInfo{
		ID:       u.ID,
		Username: u.Username,
		IsAdmin:  u.IsAdmin,
		PlanID:   u.PlanID,
	})
}

func (h *UserHandler) Create(c *gin.Context) {
	var req models.UserCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.InternalServerError(c, "Failed to hash password")
		return
	}

	var u models.User
	err = h.db.Get(&u,
		"INSERT INTO users (username, hashed_password, is_admin, plan_id) VALUES ($1, $2, $3, $4) RETURNING *",
		req.Username, hashed, req.IsAdmin, req.PlanID)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			utils.BadRequest(c, "Username already exists")
			return
		}
		utils.InternalServerError(c, "Failed to create user")
		return
	}

	utils.Created(c, models.UserInfo{
		ID:       u.ID,
		Username: u.Username,
		IsAdmin:  u.IsAdmin,
		PlanID:   u.PlanID,
	})
}

func (h *UserHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req models.UserUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var setClauses []string
	var args []interface{}
	argID := 1

	if req.Username != nil {
		setClauses = append(setClauses, fmt.Sprintf("username = $%d", argID))
		args = append(args, *req.Username)
		argID++
	}
	if req.Password != nil {
		hashed, err := utils.HashPassword(*req.Password)
		if err != nil {
			utils.InternalServerError(c, "Failed to hash password")
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("hashed_password = $%d", argID))
		args = append(args, hashed)
		argID++
	}
	if req.IsAdmin != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_admin = $%d", argID))
		args = append(args, *req.IsAdmin)
		argID++
	}
	if req.PlanID != nil {
		setClauses = append(setClauses, fmt.Sprintf("plan_id = $%d", argID))
		args = append(args, *req.PlanID)
		argID++
	}

	if len(setClauses) == 0 {
		utils.BadRequest(c, "No fields to update")
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d RETURNING *", strings.Join(setClauses, ", "), argID)

	var u models.User
	if err := h.db.Get(&u, query, args...); err != nil {
		utils.InternalServerError(c, "Failed to update user")
		return
	}

	utils.Success(c, models.UserInfo{
		ID:       u.ID,
		Username: u.Username,
		IsAdmin:  u.IsAdmin,
		PlanID:   u.PlanID,
	})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if _, err := h.db.Exec("DELETE FROM users WHERE id = $1", id); err != nil {
		utils.InternalServerError(c, "Failed to delete user")
		return
	}
	utils.NoContent(c)
}
