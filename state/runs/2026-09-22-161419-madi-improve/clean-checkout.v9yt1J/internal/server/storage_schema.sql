CREATE TABLE IF NOT EXISTS storage_providers (
 id uuid PRIMARY KEY,workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id),name text NOT NULL,kind text NOT NULL CHECK(kind IN('local','s3')),
 config jsonb NOT NULL DEFAULT '{}',enabled boolean NOT NULL DEFAULT true,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS storage_settings(id integer PRIMARY KEY CHECK(id=1),provider_id uuid REFERENCES storage_providers(id));
INSERT INTO storage_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS storage_assignments(workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,provider_id uuid REFERENCES storage_providers(id));
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS storage_provider_id uuid REFERENCES storage_providers(id);
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS object_key text NOT NULL DEFAULT '';
ALTER TABLE attachments ADD COLUMN IF NOT EXISTS checksum_sha256 text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS backup_policy (
 id integer PRIMARY KEY CHECK(id=1),enabled boolean NOT NULL DEFAULT false,owner_id uuid REFERENCES users(id),workspace_id uuid REFERENCES workspaces(id),
 provider_id uuid REFERENCES storage_providers(id),local_path text NOT NULL DEFAULT '/var/lib/madi/backups',
 interval_minutes integer NOT NULL DEFAULT 1440 CHECK(interval_minutes BETWEEN 5 AND 525600),retention_count integer NOT NULL DEFAULT 7 CHECK(retention_count BETWEEN 1 AND 1000),
 next_run timestamptz,updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO backup_policy(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS backup_artifacts (
 id uuid PRIMARY KEY,job_id uuid UNIQUE REFERENCES automation_jobs(id) ON DELETE SET NULL,
 provider_id uuid REFERENCES storage_providers(id),object_key text NOT NULL,path text NOT NULL DEFAULT '',checksum_sha256 text NOT NULL,
 size bigint NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),last_error text NOT NULL DEFAULT ''
);
ALTER TABLE backup_artifacts ADD COLUMN IF NOT EXISTS managed boolean NOT NULL DEFAULT true;
