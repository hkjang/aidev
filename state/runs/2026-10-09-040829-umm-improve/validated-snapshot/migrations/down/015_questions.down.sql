-- Reverses migrations/015_questions.sql.
--
-- Answer connections are dropped rather than remapped: the old vocabulary has no
-- relation that means the same thing, and rewriting them to something near would
-- record a claim nobody made.
DROP INDEX IF EXISTS notes_question_idx;
DROP INDEX IF EXISTS note_edges_answers_idx;

DELETE FROM note_edges WHERE relation = 'answers';

ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_relation_check;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_relation_check
  CHECK (relation IN ('related','supports','contradicts','refines','expands','follows'));

ALTER TABLE notes DROP CONSTRAINT IF EXISTS notes_kind_check;
UPDATE notes SET kind = legacy_kind WHERE legacy_kind IS NOT NULL;
-- Questions become ordinary thoughts again, since the old schema has no such kind.
UPDATE notes SET kind = 'thought' WHERE kind = 'question';
ALTER TABLE notes DROP COLUMN IF EXISTS legacy_kind;

DELETE FROM schema_migrations WHERE version = '015_questions';
