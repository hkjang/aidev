package server

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/jackc/pgx/v5"
	"github.com/reearth/ygo/crdt"
)

//go:embed collaboration.sql
var collaborationSchema string

const (
	collaborationMaxUpdate = 8 << 20
	collaborationMaxState  = 16 << 20
	collaborationSchemaID  = "madi-tiptap-v1"
)

func (s *Server) migrateCollaboration(ctx context.Context) error {
	_, err := s.DB.Exec(ctx, collaborationSchema+"\n"+collaborationJournalSchema)
	return err
}

func (s *Server) registerCollaboration() {
	s.handle("GET /api/v1/documents/{id}/collaboration", s.collaborationSocket)
	s.handle("GET /api/v1/documents/{id}/collaboration/diagnostics", s.collaborationDiagnostics)
	s.handle("POST /api/v1/documents/{id}/collaboration/compact", s.compactCollaboration)
	s.handle("GET /api/v1/documents/{id}/collaboration/history", s.collaborationHistory)
}

type collaborationMessage struct {
	connectionID string
	Type         string           `json:"type"`
	Schema       string           `json:"schema,omitempty"`
	Epoch        string           `json:"epoch,omitempty"`
	Sequence     int64            `json:"sequence,omitempty"`
	Version      int              `json:"version,omitempty"`
	State        []byte           `json:"state,omitempty"`
	Markdown     string           `json:"markdown,omitempty"`
	Title        string           `json:"title,omitempty"`
	Tags         []string         `json:"tags,omitempty"`
	Status       string           `json:"status,omitempty"`
	CanWrite     bool             `json:"can_write"`
	ID           int64            `json:"id,omitempty"`
	ClientID     int64            `json:"client_id,omitempty"`
	Clock        int64            `json:"clock,omitempty"`
	Awareness    json.RawMessage  `json:"awareness,omitempty"`
	Code         string           `json:"code,omitempty"`
	Error        string           `json:"error,omitempty"`
	Presence     []map[string]any `json:"presence,omitempty"`
	User         map[string]any   `json:"user,omitempty"`
	Committed    bool             `json:"committed,omitempty"`
	ResetReason  string           `json:"reset_reason,omitempty"`
	StateMode    string           `json:"state_mode,omitempty"`
	FromSequence int64            `json:"from_sequence,omitempty"`
}

type collaborationFault struct{ code, message string }

func (e *collaborationFault) Error() string { return e.message }

func collaborationError(code, message string) error { return &collaborationFault{code, message} }

type collaborationQuery interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Never trust a Principal captured at the upgrade: sessions, global roles,
// membership and selected-document shares may all change while a socket is idle.
func collaborationAccess(ctx context.Context, q collaborationQuery, docID, session string, lock bool) (*Principal, bool, error) {
	query := `SELECT u.id,u.email,u.name,u.role,u.kind,madi_document_allowed(u.id,d.id,true),d.workspace_id::text
	 FROM documents d JOIN workspace_members m ON m.workspace_id=d.workspace_id
	 JOIN users u ON u.id=m.user_id JOIN sessions t ON t.user_id=u.id
	 WHERE d.id=$1 AND t.token_hash=$2 AND t.expires_at>clock_timestamp()
	 AND NOT u.disabled AND u.kind='user' AND d.deleted_at IS NULL
	 AND madi_document_allowed(u.id,d.id,false)`
	if lock {
		query += " FOR SHARE OF u,m,t"
	}
	p := &Principal{}
	var write bool
	var wid string
	if err := q.QueryRow(ctx, query, docID, session).Scan(&p.ID, &p.Email, &p.Name, &p.Role, &p.Kind, &write, &wid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, collaborationError("forbidden", "세션 또는 문서 접근 권한이 만료되었습니다")
		}
		return nil, false, err
	}
	if !featureAllowed(ctx, q, p, wid, "collaboration") {
		return nil, false, collaborationError("feature_disabled", "현재 기능 정책에서 공동 편집이 꺼져 있습니다. Markdown 원문 모드에서 계속 편집할 수 있습니다.")
	}
	policyQuery := `SELECT COALESCE((data->>'enabled')::boolean,false) AND data->>'mode' IN ('block','mask') FROM protection_settings WHERE id=1`
	if lock {
		policyQuery += " FOR SHARE"
	}
	var strict bool
	if err := q.QueryRow(ctx, policyQuery).Scan(&strict); err != nil {
		return nil, false, err
	}
	if strict {
		return nil, false, collaborationError("protection_policy", "관리자 정보보호 정책에 따라 공동 편집을 사용하지 않습니다. 숨겨진 편집 이력까지 정제할 수 없어 Markdown 원문 모드에서 내용을 확인하고 저장하세요.")
	}
	return p, write, nil
}

