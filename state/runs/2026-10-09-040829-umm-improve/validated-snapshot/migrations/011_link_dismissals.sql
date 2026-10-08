-- Remembering that someone said no.
--
-- Auto-link skips pairs that are already connected, which made dismissing a
-- suggestion useless: deleting the inferred edge removed the very record that
-- kept umm from proposing it again, so the next run brought it straight back.
-- Verified on a running instance — discard a suggestion, run again, and there it
-- is. A suggestion you cannot get rid of is worse than no suggestion.
--
-- The pair is stored normalised, because a connection between two thoughts is
-- the same connection whichever way umm happened to orient it.
CREATE TABLE IF NOT EXISTS link_dismissals (
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  low_note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  high_note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  dismissed_by uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  dismissed_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (low_note_id, high_note_id),
  CHECK (low_note_id < high_note_id)
);

CREATE INDEX IF NOT EXISTS link_dismissals_space_idx ON link_dismissals(space_id);
