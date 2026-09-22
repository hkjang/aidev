package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

//go:embed knowledge_distribution.sql
var knowledgeDistributionSchema string

func (s *Server) migrateKnowledgeDistribution(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, knowledgeDistributionSchema)
	return e
}
func (s *Server) registerKnowledgeDistribution() {
	s.RegisterApprovalAdapter("knowledge_distribution", s.distributionApprovalAdapter())
	s.admin("GET /api/v1/admin/knowledge-distribution", s.getDistributionSettings)
	s.admin("PUT /api/v1/admin/knowledge-distribution", s.putDistributionSettings)
	s.admin("POST /api/v1/admin/knowledge-distribution/keys", s.createDistributionKey)
	s.admin("POST /api/v1/admin/knowledge-distribution/keys/{id}/revoke", s.revokeDistributionKey)
	s.admin("POST /api/v1/admin/knowledge-distribution/keys/{id}/trust", s.revalidateDistributionTrust)
	s.handle("GET /api/v1/knowledge/distribution/context", s.getDistributionContext)
	s.handle("POST /api/v1/migrations/sessions/{id}/signed-source", s.bindSignedMigration)
	s.handle("GET /api/v1/migrations/sessions/{id}/signed-source", s.getSignedMigration)
	s.handle("GET /api/v1/knowledge/distribution/exports", s.listDistributionExports)
	s.handle("POST /api/v1/knowledge/distribution/exports", s.createDistributionExport)
	s.handle("GET /api/v1/knowledge/distribution/exports/{id}/download", s.downloadDistributionExport)
	s.handle("POST /api/v1/knowledge/distribution/exports/{id}/revoke", s.revokeDistributionExport)
	s.handle("GET /api/v1/knowledge/distribution/exports/{id}/review", s.getDistributionReview)
	s.handle("POST /api/v1/knowledge/distribution/exports/{id}/approval", s.submitDistributionReview)
	s.handle("POST /api/v1/knowledge/distribution/exports/{id}/sign", s.signReviewedDistribution)
	s.RegisterJobHandler("knowledge.distribution", s.executeDistributionExport)
}

type distributionPolicy struct {
	Revision     int    `json:"revision"`
	Enabled      bool   `json:"enabled"`
	InstanceID   string `json:"instance_id"`
	MaxValidDays int    `json:"max_valid_days"`
}

func distributionPolicyTx(ctx context.Context, tx pgx.Tx, write bool) (distributionPolicy, error) {
	var p distributionPolicy
	lock := " FOR SHARE"
	if write {
		lock = " FOR UPDATE"
	}
	e := tx.QueryRow(ctx, `SELECT revision,enabled,instance_id::text,max_valid_days FROM knowledge_distribution_policy WHERE id=1`+lock).Scan(&p.Revision, &p.Enabled, &p.InstanceID, &p.MaxValidDays)
	return p, e
}
func distributionAdminTx(r *http.Request, tx pgx.Tx) error {
	p := current(r)
	denied := errors.New("현재 관리자 로그인 권한을 다시 확인하세요")
	if p == nil || p.Role != "admin" || p.TokenID != "" || p.ScopeRestricted || p.PluginID != "" {
		return denied
	}
	c, e := r.Cookie("madi_session")
	if e != nil {
		return denied
	}
	var role string
	var expires time.Time
	if tx.QueryRow(r.Context(), `SELECT role FROM users WHERE id=$1 AND NOT disabled FOR SHARE`, p.ID).Scan(&role) != nil || role != "admin" {
		return denied
	}
	if tx.QueryRow(r.Context(), `SELECT expires_at FROM sessions WHERE user_id=$1 AND token_hash=$2 AND expires_at>clock_timestamp() FOR SHARE`, p.ID, digest(c.Value)).Scan(&expires) != nil || !expires.After(time.Now()) {
		return denied
	}
	return nil
}

const distributionNotice = "서명은 등록된 키의 발행과 파일 무결성을 확인하며 내용의 사실성·수신망의 게시 승인을 인증하지 않습니다. 반입은 비공개 초안이며 이미 내려받은 사본을 회수할 수 없습니다."

