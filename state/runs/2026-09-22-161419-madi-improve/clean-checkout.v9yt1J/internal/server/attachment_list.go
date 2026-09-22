package server

import "net/http"

func (s *Server) listAttachments(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 404, "첨부 문서에 접근할 수 없습니다")
		return
	}
	after := r.URL.Query().Get("after")
	if after != "" && !validID(after) {
		apiError(w, 400, "첨부파일 페이지 위치를 확인하세요")
		return
	}
	v, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',a.id,'name',a.name,'size',a.size,'content_type',a.content_type,'created_at',a.created_at,'url','/api/v1/attachments/'||a.id::text) FROM attachments a JOIN documents d ON d.id=a.document_id WHERE d.id=$1 AND d.deleted_at IS NULL AND ($2='' OR (a.created_at,a.id)<(SELECT created_at,id FROM attachments WHERE id=NULLIF($2,'')::uuid AND document_id=$1)) ORDER BY a.created_at DESC,a.id DESC LIMIT 200`, id, after)
	respond(w, v, e)
}
