package server

import (
	"bytes"
	"context"
	"crypto/subtle"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/netip"
	"os"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

//go:embed public_share.sql
var publicShareSchema string

type publicShare struct {
	ID                      string     `json:"id"`
	DocumentID              string     `json:"document_id"`
	OwnerID                 string     `json:"owner_id"`
	ExpiresAt               time.Time  `json:"expires_at"`
	IPAllowlist             []string   `json:"ip_allowlist"`
	AllowDownload           bool       `json:"allow_download"`
	AllowCopy               bool       `json:"allow_copy"`
	Revision                int64      `json:"revision"`
	CreatedAt               time.Time  `json:"created_at"`
	RevokedAt               *time.Time `json:"revoked_at"`
	PasswordRequired        bool       `json:"password_required"`
	tokenHash, passwordHash string
}

func (s *Server) migratePublicShares(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, publicShareSchema)
	return e
}
func (s *Server) registerPublicShares() {
	s.handle("GET /api/v1/documents/{id}/public-shares", s.listPublicShares)
	s.handle("POST /api/v1/documents/{id}/public-shares", s.createPublicShare)
	s.handle("PUT /api/v1/documents/{id}/public-shares/{share}", s.updatePublicShare)
	s.handle("DELETE /api/v1/documents/{id}/public-shares/{share}", s.revokePublicShare)
	s.handle("POST /api/v1/documents/{id}/public-shares/{share}/rotate", s.rotatePublicShare)
	s.handle("GET /api/v1/documents/{id}/public-shares/{share}/visits", s.publicShareVisits)
	for pattern, handler := range map[string]http.HandlerFunc{
		"GET /api/v1/public-shares/{share}":                    s.readPublicShare,
		"POST /api/v1/public-shares/{share}/unlock":            s.unlockPublicShare,
		"GET /api/v1/public-shares/{share}/attachments/{file}": s.downloadPublicAttachment,
	} {
		s.apiRoutes = append(s.apiRoutes, pattern)
		s.mux.HandleFunc(pattern, handler)
	}
}
func publicShareHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, nosnippet")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
func (s *Server) loadPublicShare(ctx context.Context, id string) (publicShare, error) {
	var v publicShare
	e := s.DB.QueryRow(ctx, `SELECT id::text,document_id::text,owner_id::text,token_hash,password_hash,expires_at,ip_allowlist,allow_download,allow_copy,revision,created_at,revoked_at FROM public_shares WHERE id=$1`, id).Scan(&v.ID, &v.DocumentID, &v.OwnerID, &v.tokenHash, &v.passwordHash, &v.ExpiresAt, &v.IPAllowlist, &v.AllowDownload, &v.AllowCopy, &v.Revision, &v.CreatedAt, &v.RevokedAt)
	v.PasswordRequired = v.passwordHash != ""
	return v, e
}
func (s *Server) publicShareOwner(w http.ResponseWriter, r *http.Request) bool {
	p := current(r)
	if p == nil || p.Kind != "user" || p.TokenID != "" || p.ScopeRestricted || !s.canDocument(r.Context(), p, r.PathValue("id"), true) {
		apiError(w, 404, "공유를 관리할 문서가 없습니다")
		return false
	}
	var owner string
	if e := s.DB.QueryRow(r.Context(), "SELECT owner_id::text FROM documents WHERE id=$1 AND deleted_at IS NULL", r.PathValue("id")).Scan(&owner); e != nil || owner != p.ID {
		apiError(w, 403, "현재 문서 소유자만 외부 공유를 관리할 수 있습니다")
		return false
	}
	return true
}
func (s *Server) listPublicShares(w http.ResponseWriter, r *http.Request) {
	if !s.publicShareOwner(w, r) {
		return
	}
	rows, e := s.rows(r.Context(), `SELECT to_jsonb(p)-'token_hash'-'password_hash'||jsonb_build_object('password_required',password_hash<>'','source_owner_changed',owner_id<>$2::uuid) FROM public_shares p WHERE document_id=$1 ORDER BY created_at DESC LIMIT 200`, r.PathValue("id"), current(r).ID)
	respond(w, rows, e)
}

