-- Reverses migrations/010_memory_graph.sql.
--
-- The original relation text is restored from legacy_relation where it was
-- preserved, and the provenance that was folded into relation is put back, so a
-- rollback returns the exact strings the old code reads.
DROP INDEX IF EXISTS note_edges_space_origin_idx;
DROP INDEX IF EXISTS note_edges_target_idx;

ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_relation_check;
ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_origin_check;
ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_confidence_check;

UPDATE note_edges SET relation = 'dreamed' WHERE origin = 'dream' AND relation = 'related';
UPDATE note_edges SET relation = 'expanded' WHERE relation = 'expands';
UPDATE note_edges SET relation = legacy_relation WHERE legacy_relation IS NOT NULL;

-- The old schema allowed one edge per pair. Typed edges added since may have
-- created several, so keep the earliest of each pair before restoring it.
DELETE FROM note_edges e
USING note_edges keep
WHERE e.source_note_id = keep.source_note_id
  AND e.target_note_id = keep.target_note_id
  AND (e.created_at, e.id) > (keep.created_at, keep.id);

DROP INDEX IF EXISTS note_edges_pair_relation_key;
ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_source_note_id_target_note_id_key;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_source_note_id_target_note_id_key
  UNIQUE (source_note_id, target_note_id);

ALTER TABLE note_edges DROP COLUMN IF EXISTS legacy_relation;
ALTER TABLE note_edges DROP COLUMN IF EXISTS confidence;
ALTER TABLE note_edges DROP COLUMN IF EXISTS origin;

DELETE FROM schema_migrations WHERE version = '010_memory_graph';
