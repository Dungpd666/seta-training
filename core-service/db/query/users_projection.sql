-- name: UpsertUserProjection :exec 
INSERT INTO users_projection (user_id, username, email, role, updated_at)
VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (user_id) DO UPDATE
    SET username = EXCLUDED.username,
        email = EXCLUDED.email,
        role = EXCLUDED.role,
        deleted_at = NULL,
        updated_at = NOW();

-- name: SoftDeleteUserProjection :exec
UPDATE users_projection
SET deleted_at = NOW()
WHERE user_id = $1;
