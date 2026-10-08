-- Reverses migrations/018_presentation_links.sql.
--
-- The presentations themselves are Ptium's and are untouched; what goes is
-- umm's record of which space produced them and which thought reached which
-- slide. That record has no home in the old schema, and there is nowhere
-- honest to put it — writing it into note content would be inventing text the
-- person never typed.
DROP INDEX IF EXISTS presentation_sources_note_idx;
DROP TABLE IF EXISTS presentation_sources;

DROP INDEX IF EXISTS presentation_links_space_idx;
DROP TABLE IF EXISTS presentation_links;

DELETE FROM schema_migrations WHERE version = '018_presentation_links';
