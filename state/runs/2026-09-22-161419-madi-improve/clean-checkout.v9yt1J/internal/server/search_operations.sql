CREATE TABLE IF NOT EXISTS workspace_search_dictionary (
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 revision integer NOT NULL DEFAULT 1, entries jsonb NOT NULL DEFAULT '[]',
 updated_by uuid REFERENCES users(id),updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(jsonb_typeof(entries)='array' AND jsonb_array_length(entries)<=300)
);
-- These are local, rebuildable projections. They do not authorize any source.
CREATE TABLE IF NOT EXISTS search_folded_documents (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 document_version integer NOT NULL,normalized text NOT NULL,
 gram_vector tsvector NOT NULL,grams_complete boolean NOT NULL,
 indexed_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS search_folded_documents_grams_idx ON search_folded_documents USING GIN(gram_vector);
CREATE INDEX IF NOT EXISTS search_folded_documents_incomplete_idx ON search_folded_documents(document_id) WHERE NOT grams_complete;
CREATE TABLE IF NOT EXISTS search_folded_fragments (
 document_id uuid NOT NULL,ordinal integer NOT NULL,document_version integer NOT NULL,
 normalized text NOT NULL,gram_vector tsvector NOT NULL,grams_complete boolean NOT NULL,
 PRIMARY KEY(document_id,ordinal),
 FOREIGN KEY(document_id,ordinal) REFERENCES search_fragments(document_id,ordinal) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS search_folded_fragments_grams_idx ON search_folded_fragments USING GIN(gram_vector);
CREATE INDEX IF NOT EXISTS search_folded_fragments_incomplete_idx ON search_folded_fragments(document_id,ordinal) WHERE NOT grams_complete;

-- Canonical/backup JSON stays unchanged. The FTS candidate UNION uses this
-- expression index instead of reparsing every comment during each read.
CREATE INDEX IF NOT EXISTS comments_search_fts_idx ON comments USING GIN(to_tsvector('simple',body)) WHERE deleted_at IS NULL;

-- Each outer part is AND, each synonym in a part is OR. Values are passed as
-- JSON parameters, never interpolated into query syntax or executable SQL.
CREATE OR REPLACE FUNCTION madi_search_parts_query(parts jsonb) RETURNS tsquery LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE result tsquery;part tsquery;piece tsquery;alternatives jsonb;value text;
BEGIN
 FOR alternatives IN SELECT jsonb_array_elements(parts) LOOP
  part:=NULL;
  FOR value IN SELECT jsonb_array_elements_text(alternatives) LOOP
   piece:=plainto_tsquery('simple',value);
   IF numnode(piece)>0 THEN part:=CASE WHEN part IS NULL THEN piece ELSE part||piece END; END IF;
  END LOOP;
  IF part IS NOT NULL THEN result:=CASE WHEN result IS NULL THEN part ELSE result&&part END; END IF;
 END LOOP;
 RETURN coalesce(result,''::tsquery);
END $$;
CREATE OR REPLACE FUNCTION madi_search_folded_matches(value text,parts jsonb) RETURNS boolean LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE AS $$
DECLARE alternatives jsonb;term text;matched boolean;
BEGIN
 IF value IS NULL OR jsonb_array_length(parts)=0 THEN RETURN false;END IF;
 FOR alternatives IN SELECT jsonb_array_elements(parts) LOOP
  matched:=false;
  FOR term IN SELECT jsonb_array_elements_text(alternatives) LOOP
   IF term<>'' AND strpos(value,term)>0 THEN matched:=true;EXIT;END IF;
  END LOOP;
  IF NOT matched THEN RETURN false;END IF;
 END LOOP;
 RETURN true;
END $$;
