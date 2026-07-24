package handlers

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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
		"id":                  "m.id",
		"name":                "m.name",
		"created_at":          "m.created_at",
		"token_per_second":    "token_per_second",
		"max_connection_time": "max_connection_time",
		"param_billions":      "param_billions",
		"composite_score":     "composite_score",
		"endpoints":           "endpoints",
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
	if orderBy == "composite_score" && order == "desc" {
		nullsHandling = " NULLS LAST"
	}
	if orderBy == "param_billions" && order == "desc" {
		nullsHandling = " NULLS LAST"
	}

	searchQuery := c.Query("search")
	whereClause := ""
	var args []interface{}
	if searchQuery != "" {
		whereClause = "WHERE m.name ILIKE $1"
		args = append(args, "%"+searchQuery+"%")
	}

	paramSQL := utils.ParamBillionsSQL("m.name", "m.tag")
	compositeSQL := utils.ModelListCompositeScoreSQL()

	// endpoints count = only hosts that are actually raceable (matches proxy routing)
	query := `
		SELECT 
			m.id, m.name, m.tag, m.enabled, m.created_at,
			COUNT(case when eam.status = 'available' AND e.status = 'available'
			            AND (eh.disabled IS NOT TRUE OR eh.disabled_until IS NULL OR eh.disabled_until < NOW())
			       then 1 end) as endpoints,
			MAX(eam.token_per_second) as token_per_second,
			MIN(eam.max_connection_time) as max_connection_time,
			` + paramSQL + ` as param_billions,
			` + compositeSQL + ` as composite_score
		FROM ai_models m
		LEFT JOIN endpoint_ai_models eam ON m.id = eam.ai_model_id
		LEFT JOIN endpoints e ON e.id = eam.endpoint_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
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
	// Response shape must match frontend AIModelInfoWithEndpoint
	// (DetailDrawer eye button): total/avaliable counts + paginated endpoints.items
	id := c.Param("id")

	type modelCore struct {
		ID        int    `db:"id" json:"id"`
		Name      string `db:"name" json:"name"`
		Tag       string `db:"tag" json:"tag"`
		Enabled   bool   `db:"enabled" json:"enabled"`
		CreatedAt string `db:"created_at" json:"created_at"`
	}
	var core modelCore
	if err := h.db.Get(&core, `
		SELECT id, name, tag, enabled, created_at::text AS created_at
		FROM ai_models WHERE id = $1
	`, id); err != nil {
		utils.NotFound(c, "AI model not found")
		return
	}

	var totalCount, availableCount int
	_ = h.db.Get(&totalCount, `
		SELECT COUNT(*) FROM endpoint_ai_models WHERE ai_model_id = $1
	`, id)
	_ = h.db.Get(&availableCount, `
		SELECT COUNT(*)
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
		WHERE eam.ai_model_id = $1
		  AND eam.status = 'available'
		  AND e.status = 'available'
		  AND (eh.disabled IS NOT TRUE OR eh.disabled_until IS NULL OR eh.disabled_until < NOW())
	`, id)

	page := 1
	size := 50
	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps := c.Query("size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 {
			size = val
		}
	}
	offset := (page - 1) * size

	type endpointRow struct {
		ID                int      `db:"id"`
		URL               string   `db:"url"`
		Name              string   `db:"name"`
		CreatedAt         string   `db:"created_at"`
		// ModelOnHost = endpoint_ai_models.status (last known model test)
		ModelOnHost string `db:"model_on_host"`
		// HostStatus = endpoints.status (is the Ollama host up?)
		HostStatus        string   `db:"host_status"`
		TokenPerSecond    *float64 `db:"token_per_second"`
		MaxConnectionTime *float64 `db:"max_connection_time"`
		EAMID             int      `db:"eam_id"`
		HealthDisabled    bool     `db:"health_disabled"`
	}
	var rows []endpointRow
	err := h.db.Select(&rows, `
		SELECT
			e.id,
			e.url,
			e.name,
			e.created_at::text AS created_at,
			eam.status AS model_on_host,
			e.status AS host_status,
			eam.token_per_second,
			eam.max_connection_time,
			eam.id AS eam_id,
			COALESCE(
			  eh.disabled = TRUE AND (eh.disabled_until IS NULL OR eh.disabled_until > NOW()),
			  FALSE
			) AS health_disabled
		FROM endpoint_ai_models eam
		JOIN endpoints e ON e.id = eam.endpoint_id
		LEFT JOIN endpoint_health eh ON eh.url = e.url
		WHERE eam.ai_model_id = $1
		ORDER BY
			-- Routable first (host up + model available + health ok)
			CASE
			  WHEN eam.status = 'available' AND e.status = 'available'
			       AND NOT (COALESCE(eh.disabled, false) AND (eh.disabled_until IS NULL OR eh.disabled_until > NOW()))
			  THEN 0
			  WHEN eam.status = 'available' THEN 1
			  ELSE 2
			END,
			eam.token_per_second DESC NULLS LAST,
			e.url ASC
		LIMIT $2 OFFSET $3
	`, id, size, offset)
	if err != nil {
		rows = []endpointRow{}
	}

	// Recent performance samples per endpoint_ai_model for the status timeline
	type perfRow struct {
		ID                int      `db:"id"`
		EAMID             int      `db:"endpoint_ai_model_id"`
		Status            string   `db:"status"`
		TokenPerSecond    *float64 `db:"token_per_second"`
		MaxConnectionTime *float64 `db:"max_connection_time"`
		TotalTime         *float64 `db:"total_time"`
		CreatedAt         string   `db:"created_at"`
	}
	eamIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		eamIDs = append(eamIDs, r.EAMID)
	}
	perfsByEAM := map[int][]gin.H{}
	if len(eamIDs) > 0 {
		// One query for all visible EAMs (last 10 per link is enough for UI timeline)
		var perfs []perfRow
		// status comes from current eam status at render time; history table has metrics only
		q, args, qerr := sqlxIn(`
			SELECT id, endpoint_ai_model_id, token_per_second, max_connection_time,
			       total_time, created_at::text AS created_at
			FROM (
				SELECT amp.*,
				       ROW_NUMBER() OVER (PARTITION BY endpoint_ai_model_id ORDER BY created_at DESC) AS rn
				FROM ai_model_performances amp
				WHERE endpoint_ai_model_id IN (?)
			) t
			WHERE rn <= 10
			ORDER BY created_at DESC
		`, eamIDs)
		if qerr == nil {
			_ = h.db.Select(&perfs, q, args...)
			for _, p := range perfs {
				// Infer status from metrics presence; UI mainly needs created_at + status chips
				st := "available"
				if p.TokenPerSecond == nil || (p.TokenPerSecond != nil && *p.TokenPerSecond <= 0) {
					// keep available if we have a row — historical test existed
					st = "available"
				}
				perfsByEAM[p.EAMID] = append(perfsByEAM[p.EAMID], gin.H{
					"id":               p.ID,
					"status":           st,
					"token_per_second": p.TokenPerSecond,
					"connection_time":  p.MaxConnectionTime,
					"total_time":       p.TotalTime,
					"created_at":       p.CreatedAt,
				})
			}
		}
	}

	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		timeline := perfsByEAM[r.EAMID]
		if timeline == nil {
			timeline = []gin.H{}
		}
		// Effective status for routing: only "available" if host + model + health all OK
		effective := r.ModelOnHost
		if r.HostStatus != "available" {
			effective = "unavailable" // host down even if model was good last time
		} else if r.HealthDisabled {
			effective = "unavailable"
		}
		// At least one synthetic point so StatusTimeline has something
		if len(timeline) == 0 {
			timeline = []gin.H{{
				"id":         0,
				"status":     effective,
				"created_at": r.CreatedAt,
			}}
		}
		items = append(items, gin.H{
			"id":                  r.ID,
			"url":                 r.URL,
			"name":                r.Name,
			"created_at":          r.CreatedAt,
			"status":              effective,
			"host_status":         r.HostStatus,
			"model_on_host":       r.ModelOnHost,
			"token_per_second":    r.TokenPerSecond,
			"max_connection_time": r.MaxConnectionTime,
			"model_performances":  timeline,
		})
	}

	pages := 0
	if size > 0 {
		pages = (totalCount + size - 1) / size
	}
	if pages == 0 && totalCount == 0 {
		pages = 0
	}

	// Note: frontend typo "avaliable_endpoint_count" is intentional API contract
	utils.Success(c, gin.H{
		"id":                       core.ID,
		"name":                     core.Name,
		"tag":                      core.Tag,
		"enabled":                  core.Enabled,
		"created_at":               core.CreatedAt,
		"total_endpoint_count":     totalCount,
		"avaliable_endpoint_count": availableCount,
		"endpoints": gin.H{
			"items": items,
			"total": totalCount,
			"page":  page,
			"size":  size,
			"pages": pages,
		},
	})
}

// sqlxIn expands IN (?) placeholders — thin wrapper so we don't need sqlx import churn.
func sqlxIn(query string, args []int) (string, []interface{}, error) {
	if len(args) == 0 {
		return "", nil, strconv.ErrSyntax
	}
	// manual expand
	placeholders := make([]string, len(args))
	outArgs := make([]interface{}, len(args))
	for i, a := range args {
		placeholders[i] = "$" + strconv.Itoa(i+1)
		outArgs[i] = a
	}
	// only supports single IN (?)
	expanded := ""
	// replace first "(?)" or "IN (?)"
	const needle = "IN (?)"
	idx := -1
	for i := 0; i+len(needle) <= len(query); i++ {
		if query[i:i+len(needle)] == needle {
			idx = i
			break
		}
	}
	if idx < 0 {
		return "", nil, strconv.ErrSyntax
	}
	expanded = query[:idx] + "IN (" + joinStrings(placeholders, ", ") + ")" + query[idx+len(needle):]
	return expanded, outArgs, nil
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	out := ss[0]
	for i := 1; i < len(ss); i++ {
		out += sep + ss[i]
	}
	return out
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

// ModelRescanRequest prioritizes re-testing endpoints so a model’s eam/status
// gets refreshed (and newly capable hosts can pick up the model via full tags probe).
type ModelRescanRequest struct {
	// scope: linked (default) | available | recent | all
	//   linked    — endpoints already associated with this model (incl. dead/stale)
	//   available — currently available hosts in the general pool (discover model)
	//   recent    — hosts created in the last N days (new pool)
	//   all       — linked first, then fill remaining slots from available
	Scope string `json:"scope"`
	// limit max endpoints to queue (default 200, max 2000)
	Limit int `json:"limit"`
	// recent_days only for scope=recent (default 7)
	RecentDays int `json:"recent_days"`
	// clear_health reset health-tracker disable flags so retests actually run
	ClearHealth *bool `json:"clear_health"`
}

// Rescan queues priority endpoint tests for a model.
// Full endpoint tests re-list /api/tags and rewrite endpoint_ai_models — that's
// how a "dead" glm-5.2:cloud link becomes available again (or gets pruned).
func (h *AIModelHandler) Rescan(c *gin.Context) {
	idStr := c.Param("id")
	modelID, err := strconv.Atoi(idStr)
	if err != nil || modelID <= 0 {
		utils.BadRequest(c, "Invalid model ID")
		return
	}

	var model struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
		Tag  string `db:"tag"`
	}
	if err := h.db.Get(&model, `SELECT id, name, tag FROM ai_models WHERE id = $1`, modelID); err != nil {
		utils.NotFound(c, "AI model not found")
		return
	}

	var req ModelRescanRequest
	_ = c.ShouldBindJSON(&req) // empty body OK
	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = "linked"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	recentDays := req.RecentDays
	if recentDays <= 0 {
		recentDays = 7
	}
	clearHealth := true
	if req.ClearHealth != nil {
		clearHealth = *req.ClearHealth
	}

	var endpointIDs []int
	switch scope {
	case "linked":
		// Every host we think has this model — including unavailable endpoints
		_ = h.db.Select(&endpointIDs, `
			SELECT e.id
			FROM endpoint_ai_models eam
			JOIN endpoints e ON e.id = eam.endpoint_id
			WHERE eam.ai_model_id = $1
			ORDER BY
			  CASE WHEN e.status = 'available' THEN 0 ELSE 1 END,
			  CASE WHEN eam.status = 'available' THEN 0 ELSE 1 END,
			  eam.token_per_second DESC NULLS LAST
			LIMIT $2
		`, modelID, limit)

	case "available":
		// General healthy pool — full tags rescan may discover this model
		_ = h.db.Select(&endpointIDs, `
			SELECT e.id
			FROM endpoints e
			LEFT JOIN endpoint_health eh ON eh.url = e.url
			WHERE e.status = 'available'
			  AND (eh.disabled IS NOT TRUE OR eh.disabled_until IS NULL OR eh.disabled_until < NOW())
			ORDER BY e.id DESC
			LIMIT $1
		`, limit)

	case "recent":
		_ = h.db.Select(&endpointIDs, `
			SELECT e.id
			FROM endpoints e
			WHERE e.created_at >= NOW() - ($1 * INTERVAL '1 day')
			ORDER BY e.created_at DESC
			LIMIT $2
		`, recentDays, limit)

	case "all":
		// Linked first, then top up from available pool
		_ = h.db.Select(&endpointIDs, `
			SELECT e.id
			FROM endpoint_ai_models eam
			JOIN endpoints e ON e.id = eam.endpoint_id
			WHERE eam.ai_model_id = $1
			ORDER BY
			  CASE WHEN e.status = 'available' THEN 0 ELSE 1 END,
			  eam.token_per_second DESC NULLS LAST
			LIMIT $2
		`, modelID, limit)
		if len(endpointIDs) < limit {
			seen := map[int]bool{}
			for _, id := range endpointIDs {
				seen[id] = true
			}
			var more []int
			_ = h.db.Select(&more, `
				SELECT e.id
				FROM endpoints e
				WHERE e.status = 'available'
				ORDER BY e.id DESC
				LIMIT $1
			`, limit*2) // oversample then filter
			for _, id := range more {
				if seen[id] {
					continue
				}
				endpointIDs = append(endpointIDs, id)
				if len(endpointIDs) >= limit {
					break
				}
			}
		}

	default:
		utils.BadRequest(c, "scope must be one of: linked, available, recent, all")
		return
	}

	if len(endpointIDs) == 0 {
		utils.Success(c, gin.H{
			"model_id":   modelID,
			"model":      fmt.Sprintf("%s:%s", model.Name, model.Tag),
			"scope":      scope,
			"queued":     0,
			"message":    "No endpoints matched this scope",
		})
		return
	}

	// Priority: scheduled_at in the past so they run before normal cyclical queue
	priorityAt := time.Now().Add(-1 * time.Hour)

	// Drop existing pending tasks for these endpoints so we don't double-queue
	if q, args, err := expandIntIn(
		`DELETE FROM endpoint_test_tasks WHERE status = 'pending' AND endpoint_id IN (?)`,
		endpointIDs,
	); err == nil {
		_, _ = h.db.Exec(q, args...)
	}

	// Bulk insert priority tasks
	query := "INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES "
	var args []interface{}
	for i, eid := range endpointIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		args = append(args, eid, priorityAt, "pending")
	}
	if _, err := h.db.Exec(query, args...); err != nil {
		log.Printf("[model-rescan] failed to queue tasks for model %d: %v", modelID, err)
		utils.InternalServerError(c, "Failed to queue rescan tasks")
		return
	}

	// Clear sticky health disables so FilterHealthyEndpoints / probes don't skip them
	clearedHealth := 0
	if clearHealth {
		if q, args, err := expandIntIn(`
			UPDATE endpoint_health eh
			SET disabled = false, disabled_until = NULL, score = GREATEST(score, 40)
			FROM endpoints e
			WHERE eh.url = e.url AND e.id IN (?)
		`, endpointIDs); err == nil {
			if res, err := h.db.Exec(q, args...); err == nil {
				n, _ := res.RowsAffected()
				clearedHealth = int(n)
			}
		}
	}

	log.Printf("[model-rescan] model %s:%s scope=%s queued=%d cleared_health=%d",
		model.Name, model.Tag, scope, len(endpointIDs), clearedHealth)

	utils.Success(c, gin.H{
		"model_id":            modelID,
		"model":               fmt.Sprintf("%s:%s", model.Name, model.Tag),
		"scope":               scope,
		"queued":              len(endpointIDs),
		"priority":            true,
		"cleared_health_rows": clearedHealth,
		"message": fmt.Sprintf(
			"Queued %d endpoint retests at priority for %s:%s (full /api/tags rescan)",
			len(endpointIDs), model.Name, model.Tag,
		),
	})
}

func expandIntIn(query string, ids []int) (string, []interface{}, error) {
	if len(ids) == 0 {
		return "", nil, fmt.Errorf("empty ids")
	}
	ph := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		ph[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	const needle = "IN (?)"
	idx := strings.Index(query, needle)
	if idx < 0 {
		return "", nil, fmt.Errorf("no IN (?) in query")
	}
	out := query[:idx] + "IN (" + strings.Join(ph, ", ") + ")" + query[idx+len(needle):]
	return out, args, nil
}



// SmartModels returns the current smart model resolutions
func (h *AIModelHandler) SmartModels(c *gin.Context) {
	smartProfiles := []string{"fastest", "large", "small", "coding", "cloud", "abliterated"}
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
			LEFT JOIN endpoint_health eh ON eh.url = e.url
			WHERE ` + heuristic + `
			  AND ` + routableEndpointSQL + `
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
