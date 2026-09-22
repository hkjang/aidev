-- Ordered reads must stop after the visible-document sentinel, without first
-- sorting every document in the workspace. These contain no copied content.
CREATE INDEX IF NOT EXISTS documents_active_workspace_recent_idx
 ON documents(workspace_id,updated_at DESC,id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS documents_active_recent_idx
 ON documents(updated_at DESC,id) WHERE deleted_at IS NULL;
-- Equality prefix + existing stable keyset pagination (also works backwards).
CREATE INDEX IF NOT EXISTS comments_document_created_idx
 ON comments(document_id,created_at,id);
CREATE INDEX IF NOT EXISTS attachments_document_created_idx
 ON attachments(document_id,created_at,id);
