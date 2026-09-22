CREATE TABLE IF NOT EXISTS knowledge_evidence_policy (
 id integer PRIMARY KEY CHECK(id=1), enabled boolean NOT NULL DEFAULT false,
 retention_days integer NOT NULL DEFAULT 90 CHECK(retention_days BETWEEN 1 AND 3650),
 version integer NOT NULL DEFAULT 1, updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO knowledge_evidence_policy(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS knowledge_evidence_policy_history (
 version integer PRIMARY KEY, actor_id uuid REFERENCES users(id), enabled boolean NOT NULL,
 retention_days integer NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO knowledge_evidence_policy_history(version,enabled,retention_days)
 SELECT version,enabled,retention_days FROM knowledge_evidence_policy WHERE id=1 ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS knowledge_evidence (
 id uuid PRIMARY KEY, owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 source_refs jsonb NOT NULL CHECK(jsonb_typeof(source_refs)='array' AND jsonb_array_length(source_refs) BETWEEN 1 AND 64),
 ciphertext text NOT NULL, payload_hash text NOT NULL,
 review_status text NOT NULL DEFAULT 'unreviewed' CHECK(review_status IN ('unreviewed','supported','insufficient','misinterpreted')),
 version integer NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS knowledge_evidence_owner ON knowledge_evidence(owner_id,workspace_id,created_at DESC,id);
CREATE INDEX IF NOT EXISTS knowledge_evidence_expiry ON knowledge_evidence(expires_at);
CREATE TABLE IF NOT EXISTS knowledge_evidence_reviews (
 id uuid PRIMARY KEY, evidence_id uuid NOT NULL REFERENCES knowledge_evidence(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), version integer NOT NULL,
 status text NOT NULL CHECK(status IN ('unreviewed','supported','insufficient','misinterpreted')),
 note_ciphertext text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(evidence_id,version)
);
CREATE OR REPLACE FUNCTION madi_evidence_allowed(actor uuid,target uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM knowledge_evidence e JOIN users u ON u.id=e.owner_id AND NOT u.disabled
 JOIN workspace_members wm ON wm.workspace_id=e.workspace_id AND wm.user_id=actor
 WHERE e.id=target AND e.owner_id=actor AND e.expires_at>now()
 AND EXISTS(SELECT 1 FROM knowledge_evidence_policy WHERE id=1 AND enabled)
 AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset(e.source_refs) AS ref(id uuid,version integer,attachment_id uuid)
 WHERE NOT EXISTS(SELECT 1 FROM documents d WHERE d.id=ref.id AND d.workspace_id=e.workspace_id
 AND d.deleted_at IS NULL AND madi_document_allowed(actor,d.id,false))
 OR (ref.attachment_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM attachments a WHERE a.id=ref.attachment_id AND a.document_id=ref.id))))
$$;
CREATE OR REPLACE FUNCTION madi_erase_deleted_document_evidence()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 DELETE FROM knowledge_evidence e WHERE EXISTS(
  SELECT 1 FROM jsonb_to_recordset(e.source_refs) AS ref(id uuid) WHERE ref.id=OLD.id);
 RETURN OLD;
END $$;
DROP TRIGGER IF EXISTS madi_erase_document_evidence ON documents;
CREATE TRIGGER madi_erase_document_evidence BEFORE DELETE ON documents FOR EACH ROW EXECUTE FUNCTION madi_erase_deleted_document_evidence();
CREATE OR REPLACE FUNCTION madi_erase_deleted_attachment_evidence()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 DELETE FROM knowledge_evidence e WHERE EXISTS(SELECT 1 FROM jsonb_to_recordset(e.source_refs) AS ref(attachment_id uuid) WHERE ref.attachment_id=OLD.id);
 RETURN OLD;
END $$;
DROP TRIGGER IF EXISTS madi_erase_attachment_evidence ON attachments;
CREATE TRIGGER madi_erase_attachment_evidence BEFORE DELETE ON attachments FOR EACH ROW EXECUTE FUNCTION madi_erase_deleted_attachment_evidence();
CREATE OR REPLACE FUNCTION madi_evidence_held(target uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM knowledge_evidence e CROSS JOIN LATERAL jsonb_to_recordset(e.source_refs) AS ref(id uuid)
 JOIN knowledge_document_meta m ON m.document_id=ref.id
 WHERE e.id=target AND (m.legal_hold OR m.retain_until>now()))
$$;
