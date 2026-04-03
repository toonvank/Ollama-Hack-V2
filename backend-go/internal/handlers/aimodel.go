package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type AIModelHandler struct {
	db *database.DB
}

func NewAIModelHandler(db *database.DB) *AIModelHandler {
	return &AIModelHandler{db: db}
}

func (h *AIModelHandler) List(c *gin.Context) {
	var rowInfos []models.AIModelInfo

	// Get order_by param (default: name)
	orderBy := c.DefaultQuery("order_by", "name")
	order := c.DefaultQuery("order", "asc")

	// Validate order_by field
	validOrderFields := map[string]string{
		"id":               "m.id",
		"name":             "m.name",
		"created_at":       "m.created_at",
		"token_per_second": "token_per_second",
		"endpoints":        "endpoints",
	}

	orderField, ok := validOrderFields[orderBy]
	if !ok {
		orderField = "m.name"
	}

	// Validate order direction
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	// For token_per_second, put NULLs last when sorting desc (highest first)
	nullsHandling := ""
	if orderBy == "token_per_second" && order == "desc" {
		nullsHandling = " NULLS LAST"
	}

	query := `
		SELECT 
			m.id, m.name, m.tag, m.enabled, m.created_at,
			COUNT(case when eam.status = 'available' then 1 end) as endpoints,
			MAX(eam.token_per_second) as token_per_second
		FROM ai_models m
		LEFT JOIN endpoint_ai_models eam ON m.id = eam.ai_model_id
		GROUP BY m.id
		ORDER BY ` + orderField + " " + order + nullsHandling

	if err := h.db.Select(&rowInfos, query); err != nil {
		utils.InternalServerError(c, "Failed to fetch AI models")
		return
	}

	utils.SuccessPage(c, rowInfos, len(rowInfos), 1, 50, 1)
}

func (h *AIModelHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var info models.AIModelInfo
	query := `
		SELECT 
			m.id, m.name, m.tag, m.enabled, m.created_at,
			COUNT(case when eam.status = 'available' then 1 end) as endpoints
		FROM ai_models m
		LEFT JOIN endpoint_ai_models eam ON m.id = eam.ai_model_id
		WHERE m.id = $1
		GROUP BY m.id
	`
	if err := h.db.Get(&info, query, id); err != nil {
		utils.NotFound(c, "AI model not found")
		return
	}

	var perfs []models.AIModelPerformance
	perfQuery := `
		SELECT 
			e.id as endpoint_id, 
			e.name as endpoint_name, 
			eam.status,
			eam.token_per_second,
			eam.max_connection_time
		FROM endpoint_ai_models eam
		JOIN endpoints e ON eam.endpoint_id = e.id
		WHERE eam.ai_model_id = $1
	`
	h.db.Select(&perfs, perfQuery, id)

	detail := models.AIModelDetail{
		AIModelInfo:  info,
		Performances: perfs,
	}

	utils.Success(c, detail)
}

// Toggle enables or disables a model globally (it will be hidden from the proxy)
func (h *AIModelHandler) Toggle(c *gin.Context) {
	id := c.Param("id")
	var req models.AIModelToggle
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var info models.AIModelInfo
	err := h.db.Get(&info,
		`UPDATE ai_models SET enabled = $1 WHERE id = $2
		 RETURNING id, name, tag, enabled, created_at`,
		req.Enabled, id)
	if err != nil {
		utils.NotFound(c, "AI model not found")
		return
	}

	utils.Success(c, info)
}
