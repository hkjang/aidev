package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func migrationHexHash(v string) bool {
	data, e := hex.DecodeString(v)
	return e == nil && len(data) == sha256.Size && strings.ToLower(v) == v
}

func (s *Server) listMigrationSessions(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	if !hasIntegrationScope(current(r), "document:write") || !s.canWorkspace(r.Context(), current(r), wid, true) {
		apiError(w, 403, "이관 권한이 없습니다")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE workspace_id=$1 AND user_id=$2 ORDER BY created_at DESC LIMIT 100", wid, current(r).ID)
	if e != nil {
		respond(w, nil, e)
		return
	}
	list := []migrationSession{}
	for rows.Next() {
		v, e := scanMigrationSession(rows)
		if e != nil {
			rows.Close()
			respond(w, nil, e)
			return
		}
		list = append(list, v)
	}
	e = rows.Err()
	rows.Close()
	visible := []migrationSession{}
	for _, v := range list {
		if s.migrationSessionAllowed(r.Context(), current(r), v) {
			visible = append(visible, v)
		}
	}
	respond(w, visible, e)
}

func (s *Server) createMigrationSession(w http.ResponseWriter, r *http.Request) {
	var in struct {
		WorkspaceID string `json:"workspace_id"`
		SpaceID     string `json:"space_id"`
		SourceKey   string `json:"source_key"`
		Label       string `json:"label"`
		Format      string `json:"format"`
	}
	if decode(r, &in) != nil || !migrationSafeID(in.SourceKey) || strings.TrimSpace(in.Label) == "" || len(in.Label) > 200 || !oneOf(in.Format, "markdown", "obsidian", "notion", "html", "csv", "json") {
		apiError(w, 400, "이관 원본 식별자·이름·형식을 확인하세요")
		return
	}
	if !hasIntegrationScope(current(r), "document:write") || !s.canSpace(r.Context(), current(r), in.WorkspaceID, in.SpaceID, true) {
		apiError(w, 403, "대상 공간에 이관 권한이 없습니다")
		return
	}
	id := newID()
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,9325))", current(r).ID); e != nil {
		respond(w, nil, e)
		return
	}
	var count int
	if e = tx.QueryRow(r.Context(), "SELECT count(*) FROM migration_sessions WHERE user_id=$1 AND purged_at IS NULL AND expires_at>now()", current(r).ID).Scan(&count); e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 100 {
		apiError(w, 400, "보관 중인 이관은 개인별 100개까지입니다. 불필요한 원본을 취소하세요")
		return
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO migration_sessions(id,workspace_id,user_id,space_id,source_key,label,format,expires_at) SELECT $1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,now()+make_interval(days=>retention_days) FROM migration_settings WHERE id=1`, id, in.WorkspaceID, current(r).ID, in.SpaceID, in.SourceKey, in.Label, in.Format)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"id": id, "chunk_bytes": migrationChunkBytes}, e)
}

func (s *Server) getMigrationSession(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if ok {
		jsonResponse(w, 200, v)
	}
}

func (s *Server) listMigrationSessionItems(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		offset, _ = strconv.Atoi(raw)
	}
	if offset < 0 || offset > 100000 {
		apiError(w, 400, "조회 위치를 확인하세요")
		return
	}
	rows, e := s.DB.Query(r.Context(), "SELECT "+strings.Replace(migrationItemSelect, "prepared_data", "NULL::bytea", 1)+" FROM migration_session_items WHERE session_id=$1 ORDER BY file_path,id LIMIT 101 OFFSET $2", v.ID, offset)
	if e != nil {
		respond(w, nil, e)
		return
	}
	items := []migrationSessionItem{}
	for rows.Next() {
		item, e := scanMigrationItem(rows)
		if e != nil {
			rows.Close()
			respond(w, nil, e)
			return
		}
		item.PreparedData = nil
		items = append(items, migrationPublicItem(item))
	}
	e = rows.Err()
	rows.Close()
	more := len(items) > 100
	if more {
		items = items[:100]
	}
	respond(w, map[string]any{"items": items, "has_more": more, "next_offset": offset + len(items)}, e)
}

func (s *Server) registerMigrationSessionItems(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	var in struct {
		Items []struct {
			SourceID       string         `json:"source_id"`
			Path           string         `json:"path"`
			Kind           string         `json:"kind"`
			SHA256         string         `json:"sha256"`
			Bytes          int64          `json:"bytes"`
			ParentSourceID string         `json:"parent_source_id"`
			Metadata       map[string]any `json:"metadata"`
		} `json:"items"`
	}
	if decode(r, &in) != nil || len(in.Items) == 0 || len(in.Items) > 100 {
		apiError(w, 400, "매니페스트는 한 요청에 1~100개 항목입니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	v, e = scanMigrationSession(tx.QueryRow(r.Context(), "SELECT "+migrationSessionSelect+" FROM migration_sessions WHERE id=$1 AND expires_at>now() FOR UPDATE", v.ID))
	if e != nil || v.Status != "uploading" {
		apiError(w, 409, "업로드 단계의 유효한 이관 세션이 필요합니다")
		return
	}
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1,9325))", v.UserID); e != nil {
		respond(w, nil, e)
		return
	}
	var maxSession, maxUser, used int64
	var maxItems int
	e = tx.QueryRow(r.Context(), "SELECT max_session_bytes,max_user_bytes,max_items FROM migration_settings WHERE id=1 FOR SHARE").Scan(&maxSession, &maxUser, &maxItems)
	if e == nil {
		e = tx.QueryRow(r.Context(), "SELECT coalesce(sum(declared_bytes),0) FROM migration_sessions WHERE user_id=$1 AND purged_at IS NULL AND expires_at>now()", v.UserID).Scan(&used)
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	results := []map[string]any{}
	for _, item := range in.Items {
		if item.Metadata == nil {
			item.Metadata = map[string]any{}
		}
		for key := range item.Metadata {
			if !oneOf(key, "title", "tags", "aliases", "icon") {
				apiError(w, 400, "원본 메타데이터는 title/tags/aliases/icon만 지원합니다")
				return
			}
		}
		if len(jsonValue(item.Metadata)) > 64<<10 {
			apiError(w, 400, "원본 메타데이터는 64KB 이하입니다")
			return
		}
		limit := int64(50 << 20)
		if item.Kind == "document" {
			limit = 4 << 20
		}
		if !migrationSafeID(item.SourceID) || strings.HasPrefix(item.SourceID, "@madi-folder:") || !safeVaultPath(item.Path) || len(item.Path) > 512 || strings.HasSuffix(item.Path, "/") || strings.HasPrefix(path.Base(item.Path), ".madi-") || !oneOf(item.Kind, "document", "attachment", "csv") || item.Bytes < 0 || item.Bytes > limit || !migrationHexHash(item.SHA256) || (item.ParentSourceID != "" && !migrationSafeID(item.ParentSourceID)) {
			apiError(w, 400, "항목 식별자·상대 경로·형식·크기·SHA256을 확인하세요(문서 4MiB, 파일 50MiB)")
			return
		}
		if strings.Count(item.Path, "/") >= 20 || oneOf(strings.Split(item.Path, "/")[0], ".git", ".obsidian", "__MACOSX") {
			apiError(w, 400, "이관 경로는 20단계 이내이며 시스템 설정 폴더는 제외해야 합니다")
			return
		}
		if item.Kind == "csv" && !hasIntegrationScope(current(r), "database:write") {
			apiError(w, 403, "CSV 이관에는 database:write 권한이 필요합니다")
			return
		}
		old, e := scanMigrationItem(tx.QueryRow(r.Context(), "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE session_id=$1 AND source_id=$2", v.ID, item.SourceID))
		if e == nil {
			if old.SourceHash != item.SHA256 || old.SourceBytes != item.Bytes || old.FilePath != item.Path || old.Kind != item.Kind || old.ParentSourceID != item.ParentSourceID || str(old.Metadata, "source_metadata_hash") != digest(string(jsonValue(item.Metadata))) {
				apiError(w, 409, "같은 원본 ID의 내용이 달라졌습니다. 기존 세션을 취소하거나 새 세션으로 준비하세요")
				return
			}
			results = append(results, map[string]any{"id": old.ID, "source_id": old.SourceID, "received_bytes": old.ReceivedBytes, "chunk_count": old.ChunkCount})
			continue
		}
		if e != pgx.ErrNoRows {
			respond(w, nil, e)
			return
		}
		if v.DeclaredBytes+item.Bytes > maxSession || used+item.Bytes > maxUser || v.ItemCount+1 > maxItems {
			apiError(w, 400, "관리자가 설정한 이관 세션·개인 보관량 또는 항목 수 한도를 초과했습니다")
			return
		}
		id := newID()
		count := int((item.Bytes + migrationChunkBytes - 1) / migrationChunkBytes)
		state := "uploading"
		if item.Bytes == 0 {
			state = "pending"
		}
		cipher, err := s.encrypt(string(jsonValue(item.Metadata)))
		if err != nil {
			respond(w, nil, err)
			return
		}
		metadata := map[string]any{"source_metadata_cipher": cipher, "source_metadata_hash": digest(string(jsonValue(item.Metadata)))}
		_, e = tx.Exec(r.Context(), `INSERT INTO migration_session_items(id,session_id,source_id,source_hash,file_path,kind,source_bytes,chunk_count,target_id,parent_source_id,status,metadata) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, v.ID, item.SourceID, item.SHA256, item.Path, item.Kind, item.Bytes, count, newID(), item.ParentSourceID, state, jsonValue(metadata))
		if e != nil {
			apiError(w, 409, "같은 경로가 이미 등록되었습니다. 원본 ID와 경로를 확인하세요")
			return
		}
		v.DeclaredBytes += item.Bytes
		used += item.Bytes
		v.ItemCount++
		results = append(results, map[string]any{"id": id, "source_id": item.SourceID, "received_bytes": 0, "chunk_count": count})
	}
	_, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET declared_bytes=$2,item_count=$3,revision=revision+1,updated_at=now() WHERE id=$1", v.ID, v.DeclaredBytes, v.ItemCount)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"items": results, "chunk_bytes": migrationChunkBytes}, e)
}

