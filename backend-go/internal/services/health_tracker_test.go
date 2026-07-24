package services

import (
	"testing"
	"time"
)

func TestHealthTrackerConfig_Loading(t *testing.T) {
	config := loadHealthConfig()

	// Verify defaults
	if !config.Enabled {
		t.Error("Expected health tracking to be enabled by default")
	}
	if config.DisableThreshold != 30 {
		t.Errorf("Expected DisableThreshold=30, got %d", config.DisableThreshold)
	}
	if config.DisableDuration != 5*time.Minute {
		t.Errorf("Expected DisableDuration=5m, got %v", config.DisableDuration)
	}
	if config.ProbeInterval != 1*time.Minute {
		t.Errorf("Expected ProbeInterval=1m, got %v", config.ProbeInterval)
	}
	if config.FailPenalty != 10 {
		t.Errorf("Expected FailPenalty=10, got %d", config.FailPenalty)
	}
	if config.SuccessReward != 2 {
		t.Errorf("Expected SuccessReward=2, got %d", config.SuccessReward)
	}
	if config.MaxScore != 100 {
		t.Errorf("Expected MaxScore=100, got %d", config.MaxScore)
	}
	if config.InitialScore != 100 {
		t.Errorf("Expected InitialScore=100, got %d", config.InitialScore)
	}
}

func TestHealthTracker_RecordSuccess(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     50, // Start below max to see increase
	}, nil)
	defer tracker.Stop()

	url := "http://test-endpoint:8080"

	// Record success
	tracker.RecordSuccess(url)

	health := tracker.GetHealth(url)
	if health == nil {
		t.Fatal("Expected health entry to be created")
	}
	if health.SuccessCount != 1 {
		t.Errorf("Expected SuccessCount=1, got %d", health.SuccessCount)
	}
	if health.Score != 52 { // 50 + 2 reward
		t.Errorf("Expected Score=52 after success, got %d", health.Score)
	}
}

func TestHealthTracker_RecordFailure(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url := "http://failing-endpoint:8080"

	// Record enough failures to disable
	for i := 0; i < 8; i++ {
		tracker.RecordFailure(url)
	}

	health := tracker.GetHealth(url)
	if health == nil {
		t.Fatal("Expected health entry to be created")
	}
	if !health.Disabled {
		t.Error("Expected endpoint to be disabled after multiple failures")
	}
	if health.FailCount != 8 {
		t.Errorf("Expected FailCount=8, got %d", health.FailCount)
	}
}

func TestHealthTracker_RecordRateLimit(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url := "http://rate-limited:8080"
	initialScore := 100

	// Record rate limit
	tracker.RecordRateLimit(url)

	health := tracker.GetHealth(url)
	if health == nil {
		t.Fatal("Expected health entry to be created")
	}
	if health.RateLimitCount != 1 {
		t.Errorf("Expected RateLimitCount=1, got %d", health.RateLimitCount)
	}
	if health.Score != initialScore {
		t.Errorf("Expected Score unchanged (%d) after rate limit, got %d", initialScore, health.Score)
	}
}

func TestHealthTracker_RecordQuotaExceeded(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url := "http://quota-exceeded:8080"

	// Record quota exceeded
	tracker.RecordQuotaExceeded(url)

	health := tracker.GetHealth(url)
	if health == nil {
		t.Fatal("Expected health entry to be created")
	}
	if health.Score != 0 {
		t.Errorf("Expected Score=0, got %d", health.Score)
	}
	if !health.Disabled {
		t.Error("Expected endpoint to be disabled immediately")
	}
	if !tracker.IsDisabled(url) {
		t.Error("Expected IsDisabled to return true")
	}
}

func TestHealthTracker_IsDisabled(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  500 * time.Millisecond,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url := "http://disabled-endpoint:8080"

	// Disable the endpoint
	for i := 0; i < 8; i++ {
		tracker.RecordFailure(url)
	}

	if !tracker.IsDisabled(url) {
		t.Error("Expected endpoint to be disabled")
	}

	// Wait for disable period to expire
	time.Sleep(600 * time.Millisecond)

	if tracker.IsDisabled(url) {
		t.Error("Expected endpoint to be re-enabled after disable period")
	}

	// Cooldown expiry must clear the sticky Disabled flag (not just IsDisabled=false),
	// otherwise SQL routing that reads eh.disabled permanently blackholes the node.
	health := tracker.GetHealth(url)
	if health == nil {
		t.Fatal("Expected health entry after cooldown expiry")
	}
	if health.Disabled {
		t.Error("Expected Disabled=false after cooldown expiry, not sticky true")
	}
}

