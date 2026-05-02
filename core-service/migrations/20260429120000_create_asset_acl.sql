-- +goose Up
CREATE TABLE IF NOT EXISTS asset_acl (
    asset_id     TEXT NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL REFERENCES users_projection(user_id) ON DELETE CASCADE,
    access_level TEXT NOT NULL CHECK (access_level IN ('read', 'write')),
    PRIMARY KEY (asset_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS asset_acl;
