CREATE TABLE IF NOT EXISTS git_sync_settings (
 id integer PRIMARY KEY CHECK(id=1), data jsonb NOT NULL DEFAULT '{}',
 revision bigint NOT NULL DEFAULT 1, updated_by uuid REFERENCES users(id),
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO git_sync_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS git_sync_connections (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id),
 owner_id uuid NOT NULL REFERENCES users(id), space_id uuid REFERENCES spaces(id),
 name text NOT NULL, config jsonb NOT NULL, enabled boolean NOT NULL DEFAULT false,
 revision bigint NOT NULL DEFAULT 1, last_remote_commit text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS git_sync_connections_workspace ON git_sync_connections(workspace_id);
CREATE TABLE IF NOT EXISTS git_sync_runs (
 id uuid PRIMARY KEY, connection_id uuid NOT NULL REFERENCES git_sync_connections(id),
 workspace_id uuid NOT NULL REFERENCES workspaces(id), owner_id uuid NOT NULL REFERENCES users(id),
 actor_constraints jsonb NOT NULL DEFAULT '{}', direction text NOT NULL CHECK(direction IN ('push','pull')),
 status text NOT NULL DEFAULT 'preparing' CHECK(status IN ('preparing','preview','queued','running','succeeded','conflict','failed','cancelled','unknown')),
 config_fingerprint text NOT NULL, document_ids jsonb NOT NULL DEFAULT '[]',
 snapshot_cipher bytea, snapshot_size bigint NOT NULL DEFAULT 0,
 source_versions jsonb NOT NULL DEFAULT '{}', report jsonb NOT NULL DEFAULT '{}',
 remote_commit text NOT NULL DEFAULT '', candidate_commit text NOT NULL DEFAULT '',
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL, revision bigint NOT NULL DEFAULT 1,
 confirmed_at timestamptz, expires_at timestamptz NOT NULL DEFAULT now()+interval '1 day',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS git_sync_runs_owner ON git_sync_runs(owner_id,created_at DESC);
CREATE INDEX IF NOT EXISTS git_sync_runs_connection ON git_sync_runs(connection_id,created_at DESC);
CREATE TABLE IF NOT EXISTS git_sync_mappings (
 connection_id uuid NOT NULL REFERENCES git_sync_connections(id), path text NOT NULL,
 document_id uuid REFERENCES documents(id) ON DELETE CASCADE, attachment_id uuid REFERENCES attachments(id) ON DELETE CASCADE,
 local_hash text NOT NULL DEFAULT '', remote_hash text NOT NULL DEFAULT '',
 updated_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(connection_id,path)
);
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='git_sync_runs'::regclass AND conname='git_sync_runs_job_id_fkey' AND confdeltype<>'n') THEN
  ALTER TABLE git_sync_runs DROP CONSTRAINT git_sync_runs_job_id_fkey;
  ALTER TABLE git_sync_runs ADD CONSTRAINT git_sync_runs_job_id_fkey FOREIGN KEY(job_id) REFERENCES automation_jobs(id) ON DELETE SET NULL;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='git_sync_mappings'::regclass AND conname='git_sync_mappings_document_id_fkey' AND confdeltype<>'c') THEN
  ALTER TABLE git_sync_mappings DROP CONSTRAINT git_sync_mappings_document_id_fkey;
  ALTER TABLE git_sync_mappings ADD CONSTRAINT git_sync_mappings_document_id_fkey FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE CASCADE;
 END IF;
 IF EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='git_sync_mappings'::regclass AND conname='git_sync_mappings_attachment_id_fkey' AND confdeltype<>'c') THEN
  ALTER TABLE git_sync_mappings DROP CONSTRAINT git_sync_mappings_attachment_id_fkey;
  ALTER TABLE git_sync_mappings ADD CONSTRAINT git_sync_mappings_attachment_id_fkey FOREIGN KEY(attachment_id) REFERENCES attachments(id) ON DELETE CASCADE;
 END IF;
END $$;
