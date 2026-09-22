CREATE TABLE IF NOT EXISTS knowledge_distribution_policy (
 id integer PRIMARY KEY CHECK(id=1), revision integer NOT NULL DEFAULT 1,
 enabled boolean NOT NULL DEFAULT false,
 instance_id uuid NOT NULL DEFAULT gen_random_uuid(),
 max_valid_days integer NOT NULL DEFAULT 7 CHECK(max_valid_days BETWEEN 1 AND 365),
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO knowledge_distribution_policy(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS knowledge_distribution_policy_history (
 revision integer PRIMARY KEY, enabled boolean NOT NULL, max_valid_days integer NOT NULL,
 actor_id uuid REFERENCES users(id) ON DELETE SET NULL, created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO knowledge_distribution_policy_history(revision,enabled,max_valid_days)
SELECT revision,enabled,max_valid_days FROM knowledge_distribution_policy WHERE id=1 ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS knowledge_distribution_keys (
 id uuid PRIMARY KEY, kind text NOT NULL CHECK(kind IN ('signing','trusted')),
 label text NOT NULL CHECK(octet_length(label) BETWEEN 1 AND 200),
 source_instance uuid NOT NULL, source_key_id uuid NOT NULL,
 public_key text NOT NULL, fingerprint text NOT NULL,
 private_ciphertext text NOT NULL DEFAULT '', revision integer NOT NULL DEFAULT 1,
 created_by uuid REFERENCES users(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(), revoked_at timestamptz,
 CHECK(kind='signing' OR private_ciphertext=''),
 UNIQUE(kind,source_instance,source_key_id)
);
CREATE TABLE IF NOT EXISTS knowledge_distribution_imports (
 id uuid PRIMARY KEY, session_id uuid UNIQUE REFERENCES migration_sessions(id) ON DELETE SET NULL,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 trusted_key_id uuid NOT NULL REFERENCES knowledge_distribution_keys(id),
 bundle_id uuid NOT NULL, source_instance uuid NOT NULL, manifest_hash text NOT NULL,
 manifest_ciphertext text NOT NULL, signature text NOT NULL,
 expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 imported_at timestamptz, mapping jsonb NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS knowledge_distribution_imports_owner ON knowledge_distribution_imports(owner_id,created_at DESC);
CREATE TABLE IF NOT EXISTS knowledge_distribution_exports (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 export_id uuid REFERENCES export_runs(id) ON DELETE SET NULL,
 signing_key_id uuid NOT NULL REFERENCES knowledge_distribution_keys(id),
 receiver_instance uuid NOT NULL, policy_revision integer NOT NULL,
 document_refs jsonb NOT NULL DEFAULT '[]',
 status text NOT NULL DEFAULT 'queued' CHECK(status IN ('queued','ready','failed','revoked','expired')),
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,
 manifest_ciphertext text NOT NULL DEFAULT '', signature text NOT NULL DEFAULT '', manifest_hash text NOT NULL DEFAULT '',
 artifact_bytes bigint NOT NULL DEFAULT 0, downloads integer NOT NULL DEFAULT 0,
 expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_distribution_exports_owner ON knowledge_distribution_exports(owner_id,created_at DESC);
ALTER TABLE knowledge_distribution_exports ADD COLUMN IF NOT EXISTS document_refs jsonb NOT NULL DEFAULT '[]';
ALTER TABLE knowledge_distribution_exports ADD COLUMN IF NOT EXISTS approval_id uuid REFERENCES approval_requests(id) ON DELETE SET NULL;
ALTER TABLE knowledge_distribution_exports DROP CONSTRAINT IF EXISTS knowledge_distribution_exports_status_check;
ALTER TABLE knowledge_distribution_exports ADD CONSTRAINT knowledge_distribution_exports_status_check CHECK(status IN ('queued','awaiting_review','ready','failed','revoked','expired'));
-- Temporary, regenerable bytes are intentionally excluded from backups.
CREATE TABLE IF NOT EXISTS knowledge_distribution_artifacts (
 export_id uuid PRIMARY KEY REFERENCES knowledge_distribution_exports(id) ON DELETE CASCADE,
 ciphertext bytea NOT NULL
);
