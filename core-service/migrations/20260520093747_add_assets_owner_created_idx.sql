-- +goose Up
CREATE INDEX idx_assets_owner_created ON assets (owner_id, created_at DESC);
-- +goose Down
DROP INDEX IF EXISTS idx_assets_owner_created;
