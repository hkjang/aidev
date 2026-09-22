CREATE TABLE IF NOT EXISTS knowledge_paths (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 space_id uuid REFERENCES spaces(id) ON DELETE CASCADE,owner_id uuid NOT NULL REFERENCES users(id),
 title text NOT NULL,description text NOT NULL DEFAULT '',role_labels text[] NOT NULL DEFAULT '{}',
 visibility text NOT NULL DEFAULT 'private' CHECK(visibility IN ('private','workspace')),
 archived boolean NOT NULL DEFAULT false,revision bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_paths_workspace ON knowledge_paths(workspace_id,updated_at DESC,id);
CREATE TABLE IF NOT EXISTS knowledge_path_steps (
 id uuid PRIMARY KEY,path_id uuid NOT NULL REFERENCES knowledge_paths(id) ON DELETE CASCADE,
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal>=0 AND ordinal<100),
 kind text NOT NULL CHECK(kind IN ('read','practice','review')),
 title text NOT NULL,instruction text NOT NULL DEFAULT '',active boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS knowledge_path_steps_order ON knowledge_path_steps(path_id,ordinal) WHERE active;
CREATE TABLE IF NOT EXISTS knowledge_path_progress (
 id uuid PRIMARY KEY,step_id uuid NOT NULL REFERENCES knowledge_path_steps(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 path_revision bigint NOT NULL,document_version integer NOT NULL,
 state text NOT NULL CHECK(state IN ('confirmed','pending_review','approved','rejected','needs_recheck')),
 revision bigint NOT NULL DEFAULT 1,proof text NOT NULL DEFAULT '',
 approval_id uuid REFERENCES approval_requests(id) ON DELETE SET NULL,
 confirmed_at timestamptz,updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(step_id,user_id)
);
CREATE TABLE IF NOT EXISTS knowledge_path_events (
 id uuid PRIMARY KEY,path_id uuid NOT NULL REFERENCES knowledge_paths(id) ON DELETE CASCADE,
 step_id uuid REFERENCES knowledge_path_steps(id) ON DELETE SET NULL,
 user_id uuid REFERENCES users(id) ON DELETE SET NULL,action text NOT NULL,
 metadata jsonb NOT NULL DEFAULT '{}',proof text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE knowledge_path_events ADD COLUMN IF NOT EXISTS proof text NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS knowledge_path_events_owner ON knowledge_path_events(path_id,user_id,created_at DESC,id);

CREATE OR REPLACE FUNCTION madi_knowledge_path_allowed(actor uuid,target uuid,writing boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM knowledge_paths p JOIN workspace_members m ON m.workspace_id=p.workspace_id AND m.user_id=actor JOIN users u ON u.id=m.user_id
 WHERE p.id=target AND NOT u.disabled AND (p.visibility='workspace' OR p.owner_id=actor)
 AND madi_space_allowed(actor,p.space_id,writing)
 AND (NOT writing OR (u.role<>'viewer' AND m.role IN ('owner','admin'))))
$$;
