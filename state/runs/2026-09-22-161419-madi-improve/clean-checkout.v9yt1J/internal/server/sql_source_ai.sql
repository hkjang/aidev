CREATE TABLE IF NOT EXISTS sql_source_proposals(
 id uuid PRIMARY KEY,source_id uuid NOT NULL REFERENCES sql_sources(id) ON DELETE CASCADE,user_id uuid NOT NULL REFERENCES users(id),
 source_revision integer NOT NULL,name text NOT NULL,prompt text NOT NULL,explanation text NOT NULL DEFAULT '',plan jsonb NOT NULL,
 schema_snapshot jsonb NOT NULL,citations jsonb NOT NULL DEFAULT '[]',status text NOT NULL DEFAULT 'proposed' CHECK(status IN ('proposed','review','approved','rejected','cancelled')),
 version integer NOT NULL DEFAULT 1,query_id uuid REFERENCES sql_source_queries(id),approval_id uuid,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sql_source_proposals_owner_idx ON sql_source_proposals(source_id,user_id,created_at DESC);
ALTER TABLE sql_source_queries ADD COLUMN IF NOT EXISTS origin_proposal_id uuid;
