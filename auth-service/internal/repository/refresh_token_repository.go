package repository

import (
	"errors"

	"github.com/dungpd/seta/auth-service/internal/domain"
	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Insert(rt *domain.RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *RefreshTokenRepository) MarkRevoked(jti string) error {
	return r.db.Model(&domain.RefreshToken{}).Where("jti = ?", jti).Update("revoked", true).Error
}

func (r *RefreshTokenRepository) IsValid(jti string) (bool, error) {
	var rt domain.RefreshToken
	err := r.db.Where("jti = ? AND revoked = false AND expires_at > NOW()", jti).First(&rt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (r *RefreshTokenRepository) RevokeAllForUser(userID string) error {
	return r.db.Model(&domain.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error
}
