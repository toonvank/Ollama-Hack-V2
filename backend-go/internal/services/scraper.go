package services

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/timlzh/ollama-hack/internal/database"
)

// BackgroundScraperService periodically runs active scraping (Shodan, or direct IP scans)
type BackgroundScraperService struct {
	db       *database.DB
	interval time.Duration
	stop     chan struct{}
}

func NewBackgroundScraperService(db *database.DB) *BackgroundScraperService {
	return &BackgroundScraperService{
		db:       db,
		interval: 12 * time.Hour, // Run scans roughly twice a day
		stop:     make(chan struct{}),
	}
}

func (s *BackgroundScraperService) Start() {
	log.Println("[scraper] starting background scraper service")
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		// Do an initial run on boot in goroutine
		go s.runScraping()

		for {
			select {
			case <-ticker.C:
				s.runScraping()
			case <-s.stop:
				log.Println("[scraper] stopped background scraper service")
				return
			}
		}
	}()
}

func (s *BackgroundScraperService) Stop() {
	close(s.stop)
}

func (s *BackgroundScraperService) runScraping() {
	log.Println("[scraper] initiating active scrape jobs")
	var wg sync.WaitGroup

	shodanKey := os.Getenv("SHODAN_API_KEY")
	if shodanKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.scrapeShodan(shodanKey)
		}()
	}

	// Other scrapers like Fofa or Censys could be added here
	wg.Wait()
}

func (s *BackgroundScraperService) scrapeShodan(apiKey string) {
	log.Println("[scraper-shodan] starting shodan query for 'port:11434 product:Ollama'")
	
	client := &http.Client{Timeout: 30 * time.Second}
	url := fmt.Sprintf("https://api.shodan.io/shodan/host/search?key=%s&query=port:11434", apiKey)
	
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("[scraper-shodan] failed query: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[scraper-shodan] API returned status: %d", resp.StatusCode)
		return
	}

	var data struct {
		Matches []struct {
			IPStr string `json:"ip_str"`
			Port  int    `json:"port"`
		} `json:"matches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("[scraper-shodan] failed to decode response: %v", err)
		return
	}

	log.Printf("[scraper-shodan] found %d matches, importing newly discovered IPs", len(data.Matches))

	imported := 0
	for _, match := range data.Matches {
		endpointURL := fmt.Sprintf("http://%s:%d", match.IPStr, match.Port)
		
		// Avoid inserting duplicates
		var exists bool
		err := s.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM endpoints WHERE url = $1)", endpointURL)
		if err == nil && !exists {
			var newID int
			err = s.db.QueryRow("INSERT INTO endpoints (url, name, status) VALUES ($1, $2, 'pending') RETURNING id", endpointURL, endpointURL).Scan(&newID)
			if err == nil {
				// We don't need to insert into endpoint_test_tasks since cyclical poll catches all pending eventually,
				// but let's insert it immediately so it gets tested directly
				s.db.Exec("INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES ($1, NOW(), 'pending')", newID)
				imported++
			}
		}
	}
	
	log.Printf("[scraper-shodan] successfully imported %d new endpoints", imported)
}
