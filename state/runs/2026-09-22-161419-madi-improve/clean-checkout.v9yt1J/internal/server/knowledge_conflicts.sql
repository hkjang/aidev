CREATE TABLE IF NOT EXISTS knowledge_conflict_runs (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id),
 rule_version text NOT NULL,
 source_refs jsonb NOT NULL CHECK(jsonb_typeof(source_refs)='array'),
 diagnostics jsonb NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_conflict_runs_owner ON knowledge_conflict_runs(owner_id,workspace_id,created_at DESC);
CREATE TABLE IF NOT EXISTS knowledge_conflict_candidates (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 run_id uuid NOT NULL REFERENCES knowledge_conflict_runs(id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal BETWEEN 0 AND 63),
 left_id uuid NOT NULL, right_id uuid NOT NULL,
 left_version integer NOT NULL CHECK(left_version>0), right_version integer NOT NULL CHECK(right_version>0),
 left_hash text NOT NULL, right_hash text NOT NULL,
 ciphertext text NOT NULL, payload_hash text NOT NULL,
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 decision text NOT NULL DEFAULT 'needs_review' CHECK(decision IN ('conflict','compatible','needs_review')),
 updated_at timestamptz NOT NULL DEFAULT now(), UNIQUE(run_id,ordinal), CHECK(left_id<right_id)
);
CREATE TABLE IF NOT EXISTS knowledge_conflict_reviews (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 candidate_id uuid NOT NULL REFERENCES knowledge_conflict_candidates(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), revision integer NOT NULL CHECK(revision>1),
 decision text NOT NULL CHECK(decision IN ('conflict','compatible','needs_review')),
 note_ciphertext text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(candidate_id,revision)
);
CREATE TABLE IF NOT EXISTS knowledge_proposal_conflict_origins (
 proposal_id uuid PRIMARY KEY REFERENCES knowledge_proposals(id) ON DELETE CASCADE,
 candidate_id uuid NOT NULL REFERENCES knowledge_conflict_candidates(id) ON DELETE RESTRICT,
 candidate_revision integer NOT NULL CHECK(candidate_revision>0),
 left_id uuid NOT NULL,right_id uuid NOT NULL,
 left_version integer NOT NULL,right_version integer NOT NULL,
 left_hash text NOT NULL,right_hash text NOT NULL,
 actor_id uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_proposal_conflict_candidate ON knowledge_proposal_conflict_origins(candidate_id);
-- Report ownership never becomes proposal ownership. Proposal access is the
-- intersection of its target ACL and both current source ACLs.
CREATE OR REPLACE FUNCTION madi_proposal_origin_allowed(actor uuid,proposal uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT NOT EXISTS(SELECT 1 FROM knowledge_proposal_conflict_origins o WHERE o.proposal_id=proposal AND
  (NOT madi_document_allowed(actor,o.left_id,false) OR NOT madi_document_allowed(actor,o.right_id,false)
   OR NOT EXISTS(SELECT 1 FROM documents d WHERE d.id=o.left_id AND d.deleted_at IS NULL)
   OR NOT EXISTS(SELECT 1 FROM documents d WHERE d.id=o.right_id AND d.deleted_at IS NULL)))
$$;
