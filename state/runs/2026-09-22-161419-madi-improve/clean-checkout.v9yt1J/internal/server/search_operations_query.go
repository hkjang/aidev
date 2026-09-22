package server

import "strings"

// Keep the legacy read-model SQL as an ACL differential-test oracle. This
// deterministic transformation only adds current-version normalized candidates;
// all existing source-kind, actor, space, visibility and capability gates stay.
func legacyInterpretedSearchSQL() string {
	base := strings.Replace(universalSearchSQL, "\n)\nSELECT CASE WHEN h.kind=", attachmentSearchSQL()+"\n)\nSELECT CASE WHEN h.kind=", 1)
	sql := strings.ReplaceAll(base, "websearch_to_tsquery('simple',$3)", "CASE WHEN jsonb_array_length($19::jsonb)>0 THEN madi_search_parts_query($19::jsonb) ELSE websearch_to_tsquery('simple',$3) END")
	const normalized = `), normalized_documents AS MATERIALIZED (
 SELECT z.document_id,z.document_version FROM (
 SELECT * FROM search_folded_documents WHERE $21<>'' AND grams_complete AND gram_vector@@$21::tsquery
 UNION ALL SELECT * FROM search_folded_documents WHERE $21<>'' AND NOT grams_complete
 UNION ALL SELECT * FROM search_folded_documents WHERE $21=''
 ) z JOIN visible_documents d ON d.id=z.document_id AND d.version=z.document_version
 WHERE jsonb_array_length($20::jsonb)>0 AND madi_search_folded_matches(z.normalized,$20::jsonb)
), normalized_fragments AS MATERIALIZED (
 SELECT z.document_id,z.ordinal,z.document_version FROM (
 SELECT * FROM search_folded_fragments WHERE $21<>'' AND grams_complete AND gram_vector@@$21::tsquery
 UNION ALL SELECT * FROM search_folded_fragments WHERE $21<>'' AND NOT grams_complete
 UNION ALL SELECT * FROM search_folded_fragments WHERE $21=''
 ) z JOIN visible_documents d ON d.id=z.document_id AND d.version=z.document_version
 WHERE jsonb_array_length($20::jsonb)>0 AND madi_search_folded_matches(z.normalized,$20::jsonb)
), hits AS (`
	sql = strings.Replace(sql, "), hits AS (", normalized, 1)
	sql = strings.Replace(sql, "OR d.aliases::text ILIKE $4)", "OR d.aliases::text ILIKE $4 OR (d.id,d.version) IN (SELECT document_id,document_version FROM normalized_documents))", 1)
	sql = strings.Replace(sql, "OR f.content ILIKE $4)", "OR f.content ILIKE $4 OR (f.document_id,f.ordinal,f.document_version) IN (SELECT document_id,ordinal,document_version FROM normalized_fragments))", 1)
	sql = strings.Replace(sql, "WHEN d.title ILIKE $4 THEN 4 ELSE 0 END)", "WHEN d.title ILIKE $4 OR madi_search_folded_matches(regexp_replace(lower(normalize(d.title,NFKC)),'[[:space:]]','','g'),$20::jsonb) THEN 4 ELSE 0 END)", 1)
	return sql
}

// Hangul and ASCII nonletters have no case variants. For this deliberately
// narrow literal alphabet LIKE has the same matches as ILIKE without folding
// every character of each candidate body. Latin, combining marks and all other
// Unicode alphabets retain PostgreSQL's locale-aware ILIKE behavior.
func searchCaseIndependentLiteral(term string) bool {
	for _, r := range term {
		if r < 128 && !(r >= 'A' && r <= 'Z') && !(r >= 'a' && r <= 'z') {
			continue
		}
		if r >= 0xac00 && r <= 0xd7a3 || r >= 0x1100 && r <= 0x11ff || r >= 0x3130 && r <= 0x318f || r >= 0xa960 && r <= 0xa97f || r >= 0xd7b0 && r <= 0xd7ff {
			continue
		}
		return false
	}
	return true
}

