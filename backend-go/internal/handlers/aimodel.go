package handlers

import (
	"strconv"

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
		"id":                    "m.id",
		"name":                  "m.name",
		"created_at":            "m.created_at",
		"token_per_second":      "token_per_second",
		"max_connection_time":   "max_connection_time",
		"endpoints":             "endpoints",
	}

	orderField, ok := validOrderFields[orderBy]
	if !ok {
		orderField = "m.name"
	}

	// Validate order direction
	if order != "asc" && order != "desc" {
		order = "asc"
	}

	// Put NULLs last when sorting by performance metrics
	nullsHandling := ""
	if orderBy == "token_per_second" && order == "desc" {
		nullsHandling = " NULLS LAST"
	}
	if orderBy == "max_connection_time" && order == "asc" {
		nullsHandling = " NULLS LAST"
	}

	searchQuery := c.Query("search")
	whereClause := ""
	var args []interface{}
	if searchQuery != "" {
		whereClause = "WHERE m.name ILIKE $1"
		args = append(args, "%"+searchQuery+"%")
	}

	query := `
		SELECT 
			m.id, m.name, m.tag, m.enabled, m.created_at,
			COUNT(case when eam.status = 'available' then 1 end) as endpoints,
			MAX(eam.token_per_second) as token_per_second,
			MIN(eam.max_connection_time) as max_connection_time
		FROM ai_models m
		LEFT JOIN endpoint_ai_models eam ON m.id = eam.ai_model_id
		` + whereClause + `
		GROUP BY m.id
		ORDER BY ` + orderField + " " + order + nullsHandling

	// Count total records for pagination
	var total int
	countQuery := `SELECT COUNT(*) FROM ai_models m ` + whereClause
	if err := h.db.Get(&total, countQuery, args...); err != nil {
		utils.InternalServerError(c, "Failed to count AI models")
		return
	}

	page := 1
	size := 10
	if p := c.Query("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if s := c.Query("size"); s != "" {
		if parsedSize, err := strconv.Atoi(s); err == nil && parsedSize > 0 {
			size = parsedSize
		}
	}

	// Add limit and offset
	query += ` LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, size, (page-1)*size)

	if err := h.db.Select(&rowInfos, query, args...); err != nil {
		utils.InternalServerError(c, "Failed to fetch AI models")
		return
	}

	pages := total / size
	if total%size != 0 {
		pages++
	}
	if pages == 0 {
		pages = 1
	}

	utils.SuccessPage(c, rowInfos, total, page, size, pages)
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

// SmartModels returns the current smart model resolutions
func (h *AIModelHandler) SmartModels(c *gin.Context) {
	smartProfiles := []string{"fastest", "large", "small", "coding", "cloud"}
	results := make([]gin.H, 0, len(smartProfiles))

	for _, profile := range smartProfiles {
		heuristic, description, rankingClause := smartProfileConfig(profile)

		query := `
			SELECT 
				m.name, 
				m.tag,
				e.name as endpoint_name,
				eam.token_per_second,
				eam.max_connection_time
			FROM endpoint_ai_models eam
			JOIN endpoints e ON e.id = eam.endpoint_id
			JOIN ai_models m ON m.id = eam.ai_model_id
			WHERE ` + heuristic + `
			  AND m.enabled = TRUE
			  AND eam.status = 'available'
			  AND e.status = 'available'
			ORDER BY ` + rankingClause + `
			LIMIT 1
		`

		type resultRow struct {
			Name              string   `db:"name"`
			Tag               string   `db:"tag"`
			EndpointName      string   `db:"endpoint_name"`
			TokenPerSecond    *float64 `db:"token_per_second"`
			MaxConnectionTime *float64 `db:"max_connection_time"`
		}

		var row resultRow
		err := h.db.Get(&row, query)

		result := gin.H{
			"smart_model": "smart:" + profile,
			"description": description,
			"resolved":    false,
		}

		if err == nil {
			result["resolved"] = true
			result["model_name"] = row.Name
			result["model_tag"] = row.Tag
			result["model_full"] = row.Name + ":" + row.Tag
			result["endpoint"] = row.EndpointName
			if row.TokenPerSecond != nil {
				result["token_per_second"] = *row.TokenPerSecond
			}
			if row.MaxConnectionTime != nil {
				result["max_connection_time"] = *row.MaxConnectionTime
			}
		} else {
			result["error"] = "No available models match this profile"
		}

		results = append(results, result)
	}

	c.JSON(200, gin.H{"smart_models": results})
}
