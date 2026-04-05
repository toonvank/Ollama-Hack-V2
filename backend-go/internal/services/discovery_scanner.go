package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/utils"
)

// DiscoveryScanner actively scans IP ranges to discover Ollama endpoints
type DiscoveryScanner struct {
	db           *database.DB
	interval     time.Duration
	stop         chan struct{}
	maxWorkers   int
	scanTimeout  time.Duration
	httpTimeout  time.Duration
}

func NewDiscoveryScanner(db *database.DB) *DiscoveryScanner {
	return &DiscoveryScanner{
		db:          db,
		interval:    24 * time.Hour, // Run once per day by default
		stop:        make(chan struct{}),
		maxWorkers:  100,            // Concurrent scan workers
		scanTimeout: 2 * time.Second, // Timeout per port scan
		httpTimeout: 5 * time.Second, // Timeout per HTTP request
	}
}

func (s *DiscoveryScanner) Start() {
	log.Println("[discovery-scanner] starting Ollama endpoint discovery scanner")
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		// Initial run on boot (delayed by 1 minute to let system stabilize)
		time.Sleep(1 * time.Minute)
		go s.runDiscoveryScan()

		for {
			select {
			case <-ticker.C:
				s.runDiscoveryScan()
			case <-s.stop:
				log.Println("[discovery-scanner] stopped discovery scanner")
				return
			}
		}
	}()
}

func (s *DiscoveryScanner) Stop() {
	close(s.stop)
}

func (s *DiscoveryScanner) runDiscoveryScan() {
	// Get IP ranges from environment variable
	// Format: "192.168.1.0/24,10.0.0.0/16,172.16.0.0/12"
	ipRangesEnv := os.Getenv("SCAN_IP_RANGES")
	if ipRangesEnv == "" {
		log.Println("[discovery-scanner] SCAN_IP_RANGES not configured, skipping scan")
		return
	}

	ipRanges := strings.Split(ipRangesEnv, ",")
	log.Printf("[discovery-scanner] starting scan of %d IP range(s)", len(ipRanges))

	for _, ipRange := range ipRanges {
		ipRange = strings.TrimSpace(ipRange)
		if ipRange == "" {
			continue
		}

		log.Printf("[discovery-scanner] scanning range: %s", ipRange)
		s.scanIPRange(ipRange)
	}

	log.Println("[discovery-scanner] scan complete")
}

func (s *DiscoveryScanner) scanIPRange(cidr string) {
	ips, err := expandCIDR(cidr)
	if err != nil {
		log.Printf("[discovery-scanner] invalid CIDR %s: %v", cidr, err)
		return
	}

	log.Printf("[discovery-scanner] expanded %s to %d IPs", cidr, len(ips))

	// Channel for work distribution
	ipChan := make(chan string, len(ips))
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	// Results channel
	resultsChan := make(chan string, len(ips))

	// Worker pool
	var wg sync.WaitGroup
	for i := 0; i < s.maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				if s.scanHost(ip, 11434) {
					resultsChan <- ip
				}
			}
		}()
	}

	// Wait for all workers
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results and add to database
	discovered := 0
	for ip := range resultsChan {
		endpointURL := fmt.Sprintf("http://%s:11434", ip)
		if s.addDiscoveredEndpoint(endpointURL, ip) {
			discovered++
		}
	}

	log.Printf("[discovery-scanner] discovered %d new Ollama endpoints in %s", discovered, cidr)
}

func (s *DiscoveryScanner) scanHost(ip string, port int) bool {
	// Step 1: Check if port is open
	target := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", target, s.scanTimeout)
	if err != nil {
		return false
	}
	conn.Close()

	// Step 2: Verify it's actually Ollama by checking the response
	ctx, cancel := context.WithTimeout(context.Background(), s.httpTimeout)
	defer cancel()

	url := fmt.Sprintf("http://%s:%d", ip, port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: s.httpTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	// Check if response contains "Ollama is running"
	bodyStr := string(body)
	if strings.Contains(bodyStr, "Ollama is running") {
		log.Printf("[discovery-scanner] ✓ found Ollama at %s", url)
		return true
	}

	return false
}

func (s *DiscoveryScanner) addDiscoveredEndpoint(url, ip string) bool {
	// Check if endpoint already exists
	var exists bool
	err := s.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM endpoints WHERE url = $1)", url)
	if err != nil {
		log.Printf("[discovery-scanner] error checking existence for %s: %v", url, err)
		return false
	}

	if exists {
		return false
	}

	// Insert new endpoint
	name := fmt.Sprintf("Discovered: %s", ip)
	var newID int
	err = s.db.QueryRow(
		"INSERT INTO endpoints (url, name, status, endpoint_type) VALUES ($1, $2, 'pending', 'ollama') RETURNING id",
		url,
		name,
	).Scan(&newID)

	if err != nil {
		log.Printf("[discovery-scanner] failed to insert endpoint %s: %v", url, err)
		return false
	}

	// Schedule immediate test for new endpoint
	_, err = s.db.Exec(
		"INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES ($1, NOW(), 'pending')",
		newID,
	)
	if err != nil {
		log.Printf("[discovery-scanner] failed to schedule test for endpoint %d: %v", newID, err)
	}

	log.Printf("[discovery-scanner] added new endpoint: %s (ID: %d)", url, newID)
	return true
}

// ManualScan allows triggering a manual scan via API call
func (s *DiscoveryScanner) ManualScan(ipRange string) error {
	log.Printf("[discovery-scanner] manual scan triggered for: %s", ipRange)
	go s.scanIPRange(ipRange)
	return nil
}

// expandCIDR expands a CIDR notation to a list of IP addresses
func expandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		// Try parsing as single IP
		if net.ParseIP(cidr) != nil {
			return []string{cidr}, nil
		}
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incrementIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses for IPv4
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

// incrementIP increments an IP address
func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// ParsePortRanges parses port range strings like "80,443,8000-8100"
func ParsePortRanges(portStr string) ([]int, error) {
	var ports []int
	parts := strings.Split(portStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if strings.Contains(part, "-") {
			// Range: "8000-8100"
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid port range: %s", part)
			}

			start, err := strconv.Atoi(strings.TrimSpace(rangeParts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid start port: %s", rangeParts[0])
			}

			end, err := strconv.Atoi(strings.TrimSpace(rangeParts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid end port: %s", rangeParts[1])
			}

			for p := start; p <= end; p++ {
				ports = append(ports, p)
			}
		} else {
			// Single port
			port, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", part)
			}
			ports = append(ports, port)
		}
	}

	return ports, nil
}
