-- +goose Up
CREATE TABLE IF NOT EXISTS teams (
    team_id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    team_name TEXT NOT NULL,
    created_by TEXT NOT NULL REFERENCES users_projection(user_id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS teams;
