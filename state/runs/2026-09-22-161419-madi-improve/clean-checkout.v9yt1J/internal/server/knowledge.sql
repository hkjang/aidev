CREATE TABLE IF NOT EXISTS knowledge_document_meta (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 kind text NOT NULL DEFAULT 'page' CHECK(kind IN ('page','note','daily','meeting','decision','runbook','template','inbox','entity')),
 classification text NOT NULL DEFAULT 'internal' CHECK(classification IN ('public','internal','confidential','restricted')),
 reviewer_id uuid REFERENCES users(id),maintainer_id uuid REFERENCES users(id),
 review_period_days integer CHECK(review_period_days BETWEEN 1 AND 3650),last_reviewed_at timestamptz,
 legal_hold boolean NOT NULL DEFAULT false,retain_until timestamptz,
 system_metadata jsonb NOT NULL DEFAULT '{}',updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_review_idx ON knowledge_document_meta(last_reviewed_at);
CREATE INDEX IF NOT EXISTS knowledge_hold_idx ON knowledge_document_meta(document_id) WHERE legal_hold;
CREATE TABLE IF NOT EXISTS knowledge_document_history (
 id uuid PRIMARY KEY,document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 user_id uuid NOT NULL REFERENCES users(id),document_version integer NOT NULL,
 action text NOT NULL CHECK(action IN ('policy','reviewed')),
 before_data jsonb NOT NULL,after_data jsonb NOT NULL,created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_history_document_idx ON knowledge_document_history(document_id,created_at DESC);
