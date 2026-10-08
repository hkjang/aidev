-- What kind of failure it was, not just what the error said.
--
-- `error` holds the Go error that ended the attempt, which is the right thing
-- to keep for whoever has to fix it and the wrong thing to show the person who
-- wanted slides: it names internal hosts, Go types and SQL constraints. So the
-- list of a space's talks tooltipped things like
--
--   ptium is unreachable: Post "http://ptium.internal:8080/api/v1/presentations": dial tcp: connection refused
--
-- next to the word "실패", which says what happened and not what to do.
--
-- The kind is what decides the sentence a person reads and whether retrying can
-- possibly help — a service that was down is worth trying again, a rejected API
-- key never is. It is stored rather than derived because the error string is
-- the only thing left once the attempt is over, and parsing it back into a kind
-- would be guessing at wording that is free to change.
--
-- Empty for rows written before this, and for everything that did not fail.
ALTER TABLE presentation_links
  ADD COLUMN IF NOT EXISTS failure_kind text NOT NULL DEFAULT '';
