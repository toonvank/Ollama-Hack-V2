package handlers

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/models"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type EndpointHandler struct {
	db *database.DB
}

func NewEndpointHandler(db *database.DB) *EndpointHandler {
	return &EndpointHandler{db: db}
}

func (h *EndpointHandler) List(c *gin.Context) {
	// Parse pagination and sorting parameters
	page := 1
	pageSize := 50
	orderBy := "id"
	order := "asc"
	statusFilter := ""

	if p := c.Query("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil && val > 0 {
			page = val
		}
	}
	if ps := c.Query("size"); ps != "" {
		if val, err := strconv.Atoi(ps); err == nil && val > 0 {
			pageSize = val
		}
	}
	if ob := c.Query("order_by"); ob != "" {
		// Validate order_by field to prevent SQL injection
		allowedFields := map[string]bool{"id": true, "name": true, "url": true, "status": true, "created_at": true}
		if allowedFields[ob] {
			orderBy = ob
		}
	}
	if o := c.Query("order"); o != "" {
		if o == "asc" || o == "desc" {
			order = o
		}
	}
	if s := c.Query("status"); s != "" {
		statusFilter = s
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Build WHERE clause
	whereClause := ""
	var countArgs []interface{}
	var queryArgs []interface{}

	if statusFilter != "" {
		whereClause = "WHERE status = $1"
		countArgs = append(countArgs, statusFilter)
		queryArgs = append(queryArgs, statusFilter, pageSize, offset)
	} else {
		queryArgs = append(queryArgs, pageSize, offset)
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM endpoints " + whereClause
	if err := h.db.Get(&total, countQuery, countArgs...); err != nil {
		utils.InternalServerError(c, "Failed to count endpoints")
		return
	}

	// Fetch paginated and sorted data
	var query string
	if statusFilter != "" {
		query = fmt.Sprintf("SELECT * FROM endpoints WHERE status = $1 ORDER BY %s %s LIMIT $2 OFFSET $3", orderBy, order)
	} else {
		query = fmt.Sprintf("SELECT * FROM endpoints ORDER BY %s %s LIMIT $1 OFFSET $2", orderBy, order)
	}

	var endpoints []models.Endpoint
	if err := h.db.Select(&endpoints, query, queryArgs...); err != nil {
		utils.InternalServerError(c, "Failed to fetch endpoints")
		return
	}

	// Build response with additional data for each endpoint
	result := make([]models.EndpointWithAIModelCount, 0, len(endpoints))
	for _, ep := range endpoints {
		// Get recent performances (last 5)
		var performances []models.EndpointPerformance
		h.db.Select(&performances,
			"SELECT id, status, ollama_version, created_at FROM endpoint_performances WHERE endpoint_id = $1 ORDER BY created_at DESC LIMIT 5",
			ep.ID)
		if performances == nil {
			performances = []models.EndpointPerformance{}
		}

		// Get AI model counts
		var totalCount int
		var availableCount int
		h.db.Get(&totalCount, "SELECT COUNT(*) FROM endpoint_ai_models WHERE endpoint_id = $1", ep.ID)
		h.db.Get(&availableCount, "SELECT COUNT(*) FROM endpoint_ai_models WHERE endpoint_id = $1 AND status = 'available'", ep.ID)

		// Get latest task status
		var taskStatus *string
		var status string
		err := h.db.Get(&status, "SELECT status FROM endpoint_test_tasks WHERE endpoint_id = $1 ORDER BY created_at DESC LIMIT 1", ep.ID)
		if err == nil {
			taskStatus = &status
		}

		result = append(result, models.EndpointWithAIModelCount{
			ID:                    ep.ID,
			URL:                   ep.URL,
			Name:                  ep.Name,
			Status:                ep.Status,
			CreatedAt:             ep.CreatedAt,
			RecentPerformances:    performances,
			TotalAIModelCount:     totalCount,
			AvailableAIModelCount: availableCount,
			TaskStatus:            taskStatus,
		})
	}

	// Calculate total pages
	pages := (total + pageSize - 1) / pageSize

	utils.SuccessPage(c, result, total, page, pageSize, pages)
}

func (h *EndpointHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var endpoint models.Endpoint
	if err := h.db.Get(&endpoint, "SELECT * FROM endpoints WHERE id = $1", id); err != nil {
		utils.NotFound(c, "Endpoint not found")
		return
	}
	utils.Success(c, endpoint)
}

func (h *EndpointHandler) Create(c *gin.Context) {
	var req models.EndpointCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// Default name to URL if not provided
	if req.Name == "" {
		req.Name = req.URL
	}

	var endpoint models.Endpoint
	// In Python code, if it exists, it's updated. But we will just try to insert and conflict if unique constraint hit.
	// We'll follow a simple insert first.
	err := h.db.Get(&endpoint,
		"INSERT INTO endpoints (url, name) VALUES ($1, $2) RETURNING *",
		req.URL, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			// Get existing
			err = h.db.Get(&endpoint, "SELECT * FROM endpoints WHERE url = $1", req.URL)
			if err != nil {
				utils.InternalServerError(c, "Failed to handle existing endpoint")
				return
			}
		} else {
			utils.InternalServerError(c, "Failed to create endpoint")
			return
		}
	}

	// Schedule task
	h.scheduleTask(endpoint.ID)

	utils.Created(c, endpoint)
}

