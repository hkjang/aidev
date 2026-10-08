-- How many thoughts a length cap left out of a deck.
--
-- `excluded_count` already records the thoughts a deck did not use, but it
-- means one specific thing: their author marked them to be held out of
-- analysis. A thought dropped because the talk was capped at twenty slides was
-- not held out of anything — it simply did not fit, and it comes straight back
-- when the cap is raised. Counting the two together would tell someone their
-- space excludes thoughts it does not exclude, and would hide the one number
-- they can act on.
--
-- So it is stored rather than recomputed: the cap, the space and the graph have
-- all moved on by the time anybody looks at the list of a space's talks, and
-- recompiling to find out what an old deck left behind would answer a question
-- about that deck with a fact about today.
--
-- Zero for rows written before this, and for every deck made with no cap.
ALTER TABLE presentation_links
  ADD COLUMN IF NOT EXISTS trimmed_count integer NOT NULL DEFAULT 0;
