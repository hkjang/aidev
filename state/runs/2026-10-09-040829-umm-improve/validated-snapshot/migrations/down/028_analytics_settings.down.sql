-- Reverses migrations/028_analytics_settings.sql.
--
-- The row goes, and with it the switch: a binary that still knows the setting
-- reads a missing row as off, which is what a rolled-back installation should
-- be doing anyway.
DELETE FROM app_settings WHERE key = 'analytics';

DELETE FROM schema_migrations WHERE version = '028_analytics_settings';
