ALTER TABLE spaces
  ADD COLUMN IF NOT EXISTS ai_excluded boolean NOT NULL DEFAULT false;

ALTER TABLE notes
  ADD COLUMN IF NOT EXISTS ai_excluded boolean NOT NULL DEFAULT false;

ALTER TABLE dream_notes
  ADD COLUMN IF NOT EXISTS content text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS rationale text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS suggested_action text NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS generation integer NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS accepted_at timestamptz,
  ADD COLUMN IF NOT EXISTS dismissed_reason text NOT NULL DEFAULT '';

UPDATE dream_notes d
SET content = n.content
FROM notes n
WHERE d.note_id = n.id AND d.content = '';

-- Before v0.6 Dream notes were placed on the canvas immediately. Preserve
-- those rows as accepted so the upgrade never turns existing canvas content
-- back into an inbox candidate.
UPDATE dream_notes
SET status = 'kept', accepted_at = COALESCE(accepted_at, generated_at)
WHERE note_id IS NOT NULL AND status IN ('created', 'exposed');

ALTER TABLE dream_sources
  ADD COLUMN IF NOT EXISTS cited boolean NOT NULL DEFAULT false;

-- All source rows that predate the structured response format contributed to
-- the canvas Dream and remain visible as its historical provenance.
UPDATE dream_sources SET cited = true;

ALTER TABLE dream_feedback
  ADD COLUMN IF NOT EXISTS reason text NOT NULL DEFAULT '';

-- Preference learning is event based, so the same action must only influence
-- a Dream once even if multiple clients retry the request.
DELETE FROM dream_feedback newer
USING dream_feedback older
WHERE newer.dream_id = older.dream_id
  AND newer.user_id = older.user_id
  AND newer.action = older.action
  AND newer.id > older.id;

CREATE UNIQUE INDEX IF NOT EXISTS dream_feedback_once_idx
  ON dream_feedback(dream_id, user_id, action);

CREATE INDEX IF NOT EXISTS dream_notes_user_status_idx
  ON dream_notes(user_id, status, generated_at DESC);

CREATE INDEX IF NOT EXISTS dream_notes_space_generated_idx
  ON dream_notes(space_id, generated_at DESC);

CREATE INDEX IF NOT EXISTS notes_dream_eligible_idx
  ON notes(author_id, space_id, updated_at DESC)
  WHERE deleted_at IS NULL AND source != 'dream' AND ai_excluded = false;
