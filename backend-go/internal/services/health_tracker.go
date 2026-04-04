package services

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/timlzh/ollama-hack/internal/database"
)

// EndpointHealth holds the health status and metrics for a single endpoint
type EndpointHealth struct {
	URL           string    `json:"url"`
	Score         int       `json:"score"`
	FailCount     int       `json:"fail_count"`
	SuccessCount  int       `json:"success_count"`
	LastFail      time.Time `json:"last_fail,omitempty"`
	LastSuccess   time.Time `json:"last_success,omitempty"`
	Disabled      bool      `json:"disabled"`
	DisabledUntil time.Time `json:"disabled_until,omitempty"`
	LastProbe     time.Time `json:"last_probe,omitempty"`
}

// HealthTrackerConfig holds configuration for the health tracker
type HealthTrackerConfig struct {
	Enabled          bool
	DisableThreshold int
	DisableDuration  time.Duration
	ProbeInterval    time.Duration
	FailPenalty      int
	SuccessReward    int
	MaxScore         int
	InitialScore     int
}

// HealthTracker tracks the health of all endpoints
type HealthTracker struct {
	mu       sync.RWMutex
	health   map[string]*EndpointHealth
	config   HealthTrackerConfig
	stopChan chan struct{}
	db       *database.DB
}

// Global health tracker instance
var globalHealthTracker *HealthTracker
var healthTrackerOnce sync.Once

// InitHealthTracker initializes the global health tracker with DB
func InitHealthTracker(db *database.DB) {
	healthTrackerOnce.Do(func() {
		globalHealthTracker = NewHealthTracker(loadHealthConfig(), db)
	})
}

// GetHealthTracker returns the global health tracker instance
func GetHealthTracker() *HealthTracker {
	if globalHealthTracker == nil {
		log.Println("[health-tracker] Warning: HealthTracker accessed before initialization")
		healthTrackerOnce.Do(func() {
			globalHealthTracker = NewHealthTracker(loadHealthConfig(), nil)
		})
	}
	return globalHealthTracker
}

// loadHealthConfig loads configuration from environment variables
func loadHealthConfig() HealthTrackerConfig {
	config := HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}

	if val := os.Getenv("HEALTH_TRACKING_ENABLED"); val != "" {
		config.Enabled = val == "true" || val == "1"
	}

	if val := os.Getenv("HEALTH_DISABLE_THRESHOLD"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n >= 0 && n <= 100 {
			config.DisableThreshold = n
		}
	}

	if val := os.Getenv("HEALTH_DISABLE_DURATION"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.DisableDuration = d
		}
	}

	if val := os.Getenv("HEALTH_PROBE_INTERVAL"); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			config.ProbeInterval = d
		}
	}

	if val := os.Getenv("HEALTH_FAIL_PENALTY"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.FailPenalty = n
		}
	}

	if val := os.Getenv("HEALTH_SUCCESS_REWARD"); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			config.SuccessReward = n
		}
	}

	return config
}

// NewHealthTracker creates a new health tracker with the given configuration
func NewHealthTracker(config HealthTrackerConfig, db *database.DB) *HealthTracker {
	ht := &HealthTracker{
		health:   make(map[string]*EndpointHealth),
		config:   config,
		stopChan: make(chan struct{}),
		db:       db,
	}

	if ht.db != nil {
		ht.loadFromDB()
	}

	if config.Enabled {
		go ht.probeLoop()
		if ht.db != nil {
			go ht.persistLoop()
		}
		log.Printf("[health-tracker] Started with threshold=%d, disable_duration=%v, probe_interval=%v",
			config.DisableThreshold, config.DisableDuration, config.ProbeInterval)
	} else {
		log.Println("[health-tracker] Health tracking is disabled")
	}

	return ht
}

// getOrCreateHealth gets or creates a health entry for an endpoint
func (ht *HealthTracker) getOrCreateHealth(url string) *EndpointHealth {
	if h, exists := ht.health[url]; exists {
		return h
	}
	h := &EndpointHealth{
		URL:   url,
		Score: ht.config.InitialScore,
	}
	ht.health[url] = h
	return h
}

