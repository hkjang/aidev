CREATE TABLE IF NOT EXISTS knowledge_validity (
 document_id uuid PRIMARY KEY REFERENCES documents(id) ON DELETE CASCADE,
 revision integer NOT NULL DEFAULT 0 CHECK(revision>=0),
 updated_by uuid REFERENCES users(id), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS knowledge_validity_periods (
 document_id uuid NOT NULL REFERENCES knowledge_validity(document_id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal>=0 AND ordinal<100),
 document_version integer NOT NULL, valid_from date NOT NULL, valid_until date,
 PRIMARY KEY(document_id,ordinal),
 FOREIGN KEY(document_id,document_version) REFERENCES document_versions(document_id,version) ON DELETE CASCADE,
 CHECK(valid_until IS NULL OR valid_until>valid_from),
 CHECK(valid_from>='0001-01-01'::date AND valid_from<='9999-12-31'::date),
 CHECK(valid_until IS NULL OR valid_until<='9999-12-31'::date)
);
CREATE INDEX IF NOT EXISTS knowledge_validity_period_range ON knowledge_validity_periods USING gist(daterange(valid_from,valid_until,'[)'));
-- No btree_gist extension is required. Serialize registrations on the parent
-- and reject overlaps even for an importer that bypasses the HTTP validator.
CREATE OR REPLACE FUNCTION madi_validity_no_overlap() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 PERFORM document_id FROM knowledge_validity WHERE document_id=NEW.document_id FOR UPDATE;
 IF EXISTS(SELECT 1 FROM knowledge_validity_periods p WHERE p.document_id=NEW.document_id AND p.ordinal<>NEW.ordinal AND daterange(p.valid_from,p.valid_until,'[)') && daterange(NEW.valid_from,NEW.valid_until,'[)')) THEN
  RAISE EXCEPTION 'overlapping business validity ranges' USING ERRCODE='23P01';
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS madi_validity_no_overlap ON knowledge_validity_periods;
CREATE TRIGGER madi_validity_no_overlap BEFORE INSERT OR UPDATE ON knowledge_validity_periods FOR EACH ROW EXECUTE FUNCTION madi_validity_no_overlap();
CREATE TABLE IF NOT EXISTS knowledge_validity_history (
 document_id uuid NOT NULL REFERENCES knowledge_validity(document_id) ON DELETE CASCADE,
 revision integer NOT NULL, document_revision integer NOT NULL,
 periods jsonb NOT NULL CHECK(jsonb_typeof(periods)='array' AND jsonb_array_length(periods)<=100),
 actor_id uuid NOT NULL REFERENCES users(id), reason_ciphertext text NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(document_id,revision)
);
