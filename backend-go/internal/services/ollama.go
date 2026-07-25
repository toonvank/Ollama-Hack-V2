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
	return 45 * time.Second // was 300; generate-per-model was starving the queue
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

	client := utils.BackgroundHTTPClient(getPollTimeout())

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
	if err != nil {
		// http.Client returns a nil response for transport failures (DNS,
		// connection refusal, timeout).  The tester processes untrusted
		// discovered endpoints, so this must never take down the API server.
		log.Printf("[tester] OpenAI endpoint %s unreachable: %v", endpointURL, err)
		return result
	}
	if modelsResp.StatusCode != http.StatusOK {
		log.Printf("[tester] OpenAI endpoint %s unauthorized or unavailable (status: %d)",
			endpointURL, modelsResp.StatusCode)
		modelsResp.Body.Close()
		return result
	}
	defer modelsResp.Body.Close()

	var modelsData OpenAIModelsResponse
	if err := json.NewDecoder(modelsResp.Body).Decode(&modelsData); err != nil {
		log.Printf("[tester] failed to decode models response: %v", err)
		return result
	}

	result.EndpointStatus = StatusAvailable

	// Smoke test each model via /v1/chat/completions to measure TPS
	for _, model := range modelsData.Data {
		parts := strings.SplitN(model.ID, ":", 2)
		name := parts[0]
		tag := "latest"
		if len(parts) == 2 {
			tag = parts[1]
		}

		tps := testOpenAIModel(endpointURL, model.ID, apiKey)
		status := StatusAvailable
		if tps <= 0 {
			status = StatusUnavailable
		}

		result.Models = append(result.Models, ModelTestResult{
			ModelName:      name,
			ModelTag:       tag,
			Status:         status,
			TokenPerSecond: tps,
		})
	}

	return result
}