// The document/journal lock and CRDT projection may outlive a session. now()
// would use transaction start time, so both the current ACL/policy pass and the
// final expiry check use wall time immediately before commit. The caller already
// holds the actor/session rows FOR SHARE; logout/revocation cannot race that lock.
func collaborationCommitTx(ctx context.Context, tx pgx.Tx, docID, session string, writing bool) error {
	_, writable, err := collaborationAccess(ctx, tx, docID, session, false)
	if err != nil {
		return err
	}
	if writing && !writable {
		return collaborationError("read_only", "이 문서는 읽기 전용입니다")
	}
	var active bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM sessions WHERE token_hash=$1 AND expires_at>clock_timestamp())", session).Scan(&active); err != nil {
		return err
	}
	if !active {
		return collaborationError("forbidden", "세션 또는 문서 접근 권한이 만료되었습니다")
	}
	return tx.Commit(ctx)
}

func collaborationPolicyError(ctx context.Context, c *websocket.Conn, err error) bool {
	var fault *collaborationFault
	if !errors.As(err, &fault) || !oneOf(fault.code, "protection_policy", "feature_disabled") {
		return false
	}
	_ = wsjson.Write(ctx, c, collaborationMessage{Type: "error", Code: fault.code, Error: fault.message})
	return true
}

func collaborationUser(p *Principal) map[string]any {
	colors := []string{"#0f766e", "#7c3aed", "#b45309", "#be123c", "#1d4ed8", "#4d7c0f"}
	n := 0
	for _, c := range p.ID {
		n += int(c)
	}
	return map[string]any{"id": p.ID, "name": p.Name, "color": colors[n%len(colors)]}
}

// A single database row lock serializes writers across service replicas. We
// hold a private cache entry only after that lock. Failed/rolled-back operations
// discard it entirely; cache misses replay a disposable checkpoint and tail.
func (s *Server) collaborationState(ctx context.Context, docID, session string, in *collaborationMessage) (out collaborationMessage, err error) {
	return s.collaborationStateSince(ctx, docID, session, in, "", 0)
}

