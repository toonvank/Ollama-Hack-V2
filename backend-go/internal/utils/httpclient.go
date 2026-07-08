package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err == nil {
			privateIPBlocks = append(privateIPBlocks, block)
		}
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}
	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

type dnsCacheEntry struct {
	ips []net.IP
	exp time.Time
}

var (
	dnsCacheMu  sync.RWMutex
	dnsCache    = make(map[string]dnsCacheEntry)
	dnsCacheTTL = 60 * time.Second
)

func lookupIPCached(ctx context.Context, host string) ([]net.IP, error) {
	now := time.Now()

	dnsCacheMu.RLock()
	if entry, ok := dnsCache[host]; ok && now.Before(entry.exp) {
		ips := entry.ips
		dnsCacheMu.RUnlock()
		return ips, nil
	}
	dnsCacheMu.RUnlock()

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}

	dnsCacheMu.Lock()
	dnsCache[host] = dnsCacheEntry{ips: ips, exp: now.Add(dnsCacheTTL)}
	dnsCacheMu.Unlock()
	return ips, nil
}

// JoinEndpointURL joins a base endpoint URL with an API path without double slashes.
func JoinEndpointURL(base, path string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if path == "" {
		return base
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if base == "" {
		return path
	}
	return base + path
}

func noProxy(*http.Request) (*url.URL, error) {
	return nil, nil
}

func vpnProxyURL() string {
	if val := strings.TrimSpace(os.Getenv("VPN_HTTP_PROXY")); val != "" {
		return val
	}
	return strings.TrimSpace(os.Getenv("HTTP_PROXY"))
}

func vpnProxy(req *http.Request) (*url.URL, error) {
	proxyURL := vpnProxyURL()
	if proxyURL == "" {
		return nil, nil
	}
	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}
	return parsed, nil
}

func newHTTPTransport(dialTimeout time.Duration, proxy func(*http.Request) (*url.URL, error)) *http.Transport {
	allowLocal := os.Getenv("ALLOW_LOCAL_ENDPOINTS") == "true"

	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy: proxy,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			ips, err := lookupIPCached(ctx, host)
			if err != nil {
				return nil, err
			}

			if !allowLocal {
				for _, ip := range ips {
					if isPrivateIP(ip) {
						return nil, errors.New("SSRF Protection: connection to private/internal IP blocked. Enable ALLOW_LOCAL_ENDPOINTS=true if this is intentional.")
					}
				}
			}

			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func newHTTPClient(requestTimeout, dialTimeout time.Duration, proxy func(*http.Request) (*url.URL, error)) *http.Client {
	return &http.Client{
		Transport: newHTTPTransport(dialTimeout, proxy),
		Timeout:   requestTimeout,
	}
}

// NewHTTPClient returns a direct (non-VPN) client for background probing and discovery.
// It does not use HTTP_PROXY/VPN_HTTP_PROXY so endpoint tests keep the host egress IP.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout, 30*time.Second, noProxy)
}

// NewVPNHTTPClient routes outbound traffic through VPN_HTTP_PROXY (Gluetun HTTP proxy).
func NewVPNHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout, 30*time.Second, vpnProxy)
}

var (
	sharedProxyClient     *http.Client
	sharedProxyClientOnce sync.Once
	sharedRaceClient      *http.Client
	sharedRaceClientOnce  sync.Once
)

// SharedProxyClient returns a process-wide VPN-masked client for user proxy requests.
func SharedProxyClient() *http.Client {
	sharedProxyClientOnce.Do(func() {
		sharedProxyClient = newHTTPClient(120*time.Second, 30*time.Second, vpnProxy)
	})
	return sharedProxyClient
}

// SharedRaceClient returns a VPN-masked client tuned for endpoint racing.
func SharedRaceClient() *http.Client {
	sharedRaceClientOnce.Do(func() {
		dialTimeout := 5 * time.Second
		if val := os.Getenv("RACE_DIAL_TIMEOUT"); val != "" {
			if d, err := time.ParseDuration(val); err == nil && d > 0 {
				dialTimeout = d
			}
		}
		sharedRaceClient = newHTTPClient(120*time.Second, dialTimeout, vpnProxy)
	})
	return sharedRaceClient
}