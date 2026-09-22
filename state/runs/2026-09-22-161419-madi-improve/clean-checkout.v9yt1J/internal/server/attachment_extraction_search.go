package server

// attachmentSearchSQL is a UNION branch in the shared current-ACL search query.
// Fragment bytes are a derived projection, never an implicit AI consent.
func attachmentSearchSQL() string {
	return ` UNION ALL
 SELECT 'file',f.id::text,d.id::text,a.name,
 substring(f.text from greatest(1,strpos(lower(f.text),lower($3))-100) for 600),
 '/app/attachments/'||a.id||'?extraction='||e.id||'&fragment='||f.id,
 d.updated_at,(1+ts_rank_cd(f.search_vector,websearch_to_tsquery('simple',$3)))::float8,
 jsonb_build_object('document_title',d.title,'attachment_id',a.id,'extraction_id',e.id,'extraction_revision',e.revision,'attachment_checksum',e.checksum,'fragment_id',f.id,'ordinal',f.ordinal,'position',f.position,'content_hash',f.content_hash,'version',d.version)
 FROM (SELECT *,fts AS search_vector FROM attachment_extraction_fragments) f
 JOIN attachment_extractions e ON e.id=f.extraction_id AND e.status='ready'
 JOIN attachment_extraction_heads h ON h.extraction_id=e.id AND h.attachment_id=e.attachment_id
 JOIN attachments a ON a.id=h.attachment_id AND a.document_id=e.document_id AND a.checksum_sha256=e.checksum
 JOIN visible_documents d ON d.id=a.document_id
 JOIN attachment_extraction_settings policy ON policy.id=1 AND policy.revision=e.policy_revision AND (policy.data->>'enabled')::boolean
 WHERE ($5='' OR $5='file') AND ($3='' OR f.search_vector@@websearch_to_tsquery('simple',$3) OR f.text ILIKE $4)
 `
}
