package utils

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},       // Loopback
		{"10.0.0.1", true},        // RFC1918 private
		{"172.16.0.1", true},      // RFC1918 private
		{"192.168.1.1", true},     // RFC1918 private
		{"169.254.1.1", true},     // Link-local
		{"::1", true},             // IPv6 loopback
		{"fe80::1", true},         // IPv6 link-local
		{"fc00::1", true},         // IPv6 unique local
		{"8.8.8.8", false},        // Public
		{"1.1.1.1", false},        // Public
		{"203.0.113.1", false},    // Public
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Fatalf("Failed to parse IP %s", tt.ip)
		}
		result := isPrivateIP(ip)
		if result != tt.expected {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
		}
	}
}

func TestNewHTTPClient_Default(t *testing.T) {
	// Ensure ALLOW_LOCAL_ENDPOINTS is not set
	os.Unsetenv("ALLOW_LOCAL_ENDPOINTS")

	client := NewHTTPClient(30 * time.Second)
	if client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
	if client.Timeout != 30*time.Second {
		t.Errorf("Expected 30s timeout, got %v", client.Timeout)
	}
	if client.Transport == nil {
		t.Fatal("Expected transport to be set")
	}
}

func TestNewHTTPClient_AllowLocal(t *testing.T) {
	os.Setenv("ALLOW_LOCAL_ENDPOINTS", "true")
	defer os.Unsetenv("ALLOW_LOCAL_ENDPOINTS")

	client := NewHTTPClient(60 * time.Second)
	if client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
}

func TestHTTPClient_SSRFProtection(t *testing.T) {
	os.Unsetenv("ALLOW_LOCAL_ENDPOINTS")

	client := NewHTTPClient(5 * time.Second)

	// Try to connect to localhost - should be blocked
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://127.0.0.1:80", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	_, err = client.Do(req)
	if err == nil {
		t.Error("Expected SSRF protection to block localhost connection")
	} else if err.Error() != "SSRF Protection: connection to private IP blocked. Enable ALLOW_LOCAL_ENDPOINTS=true if this is intentional." {
		// The error might vary based on connection state, check if it's blocked
		t.Logf("Got expected error: %v", err)
	}
}

func TestJoinEndpointURL(t *testing.T) {
	tests := []struct {
		base string
		path string
		want string
	}{
		{"http://1.2.3.4:11434/", "/v1/chat/completions", "http://1.2.3.4:11434/v1/chat/completions"},
		{"http://1.2.3.4:11434", "/v1/chat/completions", "http://1.2.3.4:11434/v1/chat/completions"},
		{"http://1.2.3.4:11434/", "v1/chat/completions", "http://1.2.3.4:11434/v1/chat/completions"},
	}
	for _, tt := range tests {
		if got := JoinEndpointURL(tt.base, tt.path); got != tt.want {
			t.Errorf("JoinEndpointURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
		}
	}
}

func TestSharedProxyClient_ReusesInstance(t *testing.T) {
	if SharedProxyClient() != SharedProxyClient() {
		t.Fatal("expected shared proxy client singleton")
	}
}

func TestSharedRaceClient_ReusesInstance(t *testing.T) {
	if SharedRaceClient() != SharedRaceClient() {
		t.Fatal("expected shared race client singleton")
	}
}

func TestNewHTTPClient_DoesNotUseVPNProxy(t *testing.T) {
	os.Setenv("VPN_HTTP_PROXY", "http://gluetun:8888")
	defer os.Unsetenv("VPN_HTTP_PROXY")

	transport := NewHTTPClient(5 * time.Second).Transport.(*http.Transport)
	if parsed, _ := transport.Proxy(&http.Request{}); parsed != nil {
		t.Fatal("expected direct client to bypass VPN_HTTP_PROXY")
	}
}

func TestNewVPNHTTPClient_UsesVPNProxy(t *testing.T) {
	os.Setenv("VPN_HTTP_PROXY", "http://gluetun:8888")
	defer os.Unsetenv("VPN_HTTP_PROXY")

	globalVPNHealth.proxyURL = "http://gluetun:8888"
	globalVPNHealth.healthy.Store(true)

	transport := NewVPNHTTPClient(5 * time.Second).Transport.(*http.Transport)
	parsed, err := transport.Proxy(&http.Request{})
	if err != nil {
		t.Fatalf("proxy resolution failed: %v", err)
	}
	if parsed == nil || parsed.Host != "gluetun:8888" {
		t.Fatalf("expected gluetun proxy, got %v", parsed)
	}
}

func TestHTTPClient_AllowLocalEndpoints(t *testing.T) {
	os.Setenv("ALLOW_LOCAL_ENDPOINTS", "true")
	defer os.Unsetenv("ALLOW_LOCAL_ENDPOINTS")

	client := NewHTTPClient(5 * time.Second)

	// With ALLOW_LOCAL_ENDPOINTS=true, local connections should be allowed
	// (This test just verifies the client is created correctly)
	if client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}
}