func (h *EndpointHandler) BatchCreate(c *gin.Context) {
	var req models.EndpointBatchCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if len(req.Endpoints) == 0 {
		utils.Success(c, gin.H{"message": "No endpoints provided"})
		return
	}

	// 1. Deduplicate in request
	seenURLs := make(map[string]bool)
	var uniqueURLs []string
	urlToName := make(map[string]string)

	for _, ep := range req.Endpoints {
		if !seenURLs[ep.URL] {
			seenURLs[ep.URL] = true
			uniqueURLs = append(uniqueURLs, ep.URL)
			name := ep.Name
			if name == "" {
				name = ep.URL
			}
			urlToName[ep.URL] = name
		}
	}

	// 2. Find existing URLs
	query, args, err := sqlx.In("SELECT url, id FROM endpoints WHERE url IN (?)", uniqueURLs)
	if err != nil {
		utils.InternalServerError(c, "Failed to prepare query")
		return
	}
	query = h.db.Rebind(query)

	type URLID struct {
		URL string `db:"url"`
		ID  int    `db:"id"`
	}
	var existingRows []URLID
	if err := h.db.Select(&existingRows, query, args...); err != nil {
		utils.InternalServerError(c, "Failed to fetch existing endpoints")
		return
	}

	existingMap := make(map[string]int)
	for _, row := range existingRows {
		existingMap[row.URL] = row.ID
	}

	// 3. Insert new URLs
	var newURLs []string
	for _, u := range uniqueURLs {
		if _, exists := existingMap[u]; !exists {
			newURLs = append(newURLs, u)
		}
	}

	if len(newURLs) > 0 {
		tx, err := h.db.Beginx()
		if err != nil {
			utils.InternalServerError(c, "Transaction error")
			return
		}

		for _, u := range newURLs {
			var newID int
			err := tx.QueryRow("INSERT INTO endpoints (url, name) VALUES ($1, $2) RETURNING id", u, urlToName[u]).Scan(&newID)
			if err != nil {
				tx.Rollback()
				utils.InternalServerError(c, "Failed to insert batch")
				return
			}
			existingMap[u] = newID
		}
		tx.Commit()
	}

	// 4. Schedule tasks in bulk
	var allIDs []int
	for _, id := range existingMap {
		allIDs = append(allIDs, id)
	}

	if len(allIDs) > 0 {
		h.bulkScheduleTasks(allIDs)
	}

	utils.Success(c, gin.H{
		"message": "Batch creation successful",
		"count":   len(allIDs),
	})
}

func (h *EndpointHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req models.EndpointUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var setClauses []string
	var args []interface{}
	argID := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.URL != nil {
		setClauses = append(setClauses, fmt.Sprintf("url = $%d", argID))
		args = append(args, *req.URL)
		argID++
	}

	if len(setClauses) == 0 {
		utils.BadRequest(c, "No fields to update")
		return
	}

	args = append(args, id)
	query := fmt.Sprintf("UPDATE endpoints SET %s WHERE id = $%d RETURNING *", strings.Join(setClauses, ", "), argID)

	var endpoint models.Endpoint
	err := h.db.Get(&endpoint, query, args...)
	if err != nil {
		utils.InternalServerError(c, "Failed to update endpoint")
		return
	}

	utils.Success(c, endpoint)
}

func (h *EndpointHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	_, err := h.db.Exec("DELETE FROM endpoints WHERE id = $1", id)
	if err != nil {
		utils.InternalServerError(c, "Failed to delete endpoint")
		return
	}
	utils.NoContent(c)
}

