-- What an AI call was for, not just that one happened.
--
-- ai_calls was written for an operator: model, status, tokens, cost, latency.
-- That answers "what is this costing" and nothing else. The person whose
-- thoughts were in the prompt has a different question — when did my writing
-- go to an outside service, and what for — and the table could not answer it.
-- Twelve rows saying "gpt-4o success" do not distinguish a Dream from an AI
-- Assist from a heading proposed for a deck.
--
-- So the purpose is recorded at the call rather than inferred later. Inferring
-- it would mean guessing from the model name and the hour, which is exactly the
-- kind of answer this is meant to replace.
--
-- Empty for every row written before this. Empty means "not recorded", not
-- "unknown purpose", and the screen says so rather than inventing a label.
ALTER TABLE ai_calls
  ADD COLUMN IF NOT EXISTS purpose text NOT NULL DEFAULT '';

-- A checked vocabulary, for the same reason note_edges.relation has one: a free
-- string here would drift into whatever each call site felt like writing, and
-- a screen cannot group by that.
ALTER TABLE ai_calls DROP CONSTRAINT IF EXISTS ai_calls_purpose_check;
ALTER TABLE ai_calls ADD CONSTRAINT ai_calls_purpose_check
  CHECK (purpose IN ('','dream','assist','ask','agent','develop','deck-headings','deck-sections'));

-- The personal view reads one user's most recent calls. Without this it is a
-- sequential scan over every call the installation has ever made.
CREATE INDEX IF NOT EXISTS ai_calls_user_recent_idx ON ai_calls(user_id, created_at DESC);
