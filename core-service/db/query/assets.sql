-- name: CreateAsset :one 
INSERT INTO assets (owner_id, parent_id, type, title, content) 
VALUES ($1, $2, $3, $4, $5) 
RETURNING *;

-- name: GetAssetByID :one
SELECT * FROM assets WHERE asset_id = $1;

-- name: UpdateAsset :one
UPDATE assets
SET title = $2, content = $3
WHERE asset_id = $1
RETURNING *;

-- name: DeleteAsset :exec
DELETE FROM assets WHERE asset_id = $1;
