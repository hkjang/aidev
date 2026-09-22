CREATE TABLE IF NOT EXISTS knowledge_structured_drafts (
 id uuid PRIMARY KEY, owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 database_id uuid NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
 source_version integer NOT NULL CHECK(source_version>0), source_hash text NOT NULL,
 start_byte integer NOT NULL CHECK(start_byte>=0), end_byte integer NOT NULL CHECK(end_byte>start_byte AND end_byte-start_byte<=32768),
 schema_hash text NOT NULL, destination_hash text NOT NULL, provider_hash text NOT NULL,
 ciphertext text NOT NULL, revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 state text NOT NULL DEFAULT 'draft' CHECK(state IN ('draft','committed','expired','discarded')),
 expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 row_id uuid REFERENCES database_rows(id) ON DELETE SET NULL, values_hash text NOT NULL DEFAULT '',
 review_ciphertext text NOT NULL DEFAULT '', committed_at timestamptz,
 CHECK((state='committed')=(committed_at IS NOT NULL))
);
ALTER TABLE knowledge_structured_drafts DROP CONSTRAINT IF EXISTS knowledge_structured_drafts_state_check;
ALTER TABLE knowledge_structured_drafts ADD CONSTRAINT knowledge_structured_drafts_state_check CHECK(state IN ('draft','committed','expired','discarded'));
CREATE INDEX IF NOT EXISTS knowledge_structured_owner ON knowledge_structured_drafts(owner_id,created_at DESC,id);
CREATE INDEX IF NOT EXISTS knowledge_structured_expiry ON knowledge_structured_drafts(expires_at) WHERE state<>'committed';