func (s *Server) collaborationStateSince(ctx context.Context, docID, session string, in *collaborationMessage, sinceEpoch string, sinceSequence int64) (out collaborationMessage, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Warn("collaboration parser rejected input", "panic_type", fmt.Sprintf("%T", recovered))
			err = collaborationError("invalid_update", "편집 상태를 해석할 수 없습니다. 로컬 초안을 보관하고 다시 연결하세요")
		}
	}()
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var markdown, title, status, workspaceID string
	var tags []string
	var version int
	if err = tx.QueryRow(ctx, "SELECT markdown,version,title,status,ARRAY(SELECT jsonb_array_elements_text(tags)),workspace_id FROM documents WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", docID).Scan(&markdown, &version, &title, &status, &tags, &workspaceID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = collaborationError("forbidden", "문서가 삭제되었거나 접근할 수 없습니다")
		}
		return out, err
	}
	p, writable, err := collaborationAccess(ctx, tx, docID, session, true)
	if err != nil {
		return out, err
	}
	// Background/socket calls need the authenticated actor, just like REST
	// mutations, so chained automations never inherit a service-level identity.
	ctx = context.WithValue(ctx, principalKey, p)
	out = collaborationMessage{Type: "sync", Schema: collaborationSchemaID, Markdown: markdown, Version: version, CanWrite: writable, User: collaborationUser(p), Title: title, Tags: tags, Status: status}
	_, err = tx.Exec(ctx, `INSERT INTO document_collaboration(document_id,epoch,projected_markdown,document_version)
	 VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING`, docID, newID(), markdown, version)
	if err != nil {
		return out, err
	}
	var projection string
	var snapshotSequence int64
	var snapshotAt time.Time
	var headHash string
	if err = tx.QueryRow(ctx, "SELECT epoch,sequence,state,projected_markdown,snapshot_sequence,snapshot_at,reset_reason,head_hash FROM document_collaboration WHERE document_id=$1", docID).Scan(&out.Epoch, &out.Sequence, &out.State, &projection, &snapshotSequence, &snapshotAt, &out.ResetReason, &headHash); err != nil {
		return out, err
	}
	var runtime *collaborationRuntime
	var cached *collaborationCachedDocument
	cacheHit, cacheCommitted := false, false
	if in != nil {
		runtime, cached = s.collaborationCached(docID)
		if cached != nil {
			cacheHit = projection == markdown && cached.doc != nil && cached.epoch == out.Epoch && cached.sequence == out.Sequence && headHash != "" && cached.hash == headHash
			if cacheHit {
				runtime.cacheHits.Add(1)
			} else {
				runtime.cacheMisses.Add(1)
			}
			if !cacheHit {
				runtime.discardCached(cached)
			}
			defer func() {
				if !cacheCommitted {
					runtime.discardCached(cached)
				}
				cached.mu.Unlock()
			}()
		}
	}
	if projection != markdown {
		out.Type, out.Epoch, out.Sequence, out.State = "reset", newID(), 0, nil
		out.ResetReason = "source_replaced"
		snapshotSequence = 0
		err = collaborationResetTx(ctx, tx, docID, out.Epoch, markdown, version, out.ResetReason)
		if err != nil {
			return out, err
		}
	} else {
		out.StateMode = "full"
		if in == nil && sinceEpoch == out.Epoch && sinceSequence >= snapshotSequence && sinceSequence <= out.Sequence && len(out.State) > 0 {
			out.StateMode, out.FromSequence = "delta", sinceSequence
			out.State, err = collaborationReplay(ctx, tx, docID, out.Epoch, []byte{0, 0}, sinceSequence, out.Sequence)
		} else if cacheHit {
			out.State = cached.raw
		} else {
			out.State, err = collaborationReplay(ctx, tx, docID, out.Epoch, out.State, snapshotSequence, out.Sequence)
		}
		if err != nil {
			return out, err
		}
	}
	if in == nil {
		return out, collaborationCommitTx(ctx, tx, docID, session, false)
	}
	out.ID = in.ID
	if !writable {
		return out, collaborationError("read_only", "이 문서는 읽기 전용입니다")
	}
	if in.connectionID != "" {
		// The document row lock serializes this quota across tabs and replicas.
		var count int
		if err = tx.QueryRow(ctx, `SELECT coalesce(sum(operations),0) FROM collaboration_presence WHERE document_id=$1 AND user_id=$2 AND operation_window=date_trunc('second',now())`, docID, p.ID).Scan(&count); err != nil {
			return out, err
		}
		if count >= 40 {
			return out, collaborationError("rate_limit", "사용자별 문서 편집 속도 한도를 초과했습니다")
		}
		if _, err = tx.Exec(ctx, `UPDATE collaboration_presence SET operations=CASE WHEN operation_window=date_trunc('second',now()) THEN operations+1 ELSE 1 END,operation_window=date_trunc('second',now()) WHERE connection_id=$1`, in.connectionID); err != nil {
			return out, err
		}
	}
	if in.Epoch != out.Epoch || (in.Type == "seed" && (len(out.State) != 0 || in.Version != version)) {
		out.Type = "reset"
		return out, collaborationCommitTx(ctx, tx, docID, session, true)
	}
	if len(in.State) == 0 || len(in.State) > collaborationMaxUpdate || in.Schema != collaborationSchemaID {
		return out, collaborationError("invalid_update", "편집 스키마 또는 업데이트 크기를 확인하세요")
	}
	if len(out.State) == 0 && in.Type != "seed" {
		return out, collaborationError("seed_required", "현재 Markdown으로 공동 편집을 초기화하세요")
	}
	var doc *crdt.Doc
	if cacheHit {
		doc = cached.doc
	} else {
		doc = crdt.New(crdt.WithMaxPendingItems(1024))
		if cached != nil {
			cached.doc = doc
		}
	}
	if cached == nil {
		defer doc.Destroy()
	}
	// Materialise the XML root before decoding to prevent ambiguous root types.
	doc.GetXmlFragment("content")
	previousBody := ""
	if cacheHit {
		previousBody = cached.body
	} else if len(out.State) != 0 {
		if err = crdt.ApplyUpdateV1(doc, out.State, nil); err != nil {
			return out, err
		}
		previousBody, _, err = collaborationMarkdown(doc.GetXmlFragment("content"))
		if err != nil {
			return out, err
		}
	}
	priorVector := doc.StateVector()
	var priorState []byte
	if cacheHit {
		priorState = cached.raw
	} else {
		priorState = doc.EncodeStateAsUpdate()
	}
	if err = crdt.ApplyUpdateV1(doc, in.State, nil); err != nil {
		return out, collaborationError("invalid_update", "유효하지 않은 공동 편집 업데이트입니다")
	}
	if pending := doc.PendingStats(); pending.Items != 0 || pending.DeleteRanges != 0 {
		return out, collaborationError("missing_state", "업데이트의 선행 상태가 없습니다. 전체 편집 상태로 다시 동기화하세요")
	}
	state := doc.EncodeStateAsUpdate()
	if len(state) > collaborationMaxState+collaborationMaxUpdate {
		return out, collaborationError("state_limit", "공동 편집 이력이 16MB 한도에 도달했습니다. Markdown을 보관한 뒤 문서 복제로 새 편집 이력을 시작하세요")
	}
	body, metadata, err := collaborationMarkdown(doc.GetXmlFragment("content"))
	if err != nil {
		return out, err
	}
	front, _ := collaborationFrontMatter(markdown)
	projected := front + body
	if in.Type == "seed" {
		_, originalBody := collaborationFrontMatter(markdown)
		if !collaborationSeedEquivalent(originalBody, body) {
			return out, collaborationError("seed_mismatch", "블록 편집기 표현과 Markdown 원문이 일치하지 않습니다. 원문을 보존하기 위해 Markdown 모드에서 편집하세요.")
		}
		// Opening an editor is not an edit. Keep exact trailing newlines, list
		// markers/reference syntax and front matter until a real body change.
		projected = markdown
	} else if body == previousBody {
		// Unique IDs/selection metadata and identical rejoin updates must not
		// normalize an untouched source or create a new version/approval request.
		projected = markdown
	}
	if len(projected) > 4<<20 {
		return out, collaborationError("document_limit", "문서는 4MB 이하여야 합니다")
	}
	protection, e := s.ProtectDocumentTx(ctx, tx, p, docID, workspaceID, title, projected)
	if e != nil {
		return out, e
	}
	if protection.Changed {
		return out, ProtectionMaskRequired(protection)
	}
	metadataProtection, e := s.ProtectDocumentMetadataTx(ctx, tx, p, docID, workspaceID, metadata)
	if e != nil {
		return out, e
	}
	if metadataProtection.Changed {
		return out, collaborationError("protection_policy", "블록 메타데이터 정제가 필요합니다. Markdown 원문 모드에서 내용을 확인하고 저장하세요.")
	}
	if !bytes.Equal(state, priorState) {
		out.Sequence++
		if projected != markdown {
			// This small, non-secret setting is read in the transaction and locked
			// against a concurrent approval-policy switch.
			var raw []byte
			if err = tx.QueryRow(ctx, "SELECT data FROM settings WHERE id=1 FOR SHARE").Scan(&raw); err != nil {
				return out, err
			}
			cfg := map[string]any{}
			if err = json.Unmarshal(raw, &cfg); err != nil {
				return out, err
			}
			if boolean(cfg, "approval_enabled") {
				out.Status = "draft"
			}
			_, err = tx.Exec(ctx, `UPDATE documents SET markdown=$2,block_metadata=$3,version=version+1,updated_at=now(),
			 status=CASE WHEN $4 THEN 'draft' ELSE status END WHERE id=$1`, docID, projected, jsonValue(metadata), boolean(cfg, "approval_enabled"))
			if err != nil {
				return out, err
			}
			version++
			_, err = tx.Exec(ctx, `INSERT INTO document_versions(document_id,version,title,markdown,tags,user_id,block_metadata)
			 SELECT id,version,title,markdown,tags,$2,block_metadata FROM documents WHERE id=$1`, docID, p.ID)
			if err != nil {
				return out, err
			}
			if err = collaborationGroupHistoryTx(ctx, tx, docID, out.Epoch, p.ID, version); err != nil {
				return out, err
			}
			before := map[string]any{"id": docID, "title": title, "status": status, "version": version - 1, "tags": tags}
			after := map[string]any{"id": docID, "title": title, "status": out.Status, "version": version, "tags": tags}
			if err = s.enqueueEvent(ctx, tx, Event{Type: "document.updated", WorkspaceID: workspaceID, ResourceID: docID, ActorID: p.ID, Before: before, After: after}); err != nil {
				return out, err
			}
			if status != out.Status {
				if err = s.enqueueEvent(ctx, tx, Event{Type: "document.status_changed", WorkspaceID: workspaceID, ResourceID: docID, ActorID: p.ID, Before: before, After: after}); err != nil {
					return out, err
				}
			}
			if err = s.enqueueTaskCompletions(ctx, tx, workspaceID, docID, title, markdown, projected); err != nil {
				return out, err
			}
		} else {
			// Block IDs are derived from authenticated editor state, never supplied
			// independently by a browser snapshot.
			if _, err = tx.Exec(ctx, "UPDATE documents SET block_metadata=$2 WHERE id=$1", docID, jsonValue(metadata)); err != nil {
				return out, err
			}
		}
		if len(state) >= collaborationCompactBytes {
			out.Type, out.Epoch, out.Sequence, out.State = "reset", newID(), 0, nil
			out.ResetReason = "history_compacted"
			out.Markdown, out.Version, out.Committed = projected, version, true
			err = collaborationResetTx(ctx, tx, docID, out.Epoch, projected, version, out.ResetReason)
			if err != nil {
				return out, err
			}
			return out, collaborationCommitTx(ctx, tx, docID, session, true)
		}
		delta := crdt.EncodeStateAsUpdateV1(doc, priorVector)
		if len(delta) > collaborationMaxUpdate {
			return out, collaborationError("state_limit", "증분 저장 한도를 초과했습니다. 현재 초안을 보관하고 편집 이력을 압축하세요")
		}
		err = collaborationPersistTx(ctx, tx, docID, out.Epoch, out.Sequence, snapshotSequence, snapshotAt, version, p.ID, delta, state, projected)
		if err != nil {
			return out, err
		}
	}
	out.Type, out.State, out.Markdown, out.Version, out.Committed = "ack", state, projected, version, true
	err = collaborationCommitTx(ctx, tx, docID, session, true)
	if err == nil && cached != nil && runtime.ctx.Err() == nil {
		// Weighted admission is conservative, not an RSS guarantee: native CRDT
		// allocations can be much larger than their encoded wire representation.
		weight := int64(len(state)*8 + len(projected)*2)
		if runtime.cacheBytes.Add(weight-cached.weight) <= collaborationCacheBudget {
			cached.weight = weight
			cached.epoch, cached.sequence, cached.hash, cached.body, cached.raw = out.Epoch, out.Sequence, digest(string(state)), body, state
			cacheCommitted = true
		} else {
			runtime.cacheBytes.Add(cached.weight - weight)
		}
	}
	return out, err
}

