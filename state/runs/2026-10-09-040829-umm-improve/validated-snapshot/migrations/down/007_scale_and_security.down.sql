-- Reverses migrations/007_scale_and_security.sql.
DROP TRIGGER IF EXISTS space_events_notify ON space_events;
DROP FUNCTION IF EXISTS umm_notify_space_event();

DROP INDEX IF EXISTS notes_text_trgm_idx;
DROP INDEX IF EXISTS spaces_name_trgm_idx;
DROP INDEX IF EXISTS notes_space_updated_idx;
DROP INDEX IF EXISTS note_embeddings_algorithm_idx;
DROP INDEX IF EXISTS ai_calls_user_created_idx;
DROP INDEX IF EXISTS sessions_user_created_idx;
DROP INDEX IF EXISTS login_attempts_last_failed_idx;

ALTER TABLE note_embeddings DROP COLUMN IF EXISTS model;
ALTER TABLE sessions
  DROP COLUMN IF EXISTS user_agent,
  DROP COLUMN IF EXISTS client_ip,
  DROP COLUMN IF EXISTS last_seen_at;

DROP TABLE IF EXISTS login_attempts;

-- The settings rows survive; only the keys this migration introduced are
-- removed, so an administrator's unrelated tuning is preserved.
UPDATE app_settings
SET value = value - 'login_max_failures' - 'login_lockout_minutes' - 'api_rate_per_minute'
                  - 'ai_rate_per_minute' - 'ai_daily_limit'
WHERE key = 'security';

UPDATE app_settings
SET value = value - 'embedding_model' - 'embedding_dimensions'
WHERE key = 'ai_gateway';

DELETE FROM schema_migrations WHERE version = '007_scale_and_security';
