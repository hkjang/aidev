CREATE TABLE IF NOT EXISTS rag_generations (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name text NOT NULL,
 revision bigint NOT NULL DEFAULT 1,
 status text NOT NULL DEFAULT 'building' CHECK(status IN ('building','active','retired','disabled')),
 dimensions integer NOT NULL CHECK(dimensions BETWEEN 0 AND 8192),
 provider_fingerprint text NOT NULL,
 config_cipher text NOT NULL,
 created_by uuid NOT NULL REFERENCES users(id),
 index_job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(id,workspace_id)
);
CREATE TABLE IF NOT EXISTS rag_generation_state (
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 active_id uuid,
 revision bigint NOT NULL DEFAULT 1,
 mode text NOT NULL DEFAULT 'exact' CHECK(mode IN ('exact','ann','verify')),
 updated_at timestamptz NOT NULL DEFAULT now(),
 FOREIGN KEY(active_id,workspace_id) REFERENCES rag_generations(id,workspace_id)
);
CREATE TABLE IF NOT EXISTS rag_generation_validations (
 id uuid PRIMARY KEY,
 generation_id uuid NOT NULL REFERENCES rag_generations(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id),
 session_hash text NOT NULL,
 generation_revision bigint NOT NULL,
 state_revision bigint NOT NULL,
 settings_fingerprint text NOT NULL,
 query_hash text NOT NULL,
 cohort jsonb NOT NULL CHECK(jsonb_typeof(cohort)='array' AND jsonb_array_length(cohort) BETWEEN 1 AND 30),
 report jsonb NOT NULL,
 expires_at timestamptz NOT NULL,
 consumed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE rag_index_grants ADD COLUMN IF NOT EXISTS generation_id uuid REFERENCES rag_generations(id) ON DELETE CASCADE;
ALTER TABLE rag_vector_chunks ADD COLUMN IF NOT EXISTS generation_id uuid REFERENCES rag_generations(id) ON DELETE CASCADE;
ALTER TABLE rag_index_grants DROP CONSTRAINT IF EXISTS rag_index_grants_document_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS rag_index_grants_document_generation_idx ON rag_index_grants(document_id,generation_id) NULLS NOT DISTINCT;
CREATE INDEX IF NOT EXISTS rag_vector_chunks_generation_idx ON rag_vector_chunks(generation_id);
CREATE INDEX IF NOT EXISTS rag_generation_validations_expiry_idx ON rag_generation_validations(expires_at);
