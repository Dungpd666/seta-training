package repository

import (
	"context"

	"github.com/dungpd/seta/auth-service/internal/db"
	"github.com/dungpd/seta/auth-service/internal/domain"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepository struct {
	queries *db.Queries
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{queries: db.New(pool)}
}

func (r *RefreshTokenRepository) Insert(rt *domain.RefreshToken) error {
	return r.queries.InsertRefreshToken(context.Background(), db.InsertRefreshTokenParams{
		Jti:       rt.JTI,
		UserID:    rt.UserID,
		ExpiresAt: pgtype.Timestamptz{Time: rt.ExpiresAt, Valid: true},
	})
}

func (r *RefreshTokenRepository) MarkRevoked(jti string) error {
	return r.queries.MarkRefreshTokenRevoked(context.Background(), jti)
}

func (r *RefreshTokenRepository) IsValid(jti string) (bool, error) {
	return r.queries.IsRefreshTokenValid(context.Background(), jti)
}

func (r *RefreshTokenRepository) RevokeAllForUser(userID string) error {
	return r.queries.RevokeAllRefreshTokensForUser(context.Background(), userID)
}
