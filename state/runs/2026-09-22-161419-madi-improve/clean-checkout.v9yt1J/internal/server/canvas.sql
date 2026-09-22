CREATE TABLE IF NOT EXISTS canvases (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id),
 space_id uuid REFERENCES spaces(id), owner_id uuid NOT NULL REFERENCES users(id),
 title text NOT NULL, visibility text NOT NULL DEFAULT 'private' CHECK(visibility IN ('private','workspace','selected')),
 data jsonb NOT NULL DEFAULT '{"nodes":[],"edges":[]}', version integer NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz
);
CREATE INDEX IF NOT EXISTS canvases_workspace_updated_idx ON canvases(workspace_id,updated_at DESC);
CREATE TABLE IF NOT EXISTS canvas_shares (
 canvas_id uuid NOT NULL REFERENCES canvases(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 permission text NOT NULL CHECK(permission IN ('read','write')), PRIMARY KEY(canvas_id,user_id)
);
CREATE OR REPLACE FUNCTION madi_canvas_allowed(actor uuid, target uuid, writing boolean)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM canvases c JOIN workspace_members m ON m.workspace_id=c.workspace_id
 JOIN users u ON u.id=m.user_id WHERE c.id=target AND u.id=actor AND NOT u.disabled
 AND (NOT writing OR (u.role<>'viewer' AND m.role IN ('owner','admin','editor')))
 AND madi_space_allowed(actor,c.space_id,writing)
 AND (c.visibility='workspace' OR c.owner_id=actor OR
   (c.visibility='selected' AND EXISTS(SELECT 1 FROM canvas_shares sh WHERE sh.canvas_id=c.id AND sh.user_id=actor AND (NOT writing OR sh.permission='write')))))
$$;
