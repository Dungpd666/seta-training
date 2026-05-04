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
SELECT tm.role FROM team_members tm
JOIN users_projection up ON tm.user_id = up.user_id
WHERE tm.team_id = $1 AND tm.user_id = $2 AND up.deleted_at IS NULL;

-- name: GetUserProjectionByID :one
SELECT * FROM users_projection WHERE user_id = $1;

-- name: IsManagerOfMember :one
SELECT EXISTS (
    SELECT 1 FROM team_members tm_manager
    JOIN team_members tm_member ON tm_manager.team_id = tm_member.team_id
    JOIN users_projection up ON tm_member.user_id = up.user_id
    WHERE tm_manager.user_id = $1
    AND tm_manager.role = 'manager'
    AND tm_member.user_id = $2
    AND up.deleted_at IS NULL
);
