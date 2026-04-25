-- name: InsertRefreshToken :exec
INSERT INTO refresh_tokens (
    jti,
    user_id,
    expires_at
) VALUES (
    $1, $2, $3
);

-- name: MarkRefreshTokenRevoked :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE jti = $1;

-- name: IsRefreshTokenValid :one
SELECT EXISTS (
    SELECT 1
    FROM refresh_tokens
    WHERE jti = $1
      AND revoked = FALSE
      AND expires_at > NOW()
);

-- name: RevokeAllRefreshTokensForUser :exec
UPDATE refresh_tokens
SET revoked = TRUE
WHERE user_id = $1;
