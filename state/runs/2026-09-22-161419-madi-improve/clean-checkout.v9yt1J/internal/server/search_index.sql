CREATE TABLE IF NOT EXISTS search_index_documents (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 document_version integer NOT NULL,source_hash text NOT NULL,
 chunk_count integer NOT NULL,indexed_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS search_chunks (
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,ordinal integer NOT NULL,
 document_version integer NOT NULL,content_hash text NOT NULL,
 start_byte integer NOT NULL,end_byte integer NOT NULL,start_line integer NOT NULL,end_line integer NOT NULL,
 heading text NOT NULL,content text NOT NULL,
 search_vector tsvector GENERATED ALWAYS AS (setweight(to_tsvector('simple',heading),'A')||setweight(to_tsvector('simple',content),'B')) STORED,
 PRIMARY KEY(document_id,ordinal)
);
CREATE INDEX IF NOT EXISTS search_chunks_fts_idx ON search_chunks USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS search_chunks_content_idx ON search_chunks(document_id,content_hash);
CREATE TABLE IF NOT EXISTS search_fragments (
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,ordinal integer NOT NULL,
 document_version integer NOT NULL,kind text NOT NULL CHECK(kind IN ('block','code','task')),
 start_byte integer NOT NULL,end_byte integer NOT NULL,start_line integer NOT NULL,
 content text NOT NULL,metadata jsonb NOT NULL DEFAULT '{}',
 search_vector tsvector GENERATED ALWAYS AS (to_tsvector('simple',content)) STORED,
 PRIMARY KEY(document_id,ordinal)
);
CREATE INDEX IF NOT EXISTS search_fragments_fts_idx ON search_fragments USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS search_fragments_kind_idx ON search_fragments(kind,document_id);
CREATE INDEX IF NOT EXISTS audit_document_popularity_idx ON audit_logs(resource,created_at DESC) INCLUDE(user_id) WHERE action='DOCUMENT_READ';
CREATE TABLE IF NOT EXISTS search_index_queue (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 enqueued_at timestamptz NOT NULL DEFAULT now()
);
CREATE OR REPLACE FUNCTION madi_search_document_dirty() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.deleted_at IS NOT NULL THEN
  DELETE FROM search_chunks WHERE document_id=NEW.id;
  DELETE FROM search_fragments WHERE document_id=NEW.id;
  DELETE FROM search_index_documents WHERE document_id=NEW.id;
  DELETE FROM search_index_queue WHERE document_id=NEW.id;
 ELSE
  INSERT INTO search_index_queue(document_id) VALUES(NEW.id) ON CONFLICT(document_id) DO UPDATE SET enqueued_at=now();
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS documents_search_dirty ON documents;
CREATE TRIGGER documents_search_dirty AFTER INSERT OR UPDATE OF version,deleted_at ON documents FOR EACH ROW EXECUTE FUNCTION madi_search_document_dirty();
