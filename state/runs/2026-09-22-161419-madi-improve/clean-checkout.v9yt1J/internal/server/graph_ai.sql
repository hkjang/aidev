CREATE TABLE IF NOT EXISTS graph_ai_runs (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id),
 owner_id uuid NOT NULL REFERENCES users(id), token_id uuid REFERENCES api_keys(id) ON DELETE SET NULL,
 token_bound boolean NOT NULL DEFAULT false, actor_constraints jsonb NOT NULL DEFAULT '{}',
 session_hash text NOT NULL DEFAULT '', provider_fingerprint text NOT NULL,
 kinds text[] NOT NULL, status text NOT NULL CHECK(status IN ('running','ready','failed','cancelled')),
 error text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(),
 heartbeat_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL DEFAULT now()+interval '10 minutes',
 finished_at timestamptz
);
CREATE INDEX IF NOT EXISTS graph_ai_runs_owner_idx ON graph_ai_runs(owner_id,created_at DESC);
CREATE INDEX IF NOT EXISTS graph_ai_runs_heartbeat_idx ON graph_ai_runs(heartbeat_at) WHERE status='running';
CREATE TABLE IF NOT EXISTS graph_ai_sources (
 run_id uuid NOT NULL REFERENCES graph_ai_runs(id) ON DELETE CASCADE,
 document_id uuid NOT NULL, document_version integer NOT NULL,
 document_hash text NOT NULL, start_byte integer NOT NULL, end_byte integer NOT NULL,
 content_hash text NOT NULL, PRIMARY KEY(run_id,document_id)
);
CREATE TABLE IF NOT EXISTS graph_ai_actions (
 id uuid PRIMARY KEY, run_id uuid NOT NULL REFERENCES graph_ai_runs(id) ON DELETE CASCADE,
 kind text NOT NULL CHECK(kind IN ('relation','duplicate','topic','entity','gap')),
 payload jsonb NOT NULL, action_hash text NOT NULL,
 status text NOT NULL DEFAULT 'proposed' CHECK(status IN ('proposed','applied','rejected','cancelled')),
 result jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now(), decided_at timestamptz
);
CREATE INDEX IF NOT EXISTS graph_ai_actions_run_idx ON graph_ai_actions(run_id,created_at,id);
CREATE TABLE IF NOT EXISTS graph_ai_annotations (
 id uuid PRIMARY KEY, document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 action_id uuid UNIQUE REFERENCES graph_ai_actions(id) ON DELETE SET NULL,
 kind text NOT NULL CHECK(kind IN ('topic','relation','duplicate')),
 value jsonb NOT NULL, created_by uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now()
);
