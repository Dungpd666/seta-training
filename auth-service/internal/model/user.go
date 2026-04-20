package model

import "time"

type User struct {
	UserID       string    `gorm:"column:user_id;primaryKey"`
	Username     string    `gorm:"column:username"`
	Email        string    `gorm:"column:email"`
	PasswordHash string    `gorm:"column:password_hash"`
	Role         string    `gorm:"column:role"`
	CreatedAt    time.Time `gorm:"column:created_at"`
}

func (User) TableName() string { return "users" }
