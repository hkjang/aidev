-- v0.8.1 makes the daily AI limit a durable consume-before-call ledger rather
-- than a count-then-act check. Pending rows coordinate concurrent work across
-- all replicas; consumed rows retain usage independently of ai_calls logging.

CREATE TABLE IF NOT EXISTS ai_quota_reservations (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at timestamptz NOT NULL DEFAULT now(),
  expires_at timestamptz NOT NULL,
  consumed_at timestamptz,
  source_ai_call_id bigint REFERENCES ai_calls(id) ON DELETE SET NULL,
  CHECK (expires_at > created_at)
);

-- Keep a re-applied migration safe for prerelease databases that created the
-- first reservation-only shape before consumed usage became the source of
-- truth.
ALTER TABLE ai_quota_reservations
  ADD COLUMN IF NOT EXISTS consumed_at timestamptz,
  ADD COLUMN IF NOT EXISTS source_ai_call_id bigint REFERENCES ai_calls(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS ai_quota_reservations_user_expiry_idx
  ON ai_quota_reservations(user_id, expires_at);

CREATE INDEX IF NOT EXISTS ai_quota_reservations_expiry_idx
  ON ai_quota_reservations(expires_at);

CREATE UNIQUE INDEX IF NOT EXISTS ai_quota_reservations_source_call_idx
  ON ai_quota_reservations(source_ai_call_id);

-- Preserve the last 24 hours of usage when upgrading from v0.8.0. New calls
-- consume their quota row before contacting the gateway and do not depend on
-- ai_calls logging for enforcement.
INSERT INTO ai_quota_reservations(id,user_id,created_at,expires_at,consumed_at,source_ai_call_id)
SELECT gen_random_uuid(),user_id,created_at,created_at+interval '24 hours',created_at,id
FROM ai_calls
WHERE user_id IS NOT NULL AND created_at>now()-interval '24 hours'
ON CONFLICT(source_ai_call_id) DO NOTHING;
