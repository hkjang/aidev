CREATE TABLE IF NOT EXISTS users (
 id uuid PRIMARY KEY, email text UNIQUE NOT NULL, name text NOT NULL,
 password_hash text NOT NULL DEFAULT '', role text NOT NULL DEFAULT 'editor' CHECK(role IN ('admin','editor','viewer')),
 kind text NOT NULL DEFAULT 'user' CHECK(kind IN ('user','service')), disabled boolean NOT NULL DEFAULT false,
 preferences jsonb NOT NULL DEFAULT '{}', oidc_subject text UNIQUE, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sessions (token_hash text PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS settings (id integer PRIMARY KEY CHECK(id=1), data jsonb NOT NULL DEFAULT '{}');
INSERT INTO settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS settings_history (id uuid PRIMARY KEY, user_id uuid REFERENCES users(id), data jsonb NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS workspaces (id uuid PRIMARY KEY, name text NOT NULL, slug text UNIQUE NOT NULL, created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS workspace_members (workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,role text NOT NULL CHECK(role IN ('owner','admin','editor','commenter','viewer')),PRIMARY KEY(workspace_id,user_id));
CREATE TABLE IF NOT EXISTS documents (
 id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,parent_id uuid REFERENCES documents(id),
 title text NOT NULL,markdown text NOT NULL DEFAULT '',tags jsonb NOT NULL DEFAULT '[]',aliases jsonb NOT NULL DEFAULT '[]',icon text NOT NULL DEFAULT 'file',
 status text NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','review','published','rejected','stale','archived')),
 visibility text NOT NULL DEFAULT 'workspace' CHECK(visibility IN ('workspace','private','selected')),
 owner_id uuid NOT NULL REFERENCES users(id),version integer NOT NULL DEFAULT 1,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),deleted_at timestamptz,
 search_vector tsvector GENERATED ALWAYS AS (setweight(to_tsvector('simple',coalesce(title,'')),'A') || setweight(to_tsvector('simple',coalesce(markdown,'')),'B')) STORED
);
CREATE INDEX IF NOT EXISTS documents_workspace_idx ON documents(workspace_id,updated_at DESC);
CREATE INDEX IF NOT EXISTS documents_search_idx ON documents USING GIN(search_vector);
CREATE TABLE IF NOT EXISTS document_shares (document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,permission text NOT NULL CHECK(permission IN('read','write')),PRIMARY KEY(document_id,user_id));
CREATE TABLE IF NOT EXISTS document_versions (document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,version integer NOT NULL,title text NOT NULL,markdown text NOT NULL,tags jsonb NOT NULL DEFAULT '[]',user_id uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(document_id,version));
CREATE TABLE IF NOT EXISTS favorites (user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,PRIMARY KEY(user_id,document_id));
CREATE TABLE IF NOT EXISTS comments (id uuid PRIMARY KEY,document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,user_id uuid NOT NULL REFERENCES users(id),body text NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS attachments (id uuid PRIMARY KEY,document_id uuid NOT NULL REFERENCES documents(id),user_id uuid NOT NULL REFERENCES users(id),name text NOT NULL,content_type text NOT NULL,size bigint NOT NULL,path text NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS databases (id uuid PRIMARY KEY,workspace_id uuid NOT NULL REFERENCES workspaces(id),name text NOT NULL,properties jsonb NOT NULL DEFAULT '[]',created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS database_rows (id uuid PRIMARY KEY,database_id uuid NOT NULL REFERENCES databases(id) ON DELETE CASCADE,values jsonb NOT NULL DEFAULT '{}',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS audit_logs (id uuid PRIMARY KEY,user_id uuid REFERENCES users(id),action text NOT NULL,resource text NOT NULL DEFAULT '',ip text NOT NULL DEFAULT '',details jsonb NOT NULL DEFAULT '{}',created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS audit_created_idx ON audit_logs(created_at DESC);
CREATE TABLE IF NOT EXISTS notifications (id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,title text NOT NULL,document_id uuid REFERENCES documents(id),read_at timestamptz,created_at timestamptz NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS schema_migrations (version integer PRIMARY KEY,applied_at timestamptz NOT NULL DEFAULT now());
INSERT INTO schema_migrations(version) VALUES(1) ON CONFLICT DO NOTHING;
ALTER TABLE documents ADD COLUMN IF NOT EXISTS block_metadata jsonb NOT NULL DEFAULT '{}';
ALTER TABLE document_versions ADD COLUMN IF NOT EXISTS block_metadata jsonb NOT NULL DEFAULT '{}';
