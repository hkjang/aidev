-- Reverses migrations/016_branches.sql.
--
-- The thoughts stay; only the record of which line they belonged to goes. That
-- record has no home in the old schema, and writing it into note content would
-- be inventing text the person never typed.
DROP INDEX IF EXISTS branches_space_idx;
DROP INDEX IF EXISTS notes_branch_idx;

ALTER TABLE notes DROP COLUMN IF EXISTS branch_id;
DROP TABLE IF EXISTS branches;

DELETE FROM schema_migrations WHERE version = '016_branches';
