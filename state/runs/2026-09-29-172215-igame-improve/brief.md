- 과제: 아키텍처 백서의 비밀 저장·개인 키 회전 설명을 실제 구현과 일치시키기 (가치 3 / 위험 1 / 작업량 M)
- 왜: 백서 도입·구조도·2절은 사용자/테넌트별 DEK, 개인 API 키 암호화 저장, 자동 회전 유예를 구현된 기능처럼 설명하지만 현재 코드는 설치키 직접 암호화·개인 키 SHA-256 저장·회전 시 즉시 폐기를 구현한다. Markdown과 독자가 다운로드하는 PDF를 함께 고쳐 존재하지 않는 복구·회전 보장을 전제로 운영하는 일을 막는다.
- 수용 기준: 1) docs/architecture.md 도입·구조도·2절에서 2계층 봉투 암호화/Per-User Vault/사용자별 DEK/자동 유예 주장을 현재 계약으로 대체한다. 2) OIDC client secret·AI API key는 ENCRYPTION_KEY로 직접 AES-GCM 암호화되지만 개인 API/MCP 키는 SHA-256 검증값을 저장하고 원문은 발급 응답에만 나타난다는 차이를 명시한다. 개인 키 rotate는 트랜잭션 성공 시 기존 키를 즉시 폐기하며, 중첩 전환은 별도 키 생성→consumer 전환→이전 키 폐기라는 docs/security.md의 절차를 안내한다. 설치키 자동 회전/재암호화 도구를 새로 지원한다고 쓰지 않는다. 3) 대응 백서 PDF도 같은 설명을 담고 한글·코드·도식이 잘리지 않는다. 기존 secretbox 테스트는 실제 Box의 왕복·무작위 nonce를 증명하며, PDF는 렌더링 산출물 직접 열람으로 확인한다. 문자열 grep이나 해당 테스트만으로 API 회전 동작까지 검증했다고 주장하지 않는다.
- 건드릴 파일: docs/architecture.md:도입·1절 ASCII 구조도·2절 — 실제 비밀 저장/회전 계약 및 security.md 안내; docs/igame_Architecture_and_Security_Whitepaper.pdf — 기존 생성기로 재생성. 프로덕션 코드 0개, 의도한 변경 파일 2개.
- 검증 명령: `go test ./internal/secretbox/...`; `DOCS_PDF_DATE=2026-09-29 make docs-pdf`; `bash scripts/check-release-contract.sh`; `git diff --check`; `git diff --stat`. PDF 생성 뒤 표지/구조도/2절을 실제 뷰어로 열어 Markdown과 비교한다. PDF 테스트를 위한 소스 문자열 검사 테스트나 새 대역 테스트는 만들지 않는다.
- 위험과 피할 것: auth·session·키 제품 코드, migrations, workflows, VERSION, 의존성/lockfile, PDF 생성 스크립트 변경 금지. 문서 주장에 구현을 맞추지 않는다. make docs-pdf는 CRU PDF도 함께 쓰므로 먼저 기존 변경 여부를 확인하고 그 파일만 별도 백업한 뒤 생성 완료 후 원상복원한다(기존 사용자 변경을 덮어쓰지 않음). 백서 PDF를 낡게 남기고 Markdown만 완료 처리하지 않는다. PDF의 상대 링크 클릭 가능성은 미확인이라 링크만으로 핵심 계약 설명을 대체하지 않는다. 릴리즈 감사 실패 이력이 있으므로 빌드/릴리즈 경로 확대는 하지 않는다.
- 차선 후보: README RealmGuard 서버 재현 설명 정합성 개선 — PDF 생성 환경이 확보되지 않아 본 과제가 성립하지 않을 때만 README.md:44의 telemetry-only/비시뮬레이션 설명을 docs/realmguard.md와 internal/api/realmguard_replay.go:replayRealmGuardBattle에 맞춘다. 이 경우 과제서를 갱신하고 두 과제를 함께 수행하지 않는다.

