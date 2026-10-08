-- Two gaps, and they are the same gap twice.
--
-- `notes.kind` was never validated. It arrived from the request body like
-- `relation` used to, so a client could store anything in it — a 5000-character
-- string was accepted and kept, verified against a running instance. Nothing
-- read it back, which is why nobody noticed.
--
-- That matters now because marking a thought as a question is exactly what kind
-- is for, and a field that means anything means nothing. The vocabulary is small
-- on purpose: a slot for a kind umm does nothing with would advertise a feature
-- that does not exist.
ALTER TABLE notes ADD COLUMN IF NOT EXISTS legacy_kind text;

UPDATE notes
SET legacy_kind = kind, kind = 'thought'
WHERE kind NOT IN ('thought', 'question', 'idea');

ALTER TABLE notes DROP CONSTRAINT IF EXISTS notes_kind_check;
ALTER TABLE notes ADD CONSTRAINT notes_kind_check
  CHECK (kind IN ('thought', 'question', 'idea'));

-- Finding a question is only half of it. An open question is one nothing has
-- answered, and until now the graph had no way to say that a thought answers
-- another — supports and refines are near, but neither closes a question.
ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_relation_check;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_relation_check
  CHECK (relation IN ('related','supports','contradicts','refines','expands','follows','answers'));

-- Asking "what is still open" walks from every question looking for an incoming
-- answer, so that lookup gets an index rather than a scan of every edge.
CREATE INDEX IF NOT EXISTS note_edges_answers_idx
  ON note_edges(target_note_id) WHERE relation = 'answers';
CREATE INDEX IF NOT EXISTS notes_question_idx
  ON notes(space_id) WHERE kind = 'question' AND deleted_at IS NULL;
