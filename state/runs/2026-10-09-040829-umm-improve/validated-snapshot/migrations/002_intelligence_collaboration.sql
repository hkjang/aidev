CREATE TABLE IF NOT EXISTS note_embeddings (
  note_id uuid PRIMARY KEY REFERENCES notes(id) ON DELETE CASCADE,
  algorithm text NOT NULL DEFAULT 'umm-local-chargram-v1',
  dimensions integer NOT NULL,
  vector real[] NOT NULL,
  content_version integer NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS note_revisions (
  id bigserial PRIMARY KEY,
  note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  version integer NOT NULL,
  content text NOT NULL,
  title text NOT NULL,
  color text NOT NULL,
  kind text NOT NULL,
  x double precision NOT NULL,
  y double precision NOT NULL,
  width double precision NOT NULL,
  height double precision NOT NULL,
  rotation double precision NOT NULL,
  changed_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE(note_id,version)
);

CREATE TABLE IF NOT EXISTS dream_preferences (
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dream_type text NOT NULL,
  score double precision NOT NULL DEFAULT 0.5 CHECK(score >= 0 AND score <= 1),
  sample_count integer NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(user_id,dream_type)
);

CREATE TABLE IF NOT EXISTS notifications (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind text NOT NULL,
  title text NOT NULL,
  body text NOT NULL DEFAULT '',
  resource_type text NOT NULL DEFAULT '',
  resource_id uuid,
  read_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS notifications_user_unread_idx ON notifications(user_id,created_at DESC) WHERE read_at IS NULL;

ALTER TABLE approval_requests ADD COLUMN IF NOT EXISTS payload jsonb NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE ai_calls ADD COLUMN IF NOT EXISTS prompt_ciphertext text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS space_events (
  sequence bigserial PRIMARY KEY,
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
  event_type text NOT NULL,
  resource_id uuid,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS space_events_space_sequence_idx ON space_events(space_id,sequence);

UPDATE app_settings
SET value = value || '{"custom_days":[1,3,5],"interval_days":2}'::jsonb
WHERE key='dream';

UPDATE app_settings
SET value = value || '{"log_retention_days":90}'::jsonb
WHERE key='ai_gateway';
