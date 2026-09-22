CREATE TABLE IF NOT EXISTS connector_settings(id integer PRIMARY KEY CHECK(id=1),enabled boolean NOT NULL DEFAULT false,allowed_hosts jsonb NOT NULL DEFAULT '[]');
INSERT INTO connector_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS connector_configs(
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,space_id uuid REFERENCES spaces(id),
 owner_id uuid NOT NULL REFERENCES users(id),service_account_id uuid NOT NULL REFERENCES users(id),
 name text NOT NULL,kind text NOT NULL,base_url text NOT NULL,credentials_ciphertext text NOT NULL DEFAULT '',
 config jsonb NOT NULL DEFAULT '{}',enabled boolean NOT NULL DEFAULT false,revision integer NOT NULL DEFAULT 1,
 cursor text NOT NULL DEFAULT '',interval_minutes integer NOT NULL DEFAULT 0 CHECK(interval_minutes BETWEEN 0 AND 525600),next_run timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS connector_records(
 connector_id uuid NOT NULL REFERENCES connector_configs(id) ON DELETE CASCADE,remote_id text NOT NULL,
 document_id uuid REFERENCES documents(id) ON DELETE SET NULL,source_url text NOT NULL DEFAULT '',checksum text NOT NULL DEFAULT '',
 document_version integer NOT NULL DEFAULT 0,last_seen_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(connector_id,remote_id)
);
CREATE TABLE IF NOT EXISTS connector_runs(
 job_id uuid PRIMARY KEY REFERENCES automation_jobs(id) ON DELETE CASCADE,connector_id uuid NOT NULL REFERENCES connector_configs(id) ON DELETE CASCADE,
 revision integer NOT NULL,cursor text NOT NULL DEFAULT '',report jsonb NOT NULL DEFAULT '{}',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