func (h *EndpointHandler) BatchDelete(c *gin.Context) {
	var req models.EndpointBatchOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if len(req.EndpointIDs) == 0 {
		utils.Success(c, models.BatchOperationResult{})
		return
	}

	query, args, err := sqlx.In("DELETE FROM endpoints WHERE id IN (?)", req.EndpointIDs)
	if err != nil {
		utils.InternalServerError(c, "Failed to prepare query")
		return
	}
	query = h.db.Rebind(query)

	res, err := h.db.Exec(query, args...)
	if err != nil {
		utils.InternalServerError(c, "Failed to delete endpoints")
		return
	}

	affected, _ := res.RowsAffected()
	utils.Success(c, models.BatchOperationResult{
		SuccessCount: int(affected),
		FailedCount:  len(req.EndpointIDs) - int(affected),
		FailedIDs:    map[string]string{}, // Simplified
	})
}

func (h *EndpointHandler) BatchTest(c *gin.Context) {
	var req models.EndpointBatchOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if len(req.EndpointIDs) == 0 {
		utils.Success(c, models.BatchOperationResult{})
		return
	}

	successCount := 0
	failedCount := 0
	failedIDs := make(map[string]string)

	var validIDs []int
	for _, id := range req.EndpointIDs {
		// Verify exists
		var count int
		h.db.Get(&count, "SELECT COUNT(*) FROM endpoints WHERE id = $1", id)
		if count == 0 {
			failedCount++
			failedIDs[fmt.Sprintf("%d", id)] = "Endpoint not found"
			continue
		}
		validIDs = append(validIDs, id)
		successCount++
	}

	if len(validIDs) > 0 {
		h.bulkScheduleTasks(validIDs)
	}

	utils.Success(c, models.BatchOperationResult{
		SuccessCount: successCount,
		FailedCount:  failedCount,
		FailedIDs:    failedIDs,
	})
}

// scheduleTask acts as a simple background scheduler replacement for now
// (inserts task, which would be picked up by a cron/background job).
func (h *EndpointHandler) scheduleTask(endpointID int) {
	now := time.Now().Add(5 * time.Second)
	_, err := h.db.Exec(
		"INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES ($1, $2, $3)",
		endpointID, now, "pending",
	)
	if err != nil {
		fmt.Printf("Error scheduling task for endpoint %d: %v\n", endpointID, err)
	}
}

// bulkScheduleTasks schedules multiple tasks with a single bulk insert
func (h *EndpointHandler) bulkScheduleTasks(endpointIDs []int) {
	if len(endpointIDs) == 0 {
		return
	}
	now := time.Now().Add(5 * time.Second)

	query := "INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES "
	var args []interface{}
	for i, id := range endpointIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d, $%d)", i*3+1, i*3+2, i*3+3)
		args = append(args, id, now, "pending")
	}

	_, err := h.db.Exec(query, args...)
	if err != nil {
		fmt.Printf("Error bulk scheduling tasks: %v\n", err)
	}
}

func (h *EndpointHandler) TriggerTest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid endpoint ID")
		return
	}
	var count int
	h.db.Get(&count, "SELECT COUNT(*) FROM endpoints WHERE id = $1", id)
	if count == 0 {
		utils.NotFound(c, "Endpoint not found")
		return
	}
	h.scheduleTask(id)
	utils.Success(c, gin.H{"message": "Test triggered"})
}

func (h *EndpointHandler) GetTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid endpoint ID")
		return
	}
	var task models.EndpointTestTask
	err = h.db.Get(&task, "SELECT * FROM endpoint_test_tasks WHERE endpoint_id = $1 ORDER BY created_at DESC LIMIT 1", id)
	if err != nil {
		utils.NotFound(c, "Task not found")
		return
	}
	utils.Success(c, task)
}

func (h *EndpointHandler) BatchGetTasks(c *gin.Context) {
	var req models.EndpointBatchOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if len(req.EndpointIDs) == 0 {
		utils.Success(c, make(map[int]string))
		return
	}

	query, args, err := sqlx.In(`
		SELECT endpoint_id, status 
		FROM (
			SELECT endpoint_id, status, 
				   ROW_NUMBER() OVER(PARTITION BY endpoint_id ORDER BY created_at DESC) as rn 
			FROM endpoint_test_tasks 
			WHERE endpoint_id IN (?)
		) sub 
		WHERE rn = 1`, req.EndpointIDs)
	if err != nil {
		utils.InternalServerError(c, "Failed to prepare query")
		return
	}
	query = h.db.Rebind(query)

	type TaskStatusRow struct {
		EndpointID int    `db:"endpoint_id"`
		Status     string `db:"status"`
	}

	var rows []TaskStatusRow
	if err := h.db.Select(&rows, query, args...); err != nil {
		utils.InternalServerError(c, "Failed to fetch task statuses")
		return
	}

	statuses := make(map[int]string)
	for _, r := range rows {
		statuses[r.EndpointID] = r.Status
	}

	utils.Success(c, statuses)
}