func interpretedSearchSQL(term string, kinds ...string) string {
	sql := legacyInterpretedSearchSQL()
	if len(kinds) == 0 || kinds[0] == "" {
		// Many result kinds refer to the same document. Materialize only the
		// authorized IDs once; keep Markdown outside this CTE so a large workspace
		// does not materialize every body. Source predicates and ranking still run
		// against current documents in this same statement snapshot.
		sql = strings.Replace(sql, "), visible_documents AS NOT MATERIALIZED (\n SELECT d.* FROM documents d JOIN active_workspace", "), visible_document_ids AS MATERIALIZED (\n SELECT d.id FROM documents d JOIN active_workspace", 1)
		sql = strings.Replace(sql, "), visible_databases AS NOT MATERIALIZED (", "), visible_documents AS NOT MATERIALIZED (\n SELECT d.* FROM documents d JOIN visible_document_ids allowed ON allowed.id=d.id\n), visible_databases AS NOT MATERIALIZED (", 1)
	}
	// Candidate IDs are local SQL intermediates, never an authorization result.
	// The final document/fragment branch retains visible_documents/current ACL.
	// Rechecking the same ancestor ACL per folded fragment before that final join
	// adds repeated recursive work without changing which result can be returned.
	sql = strings.ReplaceAll(sql, ") z JOIN visible_documents d ON d.id=z.document_id AND d.version=z.document_version", ") z JOIN documents d ON d.id=z.document_id AND d.version=z.document_version AND d.workspace_id=$2 AND d.deleted_at IS NULL")
	const query = "CASE WHEN jsonb_array_length($19::jsonb)>0 THEN madi_search_parts_query($19::jsonb) ELSE websearch_to_tsquery('simple',$3) END"
	// UNION distinct IDs, not result rows: overlapping FTS, literal and folded
	// matches must produce one result with precisely the original score. Source
	// ACL and current projection version are checked again in the final joins.
	const candidates = `), matching_fragments AS MATERIALIZED (
 SELECT f.document_id,f.ordinal,f.document_version FROM search_fragments f
 WHERE ($5='' OR $5=f.kind) AND $3<>'' AND f.search_vector@@` + query + `
 UNION SELECT f.document_id,f.ordinal,f.document_version FROM search_fragments f
 WHERE ($5='' OR $5=f.kind) AND ($3='' OR f.content ILIKE $4)
 UNION SELECT document_id,ordinal,document_version FROM normalized_fragments WHERE $5='' OR $5 IN ('block','code','task')
), matching_comments AS MATERIALIZED (
 SELECT c.id FROM comments c WHERE c.deleted_at IS NULL AND ($5='' OR $5='comment') AND $3<>'' AND to_tsvector('simple',c.body)@@` + query + `
 UNION SELECT c.id FROM comments c WHERE c.deleted_at IS NULL AND ($5='' OR $5='comment') AND ($3='' OR c.body ILIKE $4)
), hits AS (`
	sql = strings.Replace(sql, "), hits AS (", candidates, 1)
	sql = strings.Replace(sql, "FROM search_fragments f JOIN visible_documents d ON d.id=f.document_id AND d.version=f.document_version\n WHERE ($5='' OR $5=f.kind) AND ($3='' OR f.search_vector@@"+query+" OR f.content ILIKE $4 OR (f.document_id,f.ordinal,f.document_version) IN (SELECT document_id,ordinal,document_version FROM normalized_fragments))", "FROM matching_fragments match JOIN search_fragments f ON f.document_id=match.document_id AND f.ordinal=match.ordinal AND f.document_version=match.document_version JOIN visible_documents d ON d.id=f.document_id AND d.version=f.document_version\n WHERE ($5='' OR $5=f.kind)", 1)
	sql = strings.Replace(sql, "FROM comments c JOIN visible_documents d ON d.id=c.document_id WHERE c.deleted_at IS NULL AND ($5='' OR $5='comment') AND ($3='' OR to_tsvector('simple',c.body)@@"+query+" OR c.body ILIKE $4)", "FROM matching_comments match JOIN comments c ON c.id=match.id JOIN visible_documents d ON d.id=c.document_id WHERE c.deleted_at IS NULL AND ($5='' OR $5='comment')", 1)
	if searchCaseIndependentLiteral(term) {
		sql = strings.ReplaceAll(sql, " ILIKE $4", " LIKE $4")
	}
	return sql
}
