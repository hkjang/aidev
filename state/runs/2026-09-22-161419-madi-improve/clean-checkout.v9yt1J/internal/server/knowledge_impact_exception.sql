CREATE TABLE IF NOT EXISTS knowledge_impact_exceptions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 review_id uuid NOT NULL REFERENCES knowledge_impact_reviews(id) ON DELETE CASCADE,
 requester_id uuid NOT NULL REFERENCES users(id),
 review_revision integer NOT NULL CHECK(review_revision>0),
 source_version integer NOT NULL CHECK(source_version>0),
 target_version integer NOT NULL CHECK(target_version>0),
 reason_ciphertext text NOT NULL, reason_hash text NOT NULL,
 valid_until timestamptz NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected','cancelled')),
 approval_id uuid REFERENCES approval_requests(id),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(), updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS knowledge_impact_exceptions_review ON knowledge_impact_exceptions(review_id,created_at DESC,id);
CREATE UNIQUE INDEX IF NOT EXISTS knowledge_impact_exceptions_pending ON knowledge_impact_exceptions(review_id) WHERE status='pending';
