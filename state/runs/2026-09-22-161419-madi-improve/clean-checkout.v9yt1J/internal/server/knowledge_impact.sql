DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='document_relations'::regclass AND conname='document_relations_type_check' AND pg_get_constraintdef(oid) LIKE '%policy%') THEN
  ALTER TABLE document_relations DROP CONSTRAINT IF EXISTS document_relations_type_check;
  ALTER TABLE document_relations ADD CONSTRAINT document_relations_type_check CHECK(type IN ('related','reference','policy','execution','data'));
 END IF;
END $$;

CREATE TABLE IF NOT EXISTS knowledge_change_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 before_version integer NOT NULL, after_version integer NOT NULL,
 body_changed boolean NOT NULL, access_changed boolean NOT NULL, metadata_changed boolean NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS knowledge_change_events_document ON knowledge_change_events(document_id,created_at DESC);
CREATE OR REPLACE FUNCTION madi_capture_knowledge_change() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 IF OLD.version IS DISTINCT FROM NEW.version OR OLD.visibility IS DISTINCT FROM NEW.visibility OR OLD.space_id IS DISTINCT FROM NEW.space_id OR OLD.parent_id IS DISTINCT FROM NEW.parent_id OR OLD.owner_id IS DISTINCT FROM NEW.owner_id THEN
  INSERT INTO knowledge_change_events(document_id,before_version,after_version,body_changed,access_changed,metadata_changed)
  VALUES(NEW.id,OLD.version,NEW.version,OLD.markdown IS DISTINCT FROM NEW.markdown,
   OLD.visibility IS DISTINCT FROM NEW.visibility OR OLD.space_id IS DISTINCT FROM NEW.space_id OR OLD.parent_id IS DISTINCT FROM NEW.parent_id OR OLD.owner_id IS DISTINCT FROM NEW.owner_id,
   OLD.title IS DISTINCT FROM NEW.title OR OLD.tags IS DISTINCT FROM NEW.tags);
 END IF; RETURN NEW; END $$;
DROP TRIGGER IF EXISTS madi_knowledge_change ON documents;
CREATE TRIGGER madi_knowledge_change AFTER UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION madi_capture_knowledge_change();

CREATE TABLE IF NOT EXISTS knowledge_impact_reviews (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), source_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 source_version integer NOT NULL, target_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 target_version integer NOT NULL, relation_type text NOT NULL CHECK(relation_type IN ('related','reference','policy','execution','data')),
 owner_id uuid NOT NULL REFERENCES users(id), created_by uuid NOT NULL REFERENCES users(id),
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','no_impact','needs_change','done','exception_requested')),
 revision integer NOT NULL DEFAULT 1, created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(source_id,source_version,target_id,relation_type), CHECK(source_id<>target_id)
);
CREATE TABLE IF NOT EXISTS knowledge_impact_review_events (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), review_id uuid NOT NULL REFERENCES knowledge_impact_reviews(id) ON DELETE CASCADE,
 actor_id uuid NOT NULL REFERENCES users(id), revision integer NOT NULL, status text NOT NULL,
 note_ciphertext text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), UNIQUE(review_id,revision)
);
CREATE OR REPLACE FUNCTION madi_impact_allowed(actor uuid,target uuid) RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM knowledge_impact_reviews r JOIN documents s ON s.id=r.source_id JOIN documents d ON d.id=r.target_id
 WHERE r.id=target AND s.deleted_at IS NULL AND d.deleted_at IS NULL AND s.workspace_id=d.workspace_id
 AND madi_document_allowed(actor,s.id,false) AND madi_document_allowed(actor,d.id,false));
$$;
