CREATE TABLE IF NOT EXISTS plugins (
  id text PRIMARY KEY,
  manifest jsonb NOT NULL,
  files jsonb NOT NULL,
  enabled boolean NOT NULL DEFAULT true,
  version integer NOT NULL DEFAULT 1,
  installed_by uuid REFERENCES users(id) ON DELETE SET NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS workspace_plugins (
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  plugin_id text NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
  enabled boolean NOT NULL DEFAULT false,
  capabilities jsonb NOT NULL DEFAULT '[]',
  version integer NOT NULL DEFAULT 1,
  updated_by uuid REFERENCES users(id) ON DELETE SET NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(workspace_id,plugin_id)
);
CREATE TABLE IF NOT EXISTS plugin_storage (
  workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
  plugin_id text NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  key text NOT NULL,
  value jsonb NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(workspace_id,plugin_id,user_id,key)
);
