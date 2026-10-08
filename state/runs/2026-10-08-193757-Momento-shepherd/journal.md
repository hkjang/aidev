# PR 처리기 노트 2026-10-08-193757-Momento-shepherd — Momento PR #30
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-190823-Momento-improve)
# 회차 노트 2026-10-08-190823-Momento-improve — Momento
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:08] base pinned — main@0beff4c
- [러너 19:08] autonomy release — 

## 정찰 노트
- 선택: Custom Dimension 저장 Alert 한 곳의 한국어 안내. 서버 여섯 코드·정규식·upsert를 확인해 DataTable 의도 추측/CIDR DB 미재현 후보보다 확실하며 프로덕션 2파일로 끝난다.
- 초안 선작성 뒤 서버 계약으로 보완했다. 이전 make verify는 no-change라 재선정하지 않고 CI 검사 명령을 빠짐없이 인계했다.
- 실측: web npm test 259/259 통과. 브라우저·DB·전체 CI·감사는 미실행; 500 원인과 UNKNOWN_SITE 실제 화면 도달성은 미확인이다.
- 주의: 이름 첫 글자/128자 규칙을 누락하지 말고, 중복 이름은 갱신이라 500을 중복으로 설명하지 않는다. 입력 차단·서버·보호 경로·다른 Alert를 같이 바꾸지 않는다.
- [러너 19:14] scout done — Custom Dimension 등록·갱신 실패를 한국어로 안내한다 (가치 3 / 위험 1 / 작업량 S)

## 구현 진행
- 과제서 채택: saveDimension 여섯 코드·PropertyKeyPattern·upsert와 기존 원문 Alert가 모두 일치. 프로덕션 2파일+테스트 1파일만 변경하며 입력 검사·서버·다른 Alert·공통 문구는 제외.
- Skill 도구가 없어 technology completion-verification/systematic-debugging/test-driven-development의 로컬 SKILL.md 정본을 읽고 적용.
- Red: 새 테스트 11개를 먼저 추가, export 없음으로 실패한 뒤 원문 반환 대조 구현으로 9개 assertion 실패(기존 259개 통과). Green: npm test 270/270. 다시 대조 구현으로 되돌려 같은 9개 실패 확인 후 복원. 로그: web-red-identity.log, web-green.log, web-revert-proof.log.

## 검증 실행 기록
- `(cd web && npm ci && npm audit && npm run lint && npm test && npm run build)` exit 0: audit 0 vulnerabilities, lint 통과, 270/270, build 통과. Vite charts/vendor 500kB 초과 경고는 남음; 번들 분할은 범위 밖.
- `(cd sdk && npm ci && npm audit && npm run typecheck && npm test && npm run build)` exit 0: audit 0 vulnerabilities, typecheck 통과, 29/29, build 통과.
- `go test -race ./cmd/... ./internal/...` exit 0; `go vet ./cmd/... ./internal/...` exit 0.
- `go run golang.org/x/vuln/cmd/govulncheck@latest ./cmd/... ./internal/...` exit 0: 호출 경로 취약점 0. 도구가 비호출 import 패키지 4건·require 모듈 20건을 함께 보고하므로 의존성 전체가 무취약하다고 주장하지 않음.
- MOMENTO_TEST_POSTGRES_DSN 미설정 확인: DB 통합 테스트는 skip. 실제 DB의 500 원인·UNKNOWN_SITE 발생 조건·Docker 빌드 미검증. HTTP 응답 대역은 서버 장애 재현이 아님.
- 브라우저 실제 명령(저장소 루트): `node /mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-190823-Momento-improve/dimension-browser.mjs`. 기존 /tmp/pptr의 puppeteer-core와 /usr/bin/google-chrome(151.0.7922.108) 재사용, 새 npm script/의존성 없음. 로컬 HTTP 서버가 web/dist와 API 응답을 제공하며 앱 api()가 APIError를 생성; 각 사례는 새 탭.
- 브라우저 첫 실행: 실패 응답 9사례 통과 뒤 성공 경로 pending 버튼을 React 렌더 전 읽어 assertion 실패(exit 1). DOM의 disabled 전환을 기다려 원인을 제거. 두 번째 실행: 앞선 검사 통과 뒤 프로덕션 SVG에 없는 data-testid로 삭제 버튼을 찾다가 실패(exit 1); 화면의 편집 버튼 옆 삭제 버튼으로 selector 수정. 앱 코드는 변경하지 않음. 첫/두 번째 로그는 browser-first-run.log, browser-second-run.log에 보존.
- 브라우저 최종 실행 exit 0, `RESULT 10/10 scenarios passed` (browser.log). 오류 9사례 모두 mapper의 message/detail과 실제 Alert DOM이 일치. 편집·저장 pending·실패 후 입력 보존/재시도·성공 폼 초기화·쿼리 무효화·삭제 동작 확인. 수용 기준 및 diff 자체 검토 완료, 범위 변경 없음.

## 구현 노트
- Custom Dimension 저장 실패만 독립 describeDimensionError + DimensionErrorAlert로 연결(프로덕션 2파일, 테스트 1파일). 이름/키 두 라벨·정확한 규칙, 선택지 안내, 중립 500 본문/detail 분리를 구현.
- TDD: 신규 11개 테스트 선작성 → 원문 반환 구현에서 9개 실패 → 270/270 통과 → 대조 구현으로 되돌려 동일 실패 확인 후 복원. 원장과 red 로그에 실제 실패 출력 보존.
- web 최종 게이트·SDK 최종 게이트·Go race/vet/govulncheck 모두 exit 0. web 270/270, SDK 29/29, npm audit 0건. Vite 청크 경고와 govulncheck 비호출 취약점 보고는 위 기록 참조.
- 브라우저: 위 실제 node 명령으로 빌드된 앱을 열어 HTTP 실패 응답 9사례+편집/성공/삭제 1사례, 10/10 통과. 본문/detail·버튼 조건·초기화·목록 재조회 확인; 하네스 두 실패는 앱 변경 없이 해결해 별도 로그 보존.
- 확신 없는 곳·검증 못 한 것: DSN 미설정으로 DB 통합 skip; 실제 DB 500 원인과 UNKNOWN_SITE 도달 조건, Docker 빌드 미검증. 응답 대역 검증을 DB 동작 증명으로 읽지 말 것.
- 입력 helperText/유효성 차단·정규화·서버·다른 Alert·공통 문구·잠금/의존성·버전은 범위 밖이라 변경하지 않음. 빌드 산출물은 삭제 후 커밋에서 제외. 커밋 b70a1b9, 최종 git diff --check 통과·git status --short 빈 출력.
- 다음 역할: 브라우저는 사례별 새 탭 사용; mapper 원문 fallback 유지. DB 통합은 별도 PostgreSQL DSN이 필요하며 보안 검사 상세 보고는 호출 경로 0건과 구분할 것.
- [러너 19:21] brief accepted — 채택 — 여섯 서버 코드·PropertyKeyPattern·upsert·원문 저장 Alert가 현재 코드와 일치하여 지정한 3파일 범위로 수용 기준을 �
- [러너 19:21] verify passed — 검증 7개 통과 (auto)
- [러너 19:22] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 19:22] pr created — https://github.com/hkjang/Momento/pull/30
