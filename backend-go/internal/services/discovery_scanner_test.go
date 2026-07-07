package services

import (
	"net"
	"testing"
	"time"
)

func TestExpandCIDR_Valid(t *testing.T) {
	ips, err := expandCIDR("192.168.1.0/30")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// /30 has 4 IPs, minus network and broadcast = 2 usable
	if len(ips) != 2 {
		t.Errorf("Expected 2 IPs, got %d", len(ips))
	}

	expected := []string{"192.168.1.1", "192.168.1.2"}
	for i, expectedIP := range expected {
		if i < len(ips) && ips[i] != expectedIP {
			t.Errorf("Expected ip[%d]=%s, got %s", i, expectedIP, ips[i])
		}
	}
}

func TestExpandCIDR_SingleIP(t *testing.T) {
	ips, err := expandCIDR("192.168.1.1")
	if err != nil {
		t.Fatalf("Expected no error for single IP, got %v", err)
	}

	if len(ips) != 1 {
		t.Errorf("Expected 1 IP, got %d", len(ips))
	}
	if ips[0] != "192.168.1.1" {
		t.Errorf("Expected 192.168.1.1, got %s", ips[0])
	}
}

func TestExpandCIDR_Invalid(t *testing.T) {
	_, err := expandCIDR("invalid-cidr")
	if err == nil {
		t.Error("Expected error for invalid CIDR")
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
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.input)
		incrementIP(ip)
		if ip.String() != tt.expected {
			t.Errorf("incrementIP(%s) = %s, want %s", tt.input, ip.String(), tt.expected)
		}
	}
}

func TestParsePortRanges(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
		hasErr   bool
	}{
		{"80,443,8080", []int{80, 443, 8080}, false},
		{"8000-8002", []int{8000, 8001, 8002}, false},
		{"80,443,8000-8002", []int{80, 443, 8000, 8001, 8002}, false},
		{"invalid", nil, true},
		{"80-abc", nil, true},
		{"abc-80", nil, true},
		{"80-443-8080", nil, true},
		{"", []int{}, false},
		{"80,,443", []int{80, 443}, false},
	}

	for _, tt := range tests {
		result, err := ParsePortRanges(tt.input)
		if tt.hasErr && err == nil {
			t.Errorf("ParsePortRanges(%q) expected error, got nil", tt.input)
		}
		if !tt.hasErr && err != nil {
			t.Errorf("ParsePortRanges(%q) unexpected error: %v", tt.input, err)
		}

		if len(result) != len(tt.expected) {
			t.Errorf("ParsePortRanges(%q) expected %d ports, got %d", tt.input, len(tt.expected), len(result))
			continue
		}

		for i, expectedPort := range tt.expected {
			if i < len(result) && result[i] != expectedPort {
				t.Errorf("ParsePortRanges(%q) port[%d]=%d, want %d", tt.input, i, result[i], expectedPort)
			}
		}
	}
}

func TestDiscoveryScanner_NewDiscoveryScanner(t *testing.T) {
	scanner := NewDiscoveryScanner(nil)

	if scanner == nil {
		t.Fatal("Expected non-nil scanner")
	}
	if scanner.db != nil {
		t.Error("Expected db to be nil")
	}
	if scanner.interval != 24*time.Hour {
		t.Errorf("Expected interval=24h, got %v", scanner.interval)
	}
	if scanner.maxWorkers != 100 {
		t.Errorf("Expected maxWorkers=100, got %d", scanner.maxWorkers)
	}
	if scanner.scanTimeout != 2*time.Second {
		t.Errorf("Expected scanTimeout=2s, got %v", scanner.scanTimeout)
	}
	if scanner.httpTimeout != 5*time.Second {
		t.Errorf("Expected httpTimeout=5s, got %v", scanner.httpTimeout)
	}
}

func TestDiscoveryScanner_Stop(t *testing.T) {
	scanner := NewDiscoveryScanner(nil)

	// Should not panic
	scanner.Stop()

	// Verify stop channel is closed
	select {
	case <-scanner.stop:
		// Expected
	default:
		t.Error("Expected stop channel to be closed")
	}
}

func TestDiscoveryScanner_ManualScan(t *testing.T) {
	scanner := NewDiscoveryScanner(nil)

	err := scanner.ManualScan("192.168.1.0/24")
	if err != nil {
		t.Errorf("ManualScan returned error: %v", err)
	}
}
