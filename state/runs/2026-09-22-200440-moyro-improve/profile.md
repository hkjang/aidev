# moyro 프로필 (2026-09-22)
- 목적: Mattermost v4 API 호환 채팅에 작업·승인·자동화·지식·MCP 를 얹은 자체 호스팅 협업 앱.
- 스택: Go(go.mod), chi, pgx v5, PostgreSQL / React 18·TypeScript·Vite·Redux·MUI / Vitest·Playwright / Docker.
- 구조:
  - server/cmd/moyro: 서버 진입점. server/cmd/fakeoidc: 테스트 IdP(prompt=none 거절, access_token 은 임의 문자열).
  - server/internal/httpapi: `/api/v4` 호환(`compat_*`)과 `/api/moyro/v1` 네이티브(`native_*`), 배선은 router.go.
    호환 핸들러는 handlers.go + compat_wave_handlers{,_early,_late,_final}.go 네 덩어리에 흩어져 있다.
  - server/internal 도메인 서비스 약 50 개(channels, posts, sidebar, preferences, workitems, automations,
    reminders, userstatus, tracking, oidcauth, mcpserver, ws, rbac, audit …).
  - **mail 패키지는 아직 main 에 없다**(internal/email 과 혼동 금지). 메일 기능은 미머지 브랜치에만 있다.
  - server/internal/store/migrations: 번호형 checksummed 마이그레이션. baseline 수정 금지.
  - server/internal/pluginhost → rpcbridge(서버 플러그인), webapp/src/plugins/runtime.ts → registry.ts(웹 확장).
  - webapp/src/{api,features,components}, webapp/e2e(브라우저 계약), docs(가이드·OpenAPI·사이트), scripts(검증·배포·캡처).
- 빌드·테스트: server/ 에서 `go vet ./...`, `go build ./...`, `go test -count=1 ./internal/<pkg>`.
  실제 DB 테스트는 `MOYRO_TEST_POSTGRES_DSN` 이 있어야 돌고 없으면 **통째로 skip** 되므로 ok 만 보고 통과라고
  하면 안 된다. 전체는 `go test -race -p 1 ./...`(수 분, -p 1 유지). 웹은 webapp/ 에서 `npm ci`, `npm test`,
  `npm run typecheck`, `npm run build`. 루트에서 `bash scripts/check-source-sizes.sh`,
  `node scripts/verify-pages.mjs`. 브라우저 e2e·이미지 빌드는 비용이 크다.
- 이번 확인(base main@960d3a1, v0.2.34 이후): 작업 트리 clean, `git log -30` 확인,
  `go test -count=1 ./internal/preferences ./internal/httpapi` 통과(preferences 는 테스트 파일 자체가 없고
  httpapi 는 DSN 미설정으로 DB 테스트 skip). 전체 go test·vet·웹·실제 DB 는 이번에 미검증.
- 관례: 영어 `feat:/fix:/test:/docs:/release:` 커밋. `/api/v4` 경로·JSON 모양·기존 오류 id 문자열 보존이 최우선.
  오류 분기는 `errors.Is` + 센티널/`pgx.ErrNoRows` 로 하고 404 본문 id 는 유지한 채 상태 코드만 가른다
  (선례: sidebar 의 `writeSidebarError`, native_ai/native_keys/native_operations 의 `errors.Is(pgx.ErrNoRows)`).
  설정은 관리자 저장 JSON + bootstrap 환경변수. 문서 정본은 docs/USER_GUIDE·ADMIN_GUIDE(+PDF·사이트). UTF-8 유지.
- 위험 구역: auth/session/principal/oidcauth/native_mcp_oauth, store/migrations, .github/workflows.
  WebSocket 단일 소유·플러그인 훅 순서 결정성 유지. webapp/dist·node_modules·생성 .js 편집 금지.
- 자주 깨지는 곳: DB 오류를 400/403/404 로 위장하는 호환 핸들러(계속 나온다), 응답과 WS 이벤트의 값 불일치,
  한 리소스 안에서 읽기 경로마다 오류 매핑이 다른 것.
- 미머지 브랜치(같은 일을 다시 하지 말 것): `origin/auto/2026-09-21-0304`(bddb463, 사이드바 재정렬 응답/이벤트
  정규화 — main 미반영이지만 **재구현 금지**), `origin/auto/2026-09-16-0812`(메일 알림),
  로컬 전용 `auto/2026-09-07-0240`(preferences 값 상한 + upsert/delete 500 분리 + ci.yml — origin 에 푸시된 적이
  없어 PR 도 반려도 없다; 상한과 CI 부분은 건드리지 말 것), `auto/2026-09-08-2109`(리마인더 상한),
  `auto/2026-09-09-0201`(프레즌스).
- 검증 함정: CI 는 PostgreSQL 15/16 으로 전체 DB 테스트를 돌린다(PG16 은 실제 플러그인 archive 변수도 제공).
  `newOperationsTestDB` 는 임의 스키마로 격리하므로 장애 테스트의 `DROP TABLE` 은 그 스키마 안에서만 한다.
  로컬 postgres 컨테이너 포트는 55432 가 점유된 적이 있어 55433 을 쓴다.
  webapp/tsconfig.json 에 `types` 가 없어 @types/node 가 끼면 useDraft.test.tsx 가 TS2345 로 깨진다 —
  Go 전용 변경의 탓으로 돌리지 말 것. Windows 에서는 `C:\Program Files\nodejs\npm.cmd` 절대 경로 사용.
- 스킬: 이번 회차에는 `pmo:estimating-and-contingency`, `technology:implementation-planning`,
  `technology:solution-exploration` 이 Skill 도구로 정상 로드됐다(이전 회차들의 "미발견" 기록은 더 이상 유효하지 않다).
