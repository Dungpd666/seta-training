  CREATE TABLE users_projection (
      user_id    TEXT PRIMARY KEY,
      username   TEXT NOT NULL,
      email      TEXT NOT NULL UNIQUE,
      role       TEXT NOT NULL,
      deleted_at TIMESTAMPTZ,
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );

  CREATE TABLE teams (
      team_id    TEXT PRIMARY KEY DEFAULT
  gen_random_uuid()::text,
      team_name  TEXT NOT NULL,
      created_by TEXT NOT NULL REFERENCES
  users_projection(user_id),
      updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
  );

  CREATE TABLE team_members (
      team_id TEXT NOT NULL REFERENCES teams(team_id) ON DELETE
  CASCADE,
      user_id TEXT NOT NULL REFERENCES
  users_projection(user_id),
      role    TEXT NOT NULL,
      PRIMARY KEY (team_id, user_id)
  );

  CREATE TABLE assets (
    asset_id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    owner_id TEXT NOT NULL,
    parent_id TEXT REFERENCES assets(asset_id) ON DELETE CASCADE,
    type VARCHAR(10) NOT NULL CHECK (type IN ('folder', 'note')),
    title TEXT NOT NULL,
    content TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

