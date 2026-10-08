-- Reverses migrations/013_thought_inbox.sql. The inbox spaces themselves are
-- left in place: they hold real thoughts, and dropping the flag only makes them
-- ordinary spaces again rather than something to clean up.
DROP INDEX IF EXISTS spaces_one_inbox_per_owner;
ALTER TABLE spaces DROP COLUMN IF EXISTS is_inbox;

DELETE FROM schema_migrations WHERE version = '013_thought_inbox';
