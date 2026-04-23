package domain

import "time"

type User struct {
	UserID       string
	Username     string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}
