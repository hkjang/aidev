CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS teams (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL UNIQUE,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username citext NOT NULL UNIQUE,
  display_name text NOT NULL,
  email citext,
  password_hash text,
  role text NOT NULL DEFAULT 'user' CHECK (role IN ('user','team_lead','admin')),
  team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
  oidc_subject text UNIQUE,
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash bytea NOT NULL UNIQUE,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS oauth_states (
  state_hash bytea PRIMARY KEY,
  return_to text NOT NULL DEFAULT '/',
  expires_at timestamptz NOT NULL
);

CREATE TABLE IF NOT EXISTS app_settings (
  key text PRIMARY KEY,
  value jsonb NOT NULL,
  encrypted boolean NOT NULL DEFAULT false,
  updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS spaces (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  color text NOT NULL DEFAULT '#FFF0A8',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS spaces_owner_idx ON spaces(owner_id);

CREATE TABLE IF NOT EXISTS space_members (
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  permission text NOT NULL DEFAULT 'view' CHECK (permission IN ('view','edit','manage')),
  PRIMARY KEY(space_id,user_id)
);

CREATE TABLE IF NOT EXISTS notes (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content text NOT NULL DEFAULT '',
  title text NOT NULL DEFAULT '',
  color text NOT NULL DEFAULT 'yellow',
  kind text NOT NULL DEFAULT 'thought',
  source text NOT NULL DEFAULT 'user' CHECK (source IN ('user','dream','api','mcp')),
  x double precision NOT NULL DEFAULT 0,
  y double precision NOT NULL DEFAULT 0,
  width double precision NOT NULL DEFAULT 240,
  height double precision NOT NULL DEFAULT 160,
  rotation double precision NOT NULL DEFAULT 0,
  version integer NOT NULL DEFAULT 1,
  archived boolean NOT NULL DEFAULT false,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS notes_space_idx ON notes(space_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS notes_author_updated_idx ON notes(author_id,updated_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS note_edges (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  source_note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  target_note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  relation text NOT NULL DEFAULT 'related',
  created_by uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(source_note_id,target_note_id)
);

CREATE TABLE IF NOT EXISTS user_preferences (
  user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  dream_enabled boolean NOT NULL DEFAULT true,
  dream_frequency text NOT NULL DEFAULT 'daily',
  dream_style text NOT NULL DEFAULT 'auto',
  dream_notifications boolean NOT NULL DEFAULT false,
  include_old_notes boolean NOT NULL DEFAULT true,
  dream_pause_until timestamptz,
  theme text NOT NULL DEFAULT 'light',
  locale text NOT NULL DEFAULT 'ko',
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  prefix text NOT NULL,
  secret_hash bytea NOT NULL UNIQUE,
  scopes text[] NOT NULL DEFAULT ARRAY['notes:read'],
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','overlap','revoked','expired')),
  expires_at timestamptz,
  overlap_until timestamptz,
  replaced_by uuid REFERENCES api_keys(id) ON DELETE SET NULL,
  last_used_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz
);
CREATE INDEX IF NOT EXISTS api_keys_user_idx ON api_keys(user_id,created_at DESC);

CREATE TABLE IF NOT EXISTS approval_requests (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  requester_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  team_id uuid REFERENCES teams(id) ON DELETE SET NULL,
  resource_type text NOT NULL,
  resource_id uuid NOT NULL,
  action text NOT NULL,
  status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected','cancelled')),
  comment text NOT NULL DEFAULT '',
  reviewer_id uuid REFERENCES users(id) ON DELETE SET NULL,
  reviewed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dream_notes (
  dream_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  note_id uuid UNIQUE REFERENCES notes(id) ON DELETE CASCADE,
  dream_type text NOT NULL,
  generated_at timestamptz NOT NULL DEFAULT now(),
  exposed_at timestamptz,
  model text NOT NULL DEFAULT '',
  prompt_version text NOT NULL DEFAULT 'dream-v1',
  quality_score double precision NOT NULL DEFAULT 0,
  status text NOT NULL DEFAULT 'created' CHECK(status IN ('created','exposed','kept','deleted')),
  source_note_count integer NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS dream_sources (
  dream_id uuid NOT NULL REFERENCES dream_notes(dream_id) ON DELETE CASCADE,
  source_note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  similarity_score double precision NOT NULL DEFAULT 0,
  rank integer NOT NULL,
  PRIMARY KEY(dream_id,source_note_id)
);

CREATE TABLE IF NOT EXISTS dream_feedback (
  id bigserial PRIMARY KEY,
  dream_id uuid NOT NULL REFERENCES dream_notes(dream_id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  action text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS dream_jobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','completed','skipped','failed')),
  attempt integer NOT NULL DEFAULT 0,
  scheduled_for date NOT NULL,
  error text NOT NULL DEFAULT '',
  started_at timestamptz,
  finished_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(user_id,scheduled_for)
);

CREATE TABLE IF NOT EXISTS ai_calls (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  dream_job_id uuid REFERENCES dream_jobs(id) ON DELETE SET NULL,
  model text NOT NULL,
  status text NOT NULL,
  input_tokens integer NOT NULL DEFAULT 0,
  output_tokens integer NOT NULL DEFAULT 0,
  cost_micros bigint NOT NULL DEFAULT 0,
  latency_ms integer NOT NULL DEFAULT 0,
  error text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id bigserial PRIMARY KEY,
  actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO app_settings(key,value) VALUES
('general', '{"service_name":"umm","public_url":"http://localhost:8080","session_hours":24,"timezone":"Asia/Seoul"}'::jsonb),
('oidc', '{"enabled":false,"issuer_url":"","client_id":"","client_secret":"","scopes":["openid","profile","email"],"admin_group":"umm-admin","team_lead_group":"umm-team-lead"}'::jsonb),
('security', '{"api_key_scopes":["notes:read","notes:write","spaces:read","dreams:read","approvals:write"],"default_key_days":90,"rotation_overlap_hours":24}'::jsonb),
('workflow', '{"enabled":false,"actions":["space_share","export"]}'::jsonb),
('dream', '{"enabled":false,"automatic":false,"schedule":"02:00","frequency":"daily","count":1,"min_notes":3,"context_days":7,"max_context_notes":20,"model":"","temperature":0.7,"token_limit":4096,"monthly_limit":30,"allow_user_disable":true,"notification":false,"quality_threshold":0.7,"quiet_mode":false}'::jsonb),
('ai_gateway', '{"base_url":"","api_key":"","timeout_seconds":45,"max_retries":2,"prompt_version":"dream-v1","input_cost_per_million":0,"output_cost_per_million":0,"log_prompt":false,"log_retention_days":90}'::jsonb)
ON CONFLICT (key) DO NOTHING;
