package models

import "time"

type User struct {
	ID             int       `db:"id" json:"id"`
	Username       string    `db:"username" json:"username"`
	HashedPassword string    `db:"hashed_password" json:"-"`
	IsAdmin        bool      `db:"is_admin" json:"is_admin"`
	PlanID         *int      `db:"plan_id" json:"plan_id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
}

type UserCreate struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=8,max=128"`
	IsAdmin  bool   `json:"is_admin"`
	PlanID   *int   `json:"plan_id"`
}

type UserUpdate struct {
	Username *string `json:"username"`
	Password *string `json:"password,omitempty" binding:"omitempty,min=8,max=128"`
	IsAdmin  *bool   `json:"is_admin"`
	PlanID   *int    `json:"plan_id"`
}

type UserInfo struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	PlanID   *int   `json:"plan_id"`
	PlanName string `json:"plan_name,omitempty"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}
