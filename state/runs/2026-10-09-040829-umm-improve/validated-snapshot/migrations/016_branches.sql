-- A thought that was tried and set aside reads exactly like a current one.
--
-- That is the gap this closes. Search, ask and the assistant all return an
-- abandoned line of thinking with the same weight as a live one, and nothing in
-- the record says which is which. The cost is not clutter — it is acting on an
-- idea you already rejected, having forgotten that you rejected it.
--
-- A branch is named and resolved by hand. umm does not infer that a line was
-- abandoned from silence: not writing about something for a month is not the
-- same as deciding against it, and treating it that way would bury work that was
-- merely paused.
CREATE TABLE IF NOT EXISTS branches (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  space_id     uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  -- The thought the line grew out of. Kept nullable because deleting that one
  -- thought should not erase the record of where the line went.
  root_note_id uuid REFERENCES notes(id) ON DELETE SET NULL,
  name         text NOT NULL,
  status       text NOT NULL DEFAULT 'open'
                 CHECK (status IN ('open', 'adopted', 'abandoned')),
  -- Why it was adopted or set aside. The decision without the reason is the part
  -- people actually lose; six months later "we chose B" without "because A cost
  -- too much" invites re-deciding the same thing.
  resolution   text NOT NULL DEFAULT '',
  resolved_at  timestamptz,
  created_by   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at   timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT branches_name_present CHECK (length(btrim(name)) > 0),
  -- A resolved branch without a reason is the failure this exists to prevent.
  CONSTRAINT branches_resolved_has_reason
    CHECK (status = 'open' OR length(btrim(resolution)) > 0),
  CONSTRAINT branches_resolved_has_time
    CHECK ((status = 'open') = (resolved_at IS NULL))
);

-- ON DELETE SET NULL, not CASCADE: removing a branch means the line is no longer
-- tracked, not that the thoughts in it never happened.
ALTER TABLE notes ADD COLUMN IF NOT EXISTS branch_id uuid
  REFERENCES branches(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS notes_branch_idx
  ON notes(branch_id) WHERE branch_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS branches_space_idx
  ON branches(space_id, created_at DESC);
