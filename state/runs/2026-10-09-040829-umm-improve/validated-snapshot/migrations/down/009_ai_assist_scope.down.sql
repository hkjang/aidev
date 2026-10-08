-- Reverses migrations/009_ai_assist_scope.sql while preserving every other
-- administrator-configured API-key scope.
UPDATE app_settings
SET value = jsonb_set(
  value,
  '{api_key_scopes}',
  COALESCE(value->'api_key_scopes', '[]'::jsonb) - 'ai:assist'
)
WHERE key = 'security';

DELETE FROM schema_migrations WHERE version = '009_ai_assist_scope';
