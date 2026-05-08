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
