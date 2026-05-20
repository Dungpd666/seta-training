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

-- name: ListAssets :many 
SELECT asset_id, owner_id, parent_id, type, title, content, created_at
FROM assets 
WHERE owner_id = $1 
ORDER BY created_at DESC 
LIMIT $2 
OFFSET $3;

-- name: CountAssetsByOwner :one 
SELECT COUNT(*) FROM assets WHERE owner_id = $1;

-- name: GetAssetDepth :one
WITH RECURSIVE ancestors AS (
    SELECT parent_id, 0 AS depth FROM assets WHERE assets.asset_id = $1
    UNION ALL
    SELECT a.parent_id, anc.depth + 1
    FROM assets a
    JOIN ancestors anc ON a.asset_id = anc.parent_id
    WHERE anc.parent_id IS NOT NULL
)
SELECT COALESCE(MAX(depth), 0)::int AS depth FROM ancestors;
