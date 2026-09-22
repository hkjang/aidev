package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/hkjang/madi/internal/extract"
	"github.com/jackc/pgx/v5"
)

//go:embed attachment_extraction.sql
var attachmentExtractionSchema string

type extractionPolicy struct {
	Revision int64          `json:"revision"`
	Data     map[string]any `json:"data"`
}
type extractionRun struct {
	ID              string         `json:"id"`
	AttachmentID    string         `json:"attachment_id"`
	DocumentID      string         `json:"document_id"`
	WorkspaceID     string         `json:"workspace_id"`
	ActorID         string         `json:"actor_id"`
	JobID           string         `json:"job_id"`
	Status          string         `json:"status"`
	Revision        int64          `json:"revision"`
	PolicyRevision  int64          `json:"policy_revision"`
	DocumentVersion int            `json:"document_version"`
	Checksum        string         `json:"checksum"`
	Object          storedObject   `json:"-"`
	Format          string         `json:"format"`
	OCRPages        []int          `json:"ocr_pages"`
	SessionHash     string         `json:"-"`
	TokenHash       string         `json:"-"`
	RequestIP       string         `json:"-"`
	Result          map[string]any `json:"result"`
	Error           string         `json:"error"`
	CreatedAt       time.Time      `json:"created_at"`
	CompletedAt     *time.Time     `json:"completed_at"`
}

const extractionRunSelect = `id::text,attachment_id::text,document_id::text,workspace_id::text,coalesce(actor_id::text,''),coalesce(job_id::text,''),status,revision,policy_revision,document_version,checksum,object_snapshot,format,ocr_pages,session_hash,token_hash,request_ip,result,error,created_at,completed_at`

func scanExtractionRun(row pgx.Row) (extractionRun, error) {
	var v extractionRun
	var object, result []byte
	e := row.Scan(&v.ID, &v.AttachmentID, &v.DocumentID, &v.WorkspaceID, &v.ActorID, &v.JobID, &v.Status, &v.Revision, &v.PolicyRevision, &v.DocumentVersion, &v.Checksum, &object, &v.Format, &v.OCRPages, &v.SessionHash, &v.TokenHash, &v.RequestIP, &result, &v.Error, &v.CreatedAt, &v.CompletedAt)
	if e == nil {
		e = json.Unmarshal(object, &v.Object)
	}
	if e == nil {
		e = json.Unmarshal(result, &v.Result)
	}
	if v.OCRPages == nil {
		v.OCRPages = []int{}
	}
	return v, e
}

func (s *Server) migrateAttachmentExtraction(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, attachmentExtractionSchema)
	return e
}

func extractionPolicyQuery(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, lock bool) (extractionPolicy, error) {
	var p extractionPolicy
	var raw []byte
	sql := "SELECT revision,data FROM attachment_extraction_settings WHERE id=1"
	if lock {
		sql += " FOR SHARE"
	}
	e := q.QueryRow(ctx, sql).Scan(&p.Revision, &raw)
	if e == nil {
		e = json.Unmarshal(raw, &p.Data)
	}
	return p, e
}

func extractionLimits(p extractionPolicy) extract.Limits {
	return extract.Limits{InputBytes: int64(number(p.Data, "max_file_bytes", 50<<20)), ExpandedBytes: 100 << 20, TextBytes: number(p.Data, "max_text_bytes", 8<<20), Fragments: number(p.Data, "max_fragments", 20000), Pages: number(p.Data, "max_pages", 500)}
}

func validateExtractionPolicy(data map[string]any) error {
	limits := map[string][2]int{"max_file_bytes": {1 << 20, 50 << 20}, "max_pages": {1, 500}, "max_text_bytes": {64 << 10, 8 << 20}, "max_fragments": {100, 20000}, "timeout_seconds": {30, 600}}
	for key, value := range data {
		if key == "enabled" || key == "ocr_enabled" {
			if _, ok := value.(bool); !ok {
				return errors.New("첨부 추출 사용 설정은 체크박스 값이어야 합니다")
			}
			continue
		}
		bound, ok := limits[key]
		if !ok {
			return errors.New("지원하지 않는 첨부 추출 설정입니다")
		}
		n, ok := value.(float64)
		if !ok || n != float64(int(n)) || n < float64(bound[0]) || n > float64(bound[1]) {
			return errors.New("첨부 추출 한도 범위를 확인하세요")
		}
	}
	for key := range limits {
		if _, ok := data[key]; !ok {
			return errors.New("첨부 추출 한도가 누락되었습니다")
		}
	}
	for _, key := range []string{"enabled", "ocr_enabled"} {
		if _, ok := data[key]; !ok {
			return errors.New("첨부 추출 사용 설정이 누락되었습니다")
		}
	}
	return nil
}

func extractionFormat(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".pdf":
		return "pdf"
	case ".docx":
		return "docx"
	case ".pptx":
		return "pptx"
	case ".xlsx":
		return "xlsx"
	case ".txt", ".md", ".csv":
		return "text"
	}
	return ""
}

func (s *Server) extractionSettings(w http.ResponseWriter, r *http.Request) {
	if current(r).TokenID != "" || current(r).ScopeRestricted {
		apiError(w, 403, "첨부 추출 정책은 관리자 로그인 세션에서 변경하세요")
		return
	}
	if r.Method == http.MethodGet {
		p, e := extractionPolicyQuery(r.Context(), s.DB, false)
		respond(w, p, e)
		return
	}
	var in extractionPolicy
	if decode(r, &in) != nil || validateExtractionPolicy(in.Data) != nil {
		apiError(w, 400, "첨부 추출 설정과 한도를 확인하세요")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	tag, e := tx.Exec(r.Context(), `UPDATE attachment_extraction_settings SET data=$1,revision=revision+1,updated_by=$2,updated_at=now() WHERE id=1 AND revision=$3`, jsonValue(in.Data), current(r).ID, in.Revision)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "다른 관리자가 정책을 변경했습니다. 새 설정을 확인하세요")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO attachment_extraction_policy_history(revision,data,actor_id) SELECT revision,data,updated_by FROM attachment_extraction_settings WHERE id=1`)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "ATTACHMENT_EXTRACTION_POLICY", "1", map[string]any{"revision": in.Revision + 1})
	p, e := extractionPolicyQuery(r.Context(), s.DB, false)
	respond(w, p, e)
}
