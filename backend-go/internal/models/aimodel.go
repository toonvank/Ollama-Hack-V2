package models

import "time"

type AIModel struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Tag       string    `db:"tag" json:"tag"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type AIModelInfo struct {
	ID             int       `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Tag            string    `json:"tag" db:"tag"`
	Enabled        bool      `json:"enabled" db:"enabled"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	Endpoints      int       `json:"endpoints" db:"endpoints"`
	TokenPerSecond *float64  `json:"token_per_second" db:"token_per_second"`
}

type AIModelToggle struct {
	Enabled bool `json:"enabled"`
}

type AIModelPerformance struct {
	EndpointID        int      `json:"endpoint_id"`
	EndpointName      string   `json:"endpoint_name"`
	Status            string   `json:"status"`
	TokenPerSecond    *float64 `json:"token_per_second"`
	MaxConnectionTime *float64 `json:"max_connection_time"`
}

type AIModelDetail struct {
	AIModelInfo
	Performances []AIModelPerformance `json:"performances"`
}
