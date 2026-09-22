CREATE TABLE IF NOT EXISTS export_settings (
 id boolean PRIMARY KEY DEFAULT true CHECK(id), enabled boolean NOT NULL DEFAULT true,
 max_archive_mb integer NOT NULL DEFAULT 100 CHECK(max_archive_mb BETWEEN 1 AND 100),
 revision bigint NOT NULL DEFAULT 1, updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO export_settings(id) VALUES(true) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS export_runs (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 token_id uuid REFERENCES api_keys(id) ON DELETE SET NULL, token_bound boolean NOT NULL DEFAULT false,
 actor_constraints jsonb NOT NULL DEFAULT '{}', session_hash text NOT NULL DEFAULT '', request_ip text NOT NULL DEFAULT '',
 format text NOT NULL CHECK(format IN ('markdown','portable','html','json','csv')),
 document_ids uuid[] NOT NULL DEFAULT '{}', database_id uuid REFERENCES databases(id) ON DELETE SET NULL,
 source_fingerprint text NOT NULL, policy_revision bigint NOT NULL,
 status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','running','ready','failed','cancelled','expired')),
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,
 filename text NOT NULL DEFAULT '', content_type text NOT NULL DEFAULT '', artifact_bytes bigint NOT NULL DEFAULT 0,
 report jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL DEFAULT now()+interval '24 hours'
);
CREATE INDEX IF NOT EXISTS export_runs_owner ON export_runs(owner_id,created_at DESC);
CREATE INDEX IF NOT EXISTS export_runs_expiry ON export_runs(expires_at);
-- Regenerable encrypted temporary artifacts: deliberately excluded from backups.
CREATE TABLE IF NOT EXISTS export_artifact_blobs (
 run_id uuid PRIMARY KEY REFERENCES export_runs(id) ON DELETE CASCADE, ciphertext bytea NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
