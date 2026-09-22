-- Markdown and its native Yjs editor state are committed together. Epochs stop
-- a disconnected editor from overwriting a subsequent REST/import/restore edit.
CREATE TABLE IF NOT EXISTS document_collaboration (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 epoch uuid NOT NULL, sequence bigint NOT NULL DEFAULT 0,
 state bytea NOT NULL DEFAULT '', projected_markdown text NOT NULL,
 document_version integer NOT NULL, updated_at timestamptz NOT NULL DEFAULT now()
);
-- Ephemeral, identity-verified awareness, shared by replicas without Redis.
CREATE TABLE IF NOT EXISTS collaboration_presence (
 connection_id uuid PRIMARY KEY, document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 epoch uuid NOT NULL, client_id bigint NOT NULL DEFAULT 0,
 clock bigint NOT NULL DEFAULT 0, state jsonb NOT NULL DEFAULT '{}',
 operations integer NOT NULL DEFAULT 0, operation_window timestamptz NOT NULL DEFAULT now(),
 touched_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS collaboration_presence_document_idx ON collaboration_presence(document_id,touched_at);
ALTER TABLE collaboration_presence ADD COLUMN IF NOT EXISTS operations integer NOT NULL DEFAULT 0;
ALTER TABLE collaboration_presence ADD COLUMN IF NOT EXISTS operation_window timestamptz NOT NULL DEFAULT now();
