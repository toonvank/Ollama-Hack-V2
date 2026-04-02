package models

import "time"

type Plan struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	RPM         int       `db:"rpm" json:"rpm"`
	RPD         int       `db:"rpd" json:"rpd"`
	IsDefault   bool      `db:"is_default" json:"is_default"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

type PlanCreate struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	RPM         int    `json:"rpm" binding:"required,min=0"`
	RPD         int    `json:"rpd" binding:"required,min=0"`
	IsDefault   bool   `json:"is_default"`
}

type PlanUpdate struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	RPM         *int    `json:"rpm" binding:"omitempty,min=0"`
	RPD         *int    `json:"rpd" binding:"omitempty,min=0"`
	IsDefault   *bool   `json:"is_default"`
}
