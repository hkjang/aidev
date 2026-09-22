CREATE TABLE IF NOT EXISTS approval_policy_clock (
 id integer PRIMARY KEY CHECK(id=1),revision bigint NOT NULL DEFAULT 1
);
INSERT INTO approval_policy_clock(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE OR REPLACE FUNCTION madi_approval_policy_clock() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF (OLD.data->'approval_enabled') IS DISTINCT FROM (NEW.data->'approval_enabled') OR
    (OLD.data->'reviewer_role') IS DISTINCT FROM (NEW.data->'reviewer_role') THEN
  UPDATE approval_policy_clock SET revision=revision+1 WHERE id=1;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS settings_approval_policy_clock ON settings;
CREATE TRIGGER settings_approval_policy_clock AFTER UPDATE ON settings FOR EACH ROW EXECUTE FUNCTION madi_approval_policy_clock();

CREATE TABLE IF NOT EXISTS approval_policies (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 space_id uuid REFERENCES spaces(id) ON DELETE CASCADE,
 resource_kind text NOT NULL CHECK(resource_kind IN ('document','runbook','sql_query_plan','impact_exception','knowledge_distribution','learning_step')),
 name text NOT NULL,enabled boolean NOT NULL DEFAULT true,stages jsonb NOT NULL,
 version bigint NOT NULL DEFAULT 1,created_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS approval_policy_scope_idx ON approval_policies(workspace_id,COALESCE(space_id::text,''),resource_kind);
CREATE TABLE IF NOT EXISTS approval_requests (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 document_id uuid REFERENCES documents(id) ON DELETE CASCADE,
 resource_kind text NOT NULL CHECK(resource_kind IN ('document','runbook','sql_query_plan','impact_exception','knowledge_distribution','learning_step')),resource_id uuid NOT NULL,
 resource_version bigint NOT NULL,resource_hash text NOT NULL,snapshot jsonb NOT NULL,
 policy_id uuid REFERENCES approval_policies(id) ON DELETE SET NULL,policy_version bigint NOT NULL,policy_revision bigint NOT NULL,
 policy_snapshot jsonb NOT NULL,requester_id uuid NOT NULL REFERENCES users(id),owner_id uuid NOT NULL REFERENCES users(id),
 status text NOT NULL CHECK(status IN ('pending','approved','rejected','cancelled','superseded','consumed')),
 current_stage integer NOT NULL DEFAULT 0,version bigint NOT NULL DEFAULT 1,
 comment text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),completed_at timestamptz
);
ALTER TABLE approval_requests DROP CONSTRAINT IF EXISTS approval_requests_policy_id_fkey;
ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_policy_id_fkey FOREIGN KEY(policy_id) REFERENCES approval_policies(id) ON DELETE SET NULL;
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='approval_policies'::regclass AND conname='approval_policies_resource_kind_check' AND pg_get_constraintdef(oid) LIKE '%impact_exception%' AND pg_get_constraintdef(oid) LIKE '%knowledge_distribution%' AND pg_get_constraintdef(oid) LIKE '%learning_step%') THEN
  ALTER TABLE approval_policies DROP CONSTRAINT IF EXISTS approval_policies_resource_kind_check;
  ALTER TABLE approval_policies ADD CONSTRAINT approval_policies_resource_kind_check CHECK(resource_kind IN ('document','runbook','sql_query_plan','impact_exception','knowledge_distribution','learning_step'));
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='approval_requests'::regclass AND conname='approval_requests_resource_kind_check' AND pg_get_constraintdef(oid) LIKE '%impact_exception%' AND pg_get_constraintdef(oid) LIKE '%knowledge_distribution%' AND pg_get_constraintdef(oid) LIKE '%learning_step%') THEN
  ALTER TABLE approval_requests DROP CONSTRAINT IF EXISTS approval_requests_resource_kind_check;
  ALTER TABLE approval_requests ADD CONSTRAINT approval_requests_resource_kind_check CHECK(resource_kind IN ('document','runbook','sql_query_plan','impact_exception','knowledge_distribution','learning_step'));
 END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS approval_pending_resource_idx ON approval_requests(resource_kind,resource_id) WHERE status='pending';
CREATE INDEX IF NOT EXISTS approval_requests_document_idx ON approval_requests(document_id,created_at DESC);
CREATE TABLE IF NOT EXISTS approval_assignments (
 request_id uuid NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
 stage_index integer NOT NULL,gate_index integer NOT NULL,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 PRIMARY KEY(request_id,stage_index,gate_index,user_id)
);
CREATE INDEX IF NOT EXISTS approval_assignment_user_idx ON approval_assignments(user_id,request_id);
CREATE TABLE IF NOT EXISTS approval_decisions (
 id uuid PRIMARY KEY,request_id uuid NOT NULL REFERENCES approval_requests(id) ON DELETE CASCADE,
 stage_index integer NOT NULL,gate_index integer NOT NULL,user_id uuid NOT NULL REFERENCES users(id),
 decision text NOT NULL CHECK(decision IN ('approved','rejected')),
 comment text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(request_id,stage_index,gate_index)
);
