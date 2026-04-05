package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	ScanInterval    time.Duration
	BackendURL      string
	BackendAPIKey   string
	MaxWorkers      int
	ScanTimeout     time.Duration
	HTTPTimeout     time.Duration
	DefaultIPRanges []string
}

type DiscoveredEndpoint struct {
	URL          string `json:"url"`
	Name         string `json:"name"`
	EndpointType string `json:"endpoint_type"`
}

type BatchCreateRequest struct {
	Endpoints []DiscoveredEndpoint `json:"endpoints"`
}

type ScanRequest struct {
	IPRange string `json:"ip_range"`
}

type ScanResponse struct {
	Status      string `json:"status"`
	Discovered  int    `json:"discovered"`
	IPRange     string `json:"ip_range,omitempty"`
	Message     string `json:"message"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at,omitempty"`
}

type DiscoveryService struct {
	config       *Config
	httpClient   *http.Client
	activeScans  map[string]*ScanStatus
	scanMutex    sync.RWMutex
	scanQueue    chan string
	shutdownChan chan struct{}
}

type ScanStatus struct {
	IPRange     string
	Status      string
	Discovered  int
	TotalIPs    int
	Scanned     int
	StartedAt   time.Time
	CompletedAt *time.Time
}

func loadConfig() *Config {
	scanIntervalStr := getEnv("SCAN_INTERVAL_HOURS", "24")
	scanInterval, _ := strconv.Atoi(scanIntervalStr)

	maxWorkersStr := getEnv("MAX_WORKERS", "100")
	maxWorkers, _ := strconv.Atoi(maxWorkersStr)

	ipRangesStr := getEnv("SCAN_IP_RANGES", "")
	var ipRanges []string
	if ipRangesStr != "" {
		ipRanges = strings.Split(ipRangesStr, ",")
		for i := range ipRanges {
			ipRanges[i] = strings.TrimSpace(ipRanges[i])
		}
	}

	return &Config{
		ScanInterval:    time.Duration(scanInterval) * time.Hour,
		BackendURL:      getEnv("BACKEND_URL", "http://backend-go:8000"),
		BackendAPIKey:   getEnv("BACKEND_API_KEY", ""),
		MaxWorkers:      maxWorkers,
		ScanTimeout:     2 * time.Second,
		HTTPTimeout:     5 * time.Second,
		DefaultIPRanges: ipRanges,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewDiscoveryService(config *Config) *DiscoveryService {
	return &DiscoveryService{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		activeScans:  make(map[string]*ScanStatus),
		scanQueue:    make(chan string, 100),
		shutdownChan: make(chan struct{}),
	}
}

func (ds *DiscoveryService) Start() {
	log.Println("🚀 Discovery Service starting...")
	log.Printf("   Backend URL: %s", ds.config.BackendURL)
	log.Printf("   Max Workers: %d", ds.config.MaxWorkers)
	log.Printf("   Scan Interval: %v", ds.config.ScanInterval)
	log.Printf("   Default IP Ranges: %v", ds.config.DefaultIPRanges)

	// Start periodic scanner
	go ds.periodicScanner()

	// Start scan queue processor
	go ds.scanQueueProcessor()

	// Start HTTP API server
	go ds.startHTTPServer()

	log.Println("✅ Discovery Service started successfully")
}

func (ds *DiscoveryService) periodicScanner() {
	ticker := time.NewTicker(ds.config.ScanInterval)
	defer ticker.Stop()

	// Initial scan after 1 minute delay
	time.Sleep(1 * time.Minute)
	ds.triggerScheduledScan()

	for {
		select {
		case <-ticker.C:
			ds.triggerScheduledScan()
		case <-ds.shutdownChan:
			log.Println("Periodic scanner shutting down")
			return
		}
	}
}

func (ds *DiscoveryService) triggerScheduledScan() {
	if len(ds.config.DefaultIPRanges) == 0 {
		log.Println("⏭️  No default IP ranges configured, skipping scheduled scan")
		return
	}

	log.Println("⏰ Triggering scheduled scan")
	for _, ipRange := range ds.config.DefaultIPRanges {
		select {
		case ds.scanQueue <- ipRange:
			log.Printf("   Queued: %s", ipRange)
		default:
			log.Printf("   Queue full, skipping: %s", ipRange)
		}
	}
}

func (ds *DiscoveryService) scanQueueProcessor() {
	for {
		select {
		case ipRange := <-ds.scanQueue:
			log.Printf("🔍 Processing scan from queue: %s", ipRange)
			ds.performScan(ipRange)
		case <-ds.shutdownChan:
			log.Println("Scan queue processor shutting down")
			return
		}
	}
}

func (ds *DiscoveryService) performScan(cidr string) {
	startTime := time.Now()
	scanStatus := &ScanStatus{
		IPRange:   cidr,
		Status:    "running",
		StartedAt: startTime,
	}

	ds.scanMutex.Lock()
	ds.activeScans[cidr] = scanStatus
	ds.scanMutex.Unlock()

	log.Printf("🔎 Starting scan of %s", cidr)

	ips, err := expandCIDR(cidr)
	if err != nil {
		log.Printf("❌ Invalid CIDR %s: %v", cidr, err)
		scanStatus.Status = "failed"
		return
	}

	scanStatus.TotalIPs = len(ips)
	log.Printf("   Expanded to %d IP addresses", len(ips))

	// Channel for discovered endpoints
	discoveredChan := make(chan string, len(ips))

	// Worker pool
	ipChan := make(chan string, len(ips))
	for _, ip := range ips {
		ipChan <- ip
	}
	close(ipChan)

	var wg sync.WaitGroup
	scannedCount := 0
	var countMutex sync.Mutex

	for i := 0; i < ds.config.MaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ip := range ipChan {
				if ds.scanHost(ip, 11434) {
					discoveredChan <- ip
				}
				countMutex.Lock()
				scannedCount++
				scanStatus.Scanned = scannedCount
				countMutex.Unlock()
			}
		}()
	}

	go func() {
		wg.Wait()
		close(discoveredChan)
	}()

	// Collect discovered endpoints
	var discovered []DiscoveredEndpoint
	for ip := range discoveredChan {
		endpoint := DiscoveredEndpoint{
			URL:          fmt.Sprintf("http://%s:11434", ip),
			Name:         fmt.Sprintf("Discovered: %s", ip),
			EndpointType: "ollama",
		}
		discovered = append(discovered, endpoint)
		scanStatus.Discovered++
		log.Printf("   ✓ Found Ollama at %s", endpoint.URL)
	}

	// Send to backend
	if len(discovered) > 0 {
		err := ds.sendToBackend(discovered)
		if err != nil {
			log.Printf("❌ Failed to send endpoints to backend: %v", err)
		} else {
			log.Printf("✅ Sent %d endpoints to backend", len(discovered))
		}
	}

	completedAt := time.Now()
	scanStatus.CompletedAt = &completedAt
	scanStatus.Status = "completed"

	duration := completedAt.Sub(startTime)
	log.Printf("✅ Scan completed: %s - found %d endpoints in %v",
		cidr, scanStatus.Discovered, duration.Round(time.Second))
}

