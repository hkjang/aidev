package server

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

//go:embed schema.sql
var schema embed.FS

type Server struct {
	DB                 *pgxpool.Pool
	EncryptionKey      []byte
	Version            string
	mux                *http.ServeMux
	apiRoutes          []string // Immutable after New; used by the complete OpenAPI catalogue.
	requests           atomic.Uint64
	errors             atomic.Uint64
	responseClasses    [6]atomic.Uint64
	operations         atomic.Pointer[operationsRuntime]
	limiterMu          sync.Mutex
	pluginLimiterMu    sync.Mutex
	pluginRequests     map[string]attempt
	limiterSweep       time.Time
	attempts           map[string]attempt
	jobsMu             sync.Mutex
	jobs               *jobRuntime
	approvalAdapters   map[string]ApprovalAdapter // Registered at startup, immutable while serving.
	collaborationMu    sync.Mutex
	collaboration      *collaborationRuntime
	documentQuerySlots chan struct{}
}
type attempt struct {
	count int
	until time.Time
}
type Principal struct {
	ID, Email, Name, Role, Kind string
	Scopes                      []string
	WorkspaceID, TokenID        string
	PluginID                    string
	ScopeRestricted             bool // A capability-constrained session is not an unrestricted cookie principal.
}
type contextKey string

const principalKey contextKey = "principal"
const requestKey contextKey = "request_id"

