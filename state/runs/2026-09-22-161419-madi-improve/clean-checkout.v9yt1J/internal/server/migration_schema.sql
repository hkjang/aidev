CREATE TABLE IF NOT EXISTS migration_imports (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id),space_id uuid REFERENCES spaces(id),
 filename text NOT NULL,format text NOT NULL,source_data bytea,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','ready','running','completed','failed','cancelled')),
 preview jsonb NOT NULL DEFAULT '{}',report jsonb NOT NULL DEFAULT '{}',
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,
 created_at timestamptz NOT NULL DEFAULT now(),expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days'
);
CREATE INDEX IF NOT EXISTS migration_imports_owner_idx ON migration_imports(workspace_id,user_id,created_at DESC);
ALTER TABLE migration_imports ADD COLUMN IF NOT EXISTS source_encrypted boolean NOT NULL DEFAULT false;
ALTER TABLE migration_imports ADD COLUMN IF NOT EXISTS source_size bigint NOT NULL DEFAULT 0;
UPDATE migration_imports SET source_size=octet_length(source_data) WHERE source_data IS NOT NULL AND NOT source_encrypted AND source_size=0;
