CREATE TABLE IF NOT EXISTS knowledge_questions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_id uuid NOT NULL UNIQUE REFERENCES documents(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id), creator_id uuid NOT NULL REFERENCES users(id),
 ciphertext text NOT NULL, source_refs jsonb NOT NULL CHECK(jsonb_typeof(source_refs)='array'),
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0), answer_version integer NOT NULL CHECK(answer_version>0),
 answer_hash text NOT NULL, review_due date NOT NULL,
 state text NOT NULL DEFAULT 'proposed' CHECK(state IN ('proposed','confirmed','archived')),
 origin text NOT NULL CHECK(origin IN ('human','personal_ai_question')), origin_hash text NOT NULL DEFAULT '',
 confirmed_by uuid REFERENCES users(id), confirmed_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_questions_owner ON knowledge_questions(owner_id,updated_at DESC,id);
CREATE TABLE IF NOT EXISTS knowledge_question_draft_receipts (
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE, message_id uuid NOT NULL REFERENCES ai_messages(id) ON DELETE CASCADE,
 document_id uuid REFERENCES documents(id) ON DELETE SET NULL, preview_hash text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(owner_id,message_id)
);
CREATE TABLE IF NOT EXISTS knowledge_question_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), question_id uuid NOT NULL REFERENCES knowledge_questions(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), revision integer NOT NULL, state text NOT NULL, note_ciphertext text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(question_id,revision)
);
CREATE OR REPLACE FUNCTION madi_question_allowed(actor uuid,target uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM knowledge_questions q JOIN documents a ON a.id=q.document_id
 WHERE q.id=target AND a.deleted_at IS NULL AND madi_document_allowed(actor,a.id,false)
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(q.source_refs) ref WHERE NOT EXISTS(
 SELECT 1 FROM documents d WHERE d.id=(ref->>'id')::uuid AND d.workspace_id=a.workspace_id AND d.deleted_at IS NULL AND madi_document_allowed(actor,d.id,false)
 AND (ref->'attachment' IS NULL OR ref->'attachment'='null'::jsonb OR EXISTS(SELECT 1 FROM attachments x WHERE x.id=(ref->'attachment'->>'attachment_id')::uuid AND x.document_id=d.id)))))
$$;
CREATE OR REPLACE FUNCTION madi_question_source_removed() RETURNS trigger LANGUAGE plpgsql AS $$
 BEGIN
 IF TG_TABLE_NAME='documents' THEN
 DELETE FROM knowledge_questions q WHERE EXISTS(SELECT 1 FROM jsonb_array_elements(q.source_refs) ref WHERE ref->>'id'=OLD.id::text);
 ELSE
 DELETE FROM knowledge_questions q WHERE EXISTS(SELECT 1 FROM jsonb_array_elements(q.source_refs) ref WHERE ref->'attachment'->>'attachment_id'=OLD.id::text);
 END IF;
 RETURN OLD;
 END $$;
DROP TRIGGER IF EXISTS madi_question_document_removed ON documents;
CREATE TRIGGER madi_question_document_removed BEFORE DELETE ON documents FOR EACH ROW EXECUTE FUNCTION madi_question_source_removed();
DROP TRIGGER IF EXISTS madi_question_attachment_removed ON attachments;
CREATE TRIGGER madi_question_attachment_removed BEFORE DELETE ON attachments FOR EACH ROW EXECUTE FUNCTION madi_question_source_removed();
