package asset

import (
	"context"
	"errors"

	"github.com/dungpd/seta/core-service/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository interface {
	Create(ctx context.Context, ownerID string, parentID *string, assetType, title string, content *string) (*Asset, error)
	GetByID(ctx context.Context, assetID string) (*Asset, error)
	Update(ctx context.Context, assetID, title string, content *string) (*Asset, error)
	Delete(ctx context.Context, assetID string) error
	GetACLEntry(ctx context.Context, assetID, userID string) (*AssetACL, error)
}

type repo struct {
	q *db.Queries
}

func NewRepository(q *db.Queries) Repository {
	return &repo{q: q}
}

func (r *repo) Create(ctx context.Context, ownerID string, parentID *string, assetType, title string, content *string) (*Asset, error) {
	var pgParentID pgtype.Text
	if parentID != nil {
		if err := pgParentID.Scan(*parentID); err != nil {
			return nil, err
		}
	}
	var pgContent pgtype.Text
	if content != nil {
		pgContent = pgtype.Text{String: *content, Valid: true}
	}

	row, err := r.q.CreateAsset(ctx, db.CreateAssetParams{
		OwnerID:  ownerID,
		ParentID: pgParentID,
		Type:     assetType,
		Title:    title,
		Content:  pgContent,
	})
	if err != nil {
		return nil, err
	}
	return rowToAsset(row), nil
}

func (r *repo) GetByID(ctx context.Context, assetID string) (*Asset, error) {
	row, err := r.q.GetAssetByID(ctx, assetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return rowToAsset(row), nil
}

func (r *repo) Update(ctx context.Context, assetID, title string, content *string) (*Asset, error) {
	var pgContent pgtype.Text
	if content != nil {
		pgContent = pgtype.Text{String: *content, Valid: true}
	}
	row, err := r.q.UpdateAsset(ctx, db.UpdateAssetParams{
		AssetID: assetID,
		Title:   title,
		Content: pgContent,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return rowToAsset(row), nil
}

func (r *repo) Delete(ctx context.Context, assetID string) error {
	return r.q.DeleteAsset(ctx, assetID)
}

func (r *repo) GetACLEntry(ctx context.Context, assetID, userID string) (*AssetACL, error) {
	row, err := r.q.GetAssetACL(ctx, assetID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &AssetACL{AssetID: row.AssetID, UserID: row.UserID, AccessLevel: row.AccessLevel}, nil
}

func rowToAsset(row db.Asset) *Asset {
	a := &Asset{
		AssetID:   row.AssetID,
		OwnerID:   row.OwnerID,
		Type:      row.Type,
		Title:     row.Title,
		CreatedAt: row.CreatedAt.Time,
	}
	if row.ParentID.Valid {
		a.ParentID = &row.ParentID.String
	}
	if row.Content.Valid {
		a.Content = &row.Content.String
	}
	return a
}
