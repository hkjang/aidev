CREATE TABLE IF NOT EXISTS document_split_receipts (
 user_id uuid NOT NULL REFERENCES users(id), request_id uuid NOT NULL,
 source_id uuid NOT NULL REFERENCES documents(id),child_id uuid NOT NULL REFERENCES documents(id),
 source_version integer NOT NULL,child_version integer NOT NULL DEFAULT 1,payload_hash text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),PRIMARY KEY(user_id,request_id)
);
CREATE INDEX IF NOT EXISTS document_split_source_idx ON document_split_receipts(source_id,created_at DESC);
