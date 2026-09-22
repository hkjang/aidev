package server

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed migration_resume_schema.sql
var migrationResumeSchema string

const migrationChunkBytes = 1 << 20

type migrationSession struct {
	ID            string         `json:"id"`
	WorkspaceID   string         `json:"workspace_id"`
	UserID        string         `json:"user_id"`
	SpaceID       string         `json:"space_id"`
	SourceKey     string         `json:"source_key"`
	Label         string         `json:"label"`
	Format        string         `json:"format"`
	Status        string         `json:"status"`
	Revision      int64          `json:"revision"`
	PlanHash      string         `json:"plan_hash"`
	DeclaredBytes int64          `json:"declared_bytes"`
	UploadedBytes int64          `json:"uploaded_bytes"`
	ItemCount     int            `json:"item_count"`
	PreparedCount int            `json:"prepared_count"`
	JobID         string         `json:"job_id"`
	Report        map[string]any `json:"report"`
	Error         string         `json:"error"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	ExpiresAt     time.Time      `json:"expires_at"`
}

const migrationSessionSelect = `id::text,workspace_id::text,user_id::text,coalesce(space_id::text,''),source_key,label,format,status,revision,plan_hash,declared_bytes,uploaded_bytes,item_count,prepared_count,coalesce(job_id::text,''),report,error,created_at,updated_at,expires_at`

func scanMigrationSession(row pgx.Row) (migrationSession, error) {
	var v migrationSession
	var raw []byte
	e := row.Scan(&v.ID, &v.WorkspaceID, &v.UserID, &v.SpaceID, &v.SourceKey, &v.Label, &v.Format, &v.Status, &v.Revision, &v.PlanHash, &v.DeclaredBytes, &v.UploadedBytes, &v.ItemCount, &v.PreparedCount, &v.JobID, &raw, &v.Error, &v.CreatedAt, &v.UpdatedAt, &v.ExpiresAt)
	if e == nil {
		e = json.Unmarshal(raw, &v.Report)
	}
	return v, e
}

type migrationSessionItem struct {
	ID              string               `json:"id"`
	SessionID       string               `json:"session_id"`
	SourceID        string               `json:"source_id"`
	SourceHash      string               `json:"source_hash"`
	FilePath        string               `json:"file_path"`
	Kind            string               `json:"kind"`
	SourceBytes     int64                `json:"source_bytes"`
	ChunkCount      int                  `json:"chunk_count"`
	ReceivedBytes   int64                `json:"received_bytes"`
	Status          string               `json:"status"`
	Checkpoint      int                  `json:"checkpoint"`
	TargetID        string               `json:"target_id,omitempty"`
	Disposition     string               `json:"disposition"`
	ExpectedVersion int64                `json:"expected_version"`
	BindingRevision int64                `json:"binding_revision"`
	TargetHash      string               `json:"target_hash"`
	ParentSourceID  string               `json:"parent_source_id"`
	Metadata        map[string]any       `json:"metadata"`
	PreparedData    []byte               `json:"-"`
	Compatibility   map[string]any       `json:"compatibility"`
	CSVTypes        []migrationCSVColumn `json:"csv_types"`
	TypesConfirmed  bool                 `json:"types_confirmed"`
	Error           string               `json:"error"`
}

const migrationItemSelect = `id::text,session_id::text,source_id,source_hash,file_path,kind,source_bytes,chunk_count,received_bytes,status,checkpoint,target_id::text,disposition,expected_version,binding_revision,target_hash,parent_source_id,metadata,prepared_data,compatibility,csv_types,types_confirmed,error`

func scanMigrationItem(row pgx.Row) (migrationSessionItem, error) {
	var v migrationSessionItem
	var metadata, compatibility, types []byte
	e := row.Scan(&v.ID, &v.SessionID, &v.SourceID, &v.SourceHash, &v.FilePath, &v.Kind, &v.SourceBytes, &v.ChunkCount, &v.ReceivedBytes, &v.Status, &v.Checkpoint, &v.TargetID, &v.Disposition, &v.ExpectedVersion, &v.BindingRevision, &v.TargetHash, &v.ParentSourceID, &metadata, &v.PreparedData, &compatibility, &types, &v.TypesConfirmed, &v.Error)
	if e == nil {
		e = json.Unmarshal(metadata, &v.Metadata)
	}
	if e == nil {
		e = json.Unmarshal(compatibility, &v.Compatibility)
	}
	if e == nil {
		e = json.Unmarshal(types, &v.CSVTypes)
	}
	return v, e
}

func migrationPublicItem(v migrationSessionItem) migrationSessionItem {
	copy := map[string]any{}
	for key, value := range v.Metadata {
		if !oneOf(key, "source_metadata_cipher", "source_metadata_hash", "previous_target_id") {
			copy[key] = value
		}
	}
	v.Metadata = copy
	v.PreparedData = nil
	v.TargetHash = ""
	return v
}

type migrationCSVColumn struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Options []string `json:"options"`
	Reason  string   `json:"reason,omitempty"`
	Empty   int      `json:"empty"`
	Samples []string `json:"samples"`
}

type migrationPrepared struct {
	Title          string        `json:"title"`
	Markdown       string        `json:"markdown,omitempty"`
	Tags           []string      `json:"tags"`
	Aliases        []string      `json:"aliases"`
	Icon           string        `json:"icon"`
	ParentID       string        `json:"parent_id,omitempty"`
	CSV            *migrationCSV `json:"csv,omitempty"`
	Attachments    []string      `json:"attachments,omitempty"`
	AttachmentData []byte        `json:"attachment_data,omitempty"`
	ContentType    string        `json:"content_type,omitempty"`
}

func (s *Server) migrationSessionAllowed(ctx context.Context, p *Principal, v migrationSession) bool {
	return p != nil && p.ID == v.UserID && hasIntegrationScope(p, "document:write") && s.canSpace(ctx, p, v.WorkspaceID, v.SpaceID, true)
}

func (s *Server) migrationSessionRequest(w http.ResponseWriter, r *http.Request) (migrationSession, bool) {
	v, e := scanMigrationSession(s.DB.QueryRow(r.Context(), "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1", r.PathValue("id")))
	if e != nil || !s.migrationSessionAllowed(r.Context(), current(r), v) {
		apiError(w, 404, "이관 세션을 찾을 수 없거나 현재 권한이 없습니다")
		return v, false
	}
	return v, true
}

func migrationSafeID(v string) bool {
	return strings.TrimSpace(v) != "" && len(v) <= 512 && !strings.ContainsAny(v, "\x00\r\n")
}

func (s *Server) migrationPreparedValue(v migrationSessionItem) (migrationPrepared, error) {
	var p migrationPrepared
	raw, e := s.decrypt(string(v.PreparedData))
	if e == nil {
		e = json.Unmarshal([]byte(raw), &p)
	}
	if e != nil {
		return p, errors.New("이관 준비 데이터를 해독하지 못했습니다")
	}
	return p, nil
}