// RecordSuccess records a successful request to an endpoint
func (ht *HealthTracker) RecordSuccess(url string) {
	if !ht.config.Enabled {
		return
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	h := ht.getOrCreateHealth(url)
	h.SuccessCount++
	h.LastSuccess = time.Now()
	h.Score += ht.config.SuccessReward
	if h.Score > ht.config.MaxScore {
		h.Score = ht.config.MaxScore
	}

	// If endpoint was disabled and score has recovered above threshold, re-enable it
	if h.Disabled && h.Score > ht.config.DisableThreshold {
		h.Disabled = false
		h.DisabledUntil = time.Time{}
		log.Printf("[health-tracker] Endpoint %s re-enabled after recovery (score: %d)", url, h.Score)
	}
}

// RecordFailure records a failed request to an endpoint
func (ht *HealthTracker) RecordFailure(url string) {
	if !ht.config.Enabled {
		return
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()

	h := ht.getOrCreateHealth(url)
	h.FailCount++
	h.LastFail = time.Now()
	h.Score -= ht.config.FailPenalty
	if h.Score < 0 {
		h.Score = 0
	}

	// Check if we should disable the endpoint
	if !h.Disabled && h.Score <= ht.config.DisableThreshold {
		h.Disabled = true
		h.DisabledUntil = time.Now().Add(ht.config.DisableDuration)
		log.Printf("[health-tracker] Endpoint %s DISABLED (score: %d, until: %v)",
			url, h.Score, h.DisabledUntil.Format(time.RFC3339))
	}
}

// IsDisabled checks if an endpoint is currently disabled
func (ht *HealthTracker) IsDisabled(url string) bool {
	if !ht.config.Enabled {
		return false
	}

	ht.mu.RLock()
	defer ht.mu.RUnlock()

	h, exists := ht.health[url]
	if !exists {
		return false
	}

	if !h.Disabled {
		return false
	}

	// Check if disable period has expired
	if time.Now().After(h.DisabledUntil) {
		return false // Will be re-enabled on next probe
	}

	return true
}

// FilterHealthyEndpoints returns only the healthy (non-disabled) endpoints
func (ht *HealthTracker) FilterHealthyEndpoints(urls []string) []string {
	if !ht.config.Enabled {
		return urls
	}

	ht.mu.RLock()
	defer ht.mu.RUnlock()

	healthy := make([]string, 0, len(urls))
	for _, url := range urls {
		h, exists := ht.health[url]
		if !exists {
			// New endpoint, assume healthy
			healthy = append(healthy, url)
			continue
		}

		if !h.Disabled {
			healthy = append(healthy, url)
			continue
		}

		// Check if disable period has expired
		if time.Now().After(h.DisabledUntil) {
			healthy = append(healthy, url)
		}
	}

	return healthy
}

// GetHealth returns the health status for a specific endpoint
func (ht *HealthTracker) GetHealth(url string) *EndpointHealth {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	if h, exists := ht.health[url]; exists {
		// Return a copy to avoid race conditions
		copy := *h
		return &copy
	}
	return nil
}

// GetAllHealth returns health status for all tracked endpoints
func (ht *HealthTracker) GetAllHealth() []EndpointHealth {
	ht.mu.RLock()
	defer ht.mu.RUnlock()

	result := make([]EndpointHealth, 0, len(ht.health))
	for _, h := range ht.health {
		result = append(result, *h)
	}
	return result
}

// GetConfig returns the current configuration
func (ht *HealthTracker) GetConfig() HealthTrackerConfig {
	return ht.config
}

func (ht *HealthTracker) loadFromDB() {
	var rows []struct {
		URL           string       `db:"url"`
		Score         int          `db:"score"`
		FailCount     int          `db:"fail_count"`
		SuccessCount  int          `db:"success_count"`
		Disabled      bool         `db:"disabled"`
		DisabledUntil sql.NullTime `db:"disabled_until"`
		LastSuccess   sql.NullTime `db:"last_success"`
		LastFail      sql.NullTime `db:"last_fail"`
		LastProbe     sql.NullTime `db:"last_probe"`
		UpdatedAt     sql.NullTime `db:"updated_at"`
	}

	err := ht.db.Select(&rows, "SELECT * FROM endpoint_health")
	if err != nil {
		log.Printf("[health-tracker] Failed to load state from DB: %v", err)
		return
	}

	ht.mu.Lock()
	defer ht.mu.Unlock()
	for _, r := range rows {
		h := &EndpointHealth{
			URL:          r.URL,
			Score:        r.Score,
			FailCount:    r.FailCount,
			SuccessCount: r.SuccessCount,
			Disabled:     r.Disabled,
		}
		if r.DisabledUntil.Valid {
			h.DisabledUntil = r.DisabledUntil.Time
		}
		if r.LastSuccess.Valid {
			h.LastSuccess = r.LastSuccess.Time
		}
		if r.LastFail.Valid {
			h.LastFail = r.LastFail.Time
		}
		if r.LastProbe.Valid {
			h.LastProbe = r.LastProbe.Time
		}
		ht.health[r.URL] = h
	}
	log.Printf("[health-tracker] Loaded %d endpoints from DB", len(rows))
}

func (ht *HealthTracker) persistLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ht.persistToDB()
		case <-ht.stopChan:
			ht.persistToDB()
			return
		}
	}
}

