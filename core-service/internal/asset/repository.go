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
	ListACLByAsset(ctx context.Context, assetID string) ([]*AssetACL, error)
	UpsertACLEntry(ctx context.Context, assetID, userID, accessLevel string) error
	DeleteACLEntry(ctx context.Context, assetID, userID string) error
	GetDescendantIDs(ctx context.Context, assetID string) ([]string, error)
	IsManagerOfOwner(ctx context.Context, callerID, ownerID string) (bool, error)
	UserExists(ctx context.Context, userID string) (bool, error)
	List(ctx context.Context, ownerID string, limit, offset int32) ([]*Asset, error)
	CountByOwner(ctx context.Context, ownerID string) (int64, error)
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
	row, err := r.q.GetAssetACL(ctx, db.GetAssetACLParams{
		AssetID: assetID,
		UserID:  userID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &AssetACL{AssetID: row.AssetID, UserID: row.UserID, AccessLevel: row.AccessLevel}, nil
}

func (r *repo) ListACLByAsset(ctx context.Context, assetID string) ([]*AssetACL, error) {
	rows, err := r.q.ListAssetACL(ctx, assetID)
	if err != nil {
		return nil, err
	}
	acls := make([]*AssetACL, len(rows))
	for i, row := range rows {
		acls[i] = &AssetACL{AssetID: row.AssetID, UserID: row.UserID, AccessLevel: row.AccessLevel}
	}
	return acls, nil
}

func (r *repo) UpsertACLEntry(ctx context.Context, assetID, userID, accessLevel string) error {
	return r.q.UpsertAssetACL(ctx, db.UpsertAssetACLParams{
		AssetID:     assetID,
		UserID:      userID,
		AccessLevel: accessLevel,
	})
}

func (r *repo) DeleteACLEntry(ctx context.Context, assetID, userID string) error {
	return r.q.DeleteAssetACLEntry(ctx, db.DeleteAssetACLEntryParams{
		AssetID: assetID,
		UserID:  userID,
	})
}

func (r *repo) GetDescendantIDs(ctx context.Context, assetID string) ([]string, error) {
	return r.q.GetDescendantIDs(ctx, pgtype.Text{String: assetID, Valid: true})
}

func (r *repo) IsManagerOfOwner(ctx context.Context, callerID, ownerID string) (bool, error) {
	return r.q.IsManagerOfMember(ctx, db.IsManagerOfMemberParams{
		ManagerID: callerID,
		MemberID:  ownerID,
	})
}

func (r *repo) UserExists(ctx context.Context, userID string) (bool, error) {
	_, err := r.q.GetUserProjectionByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *repo) List(ctx context.Context, ownerID string, limit, offset int32) ([]*Asset, error) {
	rows, err := r.q.ListAssets(ctx, db.ListAssetsParams{
		OwnerID: ownerID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}
	assets := make([]*Asset, len(rows))
	for i, row := range rows {
		assets[i] = rowToAsset(row)
	}
	return assets, nil
}

func (r *repo) CountByOwner(ctx context.Context, ownerID string) (int64, error) {
	return r.q.CountAssetsByOwner(ctx, ownerID)
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
