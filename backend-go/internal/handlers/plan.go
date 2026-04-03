package handlers

import (
	"fmt"
	"strings"

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
	utils.SuccessPage(c, plans, len(plans), 1, 50, 1)
}

func (h *PlanHandler) GetCurrentUserPlan(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var user models.User
	if err := h.db.Get(&user, "SELECT plan_id FROM users WHERE id = $1", userID); err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	if user.PlanID == nil {
		// Return null/empty plan instead of 404 - frontend expects this
		utils.Success(c, nil)
		return
	}

	var plan models.Plan
	if err := h.db.Get(&plan, "SELECT * FROM plans WHERE id = $1", *user.PlanID); err != nil {
		utils.Success(c, nil)
		return
	}

	utils.Success(c, plan)
}

func (h *PlanHandler) Get(c *gin.Context) {
	planID := c.Param("id")
	var plan models.Plan
	if err := h.db.Get(&plan, "SELECT * FROM plans WHERE id = $1", planID); err != nil {
		utils.NotFound(c, "Plan not found")
		return
	}
	utils.Success(c, plan)
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
	var setClauses []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}
	if req.RPM != nil {
		setClauses = append(setClauses, fmt.Sprintf("rpm = $%d", argID))
		args = append(args, *req.RPM)
		argID++
	}
	if req.RPD != nil {
		setClauses = append(setClauses, fmt.Sprintf("rpd = $%d", argID))
		args = append(args, *req.RPD)
		argID++
	}
	if req.IsDefault != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_default = $%d", argID))
		args = append(args, *req.IsDefault)
		argID++
	}

	if len(setClauses) == 0 {
		utils.BadRequest(c, "No fields to update")
		return
	}

	args = append(args, planID)
	query := fmt.Sprintf("UPDATE plans SET %s WHERE id = $%d RETURNING *", strings.Join(setClauses, ", "), argID)

	var plan models.Plan
	err := h.db.Get(&plan, query, args...)
	if err != nil {
		utils.InternalServerError(c, "Failed to update plan")
		return
	}

	utils.Success(c, plan)
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
