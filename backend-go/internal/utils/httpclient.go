package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
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

func newHTTPTransport(dialTimeout time.Duration) *http.Transport {
	allowLocal := os.Getenv("ALLOW_LOCAL_ENDPOINTS") == "true"

	dialer := &net.Dialer{
		Timeout:   dialTimeout,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
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

func newHTTPClient(requestTimeout, dialTimeout time.Duration) *http.Client {
	return &http.Client{
		Transport: newHTTPTransport(dialTimeout),
		Timeout:   requestTimeout,
	}
}

// NewHTTPClient returns an http.Client that prevents SSRF by blocking connections to private IPs.
// It also automatically respects HTTP_PROXY and HTTPS_PROXY environment variables out of the box.
func NewHTTPClient(timeout time.Duration) *http.Client {
	return newHTTPClient(timeout, 30*time.Second)
}

var (
	sharedProxyClient     *http.Client
	sharedProxyClientOnce sync.Once
	sharedRaceClient      *http.Client
	sharedRaceClientOnce  sync.Once
)

// SharedProxyClient returns a process-wide client for proxy requests with connection reuse.
func SharedProxyClient() *http.Client {
	sharedProxyClientOnce.Do(func() {
		sharedProxyClient = newHTTPClient(120*time.Second, 30*time.Second)
	})
	return sharedProxyClient
}

// SharedRaceClient returns a client tuned for endpoint racing: short dial timeout, shared pool.
func SharedRaceClient() *http.Client {
	sharedRaceClientOnce.Do(func() {
		dialTimeout := 5 * time.Second
		if val := os.Getenv("RACE_DIAL_TIMEOUT"); val != "" {
			if d, err := time.ParseDuration(val); err == nil && d > 0 {
				dialTimeout = d
			}
		}
		sharedRaceClient = newHTTPClient(120*time.Second, dialTimeout)
	})
	return sharedRaceClient
}