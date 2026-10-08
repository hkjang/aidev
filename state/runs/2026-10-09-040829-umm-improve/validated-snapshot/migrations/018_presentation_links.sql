-- What a space produced, and which thoughts each slide came from.
--
-- Only the link. Ptium owns presentations, slides and their content, and copying
-- any of that here would create a second copy that drifts the moment someone
-- edits the deck — leaving umm confidently showing a slide that no longer says
-- what it claims. The deck's title is stored because a list of links has to be
-- readable without a round trip to another service that may be unreachable, and
-- it is a label rather than a source of truth.
CREATE TABLE IF NOT EXISTS presentation_links (
  id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id             uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  -- Ptium's identifier for the deck. Opaque here on purpose: umm never parses
  -- it, and Ptium is free to change its shape.
  ptium_presentation_id text NOT NULL,
  title                text NOT NULL DEFAULT '',
  status               text NOT NULL DEFAULT 'pending'
                         CHECK (status IN ('pending', 'generating', 'ready', 'failed')),
  -- Why a generation failed, in Ptium's words. Empty while nothing has gone
  -- wrong. Kept so a failure is still explainable after the job is long gone.
  error                text NOT NULL DEFAULT '',
  -- The deck source umm compiled and sent. It is what makes a deck reviewable
  -- after the fact — without it, "why does slide 4 say that" has no answer that
  -- does not involve guessing at a compiler run that no longer exists.
  compiled_source      text NOT NULL DEFAULT '',
  -- How many thoughts the compiler used, and how many it deliberately left out.
  -- Stored rather than recomputed because the space keeps changing underneath.
  thought_count        integer NOT NULL DEFAULT 0,
  excluded_count       integer NOT NULL DEFAULT 0,
  created_by           uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at           timestamptz NOT NULL DEFAULT now(),
  updated_at           timestamptz NOT NULL DEFAULT now(),
  UNIQUE (space_id, ptium_presentation_id)
);

CREATE INDEX IF NOT EXISTS presentation_links_space_idx
  ON presentation_links (space_id, created_at DESC);

-- Which thoughts reached which slide.
--
-- This is the row that makes a deck traceable in both directions: a slide can
-- say where its sentences came from, and a thought can say which talks it ended
-- up in. Both matter more here than in an ordinary export, because the slides
-- carry the person's own words — being able to get back to the note is being
-- able to check that nothing was put in their mouth.
CREATE TABLE IF NOT EXISTS presentation_sources (
  presentation_link_id uuid NOT NULL REFERENCES presentation_links(id) ON DELETE CASCADE,
  -- Position in the deck rather than a Ptium slide id: a slide id changes when
  -- the deck is recompiled, and umm has no way to learn that it did. Position is
  -- what umm actually knows, because umm wrote the source that produced it.
  slide_position       integer NOT NULL,
  -- The thought whose words are on that slide. Deleting the thought leaves the
  -- deck alone — it was true that the slide came from it — so the row goes and
  -- the slide simply stops claiming a source it no longer has.
  note_id              uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  PRIMARY KEY (presentation_link_id, slide_position, note_id)
);

CREATE INDEX IF NOT EXISTS presentation_sources_note_idx
  ON presentation_sources (note_id);

INSERT INTO schema_migrations (version) VALUES ('018_presentation_links')
  ON CONFLICT DO NOTHING;
