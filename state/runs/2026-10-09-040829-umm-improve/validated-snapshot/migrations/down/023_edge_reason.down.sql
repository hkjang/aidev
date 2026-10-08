ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_reason_check;
ALTER TABLE note_edges DROP COLUMN IF EXISTS reason;