func collaborationFrontMatter(markdown string) (string, string) {
	if strings.HasPrefix(markdown, "---\n") || strings.HasPrefix(markdown, "---\r\n") {
		start := strings.IndexByte(markdown, '\n') + 1
		for offset := start; offset < len(markdown); {
			end := strings.IndexByte(markdown[offset:], '\n')
			if end < 0 {
				end = len(markdown) - offset
			} else {
				end++
			}
			line := markdown[offset : offset+end]
			offset += end
			if strings.TrimRight(line, "\r\n") == "---" {
				return markdown[:offset], markdown[offset:]
			}
		}
	}
	return "", markdown
}

func (s *Server) collaborationSocket(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	cookie, err := r.Cookie("madi_session")
	if !validID(id) || err != nil || current(r).TokenID != "" {
		apiError(w, 401, "공동 편집은 사용자 로그인 세션이 필요합니다")
		return
	}
	// Browsers always send Origin on WebSocket. Requiring it also prevents
	// accidentally relaxing cookie authentication for non-browser clients.
	if r.Header.Get("Origin") == "" || !s.sameOrigin(r) {
		apiError(w, 403, "허용되지 않은 공동 편집 연결 출처입니다")
		return
	}
	session := digest(cookie.Value)
	initial, err := s.collaborationState(r.Context(), id, session, nil)
	if err != nil {
		var fault *collaborationFault
		if errors.As(err, &fault) && oneOf(fault.code, "protection_policy", "feature_disabled") {
			c, e := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, CompressionMode: websocket.CompressionDisabled})
			if e != nil {
				return
			}
			defer c.CloseNow()
			writeCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()
			collaborationPolicyError(writeCtx, c, err)
			_ = c.Close(websocket.StatusPolicyViolation, "정보보호 정책: 원문 편집 필요")
			return
		}
		apiError(w, 403, "문서에 연결할 수 없습니다")
		return
	}
	connectionID := newID()
	// PostgreSQL advisory transaction lock makes the quota effective across replicas.
	tx, err := s.DB.Begin(r.Context())
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(726234806)"); err != nil {
		respond(w, nil, err)
		return
	}
	_, _ = tx.Exec(r.Context(), "DELETE FROM collaboration_presence WHERE touched_at<now()-interval '60 seconds'")
	var total, perDoc, perUser int
	err = tx.QueryRow(r.Context(), `SELECT count(*),count(*) FILTER(WHERE document_id=$1),count(*) FILTER(WHERE user_id=$2 AND document_id=$1) FROM collaboration_presence`, id, current(r).ID).Scan(&total, &perDoc, &perUser)
	if err != nil {
		respond(w, nil, err)
		return
	}
	if total >= 2048 || perDoc >= 64 || perUser >= 8 {
		apiError(w, 429, "공동 편집 연결 한도에 도달했습니다")
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO collaboration_presence(connection_id,document_id,user_id,epoch) VALUES($1,$2,$3,$4)`, connectionID, id, current(r).ID, initial.Epoch)
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		respond(w, nil, err)
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = s.DB.Exec(ctx, "DELETE FROM collaboration_presence WHERE connection_id=$1", connectionID)
	}()
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true, CompressionMode: websocket.CompressionDisabled})
	// InsecureSkipVerify only disables the library's redundant host check; the
	// strict scheme+host configured-origin validation was performed above.
	if err != nil {
		return
	}
	defer c.CloseNow()
	c.SetReadLimit(12 << 20)
	runtime, changedEvents, authorityEvents, unsubscribe := s.subscribeCollaboration(id)
	defer unsubscribe()
	ctx, cancel := context.WithCancel(runtime.ctx)
	defer cancel()
	// Authority checks are not blocked by a slow update/parser or a slow reader.
	// Notifications wake this independently; the timer is a fallback, not a SLA.
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		previousWritable := initial.CanWrite
		for {
			select {
			case <-ctx.Done():
				c.CloseNow()
				return
			case <-authorityEvents:
			case <-ticker.C:
			}
			checkCtx, done := context.WithTimeout(ctx, 2*time.Second)
			_, writable, e := collaborationAccess(checkCtx, s.DB, id, session, false)
			done()
			if e != nil {
				noticeCtx, stop := context.WithTimeout(ctx, 250*time.Millisecond)
				collaborationPolicyError(noticeCtx, c, e)
				stop()
				forceClose := time.AfterFunc(250*time.Millisecond, func() { _ = c.CloseNow() })
				_ = c.Close(websocket.StatusPolicyViolation, "접근 권한 또는 편집 정책이 변경되었습니다")
				forceClose.Stop()
				cancel()
				return
			}
			if writable != previousWritable {
				previousWritable = writable
				runtime.wake(id)
			}
		}
	}()
	defer func() { cancel(); <-watchDone }()
	send := func(message collaborationMessage) error {
		writeCtx, done := context.WithTimeout(ctx, 10*time.Second)
		defer done()
		// Every payload recipient is revalidated, including quiet/read-only peers.
		_, writable, e := collaborationAccess(writeCtx, s.DB, id, session, false)
		if e != nil {
			collaborationPolicyError(writeCtx, c, e)
			return e
		}
		message.CanWrite = writable
		return wsjson.Write(writeCtx, c, message)
	}
	initial.Type = "hello"
	if err = send(initial); err != nil {
		return
	}
	s.audit(r, "COLLABORATION_CONNECT", id, nil)
	incoming := make(chan collaborationMessage, 1)
	readErrors := make(chan error, 1)
	go func() {
		for {
			var message collaborationMessage
			if e := wsjson.Read(ctx, c, &message); e != nil {
				readErrors <- e
				return
			}
			select {
			case incoming <- message:
			case <-ctx.Done():
				return
			}
		}
	}()
	lastSeq, lastEpoch, lastVersion := initial.Sequence, initial.Epoch, initial.Version
	lastWritable := initial.CanWrite
	window, operations, awarenessOps := time.Now(), 0, 0
	lastTouch := time.Now()
	var previousPresence []byte
	var clientID int64
	for {
		select {
		case <-readErrors:
			return
		case <-ctx.Done():
			return
		case in := <-incoming:
			if time.Since(window) >= time.Second {
				window, operations, awarenessOps = time.Now(), 0, 0
			}
			if in.Type == "awareness" {
				awarenessOps++
				if awarenessOps > 20 {
					continue
				}
				if in.ClientID <= 0 || in.ClientID > 1<<53-1 || (clientID != 0 && in.ClientID != clientID) || in.Clock < 0 || in.Clock > 1<<53-1 || len(in.Awareness) > 8192 {
					_ = c.Close(websocket.StatusPolicyViolation, "유효하지 않은 커서 상태")
					return
				}
				clientID = in.ClientID
				p, _, e := collaborationAccess(ctx, s.DB, id, session, false)
				if e != nil {
					writeCtx, done := context.WithTimeout(ctx, 5*time.Second)
					collaborationPolicyError(writeCtx, c, e)
					done()
					_ = c.Close(websocket.StatusPolicyViolation, "접근 권한 만료")
					return
				}
				state := map[string]any{}
				if json.Unmarshal(in.Awareness, &state) != nil {
					continue
				}
				// Only cursor and server-verified identity are relayed, never arbitrary
				// peer metadata/HTML or a spoofable user name.
				state = map[string]any{"cursor": collaborationCursor(state["cursor"]), "user": collaborationUser(p)}
				_, e = s.DB.Exec(ctx, `UPDATE collaboration_presence SET client_id=$2,clock=$3,state=$4,touched_at=now() WHERE connection_id=$1 AND clock<=$3`, connectionID, clientID, in.Clock, jsonValue(state))
				if e != nil {
					return
				}
				continue
			}
			if in.Type != "update" && in.Type != "seed" {
				continue
			}
			operations++
			if operations > 20 {
				_ = send(collaborationMessage{Type: "error", Code: "rate_limit", Error: "편집 요청이 너무 빠릅니다. 로컬 초안은 보관됩니다"})
				_ = c.Close(websocket.StatusPolicyViolation, "업데이트 속도 제한")
				return
			}
			opCtx, done := context.WithTimeout(ctx, 15*time.Second)
			in.connectionID = connectionID
			state, e := s.collaborationState(opCtx, id, session, &in)
			done()
			if e != nil {
				var fault *collaborationFault
				if !errors.As(e, &fault) {
					slog.Error("collaboration update failed", "document_id", id, "error", e)
					fault = &collaborationFault{"server_error", "편집을 저장하지 못했습니다. 로컬 초안을 보관하세요"}
				}
				if fault.code == "forbidden" {
					_ = c.Close(websocket.StatusPolicyViolation, "접근 권한 만료")
					return
				}
				if send(collaborationMessage{Type: "error", ID: in.ID, Code: fault.code, Error: fault.message}) != nil {
					return
				}
				continue
			}
			if state.Type == "ack" && state.Sequence != lastSeq {
				s.audit(r, "DOCUMENT_UPDATE", id, map[string]any{"source": "collaboration", "sequence": state.Sequence, "version": state.Version})
			}
			lastEpoch, lastSeq, lastVersion = state.Epoch, state.Sequence, state.Version
			if err = send(state); err != nil {
				return
			}
			if state.Type == "reset" {
				_ = c.Close(websocket.StatusPolicyViolation, "문서의 편집 기준이 변경되었습니다")
				return
			}
		case <-changedEvents:
			pollCtx, done := context.WithTimeout(ctx, 10*time.Second)
			// The cheap check returns no document body or CRDT state unless changed.
			_, writable, e := collaborationAccess(pollCtx, s.DB, id, session, false)
			if e != nil {
				collaborationPolicyError(pollCtx, c, e)
				done()
				_ = c.Close(websocket.StatusPolicyViolation, "접근 권한 만료")
				return
			}
			var changed bool
			e = s.DB.QueryRow(pollCtx, `SELECT c.epoch<>$2 OR c.sequence<>$3 OR d.version<>$4 OR c.projected_markdown<>d.markdown FROM document_collaboration c JOIN documents d ON d.id=c.document_id WHERE d.id=$1`, id, lastEpoch, lastSeq, lastVersion).Scan(&changed)
			if e != nil {
				done()
				return
			}
			if changed || writable != lastWritable {
				state, e := s.collaborationStateSince(pollCtx, id, session, nil, lastEpoch, lastSeq)
				if e != nil {
					done()
					return
				}
				if state.Epoch != lastEpoch {
					state.Type = "reset"
				}
				lastEpoch, lastSeq, lastVersion = state.Epoch, state.Sequence, state.Version
				lastWritable = state.CanWrite
				if send(state) != nil {
					done()
					return
				}
				if state.Type == "reset" {
					done()
					_ = c.Close(websocket.StatusPolicyViolation, "문서의 편집 기준이 변경되었습니다")
					return
				}
			}
			if time.Since(lastTouch) > 10*time.Second {
				_, e = s.DB.Exec(pollCtx, "UPDATE collaboration_presence SET touched_at=now() WHERE connection_id=$1", connectionID)
				lastTouch = time.Now()
			}
			if e != nil {
				done()
				return
			}
			presence, e := s.rows(pollCtx, `SELECT jsonb_build_object('client_id',p.client_id,'clock',p.clock,'state',p.state)
			 FROM collaboration_presence p JOIN users u ON u.id=p.user_id
			 WHERE p.document_id=$1 AND p.epoch=$2 AND p.client_id<>0 AND NOT u.disabled AND p.touched_at>now()-interval '30 seconds' ORDER BY p.client_id`, id, lastEpoch)
			done()
			if e != nil {
				return
			}
			raw := jsonValue(presence)
			if !bytes.Equal(raw, previousPresence) {
				if send(collaborationMessage{Type: "presence", Presence: presence}) != nil {
					return
				}
				previousPresence = raw
			}
		}
	}
}

// Reject malformed relative positions before they reach another user's
// ProseMirror cursor plugin. Returning nil means "not currently selecting".
func collaborationCursor(raw any) any {
	object, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	validInt := func(raw any) bool { n, ok := raw.(float64); return ok && n >= 0 && n <= 1<<53-1 && math.Trunc(n) == n }
	position := func(raw any) map[string]any {
		p, ok := raw.(map[string]any)
		if !ok {
			return nil
		}
		out := map[string]any{}
		if assoc, exists := p["assoc"]; exists {
			n, ok := assoc.(float64)
			if !ok || math.Trunc(n) != n || n < -1 || n > 1 {
				return nil
			}
			out["assoc"] = n
		}
		for _, key := range []string{"item", "type"} {
			if p[key] != nil {
				v, ok := p[key].(map[string]any)
				if !ok || !validInt(v["client"]) || !validInt(v["clock"]) {
					return nil
				}
				out[key] = map[string]any{"client": v["client"], "clock": v["clock"]}
			}
		}
		if name, exists := p["tname"]; exists && name != nil {
			v, ok := name.(string)
			if !ok || len(v) > 128 {
				return nil
			}
			out["tname"] = v
		}
		if out["type"] == nil && out["tname"] == nil {
			return nil
		}
		return out
	}
	anchor, head := position(object["anchor"]), position(object["head"])
	if anchor == nil || head == nil {
		return nil
	}
	return map[string]any{"anchor": anchor, "head": head}
}
