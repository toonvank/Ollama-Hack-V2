package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/services"
	"github.com/timlzh/ollama-hack/internal/utils"
)

type DiscoveryHandler struct {
	db      *database.DB
	scanner *services.DiscoveryScanner
}

func NewDiscoveryHandler(db *database.DB, scanner *services.DiscoveryScanner) *DiscoveryHandler {
	return &DiscoveryHandler{
		db:      db,
		scanner: scanner,
	}
}

type ManualScanRequest struct {
	IPRange string `json:"ip_range" binding:"required"`
}

// TriggerManualScan triggers a manual discovery scan
func (h *DiscoveryHandler) TriggerManualScan(c *gin.Context) {
	if h.scanner == nil {
		utils.BadRequest(c, "Discovery scanner is disabled (BACKGROUND_ENDPOINT_OUTBOUND=false)")
		return
	}

	var req ManualScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	if err := h.scanner.ManualScan(req.IPRange); err != nil {
		utils.InternalServerError(c, "Failed to trigger scan: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Manual scan triggered",
		"ip_range": req.IPRange,
	})
}

// GetScanStatus returns the current status of the discovery scanner
func (h *DiscoveryHandler) GetScanStatus(c *gin.Context) {
	if h.scanner == nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "disabled",
			"message": "Discovery scanner disabled (BACKGROUND_ENDPOINT_OUTBOUND=false)",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "running",
		"message": "Discovery scanner is active",
	})
}
