-- v0.8.0 raises the ceiling on concurrent collaboration, indexed search and
-- credential abuse. Every statement is written to be safely re-runnable.

CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Indexed lexical search -----------------------------------------------------
-- The expression must stay identical to store.noteSearchExpression so the
-- planner can serve `ILIKE '%term%'` from the trigram index instead of reading
-- every note body.
CREATE INDEX IF NOT EXISTS notes_text_trgm_idx
  ON notes USING gin ((title || ' ' || content) gin_trgm_ops)
  WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS spaces_name_trgm_idx
  ON spaces USING gin (name gin_trgm_ops);

CREATE INDEX IF NOT EXISTS notes_space_updated_idx
  ON notes(space_id, updated_at DESC, id DESC)
  WHERE deleted_at IS NULL;

-- Push based collaboration stream --------------------------------------------
-- pg_notify inside a trigger is delivered by PostgreSQL at commit time, so a
-- listener never observes an event whose transaction was rolled back.
CREATE OR REPLACE FUNCTION umm_notify_space_event() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  PERFORM pg_notify('umm_space_events', NEW.space_id::text || ' ' || NEW.sequence::text);
  RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS space_events_notify ON space_events;
CREATE TRIGGER space_events_notify
  AFTER INSERT ON space_events
  FOR EACH ROW EXECUTE FUNCTION umm_notify_space_event();

-- Pluggable embeddings -------------------------------------------------------
ALTER TABLE note_embeddings
  ADD COLUMN IF NOT EXISTS model text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS note_embeddings_algorithm_idx
  ON note_embeddings(algorithm);

-- Credential abuse throttling ------------------------------------------------
CREATE TABLE IF NOT EXISTS login_attempts (
  identity text PRIMARY KEY,
  failure_count integer NOT NULL DEFAULT 0 CHECK (failure_count >= 0),
  locked_until timestamptz,
  first_failed_at timestamptz NOT NULL DEFAULT now(),
  last_failed_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS login_attempts_last_failed_idx
  ON login_attempts(last_failed_at);

-- Session inventory ----------------------------------------------------------
ALTER TABLE sessions
  ADD COLUMN IF NOT EXISTS user_agent text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS client_ip text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS last_seen_at timestamptz NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS sessions_user_created_idx
  ON sessions(user_id, created_at DESC);

-- AI spend guardrails --------------------------------------------------------
CREATE INDEX IF NOT EXISTS ai_calls_user_created_idx
  ON ai_calls(user_id, created_at DESC);

-- New settings must never clobber a value an administrator already tuned, so
-- the stored object is merged on top of the defaults.
UPDATE app_settings
SET value = '{"login_max_failures":8,"login_lockout_minutes":15,"api_rate_per_minute":600,"ai_rate_per_minute":6,"ai_daily_limit":80}'::jsonb || value
WHERE key = 'security';

UPDATE app_settings
SET value = '{"embedding_model":"","embedding_dimensions":0}'::jsonb || value
WHERE key = 'ai_gateway';
