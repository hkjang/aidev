-- Only grants are backup-worthy. Vectors and the dispatch queue are derived.
CREATE TABLE IF NOT EXISTS rag_index_grants (
 id uuid PRIMARY KEY,
 document_id uuid NOT NULL UNIQUE REFERENCES documents(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_id uuid REFERENCES api_keys(id) ON DELETE CASCADE,
 actor_constraints jsonb NOT NULL DEFAULT '{}',
 revision bigint NOT NULL DEFAULT 1,
 active boolean NOT NULL DEFAULT true,
 auto_reindex boolean NOT NULL DEFAULT false,
 provider_fingerprint text NOT NULL,
 rerank_fingerprint text NOT NULL DEFAULT '',
 expected_version integer NOT NULL,
 last_job_id uuid,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS rag_vector_indexes (
 id uuid PRIMARY KEY REFERENCES automation_jobs(id) ON DELETE CASCADE,
 grant_id uuid NOT NULL REFERENCES rag_index_grants(id) ON DELETE CASCADE,
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 document_version integer NOT NULL,
 grant_revision bigint NOT NULL,
 provider_fingerprint text NOT NULL,
 status text NOT NULL DEFAULT 'building' CHECK(status IN ('building','ready')),
 total_chunks integer NOT NULL DEFAULT 0,
 indexed_chunks integer NOT NULL DEFAULT 0,
 dimensions integer NOT NULL DEFAULT 0 CHECK(dimensions BETWEEN 0 AND 8192),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS rag_vector_indexes_document ON rag_vector_indexes(document_id,document_version,status);
CREATE TABLE IF NOT EXISTS rag_vector_chunks (
 index_id uuid NOT NULL REFERENCES rag_vector_indexes(id) ON DELETE CASCADE,
 ordinal integer NOT NULL,
 content_hash text NOT NULL,
 start_byte integer NOT NULL,
 end_byte integer NOT NULL,
 start_line integer NOT NULL,
 end_line integer NOT NULL,
 heading text NOT NULL DEFAULT '',
 embedding real[] NOT NULL CHECK(cardinality(embedding) BETWEEN 1 AND 8192),
 PRIMARY KEY(index_id,ordinal)
);
CREATE TABLE IF NOT EXISTS rag_reindex_queue (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 enqueued_at timestamptz NOT NULL DEFAULT now()
);
CREATE OR REPLACE FUNCTION madi_queue_rag_reindex() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.deleted_at IS NOT NULL THEN
  UPDATE rag_index_grants SET active=false,auto_reindex=false,revision=revision+1,updated_at=now() WHERE document_id=NEW.id AND active;
  DELETE FROM rag_vector_indexes WHERE document_id=NEW.id;
  DELETE FROM rag_reindex_queue WHERE document_id=NEW.id;
 ELSIF NEW.version IS DISTINCT FROM OLD.version THEN
  INSERT INTO rag_reindex_queue(document_id) SELECT NEW.id WHERE EXISTS(SELECT 1 FROM rag_index_grants WHERE document_id=NEW.id AND active AND auto_reindex)
  ON CONFLICT(document_id) DO UPDATE SET enqueued_at=now();
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS madi_document_rag_reindex ON documents;
CREATE TRIGGER madi_document_rag_reindex AFTER UPDATE OF version,deleted_at ON documents FOR EACH ROW EXECUTE FUNCTION madi_queue_rag_reindex();
