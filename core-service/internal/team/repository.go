package team

import (
	"context"

	"github.com/dungpd/seta/core-service/internal/db"
)

type ProjectionRepository interface {
	Upsert(ctx context.Context, u UserProjection) error
	SoftDelete(ctx context.Context, userID string) error
}

type TeamRepository interface {
	Create(ctx context.Context, teamName, createdBy string) (*Team, error)
	AddMember(ctx context.Context, teamID, userID string, role string) error
	RemoveMember(ctx context.Context, teamID, userID string) error
	GetMemberRole(ctx context.Context, teamID, userID string) (string, error)
	GetByID(ctx context.Context, teamID string) (*Team, error)
}

type projectionRepo struct {
	q *db.Queries
}

type teamRepo struct {
	q *db.Queries
}

func NewProjectionRepository(q *db.Queries) ProjectionRepository {
	return &projectionRepo{q: q}
}

func NewTeamRepository(q *db.Queries) TeamRepository {
	return &teamRepo{q: q}
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

func (r *teamRepo) Create(ctx context.Context, teamName, createdBy string) (*Team, error) {
	row, err := r.q.CreateTeam(ctx, db.CreateTeamParams{
		TeamName:  teamName,
		CreatedBy: createdBy,
	})
	if err != nil {
		return nil, err
	}
	return &Team{
		TeamID:    row.TeamID,
		TeamName:  row.TeamName,
		CreatedBy: row.CreatedBy,
	}, nil
}

func (r *teamRepo) AddMember(ctx context.Context, teamID, userID string, role string) error {
	return r.q.AddTeamMember(ctx, db.AddTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
		Role:   role,
	})
}

func (r *teamRepo) RemoveMember(ctx context.Context, teamID, userID string) error {
	return r.q.RemoveTeamMember(ctx, db.RemoveTeamMemberParams{
		TeamID: teamID,
		UserID: userID,
	})
}

func (r *teamRepo) GetMemberRole(ctx context.Context, teamID, userID string) (string, error) {
	return r.q.GetMemberRole(ctx, db.GetMemberRoleParams{
		TeamID: teamID,
		UserID: userID,
	})
}

func (r *teamRepo) GetByID(ctx context.Context, teamID string) (*Team, error) {
	row, err := r.q.GetTeamByID(ctx, teamID)
	if err != nil {
		return nil, err
	}
	return &Team{
		TeamID:    row.TeamID,
		TeamName:  row.TeamName,
		CreatedBy: row.CreatedBy,
	}, nil
}
