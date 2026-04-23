package domain

import "time"

type RefreshToken struct {
	JTI       string    `gorm:"column:jti;primaryKey"`
	UserID    string    `gorm:"column:user_id"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	Revoked   bool      `gorm:"column:revoked"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
