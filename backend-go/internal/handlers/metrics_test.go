package handlers

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLiveMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request with context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest("GET", "/metrics", nil).WithContext(ctx)

	// Give it a timeout to prevent infinite loop
	done := make(chan struct{})
	go func() {
		LiveMetrics(c)
		close(done)
	}()

	// Wait for at least one SSE message
	time.Sleep(100 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for handler to finish
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("LiveMetrics did not finish in time")
	}

	// Check SSE headers
	if w.Header().Get("Content-Type") != "text/event-stream" {
		t.Errorf("Expected Content-Type: text/event-stream, got %s", w.Header().Get("Content-Type"))
	}
	if w.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("Expected Cache-Control: no-cache, got %s", w.Header().Get("Cache-Control"))
	}

	// Check SSE format in response body
	body := w.Body.String()
	if !strings.Contains(body, "event: message") {
		t.Error("Expected SSE event message in response")
	}
	if !strings.Contains(body, "data:") {
		t.Error("Expected data payload in SSE response")
	}
	if !strings.Contains(body, "total_requests") {
		t.Error("Expected total_requests in metrics data")
	}
}
