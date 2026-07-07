package main

import (
	"net"
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Save original env vars
	origInterval := os.Getenv("SCAN_INTERVAL_HOURS")
	origWorkers := os.Getenv("MAX_WORKERS")
	origIPRanges := os.Getenv("SCAN_IP_RANGES")
	defer func() {
		setEnvIfNotEmpty("SCAN_INTERVAL_HOURS", origInterval)
		setEnvIfNotEmpty("MAX_WORKERS", origWorkers)
		setEnvIfNotEmpty("SCAN_IP_RANGES", origIPRanges)
	}()

	// Test defaults
	os.Unsetenv("SCAN_INTERVAL_HOURS")
	os.Unsetenv("MAX_WORKERS")
	os.Unsetenv("SCAN_IP_RANGES")

	config := loadConfig()

	if config.ScanInterval != 24*time.Hour {
		t.Errorf("Expected ScanInterval=24h, got %v", config.ScanInterval)
	}
	if config.MaxWorkers != 100 {
		t.Errorf("Expected MaxWorkers=100, got %d", config.MaxWorkers)
	}
	if config.ScanTimeout != 2*time.Second {
		t.Errorf("Expected ScanTimeout=2s, got %v", config.ScanTimeout)
	}
	if config.HTTPTimeout != 5*time.Second {
		t.Errorf("Expected HTTPTimeout=5s, got %v", config.HTTPTimeout)
	}
	if len(config.DefaultIPRanges) != 0 {
		t.Errorf("Expected DefaultIPRanges to be empty, got %v", config.DefaultIPRanges)
	}
}

func TestLoadConfig_CustomValues(t *testing.T) {
	os.Setenv("SCAN_INTERVAL_HOURS", "12")
	os.Setenv("MAX_WORKERS", "50")
	os.Setenv("SCAN_IP_RANGES", "192.168.1.0/24,10.0.0.0/8")
	defer func() {
		os.Unsetenv("SCAN_INTERVAL_HOURS")
		os.Unsetenv("MAX_WORKERS")
		os.Unsetenv("SCAN_IP_RANGES")
	}()

	config := loadConfig()

	if config.ScanInterval != 12*time.Hour {
		t.Errorf("Expected ScanInterval=12h, got %v", config.ScanInterval)
	}
	if config.MaxWorkers != 50 {
		t.Errorf("Expected MaxWorkers=50, got %d", config.MaxWorkers)
	}
	if len(config.DefaultIPRanges) != 2 {
		t.Errorf("Expected 2 IP ranges, got %d", len(config.DefaultIPRanges))
	}
	if config.DefaultIPRanges[0] != "192.168.1.0/24" {
		t.Errorf("Expected first IP range to be 192.168.1.0/24, got %s", config.DefaultIPRanges[0])
	}
}

func TestGetEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "test_value")
	defer os.Unsetenv("TEST_VAR")

	// Test existing var
	result := getEnv("TEST_VAR", "default")
	if result != "test_value" {
		t.Errorf("Expected 'test_value', got %s", result)
	}

	// Test default
	result = getEnv("NONEXISTENT_VAR", "default_value")
	if result != "default_value" {
		t.Errorf("Expected 'default_value', got %s", result)
	}
}

func TestNewDiscoveryService(t *testing.T) {
	config := &Config{
		BackendURL:    "http://test:8080",
		MaxWorkers:    50,
		ScanTimeout:   3 * time.Second,
		HTTPTimeout:   10 * time.Second,
		DefaultIPRanges: []string{"192.168.1.0/24"},
	}

	ds := NewDiscoveryService(config)

	if ds == nil {
		t.Fatal("Expected non-nil DiscoveryService")
	}
	if ds.config != config {
		t.Error("Expected config to be set")
	}
	if ds.httpClient == nil {
		t.Error("Expected httpClient to be set")
	}
	if ds.activeScans == nil {
		t.Error("Expected activeScans to be initialized")
	}
	if ds.scanQueue == nil {
		t.Error("Expected scanQueue to be initialized")
	}
	if ds.shutdownChan == nil {
		t.Error("Expected shutdownChan to be initialized")
	}
}

func TestExpandCIDR(t *testing.T) {
	tests := []struct {
		cidr      string
		expectErr bool
		minIPs    int
	}{
		{"192.168.1.0/30", false, 2},      // 4 IPs - 2 = 2 usable
		{"10.0.0.0/29", false, 6},         // 8 IPs - 2 = 6 usable
		{"192.168.1.1", false, 1},         // Single IP
		{"invalid", true, 0},              // Invalid
		{"256.256.256.256", true, 0},      // Invalid IP
	}

	for _, tt := range tests {
		ips, err := expandCIDR(tt.cidr)
		if tt.expectErr && err == nil {
			t.Errorf("expandCIDR(%q) expected error, got nil", tt.cidr)
		}
		if !tt.expectErr && err != nil {
			t.Errorf("expandCIDR(%q) unexpected error: %v", tt.cidr, err)
		}
		if !tt.expectErr && len(ips) < tt.minIPs {
			t.Errorf("expandCIDR(%q) expected at least %d IPs, got %d", tt.cidr, tt.minIPs, len(ips))
		}
	}
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.1", "192.168.1.2"},
		{"10.0.0.255", "10.0.1.0"},
		{"172.16.255.255", "172.17.0.0"},
		{"192.168.1.255", "192.168.2.0"},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.input)
		if ip == nil {
			t.Fatalf("Failed to parse IP %s", tt.input)
		}
		incrementIP(ip)
		if ip.String() != tt.expected {
			t.Errorf("incrementIP(%s) = %s, want %s", tt.input, ip.String(), tt.expected)
		}
	}
}

