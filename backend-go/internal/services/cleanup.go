package services

import (
	"log"
	"time"

	"github.com/timlzh/ollama-hack/internal/database"
)

// BackgroundCleanupService handles cleaning up old, unavailable nodes
type BackgroundCleanupService struct {
	db       *database.DB
	interval time.Duration
	stop     chan struct{}
}

func NewBackgroundCleanupService(db *database.DB) *BackgroundCleanupService {
	return &BackgroundCleanupService{
		db:       db,
		interval: 1 * time.Hour,
		stop:     make(chan struct{}),
	}
}

func (s *BackgroundCleanupService) Start() {
	log.Println("[cleanup] starting background dead node cleanup service")
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.cleanupDeadNodes() // run once at startup

		for {
			select {
			case <-ticker.C:
				s.cleanupDeadNodes()
			case <-s.stop:
				log.Println("[cleanup] stopped background cleanup service")
				return
			}
		}
	}()
}

func (s *BackgroundCleanupService) Stop() {
	close(s.stop)
}

func (s *BackgroundCleanupService) cleanupDeadNodes() {
	log.Println("[cleanup] checking for nodes unavailable > 3 days")

	// Only purge endpoints that have been confirmed unavailable for 3+ days.
	// Use the most recent probe time (not created_at) so VPN/DNS blips that briefly
	// flip healthy nodes to unavailable are not deleted on the next hourly run.
	query := `
		DELETE FROM endpoints e
		WHERE e.status = 'unavailable'
		AND COALESCE(
			(SELECT MAX(ett.last_tried) FROM endpoint_test_tasks ett WHERE ett.endpoint_id = e.id),
			(SELECT eh.last_fail FROM endpoint_health eh WHERE eh.url = e.url),
			e.created_at
		) < NOW() - INTERVAL '3 days'
	`

	res, err := s.db.Exec(query)
	if err != nil {
		log.Printf("[cleanup] failed to delete dead nodes: %v", err)
		return
	}

	count, _ := res.RowsAffected()
	if count > 0 {
		log.Printf("[cleanup] permanently removed %d dead nodes", count)
	}
}
