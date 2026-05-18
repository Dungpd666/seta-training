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

-- name: ListUsersFromStart :many 
SELECT user_id, username, email, password_hash, role, created_at
FROM users 
ORDER BY user_id 
LIMIT $1;

-- name: ListUsersWithCursor :many 
SELECT user_id, username, email, password_hash, role, created_at
FROM users 
WHERE user_id > $1
ORDER BY user_id
LIMIT $2;

-- name: CountUsers :one 
SELECT COUNT(*) FROM users;
