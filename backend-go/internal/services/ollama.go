package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/timlzh/ollama-hack/internal/database"
)

// EndpointStatus constants
const (
	StatusAvailable   = "available"
	StatusUnavailable = "unavailable"
	StatusFake        = "fake"
	StatusPending     = "pending"
)

// OllamaTagsResponse is the /api/tags response shape
type OllamaTagsResponse struct {
	Models []struct {
		Model string `json:"model"`
	} `json:"models"`
}

// OllamaVersionResponse is the /api/version response shape
type OllamaVersionResponse struct {
	Version string `json:"version"`
}

// OllamaGenerateResponse is one streaming chunk from /api/generate
type OllamaGenerateResponse struct {
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	EvalCount int    `json:"eval_count"`
}

// ModelTestResult holds results for a single model test
type ModelTestResult struct {
	ModelName      string
	ModelTag       string
	Status         string
	TokenPerSecond float64
	ConnectionTime float64
	TotalTime      float64
	Output         string
	OutputTokens   int
}

// EndpointTestResult aggregates the full test result for an endpoint
type EndpointTestResult struct {
	EndpointURL    string
	EndpointStatus string
	OllamaVersion  string
	Models         []ModelTestResult
}

const testPrompt = "Explain the concept of recursion in computer science. Provide a simple example and describe how the call stack works during recursive function execution."

// TestEndpoint fully tests an endpoint: version, lists models, tests each model
func TestEndpoint(endpointURL string) *EndpointTestResult {
	result := &EndpointTestResult{
		EndpointURL:    endpointURL,
		EndpointStatus: StatusUnavailable,
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// 1. Check version
	versionResp, err := client.Get(endpointURL + "/api/version")
	if err != nil || versionResp.StatusCode != http.StatusOK {
		log.Printf("[tester] endpoint %s unreachable: %v", endpointURL, err)
		return result
	}
	defer versionResp.Body.Close()
	var versionData OllamaVersionResponse
	json.NewDecoder(versionResp.Body).Decode(&versionData)
	result.OllamaVersion = versionData.Version
	result.EndpointStatus = StatusAvailable

	// 2. List models
	tagsResp, err := client.Get(endpointURL + "/api/tags")
	if err != nil || tagsResp.StatusCode != http.StatusOK {
		return result
	}
	defer tagsResp.Body.Close()
	var tagsData OllamaTagsResponse
	json.NewDecoder(tagsResp.Body).Decode(&tagsData)

	// 3. Test each model concurrently (bounded concurrency of 3 to avoid overwhelming the node)
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, 3) // max 3 concurrent tests
	isFake := false

	for _, m := range tagsData.Models {
		parts := strings.SplitN(m.Model, ":", 2)
		if len(parts) != 2 {
			continue
		}

		wg.Add(1)
		go func(name, tag string) {
			defer wg.Done()

			// If already identified as fake, don't start new tests
			mu.Lock()
			fake := isFake
			mu.Unlock()
			if fake {
				return
			}

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			mr := testModel(endpointURL, name, tag)

			mu.Lock()
			if mr.Status == StatusFake {
				isFake = true
				result.EndpointStatus = StatusFake
			}
			result.Models = append(result.Models, mr)
			mu.Unlock()

		}(parts[0], parts[1])
	}
	wg.Wait()

	return result
}

