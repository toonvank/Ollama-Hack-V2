package utils

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
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

// NewHTTPClient returns an http.Client that prevents SSRF by blocking connections to private IPs.
// It also automatically respects HTTP_PROXY and HTTPS_PROXY environment variables out of the box.
func NewHTTPClient(timeout time.Duration) *http.Client {
	allowLocal := os.Getenv("ALLOW_LOCAL_ENDPOINTS") == "true"

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, err
			}

			// Resolve IPs
			ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}

			// Enforce SSRF protection by verifying resolved IPs
			if !allowLocal {
				for _, ip := range ips {
					if isPrivateIP(ip) {
						return nil, errors.New("SSRF Protection: connection to private/internal IP blocked. Enable ALLOW_LOCAL_ENDPOINTS=true if this is intentional.")
					}
				}
			}

			// Dial the first resolved IP (Go's DefaultResolver usually sorts them well)
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}
}
