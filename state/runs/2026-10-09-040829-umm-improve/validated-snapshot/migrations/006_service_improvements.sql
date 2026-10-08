ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS onboarding_completed_at timestamptz,
  ADD COLUMN IF NOT EXISTS review_digest boolean NOT NULL DEFAULT true;

ALTER TABLE notifications
  ADD COLUMN IF NOT EXISTS resource_space_id uuid REFERENCES spaces(id) ON DELETE CASCADE,
  ADD COLUMN IF NOT EXISTS metadata jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS note_reviews (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  reviewed_at timestamptz,
  review_at timestamptz,
  pinned boolean NOT NULL DEFAULT false,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(user_id, note_id)
);
CREATE INDEX IF NOT EXISTS note_reviews_queue_idx
  ON note_reviews(user_id, pinned DESC, review_at, reviewed_at);

CREATE INDEX IF NOT EXISTS notifications_user_created_idx
  ON notifications(user_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS note_comments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  author_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  parent_id uuid REFERENCES note_comments(id) ON DELETE CASCADE,
  body text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 4000),
  resolved_at timestamptz,
  resolved_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS note_comments_note_created_idx
  ON note_comments(note_id, created_at, id) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS comment_mentions (
  comment_id uuid NOT NULL REFERENCES note_comments(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY(comment_id, user_id)
);

CREATE TABLE IF NOT EXISTS product_events (
  id bigserial PRIMARY KEY,
  user_id uuid REFERENCES users(id) ON DELETE SET NULL,
  event_name text NOT NULL,
  resource_type text NOT NULL DEFAULT '',
  resource_id uuid,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS product_events_name_created_idx
  ON product_events(event_name, created_at DESC);

CREATE TABLE IF NOT EXISTS idempotency_records (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  idempotency_key text NOT NULL CHECK (char_length(idempotency_key) BETWEEN 8 AND 128),
  method text NOT NULL,
  path text NOT NULL,
  state text NOT NULL DEFAULT 'pending' CHECK (state IN ('pending','completed')),
  response_status integer CHECK (response_status IS NULL OR response_status BETWEEN 200 AND 299),
  response_body jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL DEFAULT now() + interval '24 hours',
  PRIMARY KEY(user_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idempotency_expiry_idx ON idempotency_records(expires_at);

CREATE TABLE IF NOT EXISTS ai_eval_cases (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
  dream_type text NOT NULL DEFAULT 'connection',
  input_notes jsonb NOT NULL DEFAULT '[]'::jsonb,
  expected_terms text[] NOT NULL DEFAULT '{}',
  forbidden_terms text[] NOT NULL DEFAULT '{}',
  active boolean NOT NULL DEFAULT true,
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_eval_runs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  case_id uuid NOT NULL REFERENCES ai_eval_cases(id) ON DELETE CASCADE,
  model text NOT NULL DEFAULT '',
  prompt_version text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('passed','failed','error')),
  content text NOT NULL DEFAULT '',
  score double precision NOT NULL DEFAULT 0 CHECK (score BETWEEN 0 AND 1),
  details jsonb NOT NULL DEFAULT '{}'::jsonb,
  input_tokens integer NOT NULL DEFAULT 0,
  output_tokens integer NOT NULL DEFAULT 0,
  latency_ms integer NOT NULL DEFAULT 0,
  error text NOT NULL DEFAULT '',
  created_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ai_eval_runs_case_created_idx
  ON ai_eval_runs(case_id, created_at DESC);

CREATE TABLE IF NOT EXISTS webhook_subscriptions (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 200),
  url text NOT NULL CHECK (char_length(url) BETWEEN 1 AND 2048),
  secret_ciphertext text NOT NULL,
  events text[] NOT NULL DEFAULT '{}',
  active boolean NOT NULL DEFAULT true,
  failure_count integer NOT NULL DEFAULT 0,
  last_delivered_at timestamptz,
  last_error text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS webhook_subscriptions_owner_idx
  ON webhook_subscriptions(owner_id, created_at DESC);

CREATE TABLE IF NOT EXISTS webhook_deliveries (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  subscription_id uuid NOT NULL REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
  event_id uuid NOT NULL,
  event_type text NOT NULL,
  payload jsonb NOT NULL,
  status text NOT NULL DEFAULT 'queued' CHECK (status IN ('queued','processing','delivered','failed')),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  response_status integer,
  error text NOT NULL DEFAULT '',
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  claimed_at timestamptz,
  attempted_at timestamptz NOT NULL DEFAULT now(),
  delivered_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(subscription_id,event_id)
);
CREATE INDEX IF NOT EXISTS webhook_deliveries_subscription_idx
  ON webhook_deliveries(subscription_id, attempted_at DESC);
CREATE INDEX IF NOT EXISTS webhook_deliveries_queue_idx
  ON webhook_deliveries(status,next_attempt_at,created_at)
  WHERE status IN ('queued','processing');
CREATE INDEX IF NOT EXISTS webhook_deliveries_terminal_idx
  ON webhook_deliveries(attempted_at)
  WHERE status IN ('delivered','failed');

UPDATE app_settings
SET value = jsonb_set(
  value,
  '{api_key_scopes}',
  COALESCE(value->'api_key_scopes', '[]'::jsonb) || '["webhooks:write"]'::jsonb
)
WHERE key = 'security'
  AND NOT (COALESCE(value->'api_key_scopes', '[]'::jsonb) ? 'webhooks:write');

UPDATE app_settings
SET value = jsonb_set(
  value,
  '{api_key_scopes}',
  COALESCE(value->'api_key_scopes', '[]'::jsonb) || '["metrics:read"]'::jsonb
)
WHERE key = 'security'
  AND NOT (COALESCE(value->'api_key_scopes', '[]'::jsonb) ? 'metrics:read');
