package service

import "github.com/dungpd/seta/auth-service/internal/model"

type UserRepo interface {
	Create(*model.User) error
	FindByEmail(string) (*model.User, error)
	FindAll() ([]model.User, error)
}

type RefreshTokenRepo interface {
	Insert(*model.RefreshToken) error
	MarkRevoked(jti string) error
	IsValid(jti string) (bool, error)
	RevokeAllForUser(userID string) error
}
