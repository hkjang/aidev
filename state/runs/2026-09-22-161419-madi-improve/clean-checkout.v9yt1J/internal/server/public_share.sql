CREATE TABLE IF NOT EXISTS public_shares (
 id uuid PRIMARY KEY,document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id),token_hash text NOT NULL UNIQUE,password_hash text NOT NULL DEFAULT '',
 expires_at timestamptz NOT NULL,ip_allowlist jsonb NOT NULL DEFAULT '[]',allow_download boolean NOT NULL DEFAULT false,
 allow_copy boolean NOT NULL DEFAULT true,revision bigint NOT NULL DEFAULT 1,revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS public_shares_document_idx ON public_shares(document_id,created_at DESC);
CREATE TABLE IF NOT EXISTS public_share_visits (
 id uuid PRIMARY KEY,share_id uuid NOT NULL REFERENCES public_shares(id) ON DELETE CASCADE,
 action text NOT NULL,ip_hash text NOT NULL,result text NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS public_share_access (
 token_hash text PRIMARY KEY,share_id uuid NOT NULL REFERENCES public_shares(id) ON DELETE CASCADE,
 revision bigint NOT NULL,ip_hash text NOT NULL,expires_at timestamptz NOT NULL
);
CREATE TABLE IF NOT EXISTS public_share_rate_limits (
 bucket text PRIMARY KEY,count integer NOT NULL,expires_at timestamptz NOT NULL
);
