-- name: CreateUser :one
INSERT INTO users (
    username,
    email,
    password_hash,
    role
) VALUES (
    $1, $2, $3, $4
)
RETURNING user_id, username, email, password_hash, role, created_at;

-- name: GetUserByEmail :one
SELECT user_id, username, email, password_hash, role, created_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: ListUsers :many
SELECT user_id, username, email, password_hash, role, created_at
FROM users;