// testOpenAIModel runs a smoke test via /v1/chat/completions and returns measured TPS
func testOpenAIModel(endpointURL, modelID string, apiKey *string) float64 {
	body, _ := json.Marshal(map[string]interface{}{
		"model":      modelID,
		"messages":   []map[string]string{{"role": "user", "content": testPrompt}},
		"max_tokens": 20,
		"stream":     false,
	})

	client := utils.BackgroundHTTPClient(getPollTimeout())
	req, err := http.NewRequest("POST", endpointURL+"/v1/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return 0
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != nil && *apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+*apiKey)
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	var result struct {
		Usage struct {
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	elapsed := time.Since(start).Seconds()
	if elapsed <= 0 || result.Usage.CompletionTokens <= 0 {
		if elapsed > 0 {
			return 1.0 / elapsed
		}
		return 0
	}

	return float64(result.Usage.CompletionTokens) / elapsed
}

// TestEndpoint fully tests an endpoint: version, lists models, tests each model
func TestEndpoint(endpointURL string) *EndpointTestResult {
	result := &EndpointTestResult{
		EndpointURL:    endpointURL,
		EndpointStatus: StatusUnavailable,
	}

	client := utils.BackgroundHTTPClient(10 * time.Second)

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

	client := utils.BackgroundHTTPClient(getPollTimeout())
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
	taskChan chan pendingTask
}

func NewTester(db *database.DB) *Tester {
	return &Tester{
		db:       db,
		interval: 10 * time.Second,
		stop:     make(chan struct{}),
		taskChan: make(chan pendingTask, 500),
	}
}

func (t *Tester) Start() {
	log.Println("[tester] background tester started")

	// One-shot: never let unmeasured links sit as race-eligible
	if res, err := t.db.Exec(`
		UPDATE endpoint_ai_models
		SET status = 'unavailable'
		WHERE status = 'available'
		  AND (token_per_second IS NULL OR token_per_second <= 0)
	`); err != nil {
		log.Printf("[tester] demote unmeasured links failed: %v", err)
	} else if n, _ := res.RowsAffected(); n > 0 {
		log.Printf("[tester] demoted %d unmeasured race-eligible model links at startup", n)
	}

	// Worker pool size (TESTER_WORKERS, default 40)
	workers := 40
	if v := os.Getenv("TESTER_WORKERS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 200 {
			workers = n
		}
	}
	log.Printf("[tester] starting %d workers", workers)
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case task := <-t.taskChan:
					t.executeTask(task)
				case <-t.stop:
					return
				}
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(t.interval)
		fetchTicker := time.NewTicker(1 * time.Hour)   // Fetch every hour
		requeueTicker := time.NewTicker(1 * time.Hour) // Check for tests to re-queue every hour
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
	log.Println("[tester] checking for endpoints that need re-testing")

	intervalStr := os.Getenv("CYCLICAL_TEST_INTERVAL_HOURS")
	if intervalStr == "" {
		intervalStr = "24"
	}
	intervalInt, err := strconv.Atoi(intervalStr)
	if err != nil || intervalInt <= 0 {
		intervalInt = 24
	}

	query := fmt.Sprintf(`
		INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status)
		SELECT e.id, NOW(), 'pending'
		FROM endpoints e
		WHERE NOT EXISTS (
			SELECT 1 FROM endpoint_test_tasks ett 
			WHERE ett.endpoint_id = e.id 
			  AND (
				ett.status = 'pending' 
				OR (e.status IN ('available', 'pending') AND ett.last_tried >= NOW() - INTERVAL '%d hours')
				OR (e.status IN ('unavailable', 'fake') AND ett.last_tried >= NOW() - INTERVAL '168 hours')
			  )
		)
		LIMIT 500
	`, intervalInt)
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

// defaultExternalEndpointFeeds is the community Awesome-Ollama-Server catalog.
// Hosted site ollama.vincentko.top died (Vercel DEPLOYMENT_DISABLED), but GitHub
// Actions still publishes public/data.json every ~10h from forrany/Awesome-Ollama-Server.
const defaultExternalEndpointFeed = "https://raw.githubusercontent.com/forrany/Awesome-Ollama-Server/main/public/data.json"

// externalEndpointFeeds returns catalog URLs to import.
// EXTERNAL_ENDPOINT_FEEDS=url1,url2  (comma-separated). Empty env → default GitHub raw.
// EXTERNAL_ENDPOINT_FEEDS=off|false|none|disabled → skip import entirely.
func externalEndpointFeeds() []string {
	raw := strings.TrimSpace(os.Getenv("EXTERNAL_ENDPOINT_FEEDS"))
	if raw == "" {
		return []string{defaultExternalEndpointFeed}
	}
	switch strings.ToLower(raw) {
	case "0", "false", "off", "none", "disabled", "skip":
		return nil
	}
	var feeds []string
	for _, part := range strings.Split(raw, ",") {
		u := strings.TrimSpace(part)
		if u != "" {
			feeds = append(feeds, u)
		}
	}
	return feeds
}

func (t *Tester) fetchExternalEndpoints() {
	feeds := externalEndpointFeeds()
	if len(feeds) == 0 {
		log.Println("[tester] external endpoint feeds disabled (EXTERNAL_ENDPOINT_FEEDS=off)")
		return
	}

	// Catalog feeds (GitHub raw, etc.) are not Ollama nodes — fetch direct.
	// Actual endpoint probing still uses BackgroundHTTPClient / racer elsewhere.
	client := utils.NewHTTPClient(45 * time.Second)
	totalImported := 0
	for _, feedURL := range feeds {
		n, err := t.importEndpointsFromFeed(client, feedURL)
		if err != nil {
			log.Printf("[tester] feed %s: %v", feedURL, err)
			continue
		}
		totalImported += n
		log.Printf("[tester] feed %s: imported %d new endpoint(s)", feedURL, n)
	}
	if totalImported > 0 {
		log.Printf("[tester] successfully imported %d new external endpoints total", totalImported)
	}
}

func (t *Tester) importEndpointsFromFeed(client *http.Client, feedURL string) (int, error) {
	log.Printf("[tester] fetching external endpoints from %s", feedURL)
	resp, err := client.Get(feedURL)
	if err != nil {
		return 0, fmt.Errorf("fetch failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Awesome-Ollama-Server / legacy vincentko format: [{"server":"http://...","models":[...]}]
	var data []struct {
		Server string `json:"server"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("decode JSON: %w", err)
	}

	importedCount := 0
	for _, item := range data {
		url := strings.TrimSpace(item.Server)
		if url == "" {
			continue
		}

		var exists bool
		err := t.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM endpoints WHERE url = $1)", url)
		if err != nil || exists {
			continue
		}

		var newID int
		err = t.db.QueryRow(
			"INSERT INTO endpoints (url, name, status) VALUES ($1, $2, 'pending') RETURNING id",
			url, url,
		).Scan(&newID)
		if err == nil {
			t.db.Exec(
				"INSERT INTO endpoint_test_tasks (endpoint_id, scheduled_at, status) VALUES ($1, NOW(), 'pending')",
				newID,
			)
			importedCount++
		}
	}
	return importedCount, nil
}

type pendingTask struct {
	ID           int     `db:"id"`
	EndpointID   int     `db:"endpoint_id"`
	EndpointURL  string  `db:"url"`
	EndpointType string  `db:"endpoint_type"`
	APIKey       *string `db:"api_key"`
}

func (t *Tester) runPendingTasks() {
	// Reclaim orphaned running tasks (backend restart / hung generate)
	if _, err := t.db.Exec(`
		UPDATE endpoint_test_tasks
		SET status = 'pending', scheduled_at = NOW() - INTERVAL '1 hour'
		WHERE status = 'running'
		  AND (last_tried IS NULL OR last_tried < NOW() - INTERVAL '2 minutes')
	`); err != nil {
		log.Printf("[tester] reclaim running failed: %v", err)
	}

	var tasks []pendingTask
	err := t.db.Select(&tasks, `
		SELECT ett.id, ett.endpoint_id, e.url, e.endpoint_type, e.api_key
		FROM endpoint_test_tasks ett
		JOIN endpoints e ON e.id = ett.endpoint_id
		WHERE ett.status = 'pending' AND ett.scheduled_at <= NOW()
		ORDER BY ett.scheduled_at ASC
		LIMIT 200
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
		select {
		case t.taskChan <- task:
			log.Printf("[tester] enqueued endpoint %d (%s) for testing", task.EndpointID, task.EndpointURL)
		case <-t.stop:
			return
		}
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
	// Host dead/fake → drop all model race eligibility on this endpoint immediately
	if result.EndpointStatus != StatusAvailable {
		tx.Exec(`UPDATE endpoint_ai_models SET status = $1, token_per_second = 0 WHERE endpoint_id = $2`,
			StatusUnavailable, task.EndpointID)
	}

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
		var connectionTime *float64
		if mr.Status == StatusAvailable && mr.ConnectionTime > 0 {
			connectionTime = &mr.ConnectionTime
		}

		var linkID int
		err = tx.QueryRow(`
			INSERT INTO endpoint_ai_models (endpoint_id, ai_model_id, status, token_per_second, max_connection_time)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (endpoint_id, ai_model_id) DO UPDATE
				SET status = EXCLUDED.status,
				    token_per_second = EXCLUDED.token_per_second,
				    max_connection_time = EXCLUDED.max_connection_time
			RETURNING id`,
			task.EndpointID, modelID, mr.Status, mr.TokenPerSecond, connectionTime,
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
