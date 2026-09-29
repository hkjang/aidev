CREATE TABLE IF NOT EXISTS schema_migrations (
  version integer PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS settings (
  key text PRIMARY KEY,
  value text NOT NULL DEFAULT '',
  secret boolean NOT NULL DEFAULT false,
  updated_at timestamptz NOT NULL DEFAULT now(),
  updated_by text
);

CREATE TABLE IF NOT EXISTS organizations (
  id text PRIMARY KEY,
  external_id text UNIQUE,
  name text NOT NULL,
  parent_id text REFERENCES organizations(id) ON DELETE SET NULL,
  color text NOT NULL DEFAULT '#2563EB',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS employees (
  id text PRIMARY KEY,
  employee_no text NOT NULL UNIQUE,
  name text NOT NULL,
  email text,
  organization_id text REFERENCES organizations(id) ON DELETE SET NULL,
  title text,
  position text,
  workplace text,
  profile_image_url text,
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','leave','retired')),
  attributes jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS users (
  id text PRIMARY KEY,
  username text NOT NULL UNIQUE,
  password_hash text,
  display_name text NOT NULL,
  email text,
  employee_id text REFERENCES employees(id) ON DELETE SET NULL,
  role text NOT NULL DEFAULT 'employee' CHECK (role IN ('employee','department_manager','seat_manager','system_admin')),
  source text NOT NULL DEFAULT 'local' CHECK (source IN ('local','oidc')),
  active boolean NOT NULL DEFAULT true,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  last_login_at timestamptz
);

CREATE TABLE IF NOT EXISTS sessions (
  token_hash bytea PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  csrf_token text NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  ip_address text,
  user_agent text
);
CREATE INDEX IF NOT EXISTS sessions_user_idx ON sessions(user_id);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS oidc_states (
  state_hash bytea PRIMARY KEY,
  nonce text NOT NULL,
  verifier text NOT NULL,
  return_to text NOT NULL DEFAULT '/',
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
-- prompt=none 으로 시작한 조용한 시도인지. 콜백이 login_required 를 받았을 때
-- 이것이 참이면 오류가 아니라 "세션 없음"이라는 평범한 대답으로 다룬다.
ALTER TABLE oidc_states ADD COLUMN IF NOT EXISTS silent boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS buildings (
  id text PRIMARY KEY,
  name text NOT NULL,
  code text NOT NULL UNIQUE,
  address text,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS floors (
  id text PRIMARY KEY,
  building_id text NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
  name text NOT NULL,
  code text NOT NULL,
  sort_order integer NOT NULL DEFAULT 0,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(building_id, code)
);

CREATE TABLE IF NOT EXISTS floor_maps (
  id text PRIMARY KEY,
  floor_id text NOT NULL REFERENCES floors(id) ON DELETE CASCADE,
  version text NOT NULL,
  file_name text NOT NULL,
  content_type text NOT NULL,
  file_data bytea NOT NULL,
  width integer,
  height integer,
  status text NOT NULL DEFAULT 'uploaded' CHECK (status IN ('uploaded','analyzing','review','published','archived','failed')),
  is_active boolean NOT NULL DEFAULT false,
  created_by text REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  published_at timestamptz,
  UNIQUE(floor_id, version)
);

-- PDF 도면도 좌석 오버레이를 그릴 수 있도록 래스터 미리보기와 그 픽셀 크기를 보관한다.
ALTER TABLE floor_maps ADD COLUMN IF NOT EXISTS preview_data bytea;
ALTER TABLE floor_maps ADD COLUMN IF NOT EXISTS preview_width integer;
ALTER TABLE floor_maps ADD COLUMN IF NOT EXISTS preview_height integer;
-- 도면별 좌석 격자 보정값: {"originX":..,"originY":..,"pitchX":..,"pitchY":..}
ALTER TABLE floor_maps ADD COLUMN IF NOT EXISTS grid jsonb NOT NULL DEFAULT '{}';

CREATE TABLE IF NOT EXISTS seats (
  id text PRIMARY KEY,
  floor_map_id text NOT NULL REFERENCES floor_maps(id) ON DELETE CASCADE,
  seat_no text NOT NULL,
  type text NOT NULL DEFAULT 'fixed' CHECK (type IN ('fixed','shared','unavailable','meeting_room','executive','utility')),
  status text NOT NULL DEFAULT 'available' CHECK (status IN ('available','assigned','unavailable')),
  x double precision NOT NULL CHECK (x >= 0 AND x <= 1),
  y double precision NOT NULL CHECK (y >= 0 AND y <= 1),
  width double precision NOT NULL CHECK (width > 0 AND width <= 1),
  height double precision NOT NULL CHECK (height > 0 AND height <= 1),
  CHECK (x + width <= 1),
  CHECK (y + height <= 1),
  rotation double precision NOT NULL DEFAULT 0,
  confidence double precision,
  organization_id text REFERENCES organizations(id) ON DELETE SET NULL,
  metadata jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(floor_map_id, seat_no)
);
CREATE INDEX IF NOT EXISTS seats_map_idx ON seats(floor_map_id);

CREATE TABLE IF NOT EXISTS seat_assignments (
  id text PRIMARY KEY,
  employee_id text NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
  seat_id text NOT NULL REFERENCES seats(id) ON DELETE CASCADE,
  assigned_at timestamptz NOT NULL DEFAULT now(),
  assigned_by text REFERENCES users(id) ON DELETE SET NULL,
  reason text,
  source text NOT NULL DEFAULT 'manual',
  ended_at timestamptz
);
CREATE UNIQUE INDEX IF NOT EXISTS active_assignment_employee_idx ON seat_assignments(employee_id) WHERE ended_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS active_assignment_seat_idx ON seat_assignments(seat_id) WHERE ended_at IS NULL;

CREATE TABLE IF NOT EXISTS seat_history (
  id text PRIMARY KEY,
  employee_id text REFERENCES employees(id) ON DELETE SET NULL,
  previous_seat_id text REFERENCES seats(id) ON DELETE SET NULL,
  new_seat_id text REFERENCES seats(id) ON DELETE SET NULL,
  changed_by text REFERENCES users(id) ON DELETE SET NULL,
  reason text,
  source text NOT NULL,
  changed_at timestamptz NOT NULL DEFAULT now(),
  details jsonb NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS seat_history_changed_idx ON seat_history(changed_at DESC);

CREATE TABLE IF NOT EXISTS analysis_jobs (
  id text PRIMARY KEY,
  floor_map_id text NOT NULL REFERENCES floor_maps(id) ON DELETE CASCADE,
  status text NOT NULL DEFAULT 'queued',
  engine text NOT NULL DEFAULT 'offline-cv-v1',
  confidence_threshold double precision NOT NULL DEFAULT 0.80,
  detected_count integer NOT NULL DEFAULT 0,
  review_count integer NOT NULL DEFAULT 0,
  error text,
  created_by text REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

-- 분석 엔진 진단 정보와 사용자에게 보여줄 경고를 함께 보관한다.
ALTER TABLE analysis_jobs ADD COLUMN IF NOT EXISTS details jsonb NOT NULL DEFAULT '{}';
ALTER TABLE analysis_jobs ADD COLUMN IF NOT EXISTS warnings text[] NOT NULL DEFAULT '{}';
CREATE INDEX IF NOT EXISTS analysis_jobs_map_idx ON analysis_jobs(floor_map_id, created_at DESC);

CREATE TABLE IF NOT EXISTS api_keys (
  id text PRIMARY KEY,
  user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL,
  prefix text NOT NULL,
  secret_hash bytea NOT NULL UNIQUE,
  scopes text[] NOT NULL DEFAULT ARRAY['read'],
  version integer NOT NULL DEFAULT 1,
  rotated_from text REFERENCES api_keys(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz,
  last_used_at timestamptz,
  revoked_at timestamptz,
  grace_until timestamptz
);
CREATE INDEX IF NOT EXISTS api_keys_user_idx ON api_keys(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS audit_logs (
  id bigserial PRIMARY KEY,
  actor_user_id text REFERENCES users(id) ON DELETE SET NULL,
  action text NOT NULL,
  resource_type text NOT NULL,
  resource_id text,
  ip_address text,
  details jsonb NOT NULL DEFAULT '{}',
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_logs_created_idx ON audit_logs(created_at DESC);

CREATE TABLE IF NOT EXISTS employee_sync_runs (
  id text PRIMARY KEY,
  status text NOT NULL,
  employee_count integer NOT NULL DEFAULT 0,
  organization_count integer NOT NULL DEFAULT 0,
  error text,
  started_at timestamptz NOT NULL DEFAULT now(),
  completed_at timestamptz
);

INSERT INTO settings(key, value, secret) VALUES
 ('general.service_name', 'SeatOn', false),
 ('general.company_name', '', false),
 ('general.default_locale', 'ko-KR', false),
 ('auth.local_enabled', 'true', false),
 ('oidc.enabled', 'false', false),
 ('oidc.issuer_url', '', false),
 ('oidc.client_id', '', false),
 ('oidc.client_secret', '', true),
 ('oidc.scopes', 'openid profile email groups', false),
 ('oidc.admin_group', '/seaton-admins', false),
 ('oidc.seat_manager_group', '/seaton-seat-managers', false),
 ('oidc.auto_provision', 'true', false),
 ('oidc.auto_login', 'false', false),
 ('mcp.oauth.enabled', 'false', false),
 ('mcp.oauth.resource', '', false),
 ('mcp.oauth.audience', '', false),
 ('mcp.oauth.scopes', 'read mcp', false),
 ('security.session_hours', '8', false),
 ('security.api_key_days', '90', false),
 ('security.rotation_grace_hours', '24', false),
 ('ai.confidence_threshold', '0.80', false),
 ('ai.auto_approve_threshold', '0.95', false),
 ('ai.engine', 'cv', false),
 ('ai.vlm_base_url', '', false),
 ('ai.vlm_model', 'qwen2.5-vl-7b-instruct', false),
 ('ai.vlm_api_key', '', true),
 ('ai.vlm_timeout_seconds', '120', false),
 ('ai.vlm_max_image_side', '1600', false),
 ('ai.vlm_max_seats', '400', false),
 ('ai.vlm_tiles', '1', false),
 ('ai.vlm_json_mode', 'true', false),
 ('ai.fusion_iou', '0.35', false),
 ('hr.sync_enabled', 'false', false),
 ('hr.api_url', '', false),
 ('hr.api_token', '', true),
 ('hr.schedule', '0 2 * * *', false),
 ('tracking.enabled', 'false', false),
 ('tracking.provider', 'momento', false),
 ('tracking.momento_url', '', false),
 ('tracking.momento_site_id', '', false),
 ('tracking.momento_proxy', 'true', false),
 ('tracking.measurement_id', '', false),
 ('tracking.matomo_url', '', false),
 ('tracking.matomo_site_id', '', false),
 ('tracking.custom_snippet', '', false),
 ('tracking.allowed_hosts', '', false),
 ('tracking.include_admin', 'false', false),
 ('tracking.placement', 'head', false)
ON CONFLICT (key) DO NOTHING;

INSERT INTO schema_migrations(version) VALUES (1) ON CONFLICT DO NOTHING;
