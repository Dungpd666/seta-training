-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    user_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL CHECK (role IN ('manager', 'member')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);

-- +goose Down
DROP TABLE IF EXISTS users;