func current(r *http.Request) *Principal {
	p, _ := r.Context().Value(principalKey).(*Principal)
	return p
}
func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
func digest(v string) string { b := sha256.Sum256([]byte(v)); return hex.EncodeToString(b[:]) }
func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func apiError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]any{"error": message, "request_id": w.Header().Get("X-Request-ID")})
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 8<<20))
	if err := d.Decode(v); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return errors.New("단일 JSON 객체를 입력하세요")
	}
	return nil
}
func str(m map[string]any, k string) string   { v, _ := m[k].(string); return v }
func boolean(m map[string]any, k string) bool { v, _ := m[k].(bool); return v }
func number(m map[string]any, k string, def int) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return def
}
func jsonValue(v any) []byte { b, _ := json.Marshal(v); return b }
func validID(v string) bool {
	if len(v) != 36 {
		return false
	}
	for i, c := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

func New(ctx context.Context, db *pgxpool.Pool, key []byte, version, admin, password string, assets fs.FS) (*Server, error) {
	if len(key) != 32 {
		return nil, errors.New("ENCRYPTION_KEY는 base64로 인코딩한 32바이트 키여야 합니다")
	}
	s := &Server{DB: db, EncryptionKey: key, Version: version, mux: http.NewServeMux(), attempts: map[string]attempt{}}
	// Serialize startup migrations/bootstrap between replicas on a dedicated connection.
	conn, err := db.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock(726234801)"); err != nil {
		return nil, err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock(726234801)")
	b, _ := schema.ReadFile("schema.sql")
	if _, err = conn.Exec(ctx, string(b)); err != nil {
		return nil, err
	}
	if err = s.migrateIntegrations(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateSpaces(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateDatabaseAdvanced(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateJobs(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateCollaboration(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateCanvas(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateStorage(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledge(ctx); err != nil {
		return nil, err
	}
	if err = s.migratePlugins(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateDiscussion(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateIdentity(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateApproval(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateImports(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateConnectors(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateCaptures(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateSQLSources(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateTasks(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateTaskReadIndexes(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateNotificationDelivery(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateSearchIndex(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeGraph(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateInboundCaptures(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateRunbook(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateInformationProtection(ctx); err != nil {
		return nil, err
	}
	if err = s.migratePublicShares(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateRAGIndex(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateTemplates(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateAIHistory(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeEvidence(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgePackages(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateSearchHistory(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateWorkspaceAgent(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeImpact(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeProposals(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeConflicts(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateDocumentAccessRequests(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeValidity(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeQuestions(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeStructured(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateDocumentSplit(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateOperations(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateGitSync(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateSupport(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateExports(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgeDistribution(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateDatabaseEditing(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateAttachmentExtraction(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateSystemStatus(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateWorksets(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateKnowledgePaths(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateGraphAI(ctx); err != nil {
		return nil, err
	}
	if err = s.migrateWorkspaceAudit(ctx); err != nil {
		return nil, err
	}
	if err = s.installCollaborationNotifications(ctx); err != nil {
		return nil, err
	}
	var count int
	if err = conn.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		if len(password) < 12 {
			return nil, errors.New("BOOTSTRAP_ADMIN_PASSWORD는 12자 이상이어야 합니다")
		}
		if strings.TrimSpace(admin) == "" {
			return nil, errors.New("BOOTSTRAP_ADMIN이 필요합니다")
		}
		hash, e := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if e != nil {
			return nil, e
		}
		uid, wid, did := newID(), newID(), newID()
		tx, e := conn.Begin(ctx)
		if e != nil {
			return nil, e
		}
		defer tx.Rollback(ctx)
		if _, e = tx.Exec(ctx, "INSERT INTO users(id,email,name,password_hash,role) VALUES($1,$2,'관리자',$3,'admin')", uid, strings.ToLower(admin), string(hash)); e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO workspaces(id,name,slug) VALUES($1,'우리의 워크스페이스','workspace')", wid); e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO workspace_members VALUES($1,$2,'owner')", wid, uid); e != nil {
			return nil, e
		}
		md := "# madi에 오신 것을 환영합니다\n\n생각을 기록하고, 지식을 연결하고, 함께 성장하세요.\n\n## 처음 시작하기\n\n- [ ] 새 문서에 팀의 지식을 기록해 보세요\n- [ ] `[[문서 제목]]`으로 지식을 연결해 보세요\n- [ ] 관리자 설정에서 사내 SSO와 AI를 연결하세요\n\n## 우리의 지식, 우리의 서버\n\nmadi는 Markdown 원문을 보존합니다. 워크스페이스를 ZIP으로 내보내어 언제든 데이터를 가져갈 수 있습니다.\n\n> 모든 웹 자산은 서버에 포함되어 폐쇄망에서도 사용할 수 있습니다.\n"
		if _, e = tx.Exec(ctx, "INSERT INTO documents(id,workspace_id,title,markdown,owner_id,status,tags) VALUES($1,$2,'madi 시작 가이드',$3,$4,'published','[\"시작하기\"]')", did, wid, md, uid); e != nil {
			return nil, e
		}
		if _, e = tx.Exec(ctx, "INSERT INTO document_versions(document_id,version,title,markdown,user_id) SELECT id,version,title,markdown,owner_id FROM documents WHERE id=$1", did); e != nil {
			return nil, e
		}
		if e = tx.Commit(ctx); e != nil {
			return nil, e
		}
	}
	s.registerCore()
	s.registerDocuments()
	s.registerApproval()
	s.registerData()
	s.registerAdmin()
	s.registerIntegrations()
	s.registerDatabaseAdvanced()
	s.registerJobs()
	s.registerAutomation()
	s.registerCollaboration()
	s.registerSpaces()
	s.registerEditorBlocks()
	s.registerCanvas()
	s.registerStorage()
	s.registerKnowledge()
	s.registerPlugins()
	s.registerDiscussion()
	s.registerIdentity()
	s.registerImports()
	s.registerCaptures()
	s.registerConnectors()
	s.registerSQLSources()
	s.registerTasks()
	s.registerTaskCalendar()
	s.registerNotificationDelivery()
	s.registerSearchIndex()
	s.registerKnowledgeGraph()
	s.registerRAGSettings()
	s.registerRAGIndex()
	s.registerInboundCaptures()
	s.registerRunbook()
	s.registerInformationProtection()
	s.registerPublicShares()
	s.registerTemplates()
	s.registerAIHistory()
	s.registerKnowledgeEvidence()
	s.registerKnowledgePackages()
	s.registerKnowledgePolicyHistory()
	s.registerKnowledgeImpact()
	s.registerKnowledgeProposals()
	s.registerKnowledgeConflicts()
	s.registerAISelection()
	s.registerDocumentAccessRequests()
	s.registerDocumentAccessPreview()
	s.registerKnowledgeValidity()
	s.registerKnowledgeQuestions()
	s.registerKnowledgeStructured()
	s.registerDocumentQueries()
	s.registerKnowledgePaths()
	s.registerKnowledgeDistribution()
	s.registerAttachmentExtraction()
	s.registerSystemStatus()
	s.registerWorksets()
	s.registerDatabaseEditing()
	s.registerDocumentSplit()
	s.registerSearchAI()
	s.registerSearchHistory()
	s.registerWorkspaceAgent()
	s.registerFeatureOperations()
	s.registerBrandOperations()
	s.registerTelemetryOperations()
	s.registerGitSync()
	s.registerSupport()
	s.registerExports()
	s.registerGraphAI()
	s.registerWorkspaceAudit()
	s.registerEnterprise()
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{"status": "ok", "version": version})
	})
	s.mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if s.DB.Ping(ctx) != nil {
			apiError(w, 503, "데이터베이스 연결 대기 중")
			return
		}
		jsonResponse(w, 200, map[string]string{"status": "ready"})
	})
	s.admin("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# TYPE madi_http_requests_total counter\nmadi_http_requests_total %d\n# TYPE madi_http_errors_total counter\nmadi_http_errors_total %d\n", s.requests.Load(), s.errors.Load())
		fmt.Fprint(w, "# TYPE madi_http_responses_total counter\n")
		for class := 1; class <= 5; class++ {
			fmt.Fprintf(w, "madi_http_responses_total{status_class=\"%dxx\"} %d\n", class, s.responseClasses[class].Load())
		}
	})
	files := http.FileServer(http.FS(assets))
	s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" {
			apiError(w, 404, "경로를 찾을 수 없습니다")
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/")
		if name == "" {
			http.Redirect(w, r, "/app", http.StatusFound)
			return
		}
		if strings.HasPrefix(name, "share/") {
			w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive, nosnippet")
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Referrer-Policy", "no-referrer")
		}
		if _, e := fs.Stat(assets, name); e == nil {
			if name == "sw.js" {
				w.Header().Set("Cache-Control", "no-cache")
				w.Header().Set("Service-Worker-Allowed", "/")
			}
			if name == "plugin-sandbox.html" {
				// Only the fixed trusted shell may be framed. Uploaded plugin code
				// runs inside its network-isolated worker, never in this document.
				w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'unsafe-inline' data:; img-src data: blob:; font-src data:; connect-src 'none'; worker-src blob:; frame-src 'none'; frame-ancestors 'self'; form-action 'none'; base-uri 'none'")
				w.Header().Set("X-Frame-Options", "SAMEORIGIN")
				w.Header().Set("Cache-Control", "no-store")
			}
			if strings.HasPrefix(name, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				assetBase := strings.TrimPrefix(name, "assets/")
				if !strings.Contains(assetBase, "/") && strings.HasPrefix(assetBase, "pdf.worker.min-") && strings.HasSuffix(assetBase, ".mjs") {
					// Trusted bundled PDF worker only: image decoders need WASM,
					// not JavaScript eval or a weaker page/plugin policy.
					w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self' 'wasm-unsafe-eval'; connect-src 'self'; worker-src 'none'; object-src 'none'; base-uri 'none'")
				}
			}
			files.ServeHTTP(w, r)
			return
		}
		if !strings.HasPrefix(name, "app") && !strings.HasPrefix(name, "admin") && !strings.HasPrefix(name, "share/") && name != "login" {
			http.NotFound(w, r)
			return
		}
		b, e := fs.ReadFile(assets, "index.html")
		if e != nil {
			apiError(w, 503, "웹 빌드가 필요합니다")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(b)
	})
	return s, nil
}

type statusResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *statusResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *statusResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	if status >= 100 && status < 200 && status != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(status)
		return
	}
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += int64(n)
	return n, err
}
func (w *statusResponseWriter) FlushError() error {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}
func (w *statusResponseWriter) Flush() { _ = w.FlushError() }

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tracked := &statusResponseWriter{ResponseWriter: w}
	w = tracked
	start := time.Now()
	id := newID()
	w.Header().Set("X-Request-ID", id)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'none'; form-action 'self'")
	if strings.HasPrefix(r.URL.Path, "/api/") {
		w.Header().Set("Cache-Control", "no-store")
	}
	s.requests.Add(1)
	defer func() {
		panicked := false
		if e := recover(); e != nil {
			panicked = true
			// Panic values can contain supplied data; log the type, never the value.
			slog.Error("request panic", "request_id", id, "panic_type", fmt.Sprintf("%T", e))
			if tracked.status == 0 {
				apiError(w, 500, "요청을 처리하지 못했습니다")
			}
		}
		status := tracked.status
		if status == 0 {
			status = http.StatusOK
		}
		if class := status / 100; class >= 1 && class <= 5 {
			s.responseClasses[class].Add(1)
		}
		if status >= 400 || panicked {
			s.errors.Add(1)
		}
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		slog.Info("http", "request_id", id, "method", r.Method, "route", route, "status", status, "response_bytes", tracked.bytes, "duration_ms", time.Since(start).Milliseconds())
		s.recordOperation(id, r.Pattern, r.Method, status, start, panicked)
	}()
	r = r.WithContext(context.WithValue(r.Context(), requestKey, id))
	s.mux.ServeHTTP(w, r)
}
func (s *Server) handle(pattern string, h http.HandlerFunc) {
	s.apiRoutes = append(s.apiRoutes, pattern)
	s.mux.Handle(pattern, s.auth(h))
}
func (s *Server) admin(pattern string, h http.HandlerFunc) {
	s.handle(pattern, func(w http.ResponseWriter, r *http.Request) {
		if current(r).Role != "admin" || current(r).TokenID != "" || current(r).ScopeRestricted {
			apiError(w, 403, "서비스 관리자 권한이 필요합니다")
			return
		}
		h(w, r)
	})
}
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var p *Principal
		if r.Header.Get("Authorization") != "" {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				apiError(w, 401, "Bearer API 키 인증 형식을 확인하세요")
				return
			}
			r.Header.Set("Authorization", "Bearer "+parts[1])
			var err error
			p, err = s.tokenPrincipal(r)
			if err != nil || p == nil {
				status := integrationAuthStatus(err)
				if status == 429 {
					w.Header().Set("Retry-After", "60")
				}
				apiError(w, status, "API 키가 유효하지 않거나 사용이 제한되었습니다")
				return
			}
			if !integrationScopeAllowed(p, r) {
				apiError(w, 403, "API 키의 권한 범위를 벗어난 요청입니다")
				return
			}
		} else {
			cookie, e := r.Cookie("madi_session")
			if e != nil {
				apiError(w, 401, "로그인이 필요합니다")
				return
			}
			p = &Principal{}
			e = s.DB.QueryRow(r.Context(), "SELECT u.id,u.email,u.name,u.role,u.kind FROM sessions t JOIN users u ON u.id=t.user_id WHERE t.token_hash=$1 AND t.expires_at>now() AND NOT u.disabled AND u.kind='user'", digest(cookie.Value)).Scan(&p.ID, &p.Email, &p.Name, &p.Role, &p.Kind)
			if e != nil {
				apiError(w, 401, "세션이 만료되었습니다. 다시 로그인하세요")
				return
			}
			if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
				if r.Header.Get("X-Madi-Request") != "1" {
					apiError(w, 403, "요청 검증에 실패했습니다")
					return
				}
				if !s.sameOrigin(r) {
					apiError(w, 403, "허용되지 않은 요청 출처입니다")
					return
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey, p)))
	})
}
func (s *Server) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, e := url.Parse(origin)
	if e != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return false
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if u.Host == r.Host && u.Scheme == scheme {
		return true
	}
	cfg, e := s.settings(r.Context())
	if e != nil {
		return false
	}
	site, e := url.Parse(str(cfg, "site_url"))
	return e == nil && u.Host == site.Host && u.Scheme == site.Scheme
}
func (s *Server) createSession(w http.ResponseWriter, r *http.Request, userID string) error {
	cfg, e := s.settings(r.Context())
	if e != nil {
		return e
	}
	ttl := time.Duration(number(cfg, "session_hours", 24)) * time.Hour
	token := randomToken()
	_, e = s.DB.Exec(r.Context(), "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", digest(token), userID, time.Now().Add(ttl))
	if e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "madi_session", Value: token, Path: "/", HttpOnly: true, Secure: r.TLS != nil || strings.HasPrefix(str(cfg, "site_url"), "https://"), SameSite: http.SameSiteLaxMode, MaxAge: int(ttl.Seconds())})
	return nil
}
func (s *Server) encrypt(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	b, e := aes.NewCipher(s.EncryptionKey)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return "", e
	}
	nonce := make([]byte, g.NonceSize())
	if _, e = rand.Read(nonce); e != nil {
		return "", e
	}
	return "enc:" + base64.StdEncoding.EncodeToString(g.Seal(nonce, nonce, []byte(v), nil)), nil
}
func (s *Server) decrypt(v string) (string, error) {
	if v == "" {
		return "", nil
	}
	if !strings.HasPrefix(v, "enc:") {
		return "", errors.New("암호화된 설정 형식이 아닙니다")
	}
	raw, e := base64.StdEncoding.DecodeString(strings.TrimPrefix(v, "enc:"))
	if e != nil {
		return "", e
	}
	b, _ := aes.NewCipher(s.EncryptionKey)
	g, _ := cipher.NewGCM(b)
	if len(raw) < g.NonceSize() {
		return "", errors.New("암호화 설정 손상")
	}
	plain, e := g.Open(nil, raw[:g.NonceSize()], raw[g.NonceSize():], nil)
	return string(plain), e
}
func (s *Server) audit(r *http.Request, action, resource string, details any) {
	var uid any
	if p := current(r); p != nil {
		uid = p.ID
	} else if action == "LOGIN" && validID(resource) {
		// Local/OIDC login has not passed through session auth yet, but its resource
		// is the verified user ID. Preserve attribution for successful logins.
		uid = resource
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if details == nil {
		details = map[string]any{}
	}
	_, e := s.DB.Exec(r.Context(), "INSERT INTO audit_logs(id,user_id,action,resource,ip,details) VALUES($1,$2,$3,$4,$5,$6)", newID(), uid, action, resource, ip, jsonValue(details))
	if e != nil {
		slog.Error("audit write failed", "error", e)
	}
}
func (s *Server) canWorkspace(ctx context.Context, p *Principal, id string, write bool) bool {
	if p == nil || !validID(id) || (p.WorkspaceID != "" && p.WorkspaceID != id) {
		return false
	}
	if write && p.Role == "viewer" {
		return false
	}
	var ok bool
	e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2 AND (NOT $3 OR role IN ('owner','admin','editor')))", id, p.ID, write).Scan(&ok)
	return e == nil && ok
}
func (s *Server) canDocument(ctx context.Context, p *Principal, id string, write bool) bool {
	if !validID(id) {
		return false
	}
	if p == nil {
		return false
	}
	var ok bool
	e := s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM documents d WHERE d.id=$1 AND ($4='' OR d.workspace_id::text=$4) AND madi_document_allowed($2,d.id,$3))", id, p.ID, write, p.WorkspaceID).Scan(&ok)
	return e == nil && ok
}

var errQueryResultTooLarge = errors.New("목록 응답이 허용 크기를 초과했습니다. 검색 조건이나 페이지 크기를 줄여주세요")

const maxQueryJSONBytes = 32 << 20

func (s *Server) rows(ctx context.Context, q string, args ...any) ([]map[string]any, error) {
	rows, e := s.DB.Query(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []map[string]any{}
	totalBytes := 0
	for rows.Next() {
		var v []byte
		if e = rows.Scan(&v); e != nil {
			return nil, e
		}
		totalBytes += len(v)
		// Last-resort protection for map-backed read models. Large backup/export
		// paths use streaming pgx rows and their own explicit format budgets.
		// A 4 MiB canonical Markdown import may expand to 24 MiB when
		// JSON escapes control characters. Apply the same aggregate ceiling
		// to a single row so a valid document remains individually readable.
		if totalBytes > maxQueryJSONBytes {
			return nil, errQueryResultTooLarge
		}
		m := map[string]any{}
		if e = json.Unmarshal(v, &m); e != nil {
			return nil, e
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (s *Server) one(ctx context.Context, q string, args ...any) (map[string]any, error) {
	rows, e := s.rows(ctx, q, args...)
	if e != nil {
		return nil, e
	}
	if len(rows) == 0 {
		return nil, pgx.ErrNoRows
	}
	return rows[0], nil
}
func respond(w http.ResponseWriter, v any, e error) {
	if errors.Is(e, errQueryResultTooLarge) {
		apiError(w, http.StatusRequestEntityTooLarge, errQueryResultTooLarge.Error())
	} else if errors.Is(e, context.Canceled) {
		// A disconnected/cancelled caller is not a database failure. Do not
		// inflate the service error log or expose a driver cancellation string.
		apiError(w, http.StatusRequestTimeout, "요청이 취소되었습니다")
	} else if errors.Is(e, context.DeadlineExceeded) {
		w.Header().Set("Retry-After", "1")
		apiError(w, http.StatusServiceUnavailable, "처리 시간이 초과되었습니다. 검색 범위를 줄이거나 잠시 후 다시 시도하세요")
	} else if errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 404, "항목을 찾을 수 없습니다")
	} else if e != nil {
		// Driver errors can include private values or SQL arguments.
		var coded interface{ SQLState() string }
		code := ""
		if errors.As(e, &coded) {
			code = coded.SQLState()
		}
		slog.Error("database operation", "request_id", w.Header().Get("X-Request-ID"), "error_type", fmt.Sprintf("%T", e), "sql_state", code)
		apiError(w, 500, "데이터를 처리하지 못했습니다")
	} else {
		jsonResponse(w, 200, v)
	}
}
