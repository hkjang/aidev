CREATE TABLE IF NOT EXISTS job_settings (
 id integer PRIMARY KEY CHECK(id=1), paused boolean NOT NULL DEFAULT false,
 concurrency integer NOT NULL DEFAULT 2 CHECK(concurrency BETWEEN 1 AND 16),
 retention_days integer NOT NULL DEFAULT 90 CHECK(retention_days BETWEEN 1 AND 3650)
);
INSERT INTO job_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS automation_events (
 id uuid PRIMARY KEY, type text NOT NULL, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), token_id uuid, resource_id text NOT NULL DEFAULT '',
 resource_type text NOT NULL DEFAULT 'document', depth integer NOT NULL DEFAULT 0,
 job_id text NOT NULL DEFAULT '', automation_id text NOT NULL DEFAULT '', payload jsonb NOT NULL DEFAULT '{}',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS automation_events_workspace_idx ON automation_events(workspace_id,created_at DESC);
CREATE TABLE IF NOT EXISTS automation_jobs (
 id uuid PRIMARY KEY, kind text NOT NULL, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id), actor_id uuid NOT NULL REFERENCES users(id), token_id uuid,
 event_id uuid REFERENCES automation_events(id) ON DELETE SET NULL, resource_id text NOT NULL DEFAULT '',
 automation_id text NOT NULL DEFAULT '', target_id text NOT NULL DEFAULT '', depth integer NOT NULL DEFAULT 0,
 payload jsonb NOT NULL DEFAULT '{}', result jsonb NOT NULL DEFAULT '{}',
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','running','succeeded','failed','cancelled')),
 attempts integer NOT NULL DEFAULT 0, max_attempts integer NOT NULL DEFAULT 5 CHECK(max_attempts BETWEEN 1 AND 10),
 timeout_seconds integer NOT NULL DEFAULT 120 CHECK(timeout_seconds BETWEEN 1 AND 3600),
 run_after timestamptz NOT NULL DEFAULT now(), lease_until timestamptz, lease_id uuid,
 cancel_requested boolean NOT NULL DEFAULT false, last_error text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), finished_at timestamptz,
 UNIQUE(event_id,kind,target_id)
);
CREATE INDEX IF NOT EXISTS automation_jobs_claim_idx ON automation_jobs(status,run_after,lease_until);
CREATE INDEX IF NOT EXISTS automation_jobs_workspace_idx ON automation_jobs(workspace_id,created_at DESC);
CREATE TABLE IF NOT EXISTS automation_job_attempts (
 id bigserial PRIMARY KEY, job_id uuid NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE,
 attempt integer NOT NULL, status text NOT NULL, error text NOT NULL DEFAULT '', result jsonb NOT NULL DEFAULT '{}',
 duration_ms bigint NOT NULL DEFAULT 0, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS automation_webhooks (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id), name text NOT NULL, url text NOT NULL,
 secret_ciphertext text NOT NULL, events jsonb NOT NULL DEFAULT '[]', enabled boolean NOT NULL DEFAULT true,
 max_attempts integer NOT NULL DEFAULT 5 CHECK(max_attempts BETWEEN 1 AND 10),
 timeout_seconds integer NOT NULL DEFAULT 15 CHECK(timeout_seconds BETWEEN 1 AND 120),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS automation_rules (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id), token_id uuid, name text NOT NULL, enabled boolean NOT NULL DEFAULT true,
 trigger text NOT NULL, conditions jsonb NOT NULL DEFAULT '{}', actions jsonb NOT NULL DEFAULT '[]',
 schedule_at timestamptz, interval_minutes integer NOT NULL DEFAULT 0 CHECK(interval_minutes BETWEEN 0 AND 525600),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS automation_rules_due_idx ON automation_rules(schedule_at) WHERE enabled AND trigger='date.reached';
ALTER TABLE automation_rules ADD COLUMN IF NOT EXISTS revision integer NOT NULL DEFAULT 1;
CREATE TABLE IF NOT EXISTS automation_effects (
 job_id uuid NOT NULL REFERENCES automation_jobs(id) ON DELETE CASCADE, action_index integer NOT NULL,
 result jsonb NOT NULL DEFAULT '{}', created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(job_id,action_index)
);
ALTER TABLE automation_events ADD COLUMN IF NOT EXISTS actor_constraints jsonb NOT NULL DEFAULT '{}';
ALTER TABLE automation_jobs ADD COLUMN IF NOT EXISTS actor_constraints jsonb NOT NULL DEFAULT '{}';
-- Missing legacy credential provenance fails closed for restricted jobs.
UPDATE automation_jobs j SET actor_constraints=e.actor_constraints FROM automation_events e WHERE j.event_id=e.id AND j.actor_constraints='{}' AND e.actor_constraints<>'{}';
UPDATE automation_events SET actor_constraints=jsonb_set(actor_constraints,'{token_bound}',to_jsonb(token_id IS NOT NULL OR coalesce(actor_constraints->'scope_restricted'='true'::jsonb,false)),true) WHERE NOT actor_constraints?'token_bound';
UPDATE automation_jobs SET actor_constraints=jsonb_set(actor_constraints,'{token_bound}',to_jsonb(token_id IS NOT NULL OR coalesce(actor_constraints->'scope_restricted'='true'::jsonb,false)),true) WHERE NOT actor_constraints?'token_bound';
