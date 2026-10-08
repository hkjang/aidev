-- Reverses migrations/017_drop_unused_embedding_dimensions.sql.
--
-- Restores the key at the value 007 seeded. It did nothing then and it does
-- nothing now; this only puts the schema back where it was.
UPDATE app_settings
SET value = '{"embedding_dimensions":0}'::jsonb || value
WHERE key = 'ai_gateway';

DELETE FROM schema_migrations WHERE version = '017_drop_unused_embedding_dimensions';
