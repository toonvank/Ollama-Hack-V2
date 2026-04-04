package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/utils"
)

// EndpointStatus constants
const (
	StatusAvailable   = "available"
	StatusUnavailable = "unavailable"
	StatusFake        = "fake"
	StatusPending     = "pending"
)

// EndpointType constants
const (
	EndpointTypeOllama = "ollama"
	EndpointTypeOpenAI = "openai"
)

// OllamaTagsResponse is the /api/tags response shape
type OllamaTagsResponse struct {
	Models []struct {
		Model string `json:"model"`
	} `json:"models"`
}

// OpenAIModelsResponse is the /v1/models response shape
type OpenAIModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	} `json:"data"`
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

// getModelLockID generates a deterministic lock ID for a model name+tag pair
// to use with PostgreSQL advisory locks
func getModelLockID(name, tag string) int64 {
	h := fnv.New64a()
	h.Write([]byte(name + ":" + tag))
	return int64(h.Sum64() & 0x7FFFFFFFFFFFFFFF)
}

func getPollTimeout() time.Duration {
	if val := os.Getenv("POLL_TIMEOUT_SECS"); val != "" {
		if secs, err := strconv.Atoi(val); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 300 * time.Second // default 5 minutes
}

// TestEndpointWithType tests an endpoint based on its type
func TestEndpointWithType(endpointURL, endpointType string, apiKey *string) *EndpointTestResult {
	if endpointType == EndpointTypeOpenAI {
		return TestOpenAIEndpoint(endpointURL, apiKey)
	}
	return TestEndpoint(endpointURL)
}

// TestOpenAIEndpoint tests an OpenAI-compatible endpoint
func TestOpenAIEndpoint(endpointURL string, apiKey *string) *EndpointTestResult {
	result := &EndpointTestResult{
		EndpointURL:    endpointURL,
		EndpointStatus: StatusUnavailable,
		OllamaVersion:  "OpenAI Compatible",
	}

	client := &http.Client{Timeout: getPollTimeout()}

	// Test /v1/models endpoint
	req, err := http.NewRequest("GET", endpointURL+"/v1/models", nil)
	if err != nil {
		log.Printf("[tester] failed to create request for %s: %v", endpointURL, err)
		return result
	}

	// Add Bearer token if API key provided
	if apiKey != nil && *apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+*apiKey)
	}

	modelsResp, err := client.Do(req)
	if err != nil || modelsResp.StatusCode != http.StatusOK {
		log.Printf("[tester] OpenAI endpoint %s unreachable or unauthorized: %v (status: %v)", 
			endpointURL, err, modelsResp.StatusCode)
		return result
	}
	defer modelsResp.Body.Close()

	var modelsData OpenAIModelsResponse
	if err := json.NewDecoder(modelsResp.Body).Decode(&modelsData); err != nil {
		log.Printf("[tester] failed to decode models response: %v", err)
		return result
	}

	result.EndpointStatus = StatusAvailable

	// Convert OpenAI models to our format
	// For OpenAI endpoints, we don't test each model individually (would be expensive)
	// Just mark them as available
	for _, model := range modelsData.Data {
		// Split model ID into name:tag format
		// For most services, the ID is the full model name
		parts := strings.SplitN(model.ID, ":", 2)
		name := parts[0]
		tag := "latest"
		if len(parts) == 2 {
			tag = parts[1]
		}

		result.Models = append(result.Models, ModelTestResult{
			ModelName:      name,
			ModelTag:       tag,
			Status:         StatusAvailable,
			TokenPerSecond: 0, // Not tested for OpenAI endpoints
		})
	}

	return result
}


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

	client := &http.Client{Timeout: getPollTimeout()}
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
		requeueTicker := time.NewTicker(1 * time.Hour) // Re-queue old tests
		statsTicker := time.NewTicker(5 * time.Second) // Poll stats for frontend
		defer ticker.Stop()
		defer fetchTicker.Stop()
		defer requeueTicker.Stop()
		defer statsTicker.Stop()

		// Run fetch immediately on startup
		go t.fetchExternalEndpoints()

		var lastCompleted uint64 = utils.TestTasksCompleted.Load()
		var lastTime = time.Now()
		var currentSpeed float64 = 0

		for {
			select {
			case <-ticker.C:
				t.runPendingTasks()
			case <-fetchTicker.C:
				go t.fetchExternalEndpoints()
			case <-requeueTicker.C:
				go t.queueCyclicalTests()
			case <-statsTicker.C:
				var count int64
				err := t.db.Get(&count, "SELECT COUNT(*) FROM endpoint_test_tasks WHERE status = 'pending'")
				if err == nil {
					utils.PendingTestsQueue.Store(count)
				}
				now := time.Now()
				completedNow := utils.TestTasksCompleted.Load()
				elapsedMins := now.Sub(lastTime).Minutes()
				if elapsedMins > 0 {
					speedThisTick := float64(completedNow-lastCompleted) / elapsedMins
					if currentSpeed == 0 {
						currentSpeed = speedThisTick
					} else {
						currentSpeed = (currentSpeed * 0.8) + (speedThisTick * 0.2) // EMA
					}
					utils.TesterSpeed.Store(uint64(currentSpeed))
				}
				lastCompleted = completedNow
				lastTime = now
			case <-t.stop:
				log.Println("[tester] background tester stopped")
				return
			}
		}
	}()
}

func (t *Tester) queueCyclicalTests() {
	log.Println("[tester] checking for endpoints that need re-testing (>24h)")
	query := `
		INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status)
		SELECT e.id, NOW(), 'pending'
		FROM endpoints e
		WHERE NOT EXISTS (
			SELECT 1 FROM endpoint_test_tasks ett 
			WHERE ett.endpoint_id = e.id AND (ett.status = 'pending' OR ett.last_tried >= NOW() - INTERVAL '24 hours')
		)
		LIMIT 500
	`
	res, err := t.db.Exec(query)
	if err != nil {
		log.Printf("[tester] failed to queue cyclical tests: %v", err)
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("[tester] dynamically re-queued %d endpoints for cyclical testing", rowsAffected)
	}
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
	ID           int     `db:"id"`
	EndpointID   int     `db:"endpoint_id"`
	EndpointURL  string  `db:"url"`
	EndpointType string  `db:"endpoint_type"`
	APIKey       *string `db:"api_key"`
}

func (t *Tester) runPendingTasks() {
	var tasks []pendingTask
	err := t.db.Select(&tasks, `
		SELECT ett.id, ett.endpoint_id, e.url, e.endpoint_type, e.api_key
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
	result := TestEndpointWithType(task.EndpointURL, task.EndpointType, task.APIKey)

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
		// Acquire advisory lock to prevent concurrent upserts of the same model
		lockID := getModelLockID(mr.ModelName, mr.ModelTag)
		_, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", lockID)
		if err != nil {
			log.Printf("[tester] could not acquire advisory lock for %s:%s: %v", mr.ModelName, mr.ModelTag, err)
			continue
		}

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

	utils.TestTasksCompleted.Add(1)
	t.db.Exec("UPDATE endpoint_test_tasks SET status = 'done' WHERE id = $1", task.ID)
	log.Printf("[tester] finished task %d for endpoint %d — status: %s, models tested: %d",
		task.ID, task.EndpointID, result.EndpointStatus, len(result.Models))
}
