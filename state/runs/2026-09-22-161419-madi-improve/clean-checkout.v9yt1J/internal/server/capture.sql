CREATE TABLE IF NOT EXISTS capture_receipts (
 user_id uuid NOT NULL REFERENCES users(id), request_id uuid NOT NULL,
 workspace_id uuid NOT NULL REFERENCES workspaces(id), document_id uuid REFERENCES documents(id) ON DELETE SET NULL,
 payload_hash text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(user_id,request_id)
);
CREATE TABLE IF NOT EXISTS attachment_receipts (
 user_id uuid NOT NULL REFERENCES users(id), request_id uuid NOT NULL,
 document_id uuid REFERENCES documents(id) ON DELETE SET NULL, attachment_id uuid REFERENCES attachments(id) ON DELETE SET NULL,
 payload_hash text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY(user_id,request_id)
);
