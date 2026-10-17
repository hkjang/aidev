# BITEX 프로필 (2026-10-17)

- 목적: LLM 에이전트가 웹 자산을 정찰·점검해 취약점 발견 사항을 쌓고 내보내는, 폐쇄망(오프라인) 설치형 보안 점검 플랫폼. 한국어 UI 단일 로캘. 현재 v3.0.0.
- 스택: Go 1.26.9 (모듈명 `bitex`, 단일 바이너리 `cmd/bitex`) + PostgreSQL + Next.js/TypeScript 프런트(`web/`, React, Tailwind, `npm`). 배포는 Docker 이미지 tar.gz 한 개(비루트 10001:10001, 읽기 전용 루트).

## 구조
- `cmd/bitex` — 진입점. `server/` — HTTP·REST·MCP 서버와 작업 오케스트레이션(가장 큰 패키지, 테스트 ~50개 파일).
- `db/` — PostgreSQL 접근 계층 전체(작업·자산·발견 사항·설정·알림). 테스트 ~55개 파일.
- `agent/` — LLM 워커·프롬프트·도구 카탈로그·블랙보드. `llmpool/`(모델 풀·헬스), `llmrec/`(LLM 호출 기록), `intercept/`(모델 호출 승인·차단), `guard/`(실행 정책).
- `traffic/` — 프록시·트래픽 저장·아카이브. `evidence/` — 요청/응답 본문 증거 저장소.
- `report/` — 발견 사항 Markdown/CSV 내보내기. **테스트 0개.** `enrich/` — DNS·HTTP 자산 보강. **테스트 0개.**
- `identity/` — Keycloak OIDC·PKCE·토큰 검증·개인 키. `vault/` — 설정 자격 증명 암호화. `notify/` — 6개 채널 알림(마스킹·SSRF 가드 다수).
- `mcphttp/`, `sidequestion/`, `config/` — MCP 클라이언트, 후속 질문, 런타임 설정.
- `docs/` — 정적 소개·가이드 HTML + 한국어 검증 기록(`docs/개발-검증-계획.md` 가 사실상 로드맵·검증 대장).

## 빌드·테스트
- `go build ./...` / `go vet ./<pkg>`.
- PostgreSQL 불필요: `go test ./cmd/bitex ./config ./vault ./evidence ./guard ./intercept ./llmpool ./llmrec ./mcphttp ./notify ./sidequestion ./traffic` (릴리스 워크플로가 쓰는 정확한 목록). `./report`, `./enrich` 는 목록에 없다.
- PostgreSQL 필요: `agent`, `db`, `server`, `identity` → `BITEX_TEST_PG_DSN` 설정 후 `go run ./scripts/test-integration.go` (기본 패키지 `agent,db,server,identity`, `-packages` 로 변경). **오래 걸린다** (전체 451~452 통과 규모).
- 프런트: `cd web && npm ci --ignore-scripts && ./node_modules/.bin/tsc --noEmit && node --test src/lib/*.test.mjs`.
- 최초 `go test` 는 모듈 다운로드가 일어난다(네트워크 필요).

## 관례
- 커밋 메시지 한국어, `type(scope): 설명` 형식 (`fix(notify): …`, `feat(intercept): …`, `docs: …`). 과거 커밋은 중국어였고 v3.0.0 에서 한국어로 통일됐다.
- 코드 주석은 한국어이나 중국어 기계 번역 잔재로 깨진 문장이 많다(`report/findings.go` 등). 새 주석은 자연스러운 한국어로 쓸 것.
- 런타임 설정은 환경변수 4개: `POSTGRES_DSN`, `BOOTSTRAP_ADMIN`, `BOOTSTRAP_ADMIN_PASSWORD`, `ENCRYPTION_KEY` (`.env.example`, `config.example.json` 참조).
- 마이그레이션은 별도 디렉터리 없이 `db/` 패키지 코드 안에서 수행된다(스키마 생성·이전 로직이 Go 코드에 들어있음).
- 검증 결과는 `docs/개발-검증-계획.md` 에 실행 결과 기준으로 덧붙인다.

## 위험 구역
- `identity/`, `server/auth*.go`, `server/security_test.go` — OIDC·세션·권한. 과거 `fix/auth-init-fail-open` 회귀가 있었다.
- `vault/` — 자격 증명 암호화. 키 처리 변경은 기존 저장 데이터 복호화를 깬다.
- `db/` — 스키마 생성·이전 로직이 코드 안에 있어 변경이 곧 마이그레이션이다.
- `notify/` — 마스킹·SSRF·서명 가드가 12건 감사 수정으로 쌓인 곳. 우회 회귀가 반복됐다.
- `.github/workflows/release.yml` — 태그 푸시 시 테스트·이미지 빌드·릴리스 게시를 모두 한다. 깨지면 릴리스가 멈춘다.

## 자주 깨지는 곳
- `notify` 의 마스킹 우회·목적지 변경 판정·요약 모드 한도(a049f09, 93c1d39, 77834f0, 5081d95 연속 수정).
- `traffic` 의 인덱스 공간 회수와 삭제 후 정리(3227f2e, 2f5f7c0).
- `db` 의 커넥션 종료 순서로 인한 자산 잔존(18589b2).
- 역사적으로 테스트가 저장소 전체 TODO/FIXME 0개 수준으로 관리돼 있어, 개선 여지는 "테스트 없는 패키지"(`report`, `enrich`)에 몰려 있다.

## 검증 함정
- **push/PR CI 가 없다.** `.github/workflows` 에는 `pages.yml`(문서 배포)과 `release.yml`(`v*` 태그·수동 실행) 둘뿐이다. 일상 커밋은 아무 테스트도 거치지 않으므로 로컬 검증이 유일한 관문이다.
- `report`·`enrich` 는 릴리스 워크플로 패키지 목록과 통합 러너 기본값 양쪽에 빠져 있다. 이 두 패키지에 테스트를 추가해도 CI 에서는 돌지 않는다.
- `TestLiveContextReview` 는 외부 모델 서버가 필요해 항상 skip 된다. `TestTaskArchivePackageSkipsSymlink` 는 Windows 에서 심볼릭 링크 권한이 없으면 skip 된다.
- `server`/`db`/`agent`/`identity` 테스트는 격리 PostgreSQL 없이는 아예 돌지 않는다 — "통과" 가 아니라 "미실행" 으로 구분해 보고할 것.
