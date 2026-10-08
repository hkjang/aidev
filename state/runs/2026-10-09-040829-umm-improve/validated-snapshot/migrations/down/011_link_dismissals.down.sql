-- Reverses migrations/011_link_dismissals.sql. Dropping the table forgets which
-- suggestions people turned down, so auto-link will offer them again.
DROP INDEX IF EXISTS link_dismissals_space_idx;
DROP TABLE IF EXISTS link_dismissals;

DELETE FROM schema_migrations WHERE version = '011_link_dismissals';
