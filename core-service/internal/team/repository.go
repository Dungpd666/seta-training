package team

import (
	"context"

	"github.com/dungpd/seta/core-service/internal/db"
)

type ProjectionRepository interface {
	Upsert(ctx context.Context, u UserProjection) error
	SoftDelete(ctx context.Context, userID string) error
}

type projectionRepo struct {
	q *db.Queries
}

func NewProjectionRepository(q *db.Queries) ProjectionRepository {
	return &projectionRepo{q: q}
}

func (r *projectionRepo) Upsert(ctx context.Context, u UserProjection) error {
	return r.q.UpsertUserProjection(ctx,
		db.UpsertUserProjectionParams{
			UserID:   u.UserID,
			Username: u.Username,
			Email:    u.Email,
			Role:     u.Role,
		})
}

func (r *projectionRepo) SoftDelete(ctx context.Context, userID string) error {
	return r.q.SoftDeleteUserProjection(ctx, userID)
}
