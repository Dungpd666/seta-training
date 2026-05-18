-- name: GetAssetACL :one
SELECT asset_id, user_id, access_level
FROM asset_acl
WHERE asset_id = $1 AND user_id = $2;

-- name: UpsertAssetACL :exec
INSERT INTO asset_acl (asset_id, user_id, access_level)
VALUES ($1, $2, $3)
ON CONFLICT (asset_id, user_id) DO UPDATE SET access_level = EXCLUDED.access_level;

-- name: DeleteAssetACLEntry :exec
DELETE FROM asset_acl WHERE asset_id = $1 AND user_id = $2;

-- name: GetDescendantIDs :many
WITH RECURSIVE descendants AS (
    SELECT b.asset_id, 1 AS depth FROM assets b WHERE b.parent_id = $1
    UNION ALL
    SELECT a.asset_id, d.depth + 1 FROM assets a
    JOIN descendants d ON a.parent_id = d.asset_id
    WHERE d.depth < 20
)
SELECT asset_id FROM descendants;

-- name: ListAssetACL :many
SELECT asset_id, user_id, access_level FROM asset_acl WHERE asset_id = $1;