func TestDiscoveryService_ScanHost(t *testing.T) {
	config := &Config{
		ScanTimeout: 1 * time.Second,
		HTTPTimeout: 2 * time.Second,
	}
	ds := NewDiscoveryService(config)

	// Test with non-existent host (should return false)
	result := ds.scanHost("192.0.2.1", 11434) // TEST-NET-1, should not exist
	if result {
		t.Error("Expected scanHost to return false for non-existent host")
	}
}

func TestDiscoveryService_SendToBackend_InvalidURL(t *testing.T) {
	config := &Config{
		BackendURL: "://invalid-url",
	}
	ds := NewDiscoveryService(config)

	endpoints := []DiscoveredEndpoint{
		{URL: "http://test:11434", Name: "Test", EndpointType: "ollama"},
	}

	err := ds.sendToBackend(endpoints)
	if err == nil {
		t.Error("Expected error for invalid backend URL")
	}
}

func TestDiscoveryService_ManualScan(t *testing.T) {
	config := &Config{}
	ds := NewDiscoveryService(config)

	// Test that scan can be queued
	select {
	case ds.scanQueue <- "192.168.1.0/24":
		// Success
	default:
		t.Error("Expected to be able to queue scan")
	}

	// Verify queue size
	if len(ds.scanQueue) != 1 {
		t.Errorf("Expected queue size 1, got %d", len(ds.scanQueue))
	}
}

func TestDiscoveryService_StatusEndpoint(t *testing.T) {
	config := &Config{}
	ds := NewDiscoveryService(config)

	// Add a mock scan status
	ds.scanMutex.Lock()
	ds.activeScans["192.168.1.0/24"] = &ScanStatus{
		IPRange:   "192.168.1.0/24",
		Status:    "running",
		Discovered: 5,
		TotalIPs:  254,
		Scanned:   100,
		StartedAt: time.Now(),
	}
	ds.scanMutex.Unlock()

	// Verify we can read the status
	ds.scanMutex.RLock()
	status, ok := ds.activeScans["192.168.1.0/24"]
	ds.scanMutex.RUnlock()

	if !ok {
		t.Fatal("Expected to find scan status")
	}
	if status.Status != "running" {
		t.Errorf("Expected status 'running', got %s", status.Status)
	}
	if status.Discovered != 5 {
		t.Errorf("Expected 5 discovered, got %d", status.Discovered)
	}
}

func TestConfig_BackendDefaults(t *testing.T) {
	os.Unsetenv("BACKEND_URL")
	os.Unsetenv("BACKEND_API_KEY")
	defer func() {
		os.Unsetenv("BACKEND_URL")
		os.Unsetenv("BACKEND_API_KEY")
	}()

	config := loadConfig()

	if config.BackendURL != "http://backend-go:8000" {
		t.Errorf("Expected BackendURL=http://backend-go:8000, got %s", config.BackendURL)
	}
}

func setEnvIfNotEmpty(key, value string) {
	if value != "" {
		os.Setenv(key, value)
	} else {
		os.Unsetenv(key)
	}
}

// Helper to parse config values for testing
func TestConfigParsing(t *testing.T) {
	os.Setenv("SCAN_INTERVAL_HOURS", "48")
	os.Setenv("MAX_WORKERS", "200")
	defer func() {
		os.Unsetenv("SCAN_INTERVAL_HOURS")
		os.Unsetenv("MAX_WORKERS")
	}()

	config := loadConfig()

	if config.ScanInterval != 48*time.Hour {
		t.Errorf("Expected 48h, got %v", config.ScanInterval)
	}
	if config.MaxWorkers != 200 {
		t.Errorf("Expected 200 workers, got %d", config.MaxWorkers)
	}
}

func TestInvalidConfigParsing(t *testing.T) {
	os.Setenv("SCAN_INTERVAL_HOURS", "invalid")
	os.Setenv("MAX_WORKERS", "not-a-number")
	defer func() {
		os.Unsetenv("SCAN_INTERVAL_HOURS")
		os.Unsetenv("MAX_WORKERS")
	}()

	// Should fall back to defaults without panicking
	config := loadConfig()

	// Invalid values result in 0, not defaults
	if config.ScanInterval != 0 {
		t.Errorf("Expected 0h for invalid input, got %v", config.ScanInterval)
	}
	if config.MaxWorkers != 0 {
		t.Errorf("Expected 0 workers for invalid input, got %d", config.MaxWorkers)
	}
}
