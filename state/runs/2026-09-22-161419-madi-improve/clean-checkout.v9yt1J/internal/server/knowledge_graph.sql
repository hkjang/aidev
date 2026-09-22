CREATE TABLE IF NOT EXISTS document_relations (
 source_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 target_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 type text NOT NULL CHECK(type IN ('related','reference')),
 created_by uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(source_id,target_id,type),CHECK(source_id<>target_id)
);
CREATE INDEX IF NOT EXISTS document_relations_target_idx ON document_relations(target_id);

ALTER TABLE search_index_documents ADD COLUMN IF NOT EXISTS links jsonb NOT NULL DEFAULT '[]';
ALTER TABLE search_index_documents ADD COLUMN IF NOT EXISTS source_tags jsonb NOT NULL DEFAULT '[]';
ALTER TABLE search_index_documents ADD COLUMN IF NOT EXISTS links_indexed boolean NOT NULL DEFAULT false;
