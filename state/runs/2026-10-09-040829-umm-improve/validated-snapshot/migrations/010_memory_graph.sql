-- Memory graph foundation: give every connection a checked meaning and an
-- honest record of where it came from.
--
-- note_edges arrived with `relation` as unconstrained text supplied by the
-- client, and no field at all for provenance. Two consequences, both verified
-- against a running instance:
--
--   1. Any user could POST relation='dreamed' or 'expanded' and produce an edge
--      that claims umm's Dream layer discovered the connection. Nothing
--      distinguished it from one that really did.
--   2. A 5000-character relation string was accepted and stored, and every
--      reader — canvas, export, Dream selection — had to render it.
--
-- The modelling error underneath both is that `relation` was carrying two
-- different things: what the connection means, and who made it. This migration
-- separates them. `relation` keeps a small checked vocabulary of meanings;
-- `origin` records the maker and is never accepted from a request body.
--
-- `confidence` is added now because the connection types that follow — semantic
-- auto-link, contradiction detection — are only usable if a reader can tell a
-- certain edge from a guessed one. Human edges leave it NULL: a person who drew
-- a line is not expressing a probability.

ALTER TABLE note_edges ADD COLUMN IF NOT EXISTS origin text NOT NULL DEFAULT 'manual';
ALTER TABLE note_edges ADD COLUMN IF NOT EXISTS confidence real;

-- Split the conflated values before constraining either column. Edges written by
-- the Dream layer are the only ones whose relation encoded provenance, so they
-- are the only ones whose origin can be recovered.
--
-- This backfill cannot undo the hole it closes. An edge that a client forged as
-- 'dreamed' before this release is indistinguishable from one Dream really
-- wrote — the information needed to tell them apart was never recorded — so it
-- migrates to origin='dream' along with the genuine ones. From here forward the
-- claim cannot be made from a request body at all; existing rows are taken at
-- face value because there is nothing else to take them at.
UPDATE note_edges SET origin = 'dream' WHERE relation = 'dreamed';
UPDATE note_edges SET origin = 'development' WHERE relation = 'expanded';

-- 'dreamed' said nothing about meaning, so it becomes the generic relation.
-- 'expanded' did carry meaning — the target was developed out of the source —
-- and keeps it under a name that describes the connection rather than the tool.
UPDATE note_edges SET relation = 'related' WHERE relation = 'dreamed';
UPDATE note_edges SET relation = 'expands' WHERE relation = 'expanded';

-- Anything else came from an unvalidated request body. Rather than guess at a
-- meaning, record the generic relation and keep the original text in a column
-- that is not load-bearing, so an operator can see what was there.
ALTER TABLE note_edges ADD COLUMN IF NOT EXISTS legacy_relation text;
UPDATE note_edges
SET legacy_relation = relation, relation = 'related'
WHERE relation NOT IN ('related','supports','contradicts','refines','expands','follows');

ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_relation_check;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_relation_check
  CHECK (relation IN ('related','supports','contradicts','refines','expands','follows'));

ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_origin_check;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_origin_check
  CHECK (origin IN ('manual','agent','dream','development','import','auto'));

-- A guessed edge without a score cannot be ranked or filtered, and a score on a
-- hand-drawn edge is a number nobody produced. Tie the two together rather than
-- letting either drift.
ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_confidence_check;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_confidence_check
  CHECK (
    (confidence IS NULL AND origin <> 'auto') OR
    (confidence IS NOT NULL AND confidence >= 0 AND confidence <= 1)
  );

-- One edge per pair was right when a relation had no meaning. Now that it does,
-- two notes can legitimately both support and refine one another, and a machine
-- suggestion must not overwrite a line a person drew.
ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_source_note_id_target_note_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS note_edges_pair_relation_key
  ON note_edges(source_note_id, target_note_id, relation);

-- Walking the graph backwards — "what points at this thought" — had no index and
-- fell to a sequential scan over every edge in the database.
CREATE INDEX IF NOT EXISTS note_edges_target_idx ON note_edges(target_note_id, relation);
CREATE INDEX IF NOT EXISTS note_edges_space_origin_idx ON note_edges(space_id, origin);
