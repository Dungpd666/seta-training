-- name: CreateTeam :one 
INSERT INTO teams (team_id, team_name, created_by)
VALUES (gen_random_uuid()::text, $1, $2)
RETURNING *;

-- name: GetTeamByID :one
SELECT * FROM teams WHERE team_id = $1;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, role)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, user_id) DO UPDATE
    SET role = EXCLUDED.role;

-- name: RemoveTeamMember :exec
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: GetMemberRole :one 
SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: GetUserProjectionByID :one
SELECT * FROM users_projection WHERE user_id = $1;
