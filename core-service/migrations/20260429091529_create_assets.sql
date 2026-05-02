-- +goose Up
CREATE TABLE IF NOT EXISTS assets (
    asset_id  TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    owner_id  TEXT NOT NULL REFERENCES users_projection(user_id),
    parent_id TEXT REFERENCES assets(asset_id) ON DELETE CASCADE,
    type      VARCHAR(10) NOT NULL CHECK (type IN ('folder', 'note')),
    title     TEXT NOT NULL,
    content   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS assets CASCADE;
