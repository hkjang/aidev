CREATE TABLE IF NOT EXISTS runbook_settings (
 id integer PRIMARY KEY CHECK(id=1),enabled boolean NOT NULL DEFAULT false,revision bigint NOT NULL DEFAULT 1
);
INSERT INTO runbook_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS runbook_runners (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name text NOT NULL,kind text NOT NULL CHECK(kind IN ('awx','kubernetes')),
 enabled boolean NOT NULL DEFAULT false,revision bigint NOT NULL DEFAULT 1,
 created_by uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS runbook_runner_versions (
 runner_id uuid NOT NULL REFERENCES runbook_runners(id) ON DELETE CASCADE,revision bigint NOT NULL,
 config jsonb NOT NULL,actions jsonb NOT NULL,created_by uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(runner_id,revision)
);
CREATE TABLE IF NOT EXISTS runbook_documents (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,version bigint NOT NULL DEFAULT 1,
 purpose text NOT NULL DEFAULT '',prerequisites text NOT NULL DEFAULT '',validation text NOT NULL DEFAULT '',rollback text NOT NULL DEFAULT '',
 steps jsonb NOT NULL DEFAULT '[]',validation_steps jsonb NOT NULL DEFAULT '[]',rollback_steps jsonb NOT NULL DEFAULT '[]',
 last_tested_at timestamptz,last_tested_by uuid REFERENCES users(id),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS runbook_executions (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 document_id uuid REFERENCES documents(id) ON DELETE SET NULL,owner_id uuid NOT NULL REFERENCES users(id),
 phase text NOT NULL CHECK(phase IN ('execute','validate','rollback')),version bigint NOT NULL DEFAULT 1,
 snapshot jsonb NOT NULL,approval_required boolean NOT NULL,approval_id uuid REFERENCES approval_requests(id) ON DELETE SET NULL,
 job_id uuid REFERENCES automation_jobs(id) ON DELETE SET NULL,status text NOT NULL DEFAULT 'prepared' CHECK(status IN ('prepared','review','approved','rejected','cancelled','queued','running','succeeded','failed','unknown')),
 cancel_requested boolean NOT NULL DEFAULT false,log_bytes bigint NOT NULL DEFAULT 0,last_error text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),completed_at timestamptz
);
CREATE INDEX IF NOT EXISTS runbook_execution_document_idx ON runbook_executions(document_id,created_at DESC);
ALTER TABLE runbook_executions ADD COLUMN IF NOT EXISTS reconcile_after timestamptz NOT NULL DEFAULT now();
CREATE TABLE IF NOT EXISTS runbook_execution_steps (
 execution_id uuid NOT NULL REFERENCES runbook_executions(id) ON DELETE CASCADE,step_index integer NOT NULL,
 runner_id uuid NOT NULL REFERENCES runbook_runners(id),runner_revision bigint NOT NULL,action_id text NOT NULL,
 state text NOT NULL DEFAULT 'queued' CHECK(state IN ('queued','launching','running','succeeded','failed','cancelled','unknown')),
 external_id text NOT NULL DEFAULT '',log_cursor bigint NOT NULL DEFAULT 0,last_error text NOT NULL DEFAULT '',
 started_at timestamptz,completed_at timestamptz,PRIMARY KEY(execution_id,step_index)
);
CREATE TABLE IF NOT EXISTS runbook_events (
 id bigserial PRIMARY KEY,execution_id uuid NOT NULL REFERENCES runbook_executions(id) ON DELETE CASCADE,
 step_index integer NOT NULL DEFAULT -1,kind text NOT NULL,message text NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS runbook_event_execution_idx ON runbook_events(execution_id,id);
