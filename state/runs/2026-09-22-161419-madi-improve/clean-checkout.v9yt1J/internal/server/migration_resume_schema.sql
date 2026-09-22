CREATE TABLE IF NOT EXISTS migration_settings (
 id integer PRIMARY KEY CHECK(id=1),
 max_session_bytes bigint NOT NULL DEFAULT 1073741824 CHECK(max_session_bytes BETWEEN 1048576 AND 8589934592),
 max_user_bytes bigint NOT NULL DEFAULT 2147483648 CHECK(max_user_bytes BETWEEN 1048576 AND 17179869184),
 max_items integer NOT NULL DEFAULT 50000 CHECK(max_items BETWEEN 1 AND 100000),
 retention_days integer NOT NULL DEFAULT 7 CHECK(retention_days BETWEEN 1 AND 30),
 revision bigint NOT NULL DEFAULT 1
);
INSERT INTO migration_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS migration_sessions (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id), space_id uuid REFERENCES spaces(id),
 source_key text NOT NULL, label text NOT NULL,
 format text NOT NULL CHECK(format IN ('markdown','obsidian','notion','html','csv','json')),
 status text NOT NULL DEFAULT 'uploading' CHECK(status IN ('uploading','preparing','ready','committing','completed','failed','cancelled')),
 revision bigint NOT NULL DEFAULT 1, plan_hash text NOT NULL DEFAULT '',
 declared_bytes bigint NOT NULL DEFAULT 0, uploaded_bytes bigint NOT NULL DEFAULT 0,
 item_count integer NOT NULL DEFAULT 0, prepared_count integer NOT NULL DEFAULT 0,
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,
 request_session_hash text NOT NULL DEFAULT '', request_ip text NOT NULL DEFAULT '', request_token_hash text NOT NULL DEFAULT '',
 report jsonb NOT NULL DEFAULT '{}', error text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days', completed_at timestamptz, purged_at timestamptz
);
CREATE INDEX IF NOT EXISTS migration_sessions_owner_idx ON migration_sessions(workspace_id,user_id,created_at DESC);
ALTER TABLE migration_sessions ADD COLUMN IF NOT EXISTS purged_at timestamptz;
ALTER TABLE migration_sessions ADD COLUMN IF NOT EXISTS request_session_hash text NOT NULL DEFAULT '';
ALTER TABLE migration_sessions ADD COLUMN IF NOT EXISTS request_ip text NOT NULL DEFAULT '';
ALTER TABLE migration_sessions ADD COLUMN IF NOT EXISTS request_token_hash text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS migration_session_items (
 id uuid PRIMARY KEY, session_id uuid NOT NULL REFERENCES migration_sessions(id) ON DELETE CASCADE,
 source_id text NOT NULL, source_hash text NOT NULL, file_path text NOT NULL,
 kind text NOT NULL CHECK(kind IN ('document','attachment','csv','folder')),
 source_bytes bigint NOT NULL, chunk_count integer NOT NULL, received_bytes bigint NOT NULL DEFAULT 0,
 status text NOT NULL DEFAULT 'uploading' CHECK(status IN ('uploading','pending','prepared','failed')),
 checkpoint integer NOT NULL DEFAULT 0, target_id uuid NOT NULL,
 disposition text NOT NULL DEFAULT 'new' CHECK(disposition IN ('new','changed','unchanged','conflict')),
 expected_version bigint NOT NULL DEFAULT 0, binding_revision bigint NOT NULL DEFAULT 0, target_hash text NOT NULL DEFAULT '',
 parent_source_id text NOT NULL DEFAULT '', metadata jsonb NOT NULL DEFAULT '{}',
 prepared_data bytea, staged_object jsonb NOT NULL DEFAULT '{}', storage_fingerprint text NOT NULL DEFAULT '', compatibility jsonb NOT NULL DEFAULT '{}',
 csv_types jsonb NOT NULL DEFAULT '[]', types_confirmed boolean NOT NULL DEFAULT false,
 error text NOT NULL DEFAULT '', updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(session_id,source_id), UNIQUE(session_id,file_path)
);
CREATE INDEX IF NOT EXISTS migration_session_items_checkpoint_idx ON migration_session_items(session_id,status,id);
ALTER TABLE migration_session_items ADD COLUMN IF NOT EXISTS staged_object jsonb NOT NULL DEFAULT '{}';
ALTER TABLE migration_session_items ADD COLUMN IF NOT EXISTS storage_fingerprint text NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS migration_session_objects (
 item_id uuid NOT NULL REFERENCES migration_session_items(id) ON DELETE CASCADE,
 document_id uuid NOT NULL, attachment_id uuid NOT NULL,
 object jsonb NOT NULL, provider_fingerprint text NOT NULL,
 status text NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','ready','claimed','discarded')),
 created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(item_id,document_id)
);
CREATE TABLE IF NOT EXISTS migration_session_chunks (
 item_id uuid NOT NULL REFERENCES migration_session_items(id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal>=0), size integer NOT NULL CHECK(size BETWEEN 0 AND 1048576),
 checksum text NOT NULL, data bytea NOT NULL, PRIMARY KEY(item_id,ordinal)
);
CREATE TABLE IF NOT EXISTS migration_source_bindings (
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id), source_key text NOT NULL, source_id text NOT NULL,
 kind text NOT NULL, target_id uuid NOT NULL, source_hash text NOT NULL,
 target_version bigint NOT NULL, target_hash text NOT NULL DEFAULT '', revision bigint NOT NULL DEFAULT 1,
 session_id uuid REFERENCES migration_sessions(id) ON DELETE SET NULL,
 updated_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(workspace_id,user_id,source_key,source_id)
);
