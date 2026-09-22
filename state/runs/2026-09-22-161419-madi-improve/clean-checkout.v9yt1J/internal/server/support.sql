CREATE TABLE IF NOT EXISTS support_sessions (
 id uuid PRIMARY KEY, operator_id uuid NOT NULL REFERENCES users(id), target_id uuid NOT NULL REFERENCES users(id),
 session_hash text NOT NULL, reason text NOT NULL, status text NOT NULL DEFAULT 'active' CHECK(status IN ('active','ended','expired','revoked')),
 created_at timestamptz NOT NULL DEFAULT now(),expires_at timestamptz NOT NULL,ended_at timestamptz,
 CHECK(operator_id<>target_id),CHECK(expires_at>created_at)
);
CREATE INDEX IF NOT EXISTS support_sessions_operator ON support_sessions(operator_id,created_at DESC);
