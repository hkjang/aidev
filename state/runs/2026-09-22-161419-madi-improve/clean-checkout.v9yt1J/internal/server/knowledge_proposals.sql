CREATE TABLE IF NOT EXISTS knowledge_proposals (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id), base_version integer NOT NULL CHECK(base_version>0),
 base_hash text NOT NULL, markdown_hash text NOT NULL, ciphertext text NOT NULL,
 provenance text NOT NULL CHECK(provenance IN ('human','ai_assisted')),
 status text NOT NULL DEFAULT 'open' CHECK(status IN ('open','merged','rejected','withdrawn')),
 revision integer NOT NULL DEFAULT 1, merged_version integer,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_proposals_document ON knowledge_proposals(document_id,created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS knowledge_proposals_open_hash ON knowledge_proposals(document_id,owner_id,base_version,markdown_hash) WHERE status='open';
CREATE TABLE IF NOT EXISTS knowledge_proposal_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), proposal_id uuid NOT NULL REFERENCES knowledge_proposals(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), revision integer NOT NULL, status text NOT NULL,
 note_ciphertext text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(proposal_id,revision)
);
