package user

import (
	"errors"
	"time"
)

var (
	ErrEmailInUse         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type User struct {
	UserID       string
	Username     string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

type UserResponse struct {
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}
