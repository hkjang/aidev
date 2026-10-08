-- A picture on a thought.
--
-- umm holds writing and, until now, only writing. A whiteboard photo, a
-- screenshot of the thing being discussed, a diagram someone drew — the parts
-- of a decision that are not sentences — lived outside the space that held the
-- rest of it.
--
-- The bytes live here rather than on a disk or in object storage. umm ships as
-- one binary and one PostgreSQL, and the offline image release is the whole
-- product; adding a bucket would add an operational dependency to every
-- installation for a feature most of them will use lightly. Keeping the bytes
-- in the database also means one backup holds everything, one transaction
-- deletes a thought and its pictures together, and the access rules that
-- already govern a note govern its pictures without a second implementation.
--
-- The cost is real and bounded on purpose: images only, capped per file and per
-- thought, so a canvas cannot quietly become a file server.
CREATE TABLE IF NOT EXISTS note_attachments (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  note_id uuid NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
  -- Denormalised from the note so a permission check needs one join instead of
  -- two, and so a picture cannot outlive the space it belonged to.
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  uploaded_by uuid REFERENCES users(id) ON DELETE SET NULL,
  -- The type the bytes actually are, decided by reading them rather than by
  -- believing the upload. A file that says it is a PNG and is not is the whole
  -- reason this column is not the client's word.
  content_type text NOT NULL,
  byte_size integer NOT NULL,
  -- What the person called it, kept only to name a download. Never used to
  -- decide a type and never trusted as a path.
  filename text NOT NULL DEFAULT '',
  bytes bytea NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT note_attachments_type_check
    CHECK (content_type IN ('image/png','image/jpeg','image/gif','image/webp')),
  CONSTRAINT note_attachments_size_check
    CHECK (byte_size > 0 AND byte_size <= 5242880)
);

CREATE INDEX IF NOT EXISTS note_attachments_note_idx ON note_attachments(note_id, created_at);
