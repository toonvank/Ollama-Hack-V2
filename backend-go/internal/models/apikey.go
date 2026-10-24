package models

import (
	"strings"
	"time"
)

type APIKey struct {
	ID            int        `db:"id" json:"id"`
	Key           string     `db:"key" json:"key"`
	Name          string     `db:"name" json:"name"`
	UserID        int        `db:"user_id" json:"user_id"`
	AllowedModels *string    `db:"allowed_models" json:"allowed_models"`
	LastUsedAt    *time.Time `db:"last_used_at" json:"last_used_at"`
	CreatedAt     time.Time  `db:"created_at" json:"created_at"`
}

// ParseAllowedModels splits the comma-separated DB column into a list.
// Returns nil when unrestricted (NULL/empty).
func (k *APIKey) ParseAllowedModels() []string {
	if k.AllowedModels == nil {
		return nil
	}
	var out []string
	for _, part := range strings.Split(*k.AllowedModels, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// JoinAllowedModels collapses a filter list into the DB column value (nil = unrestricted).
func JoinAllowedModels(list []string) *string {
	if len(list) == 0 {
		return nil
	}
	var cleaned []string
	for _, part := range list {
		if p := strings.TrimSpace(part); p != "" {
			cleaned = append(cleaned, p)
		}
	}
	if len(cleaned) == 0 {
		return nil
	}
	joined := strings.Join(cleaned, ",")
	return &joined
}

type APIKeyCreate struct {
	Name          string   `json:"name" binding:"required"`
	AllowedModels []string `json:"allowed_models"`
}

type APIKeyResponse struct {
	ID            int        `json:"id"`
	Key           string     `json:"key"`
	Name          string     `json:"name"`
	UserID        int        `json:"user_id"`
	AllowedModels []string   `json:"allowed_models"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type APIKeyInfo struct {
	ID            int        `json:"id"`
	Name          string     `json:"name"`
	UserID        int        `json:"user_id"`
	AllowedModels []string   `json:"allowed_models"`
	LastUsedAt    *time.Time `json:"last_used_at"`
	CreatedAt     time.Time  `json:"created_at"`
}
