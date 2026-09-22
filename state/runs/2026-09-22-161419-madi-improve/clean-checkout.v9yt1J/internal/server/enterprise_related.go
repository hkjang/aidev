package server

// Indexed links never need to transfer source bodies. For unindexed candidate
// documents, bound the *JSON-encoded* source bytes (not just source length), so
// control characters cannot expand the response beyond the parsing budget.
// Explicit relations stay usable even when the source parsing budget is full.
const enterpriseRelatedDocumentsSQL = `WITH visible AS (
 SELECT d.id,d.title,d.version,d.updated_at,
 EXISTS(SELECT 1 FROM document_relations rel WHERE (rel.source_id=d.id AND rel.target_id=$2) OR (rel.source_id=$2 AND rel.target_id=d.id)) AS confirmed_relation,
 coalesce(x.document_version=d.version AND x.links_indexed,false) AS indexed,
 CASE WHEN x.document_version=d.version AND x.links_indexed THEN EXISTS(SELECT 1 FROM jsonb_array_elements_text(x.links) link WHERE lower(link)=ANY($4::text[])) ELSE false END AS indexed_match,
 d.markdown
 FROM documents d LEFT JOIN search_index_documents x ON x.document_id=d.id
 WHERE d.workspace_id=$1 AND d.id<>$2 AND d.deleted_at IS NULL AND madi_document_allowed($3,d.id,false)
), candidates AS (
 SELECT id,title,version,updated_at,confirmed_relation,indexed_match,
 CASE WHEN confirmed_relation OR indexed THEN '' ELSE markdown END AS markdown
 FROM visible WHERE confirmed_relation OR indexed_match OR (NOT indexed AND EXISTS(SELECT 1 FROM unnest($4::text[]) name WHERE strpos(lower(markdown),name)>0))
 ORDER BY confirmed_relation DESC,updated_at DESC,id LIMIT 2001
), bounded AS (
 SELECT *,sum(octet_length(to_jsonb(markdown)::text)) OVER(ORDER BY confirmed_relation DESC,updated_at DESC,id) AS bytes FROM candidates
) SELECT jsonb_build_object('id',id,'title',title,'version',version,'confirmed_relation',confirmed_relation,'indexed_match',indexed_match,
 'deferred',markdown<>'' AND bytes>16777216,'markdown',CASE WHEN bytes<=16777216 THEN markdown ELSE '' END)
 FROM bounded ORDER BY confirmed_relation DESC,updated_at DESC,id`
