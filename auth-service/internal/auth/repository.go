package auth

import (
	"context"

	"github.com/dungpd/seta/auth-service/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type refreshTokenRepository struct {
	queries *db.Queries
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) RefreshTokenRepo {
	return &refreshTokenRepository{queries: db.New(pool)}
}

func (r *refreshTokenRepository) Insert(ctx context.Context, rt *RefreshToken) error {
	return r.queries.InsertRefreshToken(ctx, db.InsertRefreshTokenParams{
		Jti:       rt.JTI,
		UserID:    rt.UserID,
		ExpiresAt: pgtype.Timestamptz{Time: rt.ExpiresAt, Valid: true},
	})
}

func (r *refreshTokenRepository) MarkRevoked(ctx context.Context, jti string) error {
	return r.queries.MarkRefreshTokenRevoked(ctx, jti)
}

func (r *refreshTokenRepository) IsValid(ctx context.Context, jti string) (bool, error) {
	return r.queries.IsRefreshTokenValid(ctx, jti)
}

func (r *refreshTokenRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	return r.queries.RevokeAllRefreshTokensForUser(ctx, userID)
}