func (ds *DiscoveryService) scanHost(ip string, port int) bool {
	// Step 1: TCP port check
	target := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", target, ds.config.ScanTimeout)
	if err != nil {
		return false
	}
	conn.Close()

	// Step 2: HTTP verification
	ctx, cancel := context.WithTimeout(context.Background(), ds.config.HTTPTimeout)
	defer cancel()

	url := fmt.Sprintf("http://%s:%d", ip, port)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	client := &http.Client{Timeout: ds.config.HTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	return strings.Contains(string(body), "Ollama is running")
}

func (ds *DiscoveryService) sendToBackend(endpoints []DiscoveredEndpoint) error {
	batchReq := BatchCreateRequest{
		Endpoints: endpoints,
	}

	jsonData, err := json.Marshal(batchReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v2/endpoint/batch", ds.config.BackendURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if ds.config.BackendAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+ds.config.BackendAPIKey)
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("backend returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (ds *DiscoveryService) startHTTPServer() {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	})

	// Trigger manual scan
	mux.HandleFunc("/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		select {
		case ds.scanQueue <- req.IPRange:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(ScanResponse{
				Status:    "queued",
				IPRange:   req.IPRange,
				Message:   "Scan queued successfully",
				StartedAt: time.Now().Format(time.RFC3339),
			})
		default:
			http.Error(w, "Scan queue is full", http.StatusTooManyRequests)
		}
	})

	// Get scan status
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		ds.scanMutex.RLock()
		defer ds.scanMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"active_scans": ds.activeScans,
			"queue_size":   len(ds.scanQueue),
		})
	})

	port := getEnv("PORT", "8001")
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("🌐 HTTP API listening on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

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

	// Remove network and broadcast addresses
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func main() {
	config := loadConfig()
	service := NewDiscoveryService(config)

	service.Start()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Println("\n🛑 Shutdown signal received, stopping service...")

	close(service.shutdownChan)
	time.Sleep(2 * time.Second)

	log.Println("👋 Discovery Service stopped")
}