func (s *Server) uploadMigrationSessionChunk(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	ordinal, e := strconv.Atoi(r.PathValue("chunk"))
	if e != nil || ordinal < 0 {
		apiError(w, 400, "청크 번호를 확인하세요")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, migrationChunkBytes+1)
	raw, e := io.ReadAll(r.Body)
	if e != nil || len(raw) > migrationChunkBytes {
		apiError(w, 400, "청크는 1MiB 이하여야 합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	var status string
	e = tx.QueryRow(r.Context(), "SELECT status FROM migration_sessions WHERE id=$1 AND expires_at>now() FOR UPDATE", v.ID).Scan(&status)
	if e != nil || status != "uploading" {
		apiError(w, 409, "이관 세션이 업로드 상태가 아니거나 만료되었습니다")
		return
	}
	item, e := scanMigrationItem(tx.QueryRow(r.Context(), "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE session_id=$1 AND id=$2 FOR UPDATE", v.ID, r.PathValue("item")))
	if e != nil {
		apiError(w, 404, "이관 항목을 찾을 수 없습니다")
		return
	}
	want := int64(migrationChunkBytes)
	if ordinal == item.ChunkCount-1 {
		want = item.SourceBytes - int64(ordinal*migrationChunkBytes)
	}
	if ordinal >= item.ChunkCount || int64(len(raw)) != want {
		apiError(w, 400, "청크 번호와 매니페스트의 정확한 크기가 일치하지 않습니다")
		return
	}
	hash := sha256.Sum256(raw)
	checksum := hex.EncodeToString(hash[:])
	var previous string
	e = tx.QueryRow(r.Context(), "SELECT checksum FROM migration_session_chunks WHERE item_id=$1 AND ordinal=$2", item.ID, ordinal).Scan(&previous)
	if e == nil {
		if previous != checksum {
			apiError(w, 409, "이미 받은 청크와 다른 데이터입니다")
			return
		}
		jsonResponse(w, 200, map[string]any{"ok": true, "duplicate": true, "checksum": checksum})
		return
	}
	if e != pgx.ErrNoRows {
		respond(w, nil, e)
		return
	}
	cipher, e := s.encrypt(string(raw))
	if e != nil {
		respond(w, nil, e)
		return
	}
	_, e = tx.Exec(r.Context(), "INSERT INTO migration_session_chunks(item_id,ordinal,size,checksum,data) VALUES($1,$2,$3,$4,$5)", item.ID, ordinal, len(raw), checksum, []byte(cipher))
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_session_items SET received_bytes=received_bytes+$2,status=CASE WHEN received_bytes+$2=source_bytes THEN 'pending' ELSE 'uploading' END,updated_at=now() WHERE id=$1", item.ID, len(raw))
	}
	if e == nil {
		_, e = tx.Exec(r.Context(), "UPDATE migration_sessions SET uploaded_bytes=uploaded_bytes+$2,updated_at=now() WHERE id=$1", v.ID, len(raw))
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	respond(w, map[string]any{"ok": true, "checksum": checksum}, e)
}

func (s *Server) migrationSessionItemChunks(w http.ResponseWriter, r *http.Request) {
	v, ok := s.migrationSessionRequest(w, r)
	if !ok {
		return
	}
	items, e := s.rows(r.Context(), "SELECT jsonb_build_object('ordinal',c.ordinal,'checksum',c.checksum,'size',c.size) FROM migration_session_chunks c JOIN migration_session_items i ON i.id=c.item_id WHERE i.session_id=$1 AND i.id=$2 ORDER BY c.ordinal", v.ID, r.PathValue("item"))
	respond(w, items, e)
}

func (s *Server) migrationItemRaw(ctx context.Context, item migrationSessionItem) ([]byte, error) {
	if item.SourceBytes > 50<<20 {
		return nil, errors.New("항목 원본 크기 제한을 초과했습니다")
	}
	rows, e := s.DB.Query(ctx, "SELECT ordinal,data FROM migration_session_chunks WHERE item_id=$1 ORDER BY ordinal", item.ID)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	raw := make([]byte, 0, item.SourceBytes)
	next := 0
	for rows.Next() {
		var ordinal int
		var cipher []byte
		if e = rows.Scan(&ordinal, &cipher); e != nil {
			return nil, e
		}
		if ordinal != next {
			return nil, errors.New("누락된 청크가 있습니다")
		}
		plain, e := s.decrypt(string(cipher))
		if e != nil {
			return nil, errors.New("이관 청크를 해독하지 못했습니다")
		}
		raw = append(raw, plain...)
		if int64(len(raw)) > item.SourceBytes {
			return nil, errors.New("매니페스트보다 큰 데이터입니다")
		}
		next++
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	hash := sha256.Sum256(raw)
	if int64(len(raw)) != item.SourceBytes || next != item.ChunkCount || hex.EncodeToString(hash[:]) != item.SourceHash {
		return nil, errors.New("원본 크기 또는 SHA256 검증에 실패했습니다")
	}
	return raw, nil
}

func (s *Server) migrationSetting(w http.ResponseWriter, r *http.Request) {
	if current(r).Role != "admin" || current(r).TokenID != "" || current(r).ScopeRestricted {
		apiError(w, 403, "서비스 관리자 로그인으로 설정하세요")
		return
	}
	if r.Method == http.MethodGet {
		v, e := s.one(r.Context(), "SELECT to_jsonb(c)||jsonb_build_object('chunk_bytes',$1::integer) FROM migration_settings c WHERE id=1", migrationChunkBytes)
		respond(w, v, e)
		return
	}
	var in struct {
		MaxSessionBytes int64 `json:"max_session_bytes"`
		MaxUserBytes    int64 `json:"max_user_bytes"`
		MaxItems        int   `json:"max_items"`
		RetentionDays   int   `json:"retention_days"`
		Revision        int64 `json:"revision"`
	}
	if decode(r, &in) != nil || in.MaxSessionBytes < 1<<20 || in.MaxSessionBytes > 8<<30 || in.MaxUserBytes < in.MaxSessionBytes || in.MaxUserBytes > 16<<30 || in.MaxItems < 1 || in.MaxItems > 100000 || in.RetentionDays < 1 || in.RetentionDays > 30 {
		apiError(w, 400, "세션 1MiB~8GiB, 개인 한도는 세션 이상~16GiB, 항목 1~100,000개, 보관 1~30일 범위를 확인하세요")
		return
	}
	tag, e := s.DB.Exec(r.Context(), "UPDATE migration_settings SET max_session_bytes=$1,max_user_bytes=$2,max_items=$3,retention_days=$4,revision=revision+1 WHERE id=1 AND revision=$5", in.MaxSessionBytes, in.MaxUserBytes, in.MaxItems, in.RetentionDays, in.Revision)
	if e == nil && tag.RowsAffected() != 1 {
		apiError(w, 409, "설정이 변경되었습니다. 다시 확인하세요")
		return
	}
	if e == nil {
		s.audit(r, "MIGRATION_POLICY_UPDATE", "migration_settings", map[string]any{"revision": in.Revision + 1, "max_session_bytes": in.MaxSessionBytes, "max_user_bytes": in.MaxUserBytes, "max_items": in.MaxItems, "retention_days": in.RetentionDays})
	}
	respond(w, map[string]bool{"ok": true}, e)
}

// File paths are never passed to a shell or used as writable filesystem paths.
func migrationFolderSource(dir string) string { return "@madi-folder:" + path.Clean(dir) }

func migrationMetadataCipher(s *Server, v any) ([]byte, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	cipher, e := s.encrypt(string(raw))
	return []byte(cipher), e
}
