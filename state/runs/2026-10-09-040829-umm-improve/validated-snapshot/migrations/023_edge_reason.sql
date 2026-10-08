-- Why this connection exists, in the words of whoever drew it.
--
-- 010 took free text off `relation` because that column was carrying two
-- different things at once — what a connection means, and who decided it
-- existed — and a 5000-character string had to be rendered by every reader.
-- That was a modelling error, not a ruling that a reason has no place. The
-- vocabulary answers "what kind of connection is this"; nothing yet answers
-- "why did you draw it", and six months later that is the question people
-- actually have. The `why` is what disappears first.
--
-- Its own column, so the meaning stays a checked vocabulary and the sentence
-- stays a sentence. Bounded at 200 characters: this is the line beside a
-- connection, not a second note. A person with more to say has somewhere to
-- say it already.
--
-- Empty for every connection drawn before this, and for every one whose author
-- did not feel the need. Empty is not "unknown" and must never be rendered as
-- though a reason were missing — most connections do not need one.
ALTER TABLE note_edges
  ADD COLUMN IF NOT EXISTS reason text NOT NULL DEFAULT '';

ALTER TABLE note_edges DROP CONSTRAINT IF EXISTS note_edges_reason_check;
ALTER TABLE note_edges ADD CONSTRAINT note_edges_reason_check
  CHECK (char_length(reason) <= 200);
