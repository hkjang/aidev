-- Capturing a thought used to require choosing where it belongs first: open the
-- app, pick a space, then type. That ordering is backwards for the moment a
-- thought actually arrives, and it is the reason thoughts get written somewhere
-- else and never make it in.
--
-- Every user gets one inbox space. It is an ordinary space — searchable,
-- embedded, visible to Dream, connectable — so nothing downstream needs to learn
-- about a second kind of container. The flag only marks where an unfiled thought
-- lands and stops a person from deleting the place their capture goes.
ALTER TABLE spaces ADD COLUMN IF NOT EXISTS is_inbox boolean NOT NULL DEFAULT false;

-- One inbox per owner. A partial unique index rather than a constraint on the
-- column, because every other space must remain free to exist alongside it.
CREATE UNIQUE INDEX IF NOT EXISTS spaces_one_inbox_per_owner
  ON spaces(owner_id) WHERE is_inbox;
