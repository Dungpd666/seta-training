package repository

import (
	"context"
	"time"

	"github.com/dungpd/seta/auth-service/internal/db"
	"github.com/dungpd/seta/auth-service/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	queries *db.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{queries: db.New(pool)}
}

func (r *UserRepository) Create(user *domain.User) error {
	created, err := r.queries.CreateUser(context.Background(), db.CreateUserParams{
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
	})
	if err != nil {
		return err
	}

	user.UserID = created.UserID
	user.Username = created.Username
	user.Email = created.Email
	user.PasswordHash = created.PasswordHash
	user.Role = created.Role
	user.CreatedAt = toTime(created.CreatedAt)
	return nil
}

func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	user, err := r.queries.GetUserByEmail(context.Background(), email)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &domain.User{
		UserID:       user.UserID,
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
		CreatedAt:    toTime(user.CreatedAt),
	}, nil
}

func (r *UserRepository) FindAll() ([]domain.User, error) {
	users, err := r.queries.ListUsers(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]domain.User, len(users))
	for i, user := range users {
		result[i] = domain.User{
			UserID:       user.UserID,
			Username:     user.Username,
			Email:        user.Email,
			PasswordHash: user.PasswordHash,
			Role:         user.Role,
			CreatedAt:    toTime(user.CreatedAt),
		}
	}
	return result, nil
}

func toTime(value pgtype.Timestamptz) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}
