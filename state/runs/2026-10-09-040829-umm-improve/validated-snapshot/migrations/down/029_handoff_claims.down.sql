-- Reverses migrations/029_handoff_claims.sql.
--
-- Claims live five minutes; dropping the table loses at most a handoff somebody
-- is in the middle of, and they can send again. The target list goes with it,
-- so the menu disappears until an administrator names a service again.
DROP TABLE IF EXISTS handoff_claims;
DELETE FROM app_settings WHERE key = 'handoff';

DELETE FROM schema_migrations WHERE version = '029_handoff_claims';
