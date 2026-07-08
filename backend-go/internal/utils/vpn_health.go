package utils

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// VPNStatus is exposed via /api/v2/health for monitoring.
type VPNStatus struct {
	Configured bool   `json:"configured"`
	Healthy    bool   `json:"healthy"`
	ProxyURL   string `json:"proxy_url,omitempty"`
	Mode       string `json:"mode"` // disabled | vpn | direct_fallback
	LastCheck  string `json:"last_check,omitempty"`
	LastError  string `json:"last_error,omitempty"`
}

type vpnHealth struct {
	proxyURL  string
	healthy   atomic.Bool
	lastCheck atomic.Value // time.Time
	lastError atomic.Value // string
	lastWarn  atomic.Value // time.Time
}

var globalVPNHealth vpnHealth

// StartVPNHealthProbes begins background checks when VPN_HTTP_PROXY is set.
func StartVPNHealthProbes() {
	proxyURL := vpnProxyURL()
	if proxyURL == "" {
		log.Println("[vpn] VPN_HTTP_PROXY unset — user proxy traffic uses direct egress")
		return
	}

	globalVPNHealth.proxyURL = proxyURL
	log.Printf("[vpn] VPN proxy configured (%s) — probing every 60s; direct fallback if unreachable", proxyURL)

	probe := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := dialVPNProxy(ctx, proxyURL)
		now := time.Now()
		globalVPNHealth.lastCheck.Store(now)

		if err != nil {
			wasHealthy := globalVPNHealth.healthy.Swap(false)
			globalVPNHealth.lastError.Store(err.Error())
			if wasHealthy || shouldWarnVPN(now) {
				log.Printf("[vpn] ALERT: proxy unreachable (%v) — using DIRECT egress until VPN recovers", err)
				globalVPNHealth.lastWarn.Store(now)
			}
			return
		}

		if !globalVPNHealth.healthy.Swap(true) {
			log.Printf("[vpn] proxy recovered — routing user traffic through VPN again")
		}
		globalVPNHealth.lastError.Store("")
	}

	probe()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			probe()
		}
	}()
}

func shouldWarnVPN(now time.Time) bool {
	if v := globalVPNHealth.lastWarn.Load(); v != nil {
		if last, ok := v.(time.Time); ok && now.Sub(last) < 5*time.Minute {
			return false
		}
	}
	return true
}

func dialVPNProxy(ctx context.Context, proxyURL string) error {
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}
	host := parsed.Host
	if !strings.Contains(host, ":") {
		host = net.JoinHostPort(host, "8888")
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return err
	}
	return conn.Close()
}

// GetVPNStatus returns the current VPN routing state for health checks.
func GetVPNStatus() VPNStatus {
	proxyURL := vpnProxyURL()

	st := VPNStatus{
		Configured: proxyURL != "",
		ProxyURL:   proxyURL,
	}

	if !st.Configured {
		st.Mode = "disabled"
		st.Healthy = true
		return st
	}

	st.Healthy = globalVPNHealth.healthy.Load()
	if st.Healthy {
		st.Mode = "vpn"
	} else {
		st.Mode = "direct_fallback"
	}

	if v := globalVPNHealth.lastCheck.Load(); v != nil {
		if t, ok := v.(time.Time); ok && !t.IsZero() {
			st.LastCheck = t.UTC().Format(time.RFC3339)
		}
	}
	if v := globalVPNHealth.lastError.Load(); v != nil {
		if s, ok := v.(string); ok {
			st.LastError = s
		}
	}
	return st
}

func vpnProxyOrDirect(req *http.Request) (*url.URL, error) {
	proxyURL := vpnProxyURL()
	if proxyURL == "" {
		return nil, nil
	}

	if globalVPNHealth.proxyURL == "" {
		globalVPNHealth.proxyURL = proxyURL
	}

	if !globalVPNHealth.healthy.Load() {
		return nil, nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

// VPNHomeIP returns the configured home IP to detect VPN leaks (optional env).
func VPNHomeIP() string {
	return strings.TrimSpace(os.Getenv("VPN_HOME_IP"))
}