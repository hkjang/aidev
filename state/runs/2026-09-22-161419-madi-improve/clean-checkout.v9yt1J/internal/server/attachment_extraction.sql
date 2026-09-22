CREATE TABLE IF NOT EXISTS attachment_extraction_settings (
 id integer PRIMARY KEY CHECK(id=1), revision bigint NOT NULL DEFAULT 1,
 data jsonb NOT NULL DEFAULT '{"enabled":false,"ocr_enabled":false,"max_file_bytes":52428800,"max_pages":500,"max_text_bytes":8388608,"max_fragments":20000,"timeout_seconds":300}',
 updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO attachment_extraction_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS attachment_extraction_policy_history (
 revision bigint PRIMARY KEY, data jsonb NOT NULL,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO attachment_extraction_policy_history(revision,data) SELECT revision,data FROM attachment_extraction_settings WHERE id=1 ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS attachment_extractions (
 id uuid PRIMARY KEY, attachment_id uuid NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL,
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,
 status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','ready','failed','cancelled','obsolete')),
 revision bigint NOT NULL DEFAULT 1, policy_revision bigint NOT NULL, document_version integer NOT NULL,
 checksum text NOT NULL CHECK(length(checksum)=64),
 object_snapshot jsonb NOT NULL, format text NOT NULL,
 ocr_pages integer[] NOT NULL DEFAULT '{}',
 session_hash text NOT NULL DEFAULT '', token_hash text NOT NULL DEFAULT '', request_ip text NOT NULL DEFAULT '',
 result jsonb NOT NULL DEFAULT '{}', error text NOT NULL DEFAULT '', temp_path text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), completed_at timestamptz
);
ALTER TABLE attachment_extractions ADD COLUMN IF NOT EXISTS temp_path text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS attachment_extractions_document ON attachment_extractions(document_id,created_at DESC);
CREATE INDEX IF NOT EXISTS attachment_extractions_attachment ON attachment_extractions(attachment_id,created_at DESC);
CREATE INDEX IF NOT EXISTS attachment_extractions_cleanup ON attachment_extractions(status,updated_at,id);
CREATE UNIQUE INDEX IF NOT EXISTS attachment_extractions_pending ON attachment_extractions(attachment_id) WHERE status IN ('queued','running');

-- Local derived projection. Backup retains extraction receipts/metadata above,
-- not regenerated text below. Restore invalidates all active heads and pending
-- jobs; re-extraction/OCR requires a fresh explicit request.
CREATE TABLE IF NOT EXISTS attachment_extraction_fragments (
 id uuid PRIMARY KEY, extraction_id uuid NOT NULL REFERENCES attachment_extractions(id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal>=0), text text NOT NULL CHECK(octet_length(text)<=16384),
 content_hash text NOT NULL CHECK(length(content_hash)=64), position jsonb NOT NULL,
 fts tsvector GENERATED ALWAYS AS(to_tsvector('simple',text)) STORED,
 UNIQUE(extraction_id,ordinal)
);
CREATE INDEX IF NOT EXISTS attachment_extraction_fragments_fts ON attachment_extraction_fragments USING gin(fts);
CREATE TABLE IF NOT EXISTS attachment_extraction_heads (
 attachment_id uuid PRIMARY KEY REFERENCES attachments(id) ON DELETE CASCADE,
 extraction_id uuid NOT NULL REFERENCES attachment_extractions(id) ON DELETE CASCADE,
 updated_at timestamptz NOT NULL DEFAULT now()
);
