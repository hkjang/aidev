CREATE TABLE IF NOT EXISTS teams (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 name text NOT NULL,created_at timestamptz NOT NULL DEFAULT now(),UNIQUE(workspace_id,name)
);
CREATE TABLE IF NOT EXISTS team_members (
 team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 PRIMARY KEY(team_id,user_id)
);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS parent_id uuid REFERENCES comments(id) ON DELETE CASCADE;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS block_id text NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN IF NOT EXISTS quote text NOT NULL DEFAULT '';
ALTER TABLE comments ADD COLUMN IF NOT EXISTS document_version integer;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS assigned_to uuid REFERENCES users(id);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS resolved_at timestamptz;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS resolved_by uuid REFERENCES users(id);
ALTER TABLE comments ADD COLUMN IF NOT EXISTS edited_at timestamptz;
ALTER TABLE comments ADD COLUMN IF NOT EXISTS deleted_at timestamptz;
CREATE INDEX IF NOT EXISTS comments_parent_idx ON comments(parent_id);
CREATE TABLE IF NOT EXISTS comment_reactions (
 comment_id uuid NOT NULL REFERENCES comments(id) ON DELETE CASCADE,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 reaction text NOT NULL CHECK(reaction IN ('like','thanks','idea','check','heart')),
 created_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(comment_id,user_id,reaction)
);
