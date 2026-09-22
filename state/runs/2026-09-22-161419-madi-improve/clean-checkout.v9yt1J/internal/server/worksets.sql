CREATE TABLE IF NOT EXISTS user_worksets (
 id uuid PRIMARY KEY,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL CHECK(octet_length(name) BETWEEN 1 AND 200),
 kind text NOT NULL CHECK(kind IN ('workset','reference')),
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS user_worksets_owner_workspace_idx ON user_worksets(owner_id,workspace_id,updated_at DESC);
CREATE TABLE IF NOT EXISTS workset_items (
 workset_id uuid NOT NULL REFERENCES user_worksets(id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal BETWEEN 0 AND 19),
 kind text NOT NULL CHECK(kind IN ('document','database','task')),
 resource_id uuid NOT NULL,
 context jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(context)='object' AND octet_length(context::text)<=32768),
 PRIMARY KEY(workset_id,ordinal)
);