func testModel(endpointURL, name, tag string) ModelTestResult {
	mr := ModelTestResult{
		ModelName: name,
		ModelTag:  tag,
		Status:    StatusUnavailable,
	}

	body, _ := json.Marshal(map[string]interface{}{
		"model":  fmt.Sprintf("%s:%s", name, tag),
		"prompt": testPrompt,
		"stream": true,
	})

	client := &http.Client{Timeout: 70 * time.Second}
	req, err := http.NewRequest("POST", endpointURL+"/api/generate", strings.NewReader(string(body)))
	if err != nil {
		return mr
	}
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return mr
	}
	defer resp.Body.Close()

	var outputBuilder strings.Builder
	var outputTokens int
	var connectionTime float64
	firstChunk := true

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if firstChunk {
			connectionTime = time.Since(start).Seconds()
			firstChunk = false
		}
		var chunk OllamaGenerateResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			continue
		}
		outputBuilder.WriteString(chunk.Response)

		// Fake endpoint detection
		out := strings.ToLower(outputBuilder.String())
		if strings.Contains(out, "fake-ollama") || strings.Contains(out, "server busy") {
			mr.Status = StatusFake
			return mr
		}

		if chunk.Done {
			outputTokens = chunk.EvalCount
			break
		}
	}

	totalTime := time.Since(start).Seconds()
	output := outputBuilder.String()

	if output == "" {
		return mr
	}

	if outputTokens == 0 {
		// Rough estimate: 1 token ≈ 4 chars
		outputTokens = len(output) / 4
	}

	tps := 0.0
	if totalTime > 0 {
		tps = float64(outputTokens) / totalTime
	}

	mr.Status = StatusAvailable
	mr.TokenPerSecond = tps
	mr.ConnectionTime = connectionTime
	mr.TotalTime = totalTime
	mr.Output = output
	mr.OutputTokens = outputTokens
	return mr
}

// Tester is a background goroutine-based task runner that periodically picks
// pending tasks from endpoint_test_tasks and runs them.
type Tester struct {
	db       *database.DB
	interval time.Duration
	stop     chan struct{}
}

func NewTester(db *database.DB) *Tester {
	return &Tester{
		db:       db,
		interval: 10 * time.Second,
		stop:     make(chan struct{}),
	}
}

func (t *Tester) Start() {
	log.Println("[tester] background tester started")
	go func() {
		ticker := time.NewTicker(t.interval)
		fetchTicker := time.NewTicker(1 * time.Hour) // Fetch every hour
		defer ticker.Stop()
		defer fetchTicker.Stop()

		// Run fetch immediately on startup
		go t.fetchExternalEndpoints()

		for {
			select {
			case <-ticker.C:
				t.runPendingTasks()
			case <-fetchTicker.C:
				go t.fetchExternalEndpoints()
			case <-t.stop:
				log.Println("[tester] background tester stopped")
				return
			}
		}
	}()
}

func (t *Tester) Stop() {
	close(t.stop)
}

