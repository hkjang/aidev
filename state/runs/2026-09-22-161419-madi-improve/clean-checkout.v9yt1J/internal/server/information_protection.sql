CREATE TABLE IF NOT EXISTS protection_settings (
 id integer PRIMARY KEY CHECK(id=1),data jsonb NOT NULL DEFAULT '{}',revision bigint NOT NULL DEFAULT 1,
 updated_by uuid REFERENCES users(id),updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO protection_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS protection_settings_history (
 id uuid PRIMARY KEY,revision bigint NOT NULL,data jsonb NOT NULL,user_id uuid NOT NULL REFERENCES users(id),created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS protection_events (
 id uuid PRIMARY KEY,document_id uuid,workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,
 user_id uuid REFERENCES users(id),action text NOT NULL,mode text NOT NULL,findings jsonb NOT NULL DEFAULT '[]',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS protection_events_workspace_idx ON protection_events(workspace_id,created_at DESC);

-- Hard-coded expressions only. Administrators can select detectors and literal
-- terms, never execute arbitrary regular expressions or code.
CREATE OR REPLACE FUNCTION madi_protection_analyze(input text,detectors jsonb,terms jsonb) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE output text:=coalesce(input,''); findings jsonb:='[]'; detector jsonb; term text; n integer; kind text; label text; pattern text;
BEGIN
 FOR detector IN SELECT value FROM jsonb_array_elements('[{"kind":"rrn","label":"주민번호형식","pattern":"\\m[0-9]{6}[- ]?[1-8][0-9]{6}\\M"},{"kind":"email","label":"이메일","pattern":"[A-Za-z0-9.!#$%&''*+/=?^_`{|}~-]+@[A-Za-z0-9-]+(\\.[A-Za-z0-9-]+)+"},{"kind":"phone","label":"전화번호형식","pattern":"\\m(01[016789]|0[2-6][0-9]?)[ -]?[0-9]{3,4}[ -]?[0-9]{4}\\M"},{"kind":"payment","label":"결제번호형식","pattern":"\\m[0-9]{4}([ -]?[0-9]{4}){3}\\M"},{"kind":"account","label":"계좌번호형식","pattern":"\\m[0-9]{3,6}-[0-9]{2,6}-[0-9]{3,8}\\M"}]'::jsonb) LOOP
  kind:=detector->>'kind'; label:=detector->>'label'; pattern:=detector->>'pattern';
  IF detectors ? kind THEN
   n:=regexp_count(output,pattern,1,'i');
   IF n>0 THEN
    findings:=findings||jsonb_build_array(jsonb_build_object('kind',kind,'count',n));
    output:=regexp_replace(output,pattern,'MADI_REDACTED','gi');
   END IF;
  END IF;
 END LOOP;
 FOR term IN SELECT value FROM jsonb_array_elements_text(terms) LOOP
  IF length(term)>0 THEN
   n:=(length(output)-length(replace(output,term,'')))/length(term);
   IF n>0 THEN
    findings:=findings||jsonb_build_array(jsonb_build_object('kind','custom_term','count',n));
    output:=replace(output,term,'MADI_REDACTED');
   END IF;
  END IF;
 END LOOP;
 RETURN jsonb_build_object('text',output,'findings',findings,'detected',jsonb_array_length(findings)>0);
END $$;

CREATE OR REPLACE FUNCTION madi_protection_json(input jsonb,detectors jsonb,terms jsonb,depth integer DEFAULT 0) RETURNS jsonb LANGUAGE plpgsql IMMUTABLE AS $$
DECLARE output jsonb; findings jsonb:='[]'; item record; child jsonb; scan jsonb; unmaskable boolean:=false;
BEGIN
 IF depth>30 THEN RAISE EXCEPTION 'MADI_PROTECTION_METADATA_DEPTH'; END IF;
 CASE jsonb_typeof(input)
 WHEN 'object' THEN
  output:='{}';
  FOR item IN SELECT key,value FROM jsonb_each(input) LOOP
   scan:=madi_protection_analyze(item.key,detectors,terms);
   IF (scan->>'detected')::boolean THEN findings:=findings||(scan->'findings'); unmaskable:=true; END IF;
   child:=madi_protection_json(item.value,detectors,terms,depth+1);
   output:=output||jsonb_build_object(item.key,child->'value'); findings:=findings||(child->'findings');
   unmaskable:=unmaskable OR (child->>'unmaskable')::boolean;
  END LOOP;
 WHEN 'array' THEN
  output:='[]';
  FOR item IN SELECT value FROM jsonb_array_elements(input) LOOP
   child:=madi_protection_json(item.value,detectors,terms,depth+1);
   output:=output||jsonb_build_array(child->'value'); findings:=findings||(child->'findings');
   unmaskable:=unmaskable OR (child->>'unmaskable')::boolean;
  END LOOP;
 WHEN 'string' THEN
  scan:=madi_protection_analyze(input#>>'{}',detectors,terms); output:=to_jsonb(scan->>'text'); findings:=scan->'findings';
 ELSE
  output:=input;
  IF jsonb_typeof(input)='number' THEN
   scan:=madi_protection_analyze(input::text,detectors,terms);findings:=scan->'findings';unmaskable:=(scan->>'detected')::boolean;
  END IF;
 END CASE;
 RETURN jsonb_build_object('value',coalesce(output,'null'::jsonb),'findings',findings,'unmaskable',unmaskable);
END $$;

CREATE OR REPLACE FUNCTION madi_classification_rank(value text) RETURNS integer LANGUAGE sql IMMUTABLE AS $$
 SELECT CASE value WHEN 'restricted' THEN 3 WHEN 'confidential' THEN 2 WHEN 'internal' THEN 1 ELSE 0 END
$$;
CREATE OR REPLACE FUNCTION madi_effective_classification(target uuid) RETURNS text LANGUAGE sql STABLE AS $$
 WITH RECURSIVE docs AS (
  SELECT d.id,d.parent_id,d.space_id,0 depth FROM documents d WHERE d.id=target
  UNION ALL SELECT d.id,d.parent_id,d.space_id,a.depth+1 FROM documents d JOIN docs a ON d.id=a.parent_id WHERE a.depth<20
 ), folders AS (
  SELECT s.id,s.parent_id,s.classification,0 depth FROM spaces s JOIN docs d ON s.id=d.space_id
  UNION ALL SELECT s.id,s.parent_id,s.classification,a.depth+1 FROM spaces s JOIN folders a ON s.id=a.parent_id WHERE a.depth<20
 ), ranks AS (
  SELECT madi_classification_rank(m.classification) n FROM knowledge_document_meta m JOIN docs d ON m.document_id=d.id
  UNION ALL SELECT madi_classification_rank(classification) FROM folders
 ) SELECT CASE WHEN EXISTS(SELECT 1 FROM docs WHERE depth>=20 AND parent_id IS NOT NULL) OR EXISTS(SELECT 1 FROM folders WHERE depth>=20 AND parent_id IS NOT NULL) THEN 'restricted' ELSE CASE coalesce(max(n),1) WHEN 3 THEN 'restricted' WHEN 2 THEN 'confidential' WHEN 1 THEN 'internal' ELSE 'public' END END FROM ranks
$$;

CREATE OR REPLACE FUNCTION madi_protection_enforce() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE cfg jsonb; mode text; source text; analyzed jsonb; metadata jsonb;
BEGIN
 SELECT data INTO cfg FROM protection_settings WHERE id=1 FOR SHARE;
 IF NOT coalesce((cfg->>'enabled')::boolean,false) THEN RETURN NEW; END IF;
 mode:=coalesce(cfg->>'mode','warn');
 IF mode NOT IN ('block','mask') THEN RETURN NEW; END IF;
 source:=coalesce(NEW.title,'')||E'\n'||coalesce(NEW.markdown,'');
 analyzed:=madi_protection_analyze(source,coalesce(cfg->'detectors','["rrn","email","phone","payment","account"]'),coalesce(cfg->'custom_terms','[]'));
 metadata:=madi_protection_json(jsonb_build_object('tags',to_jsonb(NEW)->'tags','aliases',to_jsonb(NEW)->'aliases','block_metadata',to_jsonb(NEW)->'block_metadata'),coalesce(cfg->'detectors','["rrn","email","phone","payment","account"]'),coalesce(cfg->'custom_terms','[]'));
 IF (analyzed->>'detected')::boolean OR jsonb_array_length(metadata->'findings')>0 THEN
  RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE=CASE WHEN mode='mask' THEN 'MADI_PROTECTION_MASK_REQUIRED' ELSE 'MADI_PROTECTION_BLOCKED' END;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS protection_documents_guard ON documents;
CREATE TRIGGER protection_documents_guard BEFORE INSERT OR UPDATE OF title,markdown,tags,aliases,block_metadata ON documents FOR EACH ROW EXECUTE FUNCTION madi_protection_enforce();
DROP TRIGGER IF EXISTS protection_versions_guard ON document_versions;
CREATE TRIGGER protection_versions_guard BEFORE INSERT OR UPDATE OF title,markdown,tags,block_metadata ON document_versions FOR EACH ROW EXECUTE FUNCTION madi_protection_enforce();

CREATE OR REPLACE FUNCTION madi_protection_audit_document() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE cfg jsonb; analyzed jsonb; mode text; metadata jsonb; findings jsonb;
BEGIN
 SELECT data INTO cfg FROM protection_settings WHERE id=1 FOR SHARE;
 mode:=coalesce(cfg->>'mode','warn');
 IF coalesce((cfg->>'enabled')::boolean,false) AND mode IN ('warn','audit') THEN
  analyzed:=madi_protection_analyze(coalesce(NEW.title,'')||E'\n'||coalesce(NEW.markdown,''),coalesce(cfg->'detectors','["rrn","email","phone","payment","account"]'),coalesce(cfg->'custom_terms','[]'));
  metadata:=madi_protection_json(jsonb_build_object('tags',NEW.tags,'aliases',NEW.aliases,'block_metadata',NEW.block_metadata),coalesce(cfg->'detectors','["rrn","email","phone","payment","account"]'),coalesce(cfg->'custom_terms','[]'));
  findings:=(analyzed->'findings')||(metadata->'findings');
  IF jsonb_array_length(findings)>0 THEN
   INSERT INTO protection_events(id,document_id,workspace_id,user_id,action,mode,findings) VALUES(gen_random_uuid(),NEW.id,NEW.workspace_id,coalesce(NULLIF(current_setting('madi.protection_actor',true),'')::uuid,NEW.owner_id),'document.detected',mode,findings);
  END IF;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS protection_documents_audit ON documents;
CREATE TRIGGER protection_documents_audit AFTER INSERT OR UPDATE OF title,markdown,tags,aliases,block_metadata ON documents FOR EACH ROW EXECUTE FUNCTION madi_protection_audit_document();
