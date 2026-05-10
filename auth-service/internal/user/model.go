package user

import (
	"errors"
	"time"
)

var (
	ErrEmailInUse         = errors.New("email already in use")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

const (
	TopicUserEvents = "user.events"

	EventUserCreated   = "USER_CREATED"
	EventUserUpdated   = "USER_UPDATED"
	EventUserDeleted   = "USER_DELETED"
	EventUsersImported = "USERS_IMPORTED"
)

type UserEvent struct {
	Type     string `json:"type"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type UsersImportedEvent struct {
	Type      string `json:"type"`
	Succeeded int    `json:"succeeded"`
	Failed    int    `json:"failed"`
}

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