func TestHealthTracker_FilterHealthyEndpoints_ExpiresStickyDisable(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  200 * time.Millisecond,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url := "http://sticky-disable:8080"
	for i := 0; i < 8; i++ {
		tracker.RecordFailure(url)
	}
	if len(tracker.FilterHealthyEndpoints([]string{url})) != 0 {
		t.Fatal("Expected endpoint filtered while cooldown active")
	}

	time.Sleep(350 * time.Millisecond)

	filtered := tracker.FilterHealthyEndpoints([]string{url})
	if len(filtered) != 1 || filtered[0] != url {
		t.Fatalf("Expected endpoint restored after cooldown, got %v", filtered)
	}
	if h := tracker.GetHealth(url); h == nil || h.Disabled {
		t.Fatalf("Expected Disabled=false after filter expiry, got %#v", h)
	}
}

func TestHealthTracker_FilterHealthyEndpoints(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	healthyURL := "http://healthy:8080"
	unhealthyURL := "http://unhealthy:8080"

	// Make one endpoint unhealthy
	for i := 0; i < 8; i++ {
		tracker.RecordFailure(unhealthyURL)
	}

	endpoints := []string{healthyURL, unhealthyURL}
	filtered := tracker.FilterHealthyEndpoints(endpoints)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 healthy endpoint, got %d", len(filtered))
	}
	if filtered[0] != healthyURL {
		t.Errorf("Expected only healthy endpoint, got %s", filtered[0])
	}
}

func TestHealthTracker_GetAllHealth(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url1 := "http://endpoint1:8080"
	url2 := "http://endpoint2:8080"

	tracker.RecordSuccess(url1)
	tracker.RecordSuccess(url2)

	allHealth := tracker.GetAllHealth()
	if len(allHealth) != 2 {
		t.Errorf("Expected 2 endpoints, got %d", len(allHealth))
	}
}

func TestHealthTracker_GetConfig(t *testing.T) {
	config := HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 50,
		DisableDuration:  10 * time.Minute,
		ProbeInterval:    2 * time.Minute,
		FailPenalty:      15,
		SuccessReward:    5,
		MaxScore:         200,
		InitialScore:     150,
	}

	tracker := NewHealthTracker(config, nil)
	defer tracker.Stop()

	gotConfig := tracker.GetConfig()
	if gotConfig.DisableThreshold != 50 {
		t.Errorf("Expected DisableThreshold=50, got %d", gotConfig.DisableThreshold)
	}
}

func TestHealthTracker_ResetEndpoint(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled:          true,
		DisableThreshold: 30,
		DisableDuration:  5 * time.Minute,
		ProbeInterval:    1 * time.Minute,
		FailPenalty:      10,
		SuccessReward:    2,
		MaxScore:         100,
		InitialScore:     100,
	}, nil)
	defer tracker.Stop()

	url := "http://reset-endpoint:8080"

	// Disable the endpoint
	for i := 0; i < 8; i++ {
		tracker.RecordFailure(url)
	}

	if !tracker.IsDisabled(url) {
		t.Fatal("Expected endpoint to be disabled")
	}

	// Reset the endpoint
	tracker.ResetEndpoint(url)

	health := tracker.GetHealth(url)
	if health == nil {
		t.Fatal("Expected health entry to exist")
	}
	if health.Disabled {
		t.Error("Expected endpoint to be re-enabled after reset")
	}
	if health.FailCount != 0 {
		t.Errorf("Expected FailCount=0 after reset, got %d", health.FailCount)
	}
	if health.Score != 100 {
		t.Errorf("Expected Score=100 after reset, got %d", health.Score)
	}
}

func TestHealthTracker_Disabled(t *testing.T) {
	tracker := NewHealthTracker(HealthTrackerConfig{
		Enabled: false,
	}, nil)

	url := "http://endpoint:8080"

	// These should be no-ops when disabled
	tracker.RecordSuccess(url)
	tracker.RecordFailure(url)
	tracker.RecordRateLimit(url)

	health := tracker.GetHealth(url)
	if health != nil {
		t.Error("Expected no health entry when tracker is disabled")
	}

	if tracker.IsDisabled(url) {
		t.Error("Expected IsDisabled to return false when tracker is disabled")
	}

	filtered := tracker.FilterHealthyEndpoints([]string{url})
	if len(filtered) != 1 {
		t.Error("Expected all endpoints to pass through when disabled")
	}
}
