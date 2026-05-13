package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
	ErrTokenRevoked = errors.New("token reused or revoked")
)

type RefreshToken struct {
	JTI       string
	UserID    string
	ExpiresAt time.Time
	Revoked   bool
}
