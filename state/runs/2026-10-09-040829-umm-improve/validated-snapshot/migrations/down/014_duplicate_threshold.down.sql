-- Reverses migrations/014_duplicate_threshold.sql. The code carries the same
-- value as a compiled-in default, so removing the key changes no behaviour.
UPDATE app_settings
SET value = value - 'duplicate_similarity'
WHERE key = 'intelligence';

DELETE FROM schema_migrations WHERE version = '014_duplicate_threshold';
