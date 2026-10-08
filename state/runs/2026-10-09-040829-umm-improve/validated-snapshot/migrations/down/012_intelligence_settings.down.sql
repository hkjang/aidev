-- Reverses migrations/012_intelligence_settings.sql. The code falls back to the
-- same values as compiled-in defaults, so removing the row changes nothing about
-- behaviour — it only takes the knobs away from the administrator.
DELETE FROM app_settings WHERE key = 'intelligence';

DELETE FROM schema_migrations WHERE version = '012_intelligence_settings';
