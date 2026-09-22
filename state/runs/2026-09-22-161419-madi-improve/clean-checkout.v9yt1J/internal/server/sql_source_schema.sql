CREATE TABLE IF NOT EXISTS sql_sources(
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id),space_id uuid REFERENCES spaces(id),
 owner_id uuid NOT NULL REFERENCES users(id),service_account_id uuid NOT NULL REFERENCES users(id),
 name text NOT NULL,kind text NOT NULL,config jsonb NOT NULL DEFAULT '{}',credentials_ciphertext text NOT NULL,
 enabled boolean NOT NULL DEFAULT false,revision integer NOT NULL DEFAULT 1,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sql_source_tables(
 source_id uuid NOT NULL REFERENCES sql_sources(id) ON DELETE CASCADE,schema_name text NOT NULL,table_name text NOT NULL,
 columns jsonb NOT NULL DEFAULT '[]',description text NOT NULL DEFAULT '',document_ids jsonb NOT NULL DEFAULT '[]',inspected_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(source_id,schema_name,table_name)
);
CREATE TABLE IF NOT EXISTS sql_source_queries(
 id uuid PRIMARY KEY,source_id uuid NOT NULL REFERENCES sql_sources(id) ON DELETE CASCADE,owner_id uuid NOT NULL REFERENCES users(id),
 name text NOT NULL,plan jsonb NOT NULL,enabled boolean NOT NULL DEFAULT true,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sql_source_runs(
 id uuid PRIMARY KEY,source_id uuid NOT NULL REFERENCES sql_sources(id) ON DELETE CASCADE,query_id uuid REFERENCES sql_source_queries(id) ON DELETE SET NULL,
 user_id uuid NOT NULL REFERENCES users(id),status text NOT NULL,row_count integer NOT NULL DEFAULT 0,duration_ms bigint NOT NULL DEFAULT 0,
 message text NOT NULL DEFAULT '',created_at timestamptz NOT NULL DEFAULT now()
);
