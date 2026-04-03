package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/services"
)

// AdminHealthStatus returns the health status of all tracked endpoints
func AdminHealthStatus(c *gin.Context) {
	tracker := services.GetHealthTracker()
	config := tracker.GetConfig()

	healthData := tracker.GetAllHealth()

	// Count statistics
	var totalEndpoints, healthyEndpoints, disabledEndpoints int
	for _, h := range healthData {
		totalEndpoints++
		if h.Disabled {
			disabledEndpoints++
		} else {
			healthyEndpoints++
		}
	}

	c.JSON(200, gin.H{
		"enabled": config.Enabled,
		"config": gin.H{
			"disable_threshold":  config.DisableThreshold,
			"disable_duration":   config.DisableDuration.String(),
			"probe_interval":     config.ProbeInterval.String(),
			"fail_penalty":       config.FailPenalty,
			"success_reward":     config.SuccessReward,
			"max_score":          config.MaxScore,
			"initial_score":      config.InitialScore,
		},
		"summary": gin.H{
			"total_endpoints":    totalEndpoints,
			"healthy_endpoints":  healthyEndpoints,
			"disabled_endpoints": disabledEndpoints,
		},
		"endpoints": healthData,
	})
}

// AdminResetEndpointHealth resets the health status of a specific endpoint
func AdminResetEndpointHealth(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(400, gin.H{"error": "url query parameter is required"})
		return
	}

	tracker := services.GetHealthTracker()
	tracker.ResetEndpoint(url)

	c.JSON(200, gin.H{
		"message": "Endpoint health reset",
		"url":     url,
	})
}
