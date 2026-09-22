ALTER TABLE database_rows ADD COLUMN IF NOT EXISTS version integer NOT NULL DEFAULT 1 CHECK(version>0);
CREATE OR REPLACE FUNCTION madi_database_row_revision() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.id<>OLD.id OR NEW.database_id<>OLD.database_id THEN RAISE EXCEPTION 'database row identity is immutable' USING ERRCODE='23514';END IF;
 IF NEW.values IS DISTINCT FROM OLD.values THEN
  IF OLD.version>=2147483647 THEN RAISE EXCEPTION 'database row revision limit' USING ERRCODE='22003';END IF;
  NEW.version:=OLD.version+1;
 ELSE NEW.version:=OLD.version;
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS madi_database_row_revision ON database_rows;
CREATE TRIGGER madi_database_row_revision BEFORE UPDATE ON database_rows FOR EACH ROW EXECUTE FUNCTION madi_database_row_revision();
CREATE TABLE IF NOT EXISTS database_views(
 id uuid PRIMARY KEY,
 database_id uuid NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 name text NOT NULL CHECK(length(name)>0),
 visibility text NOT NULL DEFAULT 'private' CHECK(visibility IN ('private','workspace')),
 data jsonb NOT NULL DEFAULT '{}' CHECK(jsonb_typeof(data)='object'),
 version integer NOT NULL DEFAULT 1 CHECK(version>0),
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS database_views_scope ON database_views(database_id,owner_id,visibility);
CREATE TABLE IF NOT EXISTS database_view_preferences(
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 database_id uuid NOT NULL REFERENCES databases(id) ON DELETE CASCADE,
 view_id uuid REFERENCES database_views(id) ON DELETE SET NULL,
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(user_id,database_id)
);
