CREATE TABLE IF NOT EXISTS document_templates (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 space_id uuid REFERENCES spaces(id) ON DELETE RESTRICT, owner_id uuid NOT NULL REFERENCES users(id),
 name text NOT NULL, description text NOT NULL DEFAULT '', category text NOT NULL DEFAULT '',
 icon text NOT NULL DEFAULT 'file', kind text NOT NULL DEFAULT 'page', markdown text NOT NULL DEFAULT '',
 tags jsonb NOT NULL DEFAULT '[]', visibility text NOT NULL DEFAULT 'private',
 version integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 CHECK(visibility IN ('private','workspace','space')),
 CHECK(visibility<>'space' OR space_id IS NOT NULL),
 CHECK(kind IN ('page','note','daily','meeting','decision','runbook')),
 CHECK(version>0), CHECK(jsonb_typeof(tags)='array')
);
CREATE INDEX IF NOT EXISTS document_templates_workspace_idx ON document_templates(workspace_id,updated_at DESC,id);
CREATE INDEX IF NOT EXISTS document_templates_owner_idx ON document_templates(owner_id,updated_at DESC);
CREATE TABLE IF NOT EXISTS document_template_versions (
 template_id uuid NOT NULL REFERENCES document_templates(id) ON DELETE CASCADE,
 version integer NOT NULL, user_id uuid NOT NULL REFERENCES users(id),
 data jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(template_id,version)
);
CREATE OR REPLACE FUNCTION madi_template_allowed(actor uuid,target uuid,writing boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM document_templates t
 JOIN workspace_members m ON m.workspace_id=t.workspace_id AND m.user_id=actor
 JOIN users u ON u.id=actor AND NOT u.disabled
 WHERE t.id=target AND (t.visibility<>'private' OR t.owner_id=actor)
 AND madi_space_allowed(actor,t.space_id,writing)
 AND (NOT writing OR (t.owner_id=actor AND u.role<>'viewer' AND m.role IN ('owner','admin','editor'))))
$$;