func (ht *HealthTracker) persistToDB() {
	if ht.db == nil {
		return
	}

	ht.mu.RLock()
	var toSave []EndpointHealth
	for _, h := range ht.health {
		toSave = append(toSave, *h)
	}
	ht.mu.RUnlock()

	for _, h := range toSave {
		_, err := ht.db.Exec(`
			INSERT INTO endpoint_health (
				url, score, success_count, fail_count, disabled, 
				disabled_until, last_success, last_fail, last_probe
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (url) DO UPDATE SET
				score = EXCLUDED.score,
				success_count = EXCLUDED.success_count,
				fail_count = EXCLUDED.fail_count,
				disabled = EXCLUDED.disabled,
				disabled_until = EXCLUDED.disabled_until,
				last_success = EXCLUDED.last_success,
				last_fail = EXCLUDED.last_fail,
				last_probe = EXCLUDED.last_probe,
				updated_at = CURRENT_TIMESTAMP
		`, h.URL, h.Score, h.SuccessCount, h.FailCount, h.Disabled, 
		nullTime(h.DisabledUntil), nullTime(h.LastSuccess), nullTime(h.LastFail), nullTime(h.LastProbe))
		if err != nil {
			log.Printf("[health-tracker] Failed to save %s: %v", h.URL, err)
		}
	}
}

func nullTime(t time.Time) sql.NullTime {
    if t.IsZero() {
        return sql.NullTime{Valid: false}
    }
    return sql.NullTime{Time: t, Valid: true}
}

// probeLoop runs periodically to check disabled endpoints for recovery
func (ht *HealthTracker) probeLoop() {
	ticker := time.NewTicker(ht.config.ProbeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ht.probeDisabledEndpoints()
		case <-ht.stopChan:
			log.Println("[health-tracker] Probe loop stopped")
			return
		}
	}
}

// probeDisabledEndpoints checks disabled endpoints to see if they've recovered
func (ht *HealthTracker) probeDisabledEndpoints() {
	ht.mu.RLock()
	var toProbe []string
	for url, h := range ht.health {
		if h.Disabled && time.Now().After(h.DisabledUntil) {
			toProbe = append(toProbe, url)
		}
	}
	ht.mu.RUnlock()

	for _, url := range toProbe {
		go ht.probeEndpoint(url)
	}
}

// probeEndpoint sends a simple health check to an endpoint
func (ht *HealthTracker) probeEndpoint(url string) {
	ht.mu.Lock()
	h, exists := ht.health[url]
	if !exists {
		ht.mu.Unlock()
		return
	}
	h.LastProbe = time.Now()
	ht.mu.Unlock()

	// Simple health check - just check if the endpoint responds
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url + "/api/version")
	if err != nil {
		log.Printf("[health-tracker] Probe failed for %s: %v", url, err)
		// Keep it disabled, extend the disable period
		ht.mu.Lock()
		if h, exists := ht.health[url]; exists && h.Disabled {
			h.DisabledUntil = time.Now().Add(ht.config.DisableDuration)
		}
		ht.mu.Unlock()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Endpoint is responding, give it a chance
		ht.mu.Lock()
		if h, exists := ht.health[url]; exists {
			h.Disabled = false
			h.DisabledUntil = time.Time{}
			// Give it a partial score boost
			h.Score = ht.config.DisableThreshold + 10
			log.Printf("[health-tracker] Probe successful for %s, re-enabled with score %d", url, h.Score)
		}
		ht.mu.Unlock()
	} else {
		log.Printf("[health-tracker] Probe got status %d for %s", resp.StatusCode, url)
		// Extend disable period
		ht.mu.Lock()
		if h, exists := ht.health[url]; exists && h.Disabled {
			h.DisabledUntil = time.Now().Add(ht.config.DisableDuration)
		}
		ht.mu.Unlock()
	}
}

// Stop stops the health tracker's background probing
func (ht *HealthTracker) Stop() {
	close(ht.stopChan)
}

// ResetEndpoint resets the health status of an endpoint (for manual intervention)
func (ht *HealthTracker) ResetEndpoint(url string) {
	ht.mu.Lock()
	defer ht.mu.Unlock()

	if h, exists := ht.health[url]; exists {
		h.Score = ht.config.InitialScore
		h.Disabled = false
		h.DisabledUntil = time.Time{}
		h.FailCount = 0
		h.SuccessCount = 0
		log.Printf("[health-tracker] Endpoint %s manually reset", url)
	}
}