func (t *Tester) fetchExternalEndpoints() {
	log.Println("[tester] fetching external endpoints from ollama.vincentko.top")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://ollama.vincentko.top/data.json")
	if err != nil {
		log.Printf("[tester] failed to fetch external endpoints: %v", err)
		return
	}
	defer resp.Body.Close()

	var data []struct {
		Server string `json:"server"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("[tester] failed to decode external endpoints JSON: %v", err)
		return
	}

	importedCount := 0
	for _, item := range data {
		url := strings.TrimSpace(item.Server)
		if url == "" {
			continue
		}

		// Check if exists
		var exists bool
		err := t.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM endpoints WHERE url = $1)", url)
		if err != nil || exists {
			continue
		}

		// Insert
		var newID int
		err = t.db.QueryRow("INSERT INTO endpoints (url, name, status) VALUES ($1, $2, 'pending') RETURNING id", url, url).Scan(&newID)
		if err == nil {
			// Schedule task immediately
			t.db.Exec("INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES ($1, NOW(), 'pending')", newID)
			importedCount++
		}
	}
	log.Printf("[tester] successfully imported %d new external endpoints", importedCount)
}

type pendingTask struct {
	ID          int    `db:"id"`
	EndpointID  int    `db:"endpoint_id"`
	EndpointURL string `db:"url"`
}

func (t *Tester) runPendingTasks() {
	var tasks []pendingTask
	err := t.db.Select(&tasks, `
		SELECT ett.id, ett.endpoint_id, e.url 
		FROM endpoint_test_tasks ett
		JOIN endpoints e ON e.id = ett.endpoint_id
		WHERE ett.status = 'pending' AND ett.scheduled_at <= NOW()
		ORDER BY ett.scheduled_at ASC
		LIMIT 100
	`)
	if err != nil {
		return
	}
	if len(tasks) == 0 {
		return
	}

	// Bulk update tasks to 'running'
	var taskIDs []int
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}

	query := "UPDATE endpoint_test_tasks SET status = 'running', last_tried = NOW() WHERE id IN ("
	var args []interface{}
	for i, id := range taskIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", i+1)
		args = append(args, id)
	}
	query += ")"
	t.db.Exec(query, args...)

	for _, task := range tasks {
		log.Printf("[tester] testing endpoint %d (%s)", task.EndpointID, task.EndpointURL)
		go t.executeTask(task)
	}
}

func (t *Tester) executeTask(task pendingTask) {
	result := TestEndpoint(task.EndpointURL)

	tx, err := t.db.Beginx()
	if err != nil {
		t.db.Exec("UPDATE endpoint_test_tasks SET status = 'failed' WHERE id = $1", task.ID)
		return
	}

	// Update endpoint status
	tx.Exec("UPDATE endpoints SET status = $1 WHERE id = $2", result.EndpointStatus, task.EndpointID)

	// Insert endpoint performance record
	var epPerfID int
	err = tx.QueryRow(`
		INSERT INTO endpoint_performances (endpoint_id, status, ollama_version)
		VALUES ($1, $2, $3) RETURNING id`,
		task.EndpointID, result.EndpointStatus, result.OllamaVersion,
	).Scan(&epPerfID)
	if err != nil {
		// Table might not exist yet — log but continue
		log.Printf("[tester] could not insert endpoint_performances: %v", err)
	}

	// Upsert models and their performances
	for _, mr := range result.Models {
		// Upsert ai_model
		var modelID int
		err = tx.QueryRow(`
			INSERT INTO ai_models (name, tag) VALUES ($1, $2)
			ON CONFLICT (name, tag) DO UPDATE SET name = EXCLUDED.name
			RETURNING id`,
			mr.ModelName, mr.ModelTag,
		).Scan(&modelID)
		if err != nil {
			log.Printf("[tester] could not upsert ai_model %s:%s: %v", mr.ModelName, mr.ModelTag, err)
			continue
		}

		// Upsert endpoint_ai_model link
		var linkID int
		err = tx.QueryRow(`
			INSERT INTO endpoint_ai_models (endpoint_id, ai_model_id, status, token_per_second)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (endpoint_id, ai_model_id) DO UPDATE
				SET status = EXCLUDED.status, token_per_second = EXCLUDED.token_per_second
			RETURNING id`,
			task.EndpointID, modelID, mr.Status, mr.TokenPerSecond,
		).Scan(&linkID)
		if err != nil {
			log.Printf("[tester] could not upsert endpoint_ai_model: %v", err)
			continue
		}

		// Insert performance record
		tx.Exec(`
			INSERT INTO ai_model_performances
				(endpoint_ai_model_id, token_per_second, max_connection_time, total_time, output, output_tokens)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			linkID, mr.TokenPerSecond, mr.ConnectionTime, mr.TotalTime,
			mr.Output, mr.OutputTokens,
		)
	}

	if err := tx.Commit(); err != nil {
		log.Printf("[tester] commit error: %v", err)
		t.db.Exec("UPDATE endpoint_test_tasks SET status = 'failed' WHERE id = $1", task.ID)
		return
	}

	t.db.Exec("UPDATE endpoint_test_tasks SET status = 'done' WHERE id = $1", task.ID)
	log.Printf("[tester] finished task %d for endpoint %d — status: %s, models tested: %d",
		task.ID, task.EndpointID, result.EndpointStatus, len(result.Models))
}
