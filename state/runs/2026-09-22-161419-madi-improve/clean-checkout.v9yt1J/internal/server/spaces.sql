CREATE TABLE IF NOT EXISTS organizations (
 id uuid PRIMARY KEY, name text NOT NULL, slug text UNIQUE NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS organization_members (
 organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 role text NOT NULL CHECK(role IN ('owner','admin','member')),
 PRIMARY KEY(organization_id,user_id)
);
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS organization_id uuid REFERENCES organizations(id);
CREATE TABLE IF NOT EXISTS spaces (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 parent_id uuid REFERENCES spaces(id), name text NOT NULL, slug text NOT NULL,
 icon text NOT NULL DEFAULT 'folder', visibility text NOT NULL DEFAULT 'workspace' CHECK(visibility IN ('workspace','restricted')),
 classification text NOT NULL DEFAULT 'internal' CHECK(classification IN ('public','internal','confidential','restricted')),
 owner_id uuid NOT NULL REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(workspace_id,slug)
);
CREATE INDEX IF NOT EXISTS spaces_parent_idx ON spaces(parent_id);
CREATE TABLE IF NOT EXISTS space_members (
 space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 role text NOT NULL CHECK(role IN ('admin','editor','commenter','viewer')),
 PRIMARY KEY(space_id,user_id)
);
ALTER TABLE documents ADD COLUMN IF NOT EXISTS space_id uuid REFERENCES spaces(id);
ALTER TABLE databases ADD COLUMN IF NOT EXISTS space_id uuid REFERENCES spaces(id);
CREATE INDEX IF NOT EXISTS documents_space_idx ON documents(space_id,updated_at DESC);
CREATE INDEX IF NOT EXISTS documents_parent_idx ON documents(parent_id);
CREATE TABLE IF NOT EXISTS workspace_settings (
 workspace_id uuid PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
 data jsonb NOT NULL DEFAULT '{}', version integer NOT NULL DEFAULT 1,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS workspace_settings_history (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id), version integer NOT NULL, data jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(workspace_id,version)
);

-- These are invoker-rights functions. They share one policy across REST, search,
-- attachments, AI, export and CRDT; neither service admin nor document owner can
-- bypass a restricted ancestor. API-key workspace/scope checks are additional.
CREATE OR REPLACE FUNCTION madi_space_allowed(actor uuid, target uuid, writing boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
 WITH RECURSIVE chain AS (
  SELECT s.*, ARRAY[s.id] AS visited, false AS cycle FROM spaces s WHERE s.id=target
  UNION ALL
  SELECT s.*, c.visited||s.id, s.id=ANY(c.visited) FROM spaces s JOIN chain c ON c.parent_id=s.id
  WHERE NOT c.cycle AND cardinality(c.visited)<21
 )
 SELECT CASE WHEN target IS NULL THEN true ELSE
  EXISTS(SELECT 1 FROM chain) AND NOT EXISTS(
   SELECT 1 FROM chain c WHERE c.cycle OR cardinality(c.visited)>20 OR
    NOT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id
     WHERE u.id=actor AND NOT u.disabled AND m.workspace_id=c.workspace_id
      AND (NOT writing OR (u.role<>'viewer' AND m.role IN ('owner','admin','editor')))) OR
    (c.visibility='restricted' AND NOT EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=c.id AND sm.user_id=actor)) OR
    (writing AND EXISTS(SELECT 1 FROM space_members sm WHERE sm.space_id=c.id AND sm.user_id=actor AND sm.role NOT IN ('admin','editor')))
  ) END
$$;

CREATE OR REPLACE FUNCTION madi_document_allowed(actor uuid, target uuid, writing boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
 WITH RECURSIVE chain AS (
  SELECT d.id,d.workspace_id,d.parent_id,d.space_id,d.visibility,d.owner_id,ARRAY[d.id] AS visited,false AS cycle
   FROM documents d WHERE d.id=target
  UNION ALL
  SELECT d.id,d.workspace_id,d.parent_id,d.space_id,d.visibility,d.owner_id,c.visited||d.id,d.id=ANY(c.visited)
   FROM documents d JOIN chain c ON d.id=c.parent_id WHERE NOT c.cycle AND cardinality(c.visited)<21
 )
 SELECT EXISTS(SELECT 1 FROM chain) AND NOT EXISTS(
  SELECT 1 FROM chain c WHERE c.cycle OR cardinality(c.visited)>20 OR
   NOT EXISTS(SELECT 1 FROM users u JOIN workspace_members m ON m.user_id=u.id
    WHERE u.id=actor AND NOT u.disabled AND m.workspace_id=c.workspace_id
     AND (NOT writing OR (u.role<>'viewer' AND m.role IN ('owner','admin','editor')))) OR
   NOT madi_space_allowed(actor,c.space_id,writing) OR
   NOT (c.visibility='workspace' OR c.owner_id=actor OR
    (c.visibility='selected' AND EXISTS(SELECT 1 FROM document_shares sh WHERE sh.document_id=c.id AND sh.user_id=actor AND (NOT writing OR sh.permission='write'))))
 )
$$;

CREATE OR REPLACE FUNCTION madi_space_comment_allowed(actor uuid,target uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
 WITH RECURSIVE chain AS (
  SELECT id,parent_id,ARRAY[id] path FROM spaces WHERE id=target
  UNION ALL SELECT s.id,s.parent_id,c.path||s.id FROM spaces s JOIN chain c ON s.id=c.parent_id WHERE NOT s.id=ANY(c.path) AND cardinality(c.path)<21
 ) SELECT madi_space_allowed(actor,target,false) AND NOT EXISTS(SELECT 1 FROM chain c JOIN space_members m ON m.space_id=c.id WHERE m.user_id=actor AND m.role='viewer')
$$;
CREATE OR REPLACE FUNCTION madi_document_comment_allowed(actor uuid,target uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
 WITH RECURSIVE chain AS (
  SELECT id,parent_id,space_id,workspace_id,ARRAY[id] path FROM documents WHERE id=target
  UNION ALL SELECT d.id,d.parent_id,d.space_id,d.workspace_id,c.path||d.id FROM documents d JOIN chain c ON d.id=c.parent_id WHERE NOT d.id=ANY(c.path) AND cardinality(c.path)<21
 ) SELECT madi_document_allowed(actor,target,false) AND EXISTS(SELECT 1 FROM users WHERE id=actor AND NOT disabled AND role<>'viewer') AND NOT EXISTS(
  SELECT 1 FROM chain c WHERE NOT madi_space_comment_allowed(actor,c.space_id) OR NOT EXISTS(SELECT 1 FROM workspace_members m WHERE m.workspace_id=c.workspace_id AND m.user_id=actor AND m.role<>'viewer')
 )
$$;
