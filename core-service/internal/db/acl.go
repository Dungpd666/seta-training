package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type AssetACL struct {
	AssetID     string
	UserID      string
	AccessLevel string
}

const getAssetACL = `
SELECT asset_id, user_id, access_level
FROM asset_acl
WHERE asset_id = $1 AND user_id = $2
`

func (q *Queries) GetAssetACL(ctx context.Context, assetID, userID string) (AssetACL, error) {
	row := q.db.QueryRow(ctx, getAssetACL, assetID, userID)
	var a AssetACL
	err := row.Scan(&a.AssetID, &a.UserID, &a.AccessLevel)
	if err != nil {
		return AssetACL{}, err
	}
	return a, nil
}

const upsertAssetACL = `
INSERT INTO asset_acl (asset_id, user_id, access_level)
VALUES ($1, $2, $3)
ON CONFLICT (asset_id, user_id) DO UPDATE SET access_level = EXCLUDED.access_level
`

type UpsertAssetACLParams struct {
	AssetID     string
	UserID      string
	AccessLevel string
}

func (q *Queries) UpsertAssetACL(ctx context.Context, arg UpsertAssetACLParams) error {
	_, err := q.db.Exec(ctx, upsertAssetACL, arg.AssetID, arg.UserID, arg.AccessLevel)
	return err
}

const deleteAssetACLEntry = `DELETE FROM asset_acl WHERE asset_id = $1 AND user_id = $2`

func (q *Queries) DeleteAssetACLEntry(ctx context.Context, assetID, userID string) error {
	_, err := q.db.Exec(ctx, deleteAssetACLEntry, assetID, userID)
	return err
}

var ErrNoRows = pgx.ErrNoRows
