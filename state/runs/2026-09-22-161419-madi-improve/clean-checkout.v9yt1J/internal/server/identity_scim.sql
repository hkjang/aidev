CREATE TABLE IF NOT EXISTS scim_users (
 id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 username text NOT NULL, external_id text NOT NULL DEFAULT '', profile jsonb NOT NULL,
 active boolean NOT NULL DEFAULT true, version bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(), deleted_at timestamptz,
 UNIQUE(workspace_id,username), UNIQUE(workspace_id,id)
);
CREATE TABLE IF NOT EXISTS scim_groups (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 display_name text NOT NULL, external_id text NOT NULL DEFAULT '', version bigint NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(workspace_id,display_name), UNIQUE(workspace_id,id)
);
CREATE TABLE IF NOT EXISTS scim_group_members (
 workspace_id uuid NOT NULL,group_id uuid NOT NULL,user_id uuid NOT NULL,
 PRIMARY KEY(group_id,user_id),
 FOREIGN KEY(workspace_id,group_id) REFERENCES scim_groups(workspace_id,id) ON DELETE CASCADE,
 FOREIGN KEY(workspace_id,user_id) REFERENCES scim_users(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS scim_users_workspace_idx ON scim_users(workspace_id,updated_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS scim_groups_workspace_idx ON scim_groups(workspace_id,updated_at);
CREATE UNIQUE INDEX IF NOT EXISTS scim_users_name_case_idx ON scim_users(workspace_id,lower(username));
CREATE UNIQUE INDEX IF NOT EXISTS scim_groups_name_case_idx ON scim_groups(workspace_id,lower(display_name));
