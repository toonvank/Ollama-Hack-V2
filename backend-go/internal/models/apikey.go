package models

import "time"

type APIKey struct {
	ID         int        `db:"id" json:"id"`
	Key        string     `db:"key" json:"key"`
	Name       string     `db:"name" json:"name"`
	UserID     int        `db:"user_id" json:"user_id"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

type APIKeyCreate struct {
	Name string `json:"name" binding:"required"`
}

type APIKeyResponse struct {
	ID         int        `json:"id"`
	Key        string     `json:"key"`
	Name       string     `json:"name"`
	UserID     int        `json:"user_id"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type APIKeyInfo struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	UserID     int        `json:"user_id"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}
