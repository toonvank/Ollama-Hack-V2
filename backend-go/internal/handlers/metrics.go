package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/utils"
)

// LiveMetrics streams proxy stats via SSE
func LiveMetrics(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	c.Writer.Flush()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		default:
			stats := gin.H{
				"total_requests":  utils.TotalRequests.Load(),
				"cache_hits":      utils.CacheHits.Load(),
				"active_requests": utils.ActiveRequests.Load(),
				"failed_requests": utils.FailedRequests.Load(),
			}

			data, _ := json.Marshal(stats)
			fmt.Fprintf(c.Writer, "event: message\ndata: %s\n\n", string(data))
			c.Writer.Flush()

			time.Sleep(1 * time.Second)
		}
	}
}
