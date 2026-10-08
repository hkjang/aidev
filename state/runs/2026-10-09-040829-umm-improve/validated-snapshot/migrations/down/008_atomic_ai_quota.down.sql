-- Reverses migrations/008_atomic_ai_quota.sql.
DROP TABLE IF EXISTS ai_quota_reservations;

DELETE FROM schema_migrations WHERE version = '008_atomic_ai_quota';
