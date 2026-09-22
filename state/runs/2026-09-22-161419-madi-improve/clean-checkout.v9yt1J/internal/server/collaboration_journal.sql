-- Legacy state rows are complete snapshots at their existing sequence. This
-- migration must not advance a live checkpoint when later updates are pending.
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='document_collaboration' AND column_name='snapshot_sequence') THEN
  ALTER TABLE document_collaboration ADD COLUMN snapshot_sequence bigint NOT NULL DEFAULT 0;
  UPDATE document_collaboration SET snapshot_sequence=sequence;
 END IF;
END $$;
ALTER TABLE document_collaboration ADD COLUMN IF NOT EXISTS snapshot_at timestamptz NOT NULL DEFAULT now();
ALTER TABLE document_collaboration ADD COLUMN IF NOT EXISTS reset_reason text NOT NULL DEFAULT '';
ALTER TABLE document_collaboration ADD COLUMN IF NOT EXISTS head_hash text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS collaboration_updates (
 document_id uuid NOT NULL REFERENCES document_collaboration(document_id) ON DELETE CASCADE,
 epoch uuid NOT NULL, sequence bigint NOT NULL CHECK(sequence>0),
 update_data bytea NOT NULL CHECK(octet_length(update_data)<=8388608),
 document_version integer NOT NULL, actor_id uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(document_id,epoch,sequence)
);
CREATE TABLE IF NOT EXISTS collaboration_history_groups (
 id uuid PRIMARY KEY, document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), epoch uuid NOT NULL,
 first_version integer NOT NULL, last_version integer NOT NULL,
 started_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(last_version>=first_version)
);
CREATE INDEX IF NOT EXISTS collaboration_history_groups_document_idx ON collaboration_history_groups(document_id,last_version DESC);

-- Notifications contain no document body, title, credentials or session hashes.
-- They are hints only: readers catch up against durable epoch/sequence values.
CREATE OR REPLACE FUNCTION madi_collaboration_notify() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE payload text;
BEGIN
 IF TG_ARGV[0]='document' THEN
  IF TG_TABLE_NAME='documents' THEN payload := coalesce(NEW.id,OLD.id)::text;
  ELSE payload := coalesce(NEW.document_id,OLD.document_id)::text; END IF;
 ELSE payload := '*'; END IF;
 PERFORM pg_notify('madi_collab_'||md5(TG_TABLE_SCHEMA),payload);
 RETURN NULL;
END $$;
