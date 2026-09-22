CREATE TABLE IF NOT EXISTS document_access_requests (
 id uuid PRIMARY KEY, document_id uuid REFERENCES documents(id) ON DELETE SET NULL,
 requested_document_id uuid NOT NULL,
 workspace_id uuid NOT NULL REFERENCES workspaces(id), requester_id uuid NOT NULL REFERENCES users(id),
 permission text NOT NULL CHECK(permission IN ('read','write')),
 reason_ciphertext text NOT NULL DEFAULT '',
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','granted','rejected','cancelled')),
 revision integer NOT NULL DEFAULT 1 CHECK(revision>0),
 resolved_by uuid REFERENCES users(id), created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
ALTER TABLE document_access_requests ADD COLUMN IF NOT EXISTS requested_document_id uuid;
UPDATE document_access_requests SET requested_document_id=document_id WHERE requested_document_id IS NULL;
ALTER TABLE document_access_requests ALTER COLUMN requested_document_id SET NOT NULL;
ALTER TABLE document_access_requests ALTER COLUMN document_id DROP NOT NULL;
ALTER TABLE document_access_requests DROP CONSTRAINT IF EXISTS document_access_requests_document_id_fkey;
ALTER TABLE document_access_requests ADD CONSTRAINT document_access_requests_document_id_fkey FOREIGN KEY(document_id) REFERENCES documents(id) ON DELETE SET NULL;
DROP INDEX IF EXISTS document_access_pending_unique;
CREATE UNIQUE INDEX document_access_pending_unique ON document_access_requests(workspace_id,requested_document_id,requester_id) WHERE status='pending';
CREATE INDEX IF NOT EXISTS document_access_requester_idx ON document_access_requests(requester_id,created_at DESC);
