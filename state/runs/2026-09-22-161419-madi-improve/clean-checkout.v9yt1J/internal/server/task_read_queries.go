package server

import (
	"context"
	_ "embed"
	"fmt"
)

//go:embed task_read_indexes.sql
var taskReadIndexesSchema string

func (s *Server) migrateTaskReadIndexes(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, taskReadIndexesSchema)
	return e
}

// LIMIT follows the current ACL, never an arbitrary set of private candidates.
// Window statistics operate on at most 2,001 visible IDs. The extra ID is a
// sentinel only; its body and tasks are not returned. UUID equality and ordered
// partial indexes let PostgreSQL stop as soon as it has enough visible IDs.
func taskDocumentQuery(actor, workspace, document string) (string, []any) {
	args := []any{actor}
	filter := "d.deleted_at IS NULL AND " + docACL
	if workspace != "" {
		args = append(args, workspace)
		filter += fmt.Sprintf(" AND d.workspace_id=$%d::uuid", len(args))
	}
	if document != "" {
		args = append(args, document)
		filter += fmt.Sprintf(" AND d.id=$%d::uuid", len(args))
	}
	query := `WITH visible AS MATERIALIZED (
 SELECT d.id,d.updated_at,octet_length(d.markdown) source_bytes
 FROM documents d WHERE ` + filter + `
 ORDER BY d.updated_at DESC,d.id LIMIT 2001
), sampled AS (
 SELECT id,updated_at,row_number() OVER(ORDER BY updated_at DESC,id) ordinal,
 sum(source_bytes) OVER(ORDER BY updated_at DESC,id ROWS UNBOUNDED PRECEDING) running_bytes
 FROM visible
), stats AS (SELECT count(*) total_visible FROM visible), bounded AS (
 SELECT id,updated_at FROM sampled WHERE ordinal<=2000 AND running_bytes<=16777216
)
SELECT CASE WHEN d.id IS NULL THEN jsonb_build_object('total_visible',stats.total_visible)
 ELSE jsonb_build_object('id',d.id,'title',d.title,'markdown',d.markdown,'version',d.version,
 'owner_id',d.owner_id,'space_id',d.space_id,'total_visible',stats.total_visible,
 'can_write',madi_document_allowed($1,d.id,true),
 'details',coalesce((SELECT jsonb_agg(to_jsonb(t)||jsonb_build_object('assignee_name',u.name,
 'assignee_available',NOT u.disabled AND madi_document_allowed(u.id,d.id,false)))
 FROM task_details t LEFT JOIN users u ON u.id=t.assignee_id WHERE t.document_id=d.id),'[]')) END
FROM stats LEFT JOIN bounded b ON true LEFT JOIN documents d ON d.id=b.id
ORDER BY b.updated_at DESC,b.id`
	return query, args
}