func (s *Server) getDistributionSettings(w http.ResponseWriter, r *http.Request) {
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	p, e := distributionPolicyTx(r.Context(), tx, false)
	if e == nil {
		e = distributionAdminTx(r, tx)
	}
	if e != nil {
		apiError(w, 403, "현재 관리자 설정 권한을 확인하세요")
		return
	}
	rows, e := tx.Query(r.Context(), `SELECT jsonb_build_object('id',id,'kind',kind,'label',label,'source_instance',source_instance,'source_key_id',source_key_id,'public_key',public_key,'fingerprint',fingerprint,'revision',revision,'created_at',created_at,'revoked_at',revoked_at) FROM knowledge_distribution_keys ORDER BY created_at DESC LIMIT 100`)
	if e != nil {
		respond(w, nil, e)
		return
	}
	keys := []any{}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			break
		}
		keys = append(keys, json.RawMessage(raw))
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	if e != nil {
		respond(w, nil, e)
		return
	}
	history := []any{}
	rows, e = tx.Query(r.Context(), `SELECT to_jsonb(h) FROM knowledge_distribution_policy_history h ORDER BY revision DESC LIMIT 100`)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			break
		}
		history = append(history, json.RawMessage(raw))
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	respond(w, map[string]any{"policy": p, "keys": keys, "history": history, "notice": distributionNotice, "max_files": distributionMaxFiles, "max_bytes": distributionMaxBytes}, e)
}
func (s *Server) putDistributionSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision     int   `json:"revision"`
		Enabled      *bool `json:"enabled"`
		MaxValidDays int   `json:"max_valid_days"`
		Consent      bool  `json:"consent"`
	}
	if decode(r, &in) != nil || in.Revision < 1 || in.Revision > 2147483646 || in.Enabled == nil || in.MaxValidDays < 1 || in.MaxValidDays > 365 || !in.Consent {
		apiError(w, 400, "현재 설정 revision·활성화·유효기간과 변경 확인이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	p, e := distributionPolicyTx(r.Context(), tx, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if p.Revision != in.Revision {
		apiError(w, 409, "배포 정책이 변경되었습니다")
		return
	}
	if e = distributionAdminTx(r, tx); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	_, e = tx.Exec(r.Context(), `UPDATE knowledge_distribution_policy SET enabled=$1,max_valid_days=$2,revision=revision+1,updated_at=now() WHERE id=1`, *in.Enabled, in.MaxValidDays)
	if e == nil {
		_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_distribution_policy_history(revision,enabled,max_valid_days,actor_id) VALUES($1,$2,$3,$4)`, in.Revision+1, *in.Enabled, in.MaxValidDays, current(r).ID)
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e == nil {
		s.audit(r, "DISTRIBUTION_POLICY_UPDATE", "1", map[string]any{"revision": in.Revision + 1, "enabled": *in.Enabled, "max_valid_days": in.MaxValidDays})
	}
	respond(w, map[string]any{"revision": in.Revision + 1}, e)
}
func (s *Server) createDistributionKey(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Kind           string `json:"kind"`
		Label          string `json:"label"`
		SourceInstance string `json:"source_instance"`
		SourceKeyID    string `json:"source_key_id"`
		PublicKey      string `json:"public_key"`
		Fingerprint    string `json:"fingerprint"`
		Consent        bool   `json:"consent"`
	}
	if decode(r, &in) != nil || !oneOf(in.Kind, "signing", "trusted") || len(strings.TrimSpace(in.Label)) == 0 || len(in.Label) > 200 || !in.Consent {
		apiError(w, 400, "키 종류·이름과 키 등록 확인이 필요합니다")
		return
	}
	id := newID()
	var sealed, public, fingerprint string
	if in.Kind == "trusted" {
		key, e := base64.StdEncoding.Strict().DecodeString(in.PublicKey)
		if e != nil || len(key) != ed25519.PublicKeySize || !validID(in.SourceInstance) || !validID(in.SourceKeyID) || in.Fingerprint != digest(string(key)) {
			apiError(w, 400, "반출망에서 별도로 확인한 instance·키 ID·32바이트 공개키·SHA256 지문을 입력하세요")
			return
		}
		public = in.PublicKey
		fingerprint = in.Fingerprint
	} else {
		pub, private, e := ed25519.GenerateKey(rand.Reader)
		if e != nil {
			respond(w, nil, e)
			return
		}
		sealed, e = s.encrypt(base64.StdEncoding.EncodeToString(private))
		if e != nil {
			respond(w, nil, e)
			return
		}
		public = base64.StdEncoding.EncodeToString(pub)
		fingerprint = digest(string(pub))
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	p, e := distributionPolicyTx(r.Context(), tx, true)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if e = distributionAdminTx(r, tx); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	var count int
	if e = tx.QueryRow(r.Context(), `SELECT count(*) FROM knowledge_distribution_keys`).Scan(&count); e != nil {
		respond(w, nil, e)
		return
	}
	if count >= 100 {
		apiError(w, 409, "키 이력을 포함하여 최대100개입니다. 관리자 운영 절차로 키 이력을 보존·이관하세요")
		return
	}
	if in.Kind == "signing" {
		in.SourceInstance = p.InstanceID
		in.SourceKeyID = id
	}
	_, e = tx.Exec(r.Context(), `INSERT INTO knowledge_distribution_keys(id,kind,label,source_instance,source_key_id,public_key,fingerprint,private_ciphertext,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`, id, in.Kind, in.Label, in.SourceInstance, in.SourceKeyID, public, fingerprint, sealed, current(r).ID)
	if e == nil {
		var actual string
		e = tx.QueryRow(r.Context(), `SELECT id::text FROM knowledge_distribution_keys WHERE kind=$1 AND source_instance=$2 AND source_key_id=$3`, in.Kind, in.SourceInstance, in.SourceKeyID).Scan(&actual)
		if e == nil && actual != id {
			apiError(w, 409, "이미 등록된 키 ID입니다. 키 자료를 덮어쓰지 않습니다")
			return
		}
	}
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "DISTRIBUTION_KEY_CREATE", id, map[string]any{"kind": in.Kind, "fingerprint": fingerprint})
	jsonResponse(w, 201, map[string]any{"id": id, "kind": in.Kind, "source_instance": in.SourceInstance, "source_key_id": in.SourceKeyID, "public_key": public, "fingerprint": fingerprint, "revision": 1})
}
func (s *Server) revokeDistributionKey(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision     int    `json:"revision"`
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || !validID(r.PathValue("id")) || in.Revision < 1 || in.Revision > 2147483646 || in.Confirmation != "REVOKE" {
		apiError(w, 400, "키 revision과 REVOKE 확인이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = distributionPolicyTx(r.Context(), tx, true); e == nil {
		e = distributionAdminTx(r, tx)
	}
	if e != nil {
		apiError(w, 403, "현재 관리자 권한을 확인하세요")
		return
	}
	tag, e := tx.Exec(r.Context(), `UPDATE knowledge_distribution_keys SET revoked_at=clock_timestamp(),revision=revision+1 WHERE id=$1 AND revision=$2 AND revoked_at IS NULL`, r.PathValue("id"), in.Revision)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "키가 이미 변경됐거나 폐기됐습니다")
		return
	}
	e = tx.Commit(r.Context())
	if e == nil {
		s.audit(r, "DISTRIBUTION_KEY_REVOKE", r.PathValue("id"), map[string]any{"revision": in.Revision + 1})
	}
	respond(w, map[string]any{"revoked": true}, e)
}
func (s *Server) revalidateDistributionTrust(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Revision     int    `json:"revision"`
		Fingerprint  string `json:"fingerprint"`
		Confirmation string `json:"confirmation"`
	}
	if decode(r, &in) != nil || !validID(r.PathValue("id")) || in.Revision < 1 || in.Revision > 2147483646 || !migrationHexHash(in.Fingerprint) || in.Confirmation != "TRUST" {
		apiError(w, 400, "현재 키 revision·별도로 재확인한 SHA256 지문·TRUST 확인이 필요합니다")
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = distributionPolicyTx(r.Context(), tx, true); e == nil {
		e = distributionAdminTx(r, tx)
	}
	if e != nil {
		apiError(w, 403, "현재 관리자 권한을 확인하세요")
		return
	}
	tag, e := tx.Exec(r.Context(), `UPDATE knowledge_distribution_keys SET revoked_at=NULL,revision=revision+1 WHERE id=$1 AND kind='trusted' AND revision=$2 AND fingerprint=$3 AND revoked_at IS NOT NULL`, r.PathValue("id"), in.Revision, in.Fingerprint)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "키 상태·지문이 다르거나 재확인할 외부 신뢰 키가 아닙니다")
		return
	}
	e = tx.Commit(r.Context())
	if e == nil {
		s.audit(r, "DISTRIBUTION_TRUST_REVALIDATE", r.PathValue("id"), map[string]any{"revision": in.Revision + 1, "fingerprint": in.Fingerprint})
	}
	respond(w, map[string]any{"trusted": true, "revision": in.Revision + 1}, e)
}
func (s *Server) getDistributionContext(w http.ResponseWriter, r *http.Request) {
	wid := r.URL.Query().Get("workspace_id")
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	p, e := distributionPolicyTx(r.Context(), tx, false)
	if e == nil {
		e = s.knowledgeActorTx(r, tx, wid, "document:read")
	}
	if e != nil {
		apiError(w, 403, "현재 워크스페이스 조회 권한을 확인하세요")
		return
	}
	keys := []any{}
	rows, e := tx.Query(r.Context(), `SELECT jsonb_build_object('id',id,'label',label,'fingerprint',fingerprint) FROM knowledge_distribution_keys WHERE kind='signing' AND revoked_at IS NULL ORDER BY created_at DESC LIMIT 100`)
	if e != nil {
		respond(w, nil, e)
		return
	}
	for rows.Next() {
		var raw []byte
		if e = rows.Scan(&raw); e != nil {
			break
		}
		keys = append(keys, json.RawMessage(raw))
	}
	if e == nil {
		e = rows.Err()
	}
	rows.Close()
	respond(w, map[string]any{"policy": p, "signing_keys": keys, "notice": distributionNotice}, e)
}
