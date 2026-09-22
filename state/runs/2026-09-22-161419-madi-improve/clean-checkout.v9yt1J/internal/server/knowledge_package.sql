CREATE TABLE IF NOT EXISTS knowledge_package_policy (
 id integer PRIMARY KEY CHECK(id=1), enabled boolean NOT NULL DEFAULT true,
 retention_hours integer NOT NULL DEFAULT 24 CHECK(retention_hours BETWEEN 1 AND 168),
 token_counter text NOT NULL DEFAULT 'estimate' CHECK(token_counter IN ('estimate','responses')),
 allow_http boolean NOT NULL DEFAULT false, version integer NOT NULL DEFAULT 1,
 updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO knowledge_package_policy(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS knowledge_package_policy_history (
 version integer PRIMARY KEY, actor_id uuid REFERENCES users(id), enabled boolean NOT NULL,
 retention_hours integer NOT NULL, token_counter text NOT NULL, allow_http boolean NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO knowledge_package_policy_history(version,enabled,retention_hours,token_counter,allow_http)
 SELECT version,enabled,retention_hours,token_counter,allow_http FROM knowledge_package_policy WHERE id=1 ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS knowledge_packages (
 id uuid PRIMARY KEY, owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 source_refs jsonb NOT NULL CHECK(jsonb_typeof(source_refs)='array' AND jsonb_array_length(source_refs) BETWEEN 1 AND 64),
 ciphertext text NOT NULL, payload_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), expires_at timestamptz NOT NULL,
 export_count integer NOT NULL DEFAULT 0, last_exported_at timestamptz
);
CREATE INDEX IF NOT EXISTS knowledge_packages_owner ON knowledge_packages(owner_id,workspace_id,created_at DESC,id);
CREATE INDEX IF NOT EXISTS knowledge_packages_expiry ON knowledge_packages(expires_at);
CREATE OR REPLACE FUNCTION madi_package_allowed(actor uuid,target uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM knowledge_packages p JOIN users u ON u.id=p.owner_id AND NOT u.disabled
 JOIN workspace_members wm ON wm.workspace_id=p.workspace_id AND wm.user_id=actor
 WHERE p.id=target AND p.owner_id=actor AND p.expires_at>now()
 AND EXISTS(SELECT 1 FROM knowledge_package_policy WHERE id=1 AND enabled)
 AND NOT EXISTS(SELECT 1 FROM jsonb_to_recordset(p.source_refs) AS ref(id uuid,version integer,attachment_id uuid)
 WHERE NOT EXISTS(SELECT 1 FROM documents d WHERE d.id=ref.id AND d.workspace_id=p.workspace_id
 AND d.deleted_at IS NULL AND madi_document_allowed(actor,d.id,false))
 OR (ref.attachment_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM attachments a WHERE a.id=ref.attachment_id AND a.document_id=ref.id))))
$$;
CREATE OR REPLACE FUNCTION madi_erase_deleted_document_packages()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 DELETE FROM knowledge_packages p WHERE EXISTS(SELECT 1 FROM jsonb_to_recordset(p.source_refs) AS ref(id uuid) WHERE ref.id=OLD.id);
 RETURN OLD;
END $$;
DROP TRIGGER IF EXISTS madi_erase_document_packages ON documents;
CREATE TRIGGER madi_erase_document_packages BEFORE DELETE ON documents FOR EACH ROW EXECUTE FUNCTION madi_erase_deleted_document_packages();
CREATE OR REPLACE FUNCTION madi_erase_deleted_attachment_packages()
RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 DELETE FROM knowledge_packages p WHERE EXISTS(SELECT 1 FROM jsonb_to_recordset(p.source_refs) AS ref(attachment_id uuid) WHERE ref.attachment_id=OLD.id);
 RETURN OLD;
END $$;
DROP TRIGGER IF EXISTS madi_erase_attachment_packages ON attachments;
CREATE TRIGGER madi_erase_attachment_packages BEFORE DELETE ON attachments FOR EACH ROW EXECUTE FUNCTION madi_erase_deleted_attachment_packages();