범위와 확인 근거 (main@fb1f649)
- 실제 읽은 internal/secretbox/secretbox.go:New/Seal/Open은 입력 설치키로 AES cipher/GCM을 생성하고 nonce 및 AAD igame:v1과 v1: 포맷을 사용한다. cmd/igame/main.go:59는 cfg.EncryptionKey를 New로 전달한다.
- internal/api/admin.go:putOIDCSetting/putAISetting은 Secrets.Seal로 공급자 비밀을 저장한다. internal/api/apikeys.go:createAPIKey/rotateAPIKey는 sha256.Sum256을 key_hash로 저장하고 rotate는 UPDATE revoked_at과 새 INSERT를 같은 tx에서 commit한다. 원문 응답과 commit 순서까지 읽었다. 실제 API 회전 통합 실행은 이번 정찰에서 미확인이다.
- docs/security.md:세 가지 키 계층·개인 키 회전 문단과 docs/api.md의 POST /api/v1/me/api-keys/{id}/rotate 설명은 이미 즉시 회전 계약이므로 수정 대상이 아니다.
- scripts/build-docs-pdf.sh의 MANUALS는 CRU와 백서 2개를 생성한다. scripts/docs-pdf/render.mjs는 첫 문단을 표지로 소비하므로 본문 2절뿐 아니라 도입 수정이 필수다. print.mjs는 Chromium으로 출력하고 외부 자산 요청을 실패 처리한다.

실행 계획과 체크포인트 (모두 미착수; 사람 승인 없이 자체 점검)
1. 위 코드와 security.md를 다시 대조해 백서 3개 지점을 수정한다. 증거: `git diff -- docs/architecture.md`를 문장별로 읽고 각 주장을 위 함수에 대응시킨다. 체크포인트: 추측한 기능이 들어갔으면 다음 단계 전에 수정.
2. 백서 PDF를 재생성하고 의도하지 않은 CRU 산출물 변경을 원상복원한다. 증거: `DOCS_PDF_DATE=2026-09-29 make docs-pdf` 성공 및 PDF 실제 열람. 체크포인트: 한글/도식/2절 내용이 읽히고 기존 설명이 남지 않아야 함. 환경이 막히면 조용히 범위를 줄이지 말고 과제서와 기록을 갱신.
3. 위 Go 테스트·release contract·diff 검사를 실행하고 변경 파일 2개를 확인한다. 체크포인트: 실패 원인이 기존 환경인지 변경인지 기록하며 테스트 결과로 문서 밖 동작을 과장하지 않는다.

대안 비교와 추정 근거
- 선택: 백서+PDF 수정. 운영자가 보는 실제 배포 설명을 바로잡고 제품 동작 변경이 없어 위험이 작다.
- Markdown만 수정: 비용은 작지만 기존 PDF가 계속 잘못된 정본으로 제공되므로 채택하지 않는다.
- DEK/자동 유예를 실제 구현: 기존 문구를 만족하지만 암호화 마이그레이션·인증 정책 변경이라 45분 범위 밖이다.
- 현상 유지/보안 가이드 링크만 추가: 비용은 거의 없지만 모순된 명시적 보장을 남겨 채택하지 않는다.
- 가장 큰 가정: 기존 Docker 기반 PDF 생성 환경과 apt/npm 다운로드가 동작한다. 고정 Playwright 이미지가 로컬에 있음은 확인했지만 전체 생성은 미실행이다.
- Bottom-up: 코드/문구 대조 5~7분 + Markdown 수정 5~7분 + PDF 생성·열람 10~15분 + 검사/정리 3~5분 = 기본 23~34분. 알려진 PDF 다운로드·레이아웃 편차 예비 5~10분을 별도 두어 총 28~44분, 주관적 신뢰도 중간(통계적 보장 아님). 관리 예비는 이 과제에 배정하지 않으며 새 PDF 도구 도입은 범위 변경이다. 비교 가능한 문서+PDF 작업의 소요 기록이 없어 유사 추정은 숫자로 꾸미지 않았다.

정찰 검증 결과
- `go test ./internal/secretbox/...` PASS, `go test ./internal/api/... ./internal/database/... -count=1` PASS. IGAME_TEST_DSN 없는 실행이므로 PostgreSQL 회귀는 skip이며 DB 계약 검증이 아니다.
- `bash scripts/check-release-contract.sh` PASS(v0.7.24), `git diff --check` PASS, 작업 트리 변경 없음.
- `node scripts/docs-pdf/render.mjs docs/architecture.md /mnt/c/Users/USER/projects/aidev/state/runs/2026-09-29-172215-igame-improve/architecture-preview.html 0.7.24 2026-09-29` PASS. 원문 기준 HTML 생성만 확인했고 PDF 전체 재생성·시각 검수는 구현자 단계다.
- 요청한 3개 조직 스킬은 Skill 도구 부재로 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/ 아래 SKILL.md를 직접 읽어 적용했다. estimating-and-contingency/references/sources.md도 읽었으나 외부 기관의 수치·신뢰수준을 차용하지 않았다.
