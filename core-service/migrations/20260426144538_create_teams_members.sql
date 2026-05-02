-- +goose Up
CREATE TABLE IF NOT EXISTS team_members (
    team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users_projection(user_id),
    role TEXT NOT NULL CHECK (role IN ('manager', 'member')),
    PRIMARY KEY (team_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS team_members;
