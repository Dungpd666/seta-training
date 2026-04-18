package model

import (
	"time"
)

type User struct {
	ID           string `gorm:"type:uuid;default:gen_random_uuid()"`
	Username     string `gorm:"not null"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null; check:role IN('manager', 'member')"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
