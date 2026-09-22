CREATE TABLE IF NOT EXISTS ai_conversations (
 id uuid PRIMARY KEY, workspace_id uuid NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 owner_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 title text NOT NULL, version integer NOT NULL DEFAULT 1 CHECK(version>0),
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS ai_conversations_owner_idx ON ai_conversations(owner_id,workspace_id,updated_at DESC,id);
CREATE TABLE IF NOT EXISTS ai_messages (
 id uuid PRIMARY KEY, conversation_id uuid NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
 ordinal integer NOT NULL CHECK(ordinal>0), question text NOT NULL, answer text NOT NULL,
 action text NOT NULL, sources jsonb NOT NULL DEFAULT '[]', grant_refs jsonb NOT NULL DEFAULT '[]',
 provider_fingerprint text NOT NULL, model text NOT NULL, created_at timestamptz NOT NULL DEFAULT now(),
 UNIQUE(conversation_id,ordinal), CHECK(jsonb_typeof(sources)='array'), CHECK(jsonb_typeof(grant_refs)='array')
);
CREATE INDEX IF NOT EXISTS ai_messages_conversation_idx ON ai_messages(conversation_id,ordinal);
CREATE INDEX IF NOT EXISTS ai_messages_search_idx ON ai_messages USING gin(to_tsvector('simple',question || ' ' || answer));
CREATE OR REPLACE FUNCTION madi_ai_conversation_allowed(actor uuid,target uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
 SELECT EXISTS(SELECT 1 FROM ai_conversations c JOIN users u ON u.id=c.owner_id AND NOT u.disabled
 JOIN workspace_members wm ON wm.workspace_id=c.workspace_id AND wm.user_id=actor
 WHERE c.id=target AND c.owner_id=actor AND NOT EXISTS(
  SELECT 1 FROM ai_messages m CROSS JOIN LATERAL jsonb_array_elements(m.sources) ref
  WHERE m.conversation_id=c.id AND (NOT EXISTS(SELECT 1 FROM documents d
   WHERE d.id=NULLIF(ref->>'id','')::uuid AND d.workspace_id=c.workspace_id AND d.deleted_at IS NULL AND madi_document_allowed(actor,d.id,false))
   OR (NULLIF(ref->>'attachment_id','') IS NOT NULL AND NOT EXISTS(SELECT 1 FROM attachments a WHERE a.id=(ref->>'attachment_id')::uuid AND a.document_id=(ref->>'id')::uuid)))))
$$;
