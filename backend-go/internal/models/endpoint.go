package models

import "time"

type Endpoint struct {
	ID        int       `db:"id" json:"id"`
	URL       string    `db:"url" json:"url"`
	Name      string    `db:"name" json:"name"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
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
