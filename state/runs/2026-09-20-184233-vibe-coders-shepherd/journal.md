# PR 처리기 노트 2026-09-20-184233-vibe-coders-shepherd — vibe-coders PR #22
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-20-181357-vibe-coders-improve)
# 회차 노트 2026-09-20-181357-vibe-coders-improve — vibe-coders
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:14] base pinned — master@d67267b
- [러너 18:14] autonomy release — 

## 정찰 노트
- 수정 과제 선택: TS5072를 현재 명령과 직전 verify 로그에서 확인; 신규 기능보다 릴리즈를 계속 막는 미병합 964d2b0의 세 파일 복원이 우선이다.
- 워크플로는 정상 인자이며 외부 run_verify가 --silent를 전달한다. 보호 파일·외부 러너·MCP 브랜치 전체를 바꾸지 말고 실제 타입 오류가 계속 실패하는 증거를 남길 것.
- 정찰은 코드 미수정·커밋 없음; 수정 후 동일 명령 통과는 구현자 몫이다. 재현 중 pnpm이 node_modules를 자동 설치했으며 추적 파일 변경은 없다.
- 요청 회사 스킬 3종은 도구/로컬 검색에서 미발견, 고유 형식 미확인. 보류 12건은 현재도 미해결 또는 선행 병합 대기여서 pending을 유지했다.
- [러너 18:18] scout done — 수정 과제 — 미병합된 pnpm 전달 옵션 호환 수정만 복원하여 typecheck·build 검증 실패 해소 (가치 5 / 위험 1 /

## 구현 노트
- 3a17c4f: 미병합 964d2b0의 package.json·run-tool.mjs·run-tool.test.mjs만 복원하고 실제 실패 보존 테스트 2개를 추가했다.
- 수정 전 동일 --silent 명령은 typecheck exit 1/TS5072, build exit 1/CACError; 수정 후 모두 exit 0이며 dist/index.html·assets 생성 확인.
- 실제 포함 fixture의 TS2322에서 typecheck/build 모두 exit 2·Vite 미진행을 확인하고 finally 제거; 이후 정상 명령 재통과. node:test는 package.json test에 배선됨.
- pnpm frozen install·test(908 Vitest+10 node:test)·typecheck/build(옵션 유무 모두)·lint·format·OpenAPI와 go build/vet/test 모두 통과; 로그는 이번 결과 디렉터리에 저장.
- 확신 없는 곳·검증 못 한 것: Docker/E2E/전체 race/PostgreSQL CI/취약점·SBOM 미실행; 요청 회사 스킬 3종은 도구·파일 미발견으로 고유 형식 미확인.
- 보호 파일·외부 러너·lockfile·앱·Go는 변경하지 않았고 tsc -b·strict·&&는 유지; 푸시·병합·릴리즈는 이번 범위 밖이다.
- 다음 역할: 타입 오류 회귀는 잠시 src에 fixture를 만들므로 같은 worktree에서 typecheck/build와 병렬 실행하지 말 것. 산출물은 기존 gitignore 대상이며 커밋은 세 파일뿐이다.
- [러너 18:25] brief accepted — 채택 — 현재 스크립트·워크플로와 외부 run_verify의 인자 전달 원인이 일치하여 지정된 세 파일만 복원하고 실제 실패 보
- [러너 18:28] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- reject (medium), 부서 차단 없음: 수리는 internal/proxy/tracking.go:242의 요청별 Transport 누수와 전체 10초 deadline 누락부터 확인.
- 실제 master=eee0ed4 비교는 추적 포함 26파일; pinned d67267b 이후 래퍼 3파일 자체에서는 결함 미발견.
- 래퍼 실행 테스트 5개 및 추적 관련 Go 선별 테스트 통과; TS2322 fixture 테스트는 읽기 검토만, 연결 누적·지연 응답은 정적 확인.
- Docker/E2E/race/PostgreSQL/취약점·SBOM/PDF 렌더링 미확인; /admin CSP 설명 불일치와 기준 브랜치 차이는 다음 단계에 전달.
- [러너 18:30] review rejected — 리뷰 거절: internal/proxy/tracking.go:242 [P1] Momento 프록시 요청마다 새 http.Transport를 생성하지만 CloseIdleConnections를 호출하지 않고 IdleConnTimeout도 0이다. keep-ali

## 수리 노트
- P1/P2 모두 맞음: 회귀 테스트에서 연결 잔류와 13초 본문 대기를 재현; 틀린 지적 없음. 해당 코드는 pinned 기준에도 존재하지만 명시된 수리 대상으로 수정.
- 요청 종료 시 Transport 유휴 연결 정리 및 전체 전달 요청에 10초 context deadline; 실제 수집기 연결 종료·본문 중단·취소 테스트 추가.
- 수정 후 전체 Go test/build/vet 및 Momento race 통과; f201cf5 새 커밋, push 없음.
- 확신 없는 곳: 전체 race/PostgreSQL CI/Docker/E2E 미실행; 요청 회사 스킬 3종의 도구·파일 미발견으로 고유 형식 미확인.
- [러너 18:37] repair done — - P1/P2 모두 재현: 정상 응답 뒤 연결 잔류, 헤더 후 본문 정지 시 10초를 넘어 13초 클라이언트 제한까지 대기. - 요청별 Transport에 defer CloseIdleConnections를 �
- [러너 18:37] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 18:37] pr created — https://github.com/hkjang/vibe-coders/pull/22
