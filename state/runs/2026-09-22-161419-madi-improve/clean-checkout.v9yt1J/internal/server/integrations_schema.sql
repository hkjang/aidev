CREATE TABLE IF NOT EXISTS api_keys (
 id uuid PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL,
 prefix text NOT NULL,
 token_hash text NOT NULL UNIQUE,
 scopes text[] NOT NULL,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL,
 last_used_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),
 revoked_at timestamptz,
 ip_allowlist text[] NOT NULL DEFAULT '{}',
 rate_limit integer NOT NULL DEFAULT 120 CHECK (rate_limit BETWEEN 1 AND 10000),
 rate_window timestamptz NOT NULL DEFAULT now(),
 rate_count integer NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS api_keys_user_idx ON api_keys(user_id, created_at DESC);
CREATE TABLE IF NOT EXISTS oidc_attempts (
 state_hash text PRIMARY KEY,
 browser_hash text NOT NULL,
 nonce text NOT NULL,
 verifier text NOT NULL,
 issuer text NOT NULL,
 client_id text NOT NULL,
 redirect_uri text NOT NULL,
 expires_at timestamptz NOT NULL
);
ALTER TABLE oidc_attempts ADD COLUMN IF NOT EXISTS silent boolean NOT NULL DEFAULT false;
ALTER TABLE oidc_attempts ADD COLUMN IF NOT EXISTS return_to text NOT NULL DEFAULT '/app';
