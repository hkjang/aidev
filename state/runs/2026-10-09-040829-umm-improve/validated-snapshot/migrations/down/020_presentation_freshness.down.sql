-- Reverses migrations/020_presentation_freshness.sql.
--
-- Losing the fingerprints means every deck goes back to being of unknown
-- freshness, which is what it was before. Nothing about the decks themselves
-- changes.
ALTER TABLE presentation_sources DROP COLUMN IF EXISTS note_fingerprint;

DELETE FROM schema_migrations WHERE version = '020_presentation_freshness';