type publicShareInput struct {
	ExpiresAt     time.Time `json:"expires_at"`
	Password      string    `json:"password"`
	ClearPassword bool      `json:"clear_password"`
	IPAllowlist   []string  `json:"ip_allowlist"`
	AllowDownload bool      `json:"allow_download"`
	AllowCopy     bool      `json:"allow_copy"`
	Confirm       bool      `json:"confirm_public"`
	Revision      int64     `json:"revision"`
}

func (s *Server) publicShareInput(r *http.Request, existing *publicShare) (publicShareInput, map[string]any, string, error) {
	var in publicShareInput
	if decode(r, &in) != nil || !in.Confirm {
		return in, nil, "", errors.New("외부 공유 위험을 확인하고 명시적으로 동의하세요")
	}
	cfg, _, e := s.protectionSettings(r.Context())
	if e != nil {
		return in, nil, "", e
	}
	if !boolean(cfg, "public_shares_enabled") {
		return in, nil, "", errors.New("관리자가 외부 공유를 허용하지 않았습니다")
	}
	now := time.Now()
	created := now
	if existing != nil {
		created = existing.CreatedAt
	}
	if !in.ExpiresAt.After(now) || in.ExpiresAt.After(created.Add(time.Duration(number(cfg, "public_share_max_days", 30))*24*time.Hour)) {
		return in, nil, "", errors.New("공유 만료 시각이 관리자 허용 기간을 벗어났습니다")
	}
	if in.AllowDownload && !boolean(cfg, "public_share_allow_download") {
		return in, nil, "", errors.New("관리자가 외부 공유 다운로드를 허용하지 않았습니다")
	}
	if len(in.IPAllowlist) > 50 {
		return in, nil, "", errors.New("IP 허용 목록은 50개 이하여야 합니다")
	}
	for _, ip := range in.IPAllowlist {
		_, addressError := netip.ParseAddr(ip)
		_, prefixError := netip.ParsePrefix(ip)
		if addressError != nil && prefixError != nil {
			return in, nil, "", errors.New("공유 IP 또는 CIDR 형식을 확인하세요")
		}
	}
	if in.IPAllowlist == nil {
		in.IPAllowlist = []string{}
	}
	hash := ""
	if existing != nil {
		hash = existing.passwordHash
	}
	if in.ClearPassword {
		hash = ""
	}
	if in.Password != "" {
		if len(in.Password) < 8 || len(in.Password) > 72 {
			return in, nil, "", errors.New("공유 암호는 8~72바이트여야 합니다")
		}
		encoded, e := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
		if e != nil {
			return in, nil, "", e
		}
		hash = string(encoded)
	}
	if boolean(cfg, "public_share_require_password") && hash == "" {
		return in, nil, "", errors.New("관리자 정책상 공개 공유에 암호가 필요합니다")
	}
	return in, cfg, hash, nil
}
func (s *Server) publicShareDocumentAllowed(ctx context.Context, id, owner string, cfg map[string]any) bool {
	var allowed bool
	var classification string
	e := s.DB.QueryRow(ctx, `SELECT NOT u.disabled AND u.kind='user' AND d.owner_id=$2 AND d.deleted_at IS NULL AND madi_document_allowed($2,d.id,false),madi_effective_classification(d.id) FROM documents d JOIN users u ON u.id=$2 WHERE d.id=$1`, id, owner).Scan(&allowed, &classification)
	if e != nil || !allowed || classificationRank(classification) > classificationRank(str(cfg, "public_share_max_classification")) {
		return false
	}
	if boolean(cfg, "enabled") && oneOf(str(cfg, "mode"), "block", "mask") {
		var detected bool
		e = s.DB.QueryRow(ctx, `SELECT (madi_protection_analyze(title||E'\n'||markdown,$2,$3)->>'detected')::boolean OR jsonb_array_length(madi_protection_json(jsonb_build_object('tags',tags,'aliases',aliases,'block_metadata',block_metadata),$2,$3)->'findings')>0 FROM documents WHERE id=$1`, id, jsonValue(cfg["detectors"]), jsonValue(cfg["custom_terms"])).Scan(&detected)
		if e != nil || detected {
			return false
		}
	}
	return true
}
func (s *Server) createPublicShare(w http.ResponseWriter, r *http.Request) {
	if !s.publicShareOwner(w, r) {
		return
	}
	in, cfg, hash, e := s.publicShareInput(r, nil)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	id := r.PathValue("id")
	if !s.publicShareDocumentAllowed(r.Context(), id, current(r).ID, cfg) {
		apiError(w, 403, "현재 문서 분류 또는 상위 공간 권한상 외부 공유할 수 없습니다")
		return
	}
	token, sid := integrationSecret(), newID()
	_, e = s.DB.Exec(r.Context(), `INSERT INTO public_shares(id,document_id,owner_id,token_hash,password_hash,expires_at,ip_allowlist,allow_download,allow_copy) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, sid, id, current(r).ID, integrationHash(token), hash, in.ExpiresAt, jsonValue(in.IPAllowlist), in.AllowDownload, in.AllowCopy)
	if e != nil {
		respond(w, nil, e)
		return
	}
	value, e := s.loadPublicShare(r.Context(), sid)
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.audit(r, "PUBLIC_SHARE_CREATE", sid, map[string]any{"document_id": id})
	jsonResponse(w, 200, map[string]any{"share": value, "url": "/share/" + sid + "#" + token})
}
func (s *Server) managedPublicShare(w http.ResponseWriter, r *http.Request) (publicShare, bool) {
	if !s.publicShareOwner(w, r) {
		return publicShare{}, false
	}
	v, e := s.loadPublicShare(r.Context(), r.PathValue("share"))
	if e != nil || v.DocumentID != r.PathValue("id") {
		apiError(w, 404, "공유 링크를 찾을 수 없습니다")
		return v, false
	}
	return v, true
}
func (s *Server) updatePublicShare(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedPublicShare(w, r)
	if !ok {
		return
	}
	if v.OwnerID != current(r).ID {
		apiError(w, 409, "이전 소유자의 공유는 폐기한 뒤 새로 만들어야 합니다")
		return
	}
	in, cfg, hash, e := s.publicShareInput(r, &v)
	if e != nil {
		apiError(w, 400, e.Error())
		return
	}
	if !s.publicShareDocumentAllowed(r.Context(), v.DocumentID, current(r).ID, cfg) {
		apiError(w, 403, "현재 분류 정책상 공유할 수 없습니다")
		return
	}
	tag, e := s.DB.Exec(r.Context(), `UPDATE public_shares SET password_hash=$2,expires_at=$3,ip_allowlist=$4,allow_download=$5,allow_copy=$6,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$7 AND revoked_at IS NULL`, v.ID, hash, in.ExpiresAt, jsonValue(in.IPAllowlist), in.AllowDownload, in.AllowCopy, in.Revision)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "공유 설정이 바뀌었거나 폐기되었습니다")
		return
	}
	s.audit(r, "PUBLIC_SHARE_UPDATE", v.ID, nil)
	value, e := s.loadPublicShare(r.Context(), v.ID)
	respond(w, value, e)
}
func (s *Server) revokePublicShare(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedPublicShare(w, r)
	if !ok {
		return
	}
	_, e := s.DB.Exec(r.Context(), "UPDATE public_shares SET revoked_at=now(),revision=revision+1,updated_at=now() WHERE id=$1", v.ID)
	if e == nil {
		s.audit(r, "PUBLIC_SHARE_REVOKE", v.ID, nil)
	}
	respond(w, map[string]bool{"ok": true}, e)
}
func (s *Server) rotatePublicShare(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedPublicShare(w, r)
	if !ok {
		return
	}
	if v.OwnerID != current(r).ID {
		apiError(w, 409, "이전 소유자의 공유는 폐기한 뒤 새로 만들어야 합니다")
		return
	}
	var in struct {
		Revision int64 `json:"revision"`
	}
	if decode(r, &in) != nil {
		apiError(w, 400, "현재 공유 버전을 확인하세요")
		return
	}
	token := integrationSecret()
	tag, e := s.DB.Exec(r.Context(), "UPDATE public_shares SET token_hash=$2,revision=revision+1,updated_at=now() WHERE id=$1 AND revision=$3 AND revoked_at IS NULL", v.ID, integrationHash(token), in.Revision)
	if e != nil {
		respond(w, nil, e)
		return
	}
	if tag.RowsAffected() != 1 {
		apiError(w, 409, "공유 설정이 바뀌었거나 폐기되었습니다")
		return
	}
	s.audit(r, "PUBLIC_SHARE_ROTATE", v.ID, nil)
	jsonResponse(w, 200, map[string]any{"url": "/share/" + v.ID + "#" + token, "revision": v.Revision + 1})
}
func (s *Server) publicShareVisits(w http.ResponseWriter, r *http.Request) {
	v, ok := s.managedPublicShare(w, r)
	if !ok {
		return
	}
	rows, e := s.rows(r.Context(), "SELECT to_jsonb(v) FROM public_share_visits v WHERE share_id=$1 ORDER BY created_at DESC LIMIT 200", v.ID)
	respond(w, rows, e)
}
func (s *Server) publicShareRate(ctx context.Context, bucket string, max int, duration time.Duration) bool {
	var count int
	e := s.DB.QueryRow(ctx, `INSERT INTO public_share_rate_limits(bucket,count,expires_at) VALUES($1,1,now()+$3::integer * interval '1 second') ON CONFLICT(bucket) DO UPDATE SET count=CASE WHEN public_share_rate_limits.expires_at<=now() THEN 1 ELSE public_share_rate_limits.count+1 END,expires_at=CASE WHEN public_share_rate_limits.expires_at<=now() THEN excluded.expires_at ELSE public_share_rate_limits.expires_at END WHERE public_share_rate_limits.expires_at<=now() OR public_share_rate_limits.count<$2 RETURNING count`, digest(bucket), max, int(duration.Seconds())).Scan(&count)
	return e == nil
}
func (s *Server) publicShareAccess(w http.ResponseWriter, r *http.Request, requireUnlock bool) (publicShare, map[string]any, bool) {
	publicShareHeaders(w)
	if !s.publicShareRate(r.Context(), "public:"+integrationClientIP(r), 120, time.Minute) {
		apiError(w, 429, "공개 공유 요청이 많습니다. 잠시 뒤 다시 시도하세요")
		return publicShare{}, nil, false
	}
	id, token := r.PathValue("share"), r.Header.Get("X-Madi-Share-Token")
	if !validID(id) || len(token) != 43 {
		apiError(w, 404, "공유 링크가 없거나 사용할 수 없습니다")
		return publicShare{}, nil, false
	}
	v, e := s.loadPublicShare(r.Context(), id)
	cfg, _, policyError := s.protectionSettings(r.Context())
	if e != nil || policyError != nil || subtle.ConstantTimeCompare([]byte(v.tokenHash), []byte(integrationHash(token))) != 1 || v.RevokedAt != nil || !time.Now().Before(v.ExpiresAt) || v.ExpiresAt.After(v.CreatedAt.Add(time.Duration(number(cfg, "public_share_max_days", 30))*24*time.Hour)) || !boolean(cfg, "public_shares_enabled") || !integrationIPAllowed(integrationClientIP(r), v.IPAllowlist) || !s.publicShareDocumentAllowed(r.Context(), v.DocumentID, v.OwnerID, cfg) || (boolean(cfg, "public_share_require_password") && !v.PasswordRequired) {
		apiError(w, 404, "공유 링크가 없거나 사용할 수 없습니다")
		return v, cfg, false
	}
	if v.PasswordRequired && requireUnlock {
		grant := r.Header.Get("X-Madi-Share-Access")
		var valid bool
		if len(grant) == 43 {
			_ = s.DB.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM public_share_access WHERE token_hash=$1 AND share_id=$2 AND revision=$3 AND ip_hash=$4 AND expires_at>now())", integrationHash(grant), v.ID, v.Revision, digest(integrationClientIP(r))).Scan(&valid)
		}
		if !valid {
			jsonResponse(w, 401, map[string]any{"error": "공유 암호를 입력하세요", "code": "share_password_required"})
			return v, cfg, false
		}
	}
	return v, cfg, true
}
func (s *Server) recordPublicVisit(r *http.Request, v publicShare, action, result string) {
	_, _ = s.DB.Exec(r.Context(), "INSERT INTO public_share_visits(id,share_id,action,ip_hash,result) VALUES($1,$2,$3,$4,$5)", newID(), v.ID, action, digest(integrationClientIP(r)), result)
}
func (s *Server) unlockPublicShare(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2048)
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(15 * time.Second))
	defer controller.SetReadDeadline(time.Time{})
	v, _, ok := s.publicShareAccess(w, r, false)
	if !ok {
		return
	}
	if !s.publicShareRate(r.Context(), "unlock:"+v.ID+":"+integrationClientIP(r), 5, 5*time.Minute) {
		apiError(w, 429, "암호 입력 횟수를 초과했습니다. 5분 뒤 다시 시도하세요")
		return
	}
	var in struct {
		Password string `json:"password"`
	}
	if decode(r, &in) != nil || len(in.Password) > 72 || !v.PasswordRequired || bcrypt.CompareHashAndPassword([]byte(v.passwordHash), []byte(in.Password)) != nil {
		s.recordPublicVisit(r, v, "unlock", "denied")
		apiError(w, 401, "공유 암호가 올바르지 않습니다")
		return
	}
	grant := integrationSecret()
	expires := time.Now().Add(time.Hour)
	if v.ExpiresAt.Before(expires) {
		expires = v.ExpiresAt
	}
	_, e := s.DB.Exec(r.Context(), "INSERT INTO public_share_access(token_hash,share_id,revision,ip_hash,expires_at) VALUES($1,$2,$3,$4,$5)", integrationHash(grant), v.ID, v.Revision, digest(integrationClientIP(r)), expires)
	if e != nil {
		respond(w, nil, e)
		return
	}
	s.recordPublicVisit(r, v, "unlock", "allowed")
	jsonResponse(w, 200, map[string]any{"access_token": grant, "expires_at": expires})
}
func (s *Server) readPublicShare(w http.ResponseWriter, r *http.Request) {
	v, cfg, ok := s.publicShareAccess(w, r, true)
	if !ok {
		return
	}
	result, e := s.one(r.Context(), "SELECT jsonb_build_object('title',title,'markdown',markdown,'version',version,'updated_at',updated_at,'classification',madi_effective_classification(id)) FROM documents WHERE id=$1 AND deleted_at IS NULL AND owner_id=$2 AND madi_document_allowed($2,id,false)", v.DocumentID, v.OwnerID)
	if e != nil {
		apiError(w, 404, "공유 문서에 접근할 수 없습니다")
		return
	}
	result["allow_copy"] = v.AllowCopy
	result["allow_download"] = v.AllowDownload && boolean(cfg, "public_share_allow_download")
	result["expires_at"] = v.ExpiresAt
	result["watermark"] = boolean(cfg, "watermark_enabled") && classificationRank(str(result, "classification")) >= classificationRank(str(cfg, "watermark_min_classification"))
	result["viewed_at"] = time.Now().UTC()
	result["attachments"] = []any{}
	if result["allow_download"] == true {
		files, e := s.rows(r.Context(), "SELECT jsonb_build_object('id',id,'name',name,'size',size) FROM attachments WHERE document_id=$1 ORDER BY name", v.DocumentID)
		if e != nil {
			respond(w, nil, e)
			return
		}
		result["attachments"] = files
	}
	if current, policy, ok := s.publicShareAccess(w, r, true); !ok {
		return
	} else if current.Revision != v.Revision || !reflect.DeepEqual(policy, cfg) {
		apiError(w, 409, "공유 설정이 변경되었습니다. 다시 확인하세요")
		return
	}
	var sameVersion bool
	if e := s.DB.QueryRow(r.Context(), "SELECT version=$2 FROM documents WHERE id=$1", v.DocumentID, result["version"]).Scan(&sameVersion); e != nil || !sameVersion {
		apiError(w, 409, "공유 문서가 변경되었습니다. 다시 확인하세요")
		return
	}
	delete(result, "version")
	s.recordPublicVisit(r, v, "read", "allowed")
	jsonResponse(w, 200, result)
}
func (s *Server) downloadPublicAttachment(w http.ResponseWriter, r *http.Request) {
	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Now().Add(90 * time.Second))
	defer controller.SetWriteDeadline(time.Time{})
	v, cfg, ok := s.publicShareAccess(w, r, true)
	if !ok {
		return
	}
	if !v.AllowDownload || !boolean(cfg, "public_share_allow_download") {
		apiError(w, 403, "이 공유에서는 첨부 다운로드를 허용하지 않습니다")
		return
	}
	var name, contentType string
	var object storedObject
	e := s.DB.QueryRow(r.Context(), "SELECT name,content_type,path,coalesce(storage_provider_id::text,''),object_key,checksum_sha256,size FROM attachments WHERE id=$1 AND document_id=$2", r.PathValue("file"), v.DocumentID).Scan(&name, &contentType, &object.Path, &object.ProviderID, &object.Key, &object.Checksum, &object.Size)
	if e != nil {
		apiError(w, 404, "공유 첨부파일을 찾을 수 없습니다")
		return
	}
	f, e := s.materializeObject(r.Context(), object, 50<<20)
	if e != nil {
		apiError(w, 404, "공유 첨부파일을 읽을 수 없습니다")
		return
	}
	defer f.Close()
	defer os.Remove(f.Name())
	if !s.publicAttachmentProtectionAllowed(r.Context(), cfg, name, contentType, f) {
		apiError(w, 403, "현재 정보보호 정책상 공유 첨부파일을 제공할 수 없습니다")
		return
	}
	if current, policy, allowed := s.publicShareAccess(w, r, true); !allowed {
		return
	} else if current.Revision != v.Revision || !reflect.DeepEqual(policy, cfg) {
		apiError(w, 409, "공유 설정이 변경되었습니다. 다시 확인하세요")
		return
	}
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprint(object.Size))
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	s.recordPublicVisit(r, v, "download", "allowed")
	buffer := make([]byte, 64<<10)
	checked := time.Now()
	for {
		if time.Since(checked) > time.Second {
			current, e := s.loadPublicShare(r.Context(), v.ID)
			policy, _, pe := s.protectionSettings(r.Context())
			if e != nil || pe != nil || current.Revision != v.Revision || !reflect.DeepEqual(policy, cfg) || current.RevokedAt != nil || !time.Now().Before(current.ExpiresAt) || !boolean(policy, "public_shares_enabled") || !boolean(policy, "public_share_allow_download") || !s.publicShareDocumentAllowed(r.Context(), v.DocumentID, v.OwnerID, policy) {
				return
			}
			checked = time.Now()
		}
		n, e := f.Read(buffer)
		if n > 0 {
			if _, writeError := w.Write(buffer[:n]); writeError != nil {
				return
			}
		}
		if e == io.EOF {
			return
		}
		if e != nil || r.Context().Err() != nil {
			return
		}
	}
}

// Public delivery must not expose an old, unscanned attachment merely because
// the current document text is clean. Scan the materialized immutable bytes and
// fail closed; never silently create a second masked version of the download.
func (s *Server) publicAttachmentProtectionAllowed(ctx context.Context, cfg map[string]any, name, contentType string, f *os.File) bool {
	if !boolean(cfg, "enabled") {
		return true
	}
	data, e := io.ReadAll(io.LimitReader(f, (4<<20)+1))
	if e != nil {
		return false
	}
	if _, e = f.Seek(0, io.SeekStart); e != nil {
		return false
	}
	kind, _, _ := mime.ParseMediaType(contentType)
	supported := (strings.HasPrefix(kind, "text/") || oneOf(kind, "application/json", "application/xml", "application/yaml", "application/x-yaml", "application/javascript") || strings.HasSuffix(kind, "+json") || strings.HasSuffix(kind, "+xml")) && len(data) <= 4<<20 && utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
	if !supported && str(cfg, "unscannable") == "block" {
		return false
	}
	if !oneOf(str(cfg, "mode"), "block", "mask") {
		return true
	}
	input := name
	if supported {
		input += "\n" + string(data)
	}
	var detected bool
	e = s.DB.QueryRow(ctx, "SELECT (madi_protection_analyze($1,$2,$3)->>'detected')::boolean", input, jsonValue(cfg["detectors"]), jsonValue(cfg["custom_terms"])).Scan(&detected)
	return e == nil && !detected
}
