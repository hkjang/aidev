-- Whether a slide still says what its thought says.
--
-- A deck made from a space goes stale when the person rewrites a thought it
-- quotes. Catching that is the difference between a deck that was true once and
-- one that is still true, which matters here more than in an ordinary export
-- because the slides carry their own sentences.
--
-- The obvious signal is the note's version, and it is the wrong one: notes bump
-- their version on any update, and x, y, width, height and rotation all live in
-- that same statement. On a canvas where dragging a note is the commonest thing
-- a person does, every deck would be permanently stale and the warning would
-- mean nothing.
--
-- So what is recorded is a fingerprint of the words. Moving a note does not
-- change it; rewriting one does. md5 is computed by the database on both sides
-- — writing the row and checking it — so there is one definition of "the same
-- words" rather than one here and another in Go that can drift apart. It is
-- used as a fingerprint and never as a security primitive.
ALTER TABLE presentation_sources
  ADD COLUMN IF NOT EXISTS note_fingerprint text NOT NULL DEFAULT '';

-- Existing rows have no fingerprint and must not be reported as changed: they
-- were compiled before umm recorded one, and claiming a slide is stale when
-- nobody knows is worse than saying nothing. An empty fingerprint means
-- "unknown", and every check treats it that way.
COMMENT ON COLUMN presentation_sources.note_fingerprint IS
  'md5 of the quoted thought''s title and content when the slide was compiled. Empty means unknown, never stale.';

INSERT INTO schema_migrations (version) VALUES ('020_presentation_freshness')
  ON CONFLICT DO NOTHING;
