package utils

import (
	"net/http"
	"os"
	"testing"
	"time"
)

func TestGetVPNStatus_Disabled(t *testing.T) {
	os.Unsetenv("VPN_HTTP_PROXY")
	os.Unsetenv("HTTP_PROXY")

	st := GetVPNStatus()
	if st.Configured {
		t.Fatal("expected VPN not configured")
	}
	if st.Mode != "disabled" {
		t.Fatalf("expected mode disabled, got %s", st.Mode)
	}
}

func TestVpnProxyOrDirect_FailOpenWhenUnhealthy(t *testing.T) {
	os.Setenv("VPN_HTTP_PROXY", "http://127.0.0.1:1")
	defer os.Unsetenv("VPN_HTTP_PROXY")

	globalVPNHealth.proxyURL = "http://127.0.0.1:1"
	globalVPNHealth.healthy.Store(false)

	parsed, err := vpnProxyOrDirect(&http.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed != nil {
		t.Fatal("expected direct fallback when VPN unhealthy")
	}
}

func TestVpnProxyOrDirect_UsesProxyWhenHealthy(t *testing.T) {
	os.Setenv("VPN_HTTP_PROXY", "http://gluetun:8888")
	defer os.Unsetenv("VPN_HTTP_PROXY")

	globalVPNHealth.proxyURL = "http://gluetun:8888"
	globalVPNHealth.healthy.Store(true)

	parsed, err := vpnProxyOrDirect(&http.Request{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed == nil || parsed.Host != "gluetun:8888" {
		t.Fatalf("expected gluetun proxy, got %v", parsed)
	}
}

func TestShouldWarnVPN_RateLimit(t *testing.T) {
	now := time.Now()
	globalVPNHealth.lastWarn.Store(now)
	if shouldWarnVPN(now.Add(time.Minute)) {
		t.Fatal("expected warn rate limit within 5 minutes")
	}
	if !shouldWarnVPN(now.Add(6 * time.Minute)) {
		t.Fatal("expected warn allowed after 5 minutes")
	}
}