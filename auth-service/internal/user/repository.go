package user

import (
	"context"
	"errors"
	"time"

	"github.com/dungpd/seta/auth-service/internal/db"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgUniqueViolation = "23505"

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindPage(ctx context.Context, cursor string, limit int32) ([]User, error)
	Count(ctx context.Context) (int64, error)
}

type repository struct {
	queries *db.Queries
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &repository{queries: db.New(pool)}
}

func (r *repository) Create(ctx context.Context, u *User) error {
	created, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Username:     u.Username,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return ErrEmailInUse
		}
		return err
	}

	u.UserID = created.UserID
	u.Username = created.Username
	u.Email = created.Email
	u.PasswordHash = created.PasswordHash
	u.Role = created.Role
	u.CreatedAt = toTime(created.CreatedAt)
	return nil
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	return &User{
		UserID:       row.UserID,
		Username:     row.Username,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		CreatedAt:    toTime(row.CreatedAt),
	}, nil
}

func (r *repository) FindPage(ctx context.Context, cursor string, limit int32) ([]User, error) {
	var rows []db.User
	var err error
	if cursor == "" {
		rows, err = r.queries.ListUsersFromStart(ctx, limit)
	} else {
		rows, err = r.queries.ListUsersWithCursor(ctx, db.ListUsersWithCursorParams{
			UserID: cursor,
			Limit:  limit,
		})
	}
	if err != nil {
		return nil, err
	}

	result := make([]User, len(rows))
	for i, row := range rows {
		result[i] = User{
			UserID:       row.UserID,
			Username:     row.Username,
			Email:        row.Email,
			PasswordHash: row.PasswordHash,
			Role:         row.Role,
			CreatedAt:    toTime(row.CreatedAt),
		}
	}
	return result, nil
}

func (r *repository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountUsers(ctx)
}

func toTime(value pgtype.Timestamptz) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}
