CREATE TABLE IF NOT EXISTS user_feature_flags (
 user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
 data jsonb NOT NULL DEFAULT '{}', version bigint NOT NULL DEFAULT 1,
 updated_by uuid NOT NULL REFERENCES users(id), updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK(jsonb_typeof(data)='object'), CHECK(version>0)
);
CREATE TABLE IF NOT EXISTS user_feature_flags_history (
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 version bigint NOT NULL,data jsonb NOT NULL,actor_id uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(user_id,version)
);
CREATE TABLE IF NOT EXISTS workspace_brand_assets (
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 digest text NOT NULL, data bytea NOT NULL, width integer NOT NULL,height integer NOT NULL,
 created_by uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(workspace_id,digest),CHECK(digest~'^[0-9a-f]{64}$'),CHECK(octet_length(data)<=524288),
 CHECK(width BETWEEN 1 AND 2048),CHECK(height BETWEEN 1 AND 2048)
);
CREATE TABLE IF NOT EXISTS operations_http_errors (
 request_id uuid PRIMARY KEY,route text NOT NULL,method text NOT NULL,status integer NOT NULL,
 duration_ms bigint NOT NULL,kind text NOT NULL DEFAULT 'http',created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(status BETWEEN 400 AND 599),CHECK(kind IN ('http','panic'))
);
CREATE INDEX IF NOT EXISTS operations_http_errors_created_idx ON operations_http_errors(created_at DESC);
CREATE OR REPLACE FUNCTION madi_feature_allowed(actor uuid,wid uuid,feature text)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT feature IN ('canvas','plugins','collaboration','database-formula','ai-graph','workspace-agents','document-queries') AND coalesce((
 SELECT coalesce((s.data->'feature_flags'->>feature)::boolean,true)
 AND coalesce((ws.data->'feature_flags'->>feature)::boolean,true)
 AND coalesce((uf.data->>feature)::boolean,true)
 FROM users u JOIN workspace_members m ON m.user_id=u.id AND m.workspace_id=wid
 CROSS JOIN settings s LEFT JOIN workspace_settings ws ON ws.workspace_id=m.workspace_id
 LEFT JOIN user_feature_flags uf ON uf.user_id=u.id WHERE u.id=actor AND NOT u.disabled AND s.id=1
 ),false)
$$;
