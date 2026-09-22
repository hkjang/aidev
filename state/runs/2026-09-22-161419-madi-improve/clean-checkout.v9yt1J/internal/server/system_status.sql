CREATE TABLE IF NOT EXISTS system_status_policy (
 id integer PRIMARY KEY CHECK(id=1), revision bigint NOT NULL DEFAULT 1,
 enabled boolean NOT NULL DEFAULT false,
 max_ttl_seconds integer NOT NULL DEFAULT 86400 CHECK(max_ttl_seconds BETWEEN 60 AND 604800),
 max_observation_age_seconds integer NOT NULL DEFAULT 86400 CHECK(max_observation_age_seconds BETWEEN 60 AND 604800),
 retention_days integer NOT NULL DEFAULT 90 CHECK(retention_days BETWEEN 8 AND 3650),
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO system_status_policy(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS system_status_policy_history (
 revision bigint PRIMARY KEY, actor_id uuid REFERENCES users(id), policy jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO system_status_policy_history(revision,policy)
 SELECT revision,to_jsonb(p)-'id'-'updated_at' FROM system_status_policy p WHERE id=1 ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS system_status_cards (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 revision bigint NOT NULL DEFAULT 1, document_version integer NOT NULL,
 verification_epoch bigint NOT NULL DEFAULT 1, report_revision bigint NOT NULL DEFAULT 0,
 owner_id uuid NOT NULL REFERENCES users(id), reporter_ids uuid[] NOT NULL DEFAULT '{}',
 ttl_seconds integer NOT NULL CHECK(ttl_seconds BETWEEN 60 AND 604800),
 ciphertext text NOT NULL, updated_by uuid NOT NULL REFERENCES users(id),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS system_status_workspace_idx ON system_status_cards(workspace_id,updated_at DESC,document_id);
CREATE TABLE IF NOT EXISTS system_status_reports (
 id uuid PRIMARY KEY, document_id uuid NOT NULL REFERENCES system_status_cards(document_id) ON DELETE CASCADE,
 request_id uuid NOT NULL, payload_hash text NOT NULL,
 revision bigint NOT NULL, card_revision bigint NOT NULL, verification_epoch bigint NOT NULL,
 document_version integer NOT NULL, policy_revision bigint NOT NULL,
 actor_id uuid NOT NULL REFERENCES users(id), actor_kind text NOT NULL,
 token_id uuid REFERENCES api_keys(id) ON DELETE SET NULL,
 observed_at timestamptz NOT NULL, received_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 ciphertext text NOT NULL,
 UNIQUE(document_id,request_id), UNIQUE(document_id,revision)
);
CREATE INDEX IF NOT EXISTS system_status_reports_document_idx ON system_status_reports(document_id,revision DESC);
CREATE INDEX IF NOT EXISTS system_status_reports_retention_idx ON system_status_reports(received_at);
