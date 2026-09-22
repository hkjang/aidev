CREATE TABLE IF NOT EXISTS search_history_preferences (
 user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 enabled boolean NOT NULL DEFAULT false,retention_days integer NOT NULL DEFAULT 30 CHECK(retention_days BETWEEN 7 AND 365),
 revision integer NOT NULL DEFAULT 1,updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS search_history_entries (
 id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 query text NOT NULL,filters jsonb NOT NULL DEFAULT '{}',query_hash text NOT NULL,day date NOT NULL,
 searches integer NOT NULL DEFAULT 1 CHECK(searches BETWEEN 1 AND 10000),
 zero_results integer NOT NULL DEFAULT 0 CHECK(zero_results BETWEEN 0 AND 10000),
 last_count integer NOT NULL CHECK(last_count BETWEEN 0 AND 100),revision integer NOT NULL DEFAULT 1,
 first_seen timestamptz NOT NULL DEFAULT now(),last_seen timestamptz NOT NULL DEFAULT now(),expires_at timestamptz NOT NULL,
 UNIQUE(user_id,workspace_id,day,query_hash)
);
CREATE INDEX IF NOT EXISTS search_history_owner_idx ON search_history_entries(user_id,workspace_id,last_seen DESC);
CREATE INDEX IF NOT EXISTS search_history_expiry_idx ON search_history_entries(expires_at);
