CREATE TABLE IF NOT EXISTS identity_links (
 id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 provider text NOT NULL CHECK(provider IN ('oidc','ldap','saml','scim')),issuer text NOT NULL,subject text NOT NULL,
 groups jsonb NOT NULL DEFAULT '[]',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(provider,issuer,subject),UNIQUE(provider,issuer,user_id)
);
CREATE TABLE IF NOT EXISTS identity_grants (
 provider text NOT NULL,issuer text NOT NULL,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,role text NOT NULL CHECK(role IN ('admin','editor','commenter','viewer')),
 PRIMARY KEY(provider,issuer,user_id,workspace_id)
);
CREATE TABLE IF NOT EXISTS identity_managed_members (
 workspace_id uuid NOT NULL,user_id uuid NOT NULL,role text NOT NULL,
 PRIMARY KEY(workspace_id,user_id),
 FOREIGN KEY(workspace_id,user_id) REFERENCES workspace_members(workspace_id,user_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS saml_authn_requests (
 request_id text PRIMARY KEY,state_hash text NOT NULL UNIQUE,browser_hash text NOT NULL,configuration_hash text NOT NULL,
 expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS saml_authn_expiry_idx ON saml_authn_requests(expires_at);
CREATE TABLE IF NOT EXISTS saml_assertion_replays (
 issuer text NOT NULL,assertion_hash text NOT NULL,expires_at timestamptz NOT NULL,
 PRIMARY KEY(issuer,assertion_hash)
);
CREATE INDEX IF NOT EXISTS saml_replay_expiry_idx ON saml_assertion_replays(expires_at);
