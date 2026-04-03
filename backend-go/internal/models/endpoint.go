package models

import "time"

type Endpoint struct {
	ID        int       `db:"id" json:"id"`
	URL       string    `db:"url" json:"url"`
	Name      string    `db:"name" json:"name"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// EndpointPerformance represents a performance record
type EndpointPerformance struct {
	ID            int       `db:"id" json:"id"`
	Status        string    `db:"status" json:"status"`
	OllamaVersion *string   `db:"ollama_version" json:"ollama_version"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

// EndpointWithAIModelCount is the response for list endpoint
type EndpointWithAIModelCount struct {
	ID                     int                   `json:"id"`
	URL                    string                `json:"url"`
	Name                   string                `json:"name"`
	Status                 string                `json:"status,omitempty"`
	CreatedAt              time.Time             `json:"created_at"`
	RecentPerformances     []EndpointPerformance `json:"recent_performances"`
	TotalAIModelCount      int                   `json:"total_ai_model_count"`
	AvailableAIModelCount  int                   `json:"avaliable_ai_model_count"` // Note: frontend typo "avaliable"
	TaskStatus             *string               `json:"task_status"`
}

type EndpointCreate struct {
	URL  string `json:"url" binding:"required"`
	Name string `json:"name"`
}

type EndpointUpdate struct {
	Name *string `json:"name"`
	URL  *string `json:"url"`
}

type EndpointBatchCreate struct {
	Endpoints []EndpointCreate `json:"endpoints" binding:"required"`
}

type BatchOperationResult struct {
	SuccessCount int               `json:"success_count"`
	FailedCount  int               `json:"failed_count"`
	FailedIDs    map[string]string `json:"failed_ids"`
}

type EndpointBatchOperation struct {
	EndpointIDs []int `json:"endpoint_ids" binding:"required"`
}

type EndpointTestTask struct {
	ID          int        `db:"id" json:"id"`
	EndpointID  int        `db:"endpoint_id" json:"endpoint_id"`
	Status      string     `db:"status" json:"status"`
	ScheduledAt time.Time  `db:"scheduled_at" json:"scheduled_at"`
	LastTried   *time.Time `db:"last_tried" json:"last_tried,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}
