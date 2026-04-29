-- +goose Up
CREATE TABLE IF NOT EXISTS asset_acl (
    asset_id     UUID NOT NULL REFERENCES assets(asset_id) ON DELETE CASCADE,
    user_id      UUID NOT NULL,
    access_level VARCHAR(5) NOT NULL CHECK (access_level IN ('read', 'write')),
    PRIMARY KEY (asset_id, user_id)
);

-- +goose Down
DROP TABLE asset_acl;
