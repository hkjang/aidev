# AgentHub 프로필 (2026-09-20)
- 목적: 사내 AI 에이전트 런타임(Pod)을 띄우고 정책·DLP·승인·감사로 통제하는 컨트롤 플레인 + 관리자/사용자 콘솔.
- 스택: Go(chi, pgx, go-oidc) · PostgreSQL · React/TypeScript(Vite, web/) · Kubernetes 배포(deploy/kubernetes, kustomize) · Playwright/node 스크립트(web/scripts).
- 구조: `cmd/`(control-plane, worker, runtime-proxy=Pod 게이트웨이 — base 이미지 소스), `internal/api`(라우터·핸들러·live 테스트), `internal/store`(pgx, 마이그레이션), `internal/dlp`(스캐너, base 이미지 소스), `internal/mail`, `internal/tracking`, `web/`(콘솔), `docs/`(ADMIN_GUIDE.md·USER_GUIDE.md 정본 + PDF), `scripts/`(release-catalog-images.sh), 런타임 이미지 Dockerfile.* 과 *_VERSION 파일.
- 빌드·테스트: `go build ./... && go vet ./...`; `go test -race ./cmd/... ./internal/...`(CI 와 동일, DB 없이도 통과); live 테스트는 `AGENTHUB_TEST_DSN`·`AGENTHUB_ENCRYPTION_KEY` 를 주면 실행되며 `-p 1` 권장; web 은 `npm ci && npm run lint && npm run build`; `scripts/release-catalog-images.sh check-versions|validate`; `kubectl kustomize deploy/kubernetes`.
- 관례: 커밋 제목은 한국어 문장 또는 영어 소문자 conventional(`fix:`/`feat:`), 트레일러 없음. 설정은 system_settings 의 키 하나에 JSON 블롭(예: `mail`, `mcp.oauth`, `authentication`), 비밀은 행의 secret 슬롯. 문서는 docs/ADMIN_GUIDE.md·USER_GUIDE.md 가 정본이고 PDF 는 md2pdf.mjs 로 재생성. cmd/runtime-proxy·internal/dlp 를 건드리면 BASE_VERSION 상향(5곳), 릴리즈 VERSION 은 건드리지 않음.
- 위험 구역: `internal/api/auth.go`(OIDC·세션), `mcpoauth.go`(토큰 검사·거절 메시지를 live 테스트가 고정), `internal/store` 마이그레이션, 정책 지점의 actor(task.OwnerID; agent.OwnerID 는 SCMTokenFor 만), 감사 details(원문 금지, 식별자만).
- 자주 깨지는 곳: Pod↔컨트롤 플레인 문서 어긋남(엄격 디코더), 같은 값을 읽는 경로가 둘인 곳(분류기/렌더러), 문서의 API 메서드(테스트가 라우터와 대조), 캡처 스크립트의 설정 덮어쓰기(백업·복원 필수).
- 검증 함정: live 테스트는 DSN 없으면 건너뛰어 CI 에서 돌지 않음; 병렬 실행 시 store 의 pinned-server·never-started 테스트가 동시 마이그레이션에 흔들림(`-p 1`); 클러스터·Keycloak·SMTP 는 이 환경에 없어 대역으로 대신; `GET /admin/settings/{key}` 는 없음(405) — 전체 GET 에서 꺼낼 것.
