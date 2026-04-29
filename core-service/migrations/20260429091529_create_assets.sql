-- +goose Up
CREATE TABLE IF NOT EXISTS assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id UUID NOT NULL,
    parent_id UUID REFERENCES assets(asset_id) ON DELETE CASCADE,
    type VARCHAR(10) NOT NULL CHECK (type IN ('folder', 'note')),
    title TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- +goose Down 
DROP TABLE assets; 
