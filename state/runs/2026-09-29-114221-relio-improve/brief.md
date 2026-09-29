- 과제: make test에 기존 프런트 회귀 테스트를 포함하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: README.md의 개발 절이 안내하는 `make test`는 Go test/vet와 프런트 설치·typecheck만 실행하여, 이미 CI가 실행하는 로그인 렌더링·SSO·멤버 목록·날짜 회귀 테스트를 빠뜨린다. 기존 `npm test`를 같은 진입점에 연결하면 개발자가 테스트 성공을 보고도 이 회귀 검증을 건너뛰는 공백을 없앤다.
- 수용 기준:
  1) `make test`가 기존 Go test/vet 및 npm ci/typecheck를 유지하면서 `web` 디렉터리에서 `npm test`도 실행한다. `make -n test` 출력에도 이 명령이 보인다.
  2) 기존 web/test 4파일 전체가 실행·통과하고, 임시로 실패시키면 `make test`가 0이 아닌 종료 코드를 반환한다. 실패를 무시하는 접두사 `-`, `|| true`, 백그라운드 실행을 넣지 않는다.
  3) README 개발 절에 `make test`가 백엔드 테스트·정적 분석·프런트 타입 검사·회귀 테스트를 수행한다는 한 문장을 추가한다. 기존 Node.js 24+ 요구를 유지한다.
  4) 기존 테스트·제품 동작·의존성·CI/릴리즈 워크플로는 바꾸지 않는다. 최종 diff는 Makefile과 README.md 두 파일 이내다.
- 건드릴 파일:
  - `Makefile:test`(16행 부근) — 현재 `cd web && npm ci && npm run typecheck` 체인 끝에 `&& npm test`를 연결한다. Go 명령과 기존 실패 전달을 유지한다.
  - `README.md:개발`(158행부터) — 위 로컬 검증 명령이 포함하는 범위를 한 문장으로 설명한다.
  - 프로덕션 코드 파일 0개. 새 영구 테스트 파일은 필요 없다.
- 검증 명령:
  - 저장소 루트에서 `make -n test` — 기존 순서와 npm test 포함 확인.
  - `make test` — README 요구 환경 Go 1.24+·Node 24+에서 실행. npm ci를 포함하므로 npm 레지스트리 접근 또는 충분한 설치 캐시가 필요하다.
  - `npm --prefix web test` — 필요할 때 프런트만 분리 검증. package.json에 이미 존재하는 Node 내장 러너 명령이다.
  - `git diff --check` 및 `git status --short` — 두 파일만 남았는지 확인.
  - 실패 전달은 일회성 임시 `web/test/zz-make-failure.test.ts`에 실패하는 node:test 하나를 두고 `make test`의 비영 종료를 확인한 뒤 반드시 삭제한다. 영구 테스트나 기존 auth 테스트 수정으로 남기지 않는다. 삭제 뒤 `npm --prefix web test`가 다시 통과하는지 확인한다.
- 위험과 피할 것: `internal/auth`, `internal/oidc`, `migrations/`, `.github/workflows/` 및 web/src는 범위 밖이다. 과거 가이드 캡처 실행파일 선택 과제의 no-change를 반복하지 않는다. release.yml에도 npm test 누락이 있지만 이번 회차에 함께 고치지 않는다. 네트워크 설치 실패를 제품 코드 결함으로 보고 의존성·lockfile을 바꾸지 않는다. web build는 이 변경의 필수 검증이 아니며 embed 앵커를 건드릴 필요가 없다.
- 차선 후보: `docs/ROADMAP_PLAN.md` 3.1 VOC 계획에 이미 제공되는 기능과 남은 계획을 구분하는 현재 상태 문단 추가 (가치 2 / 위험 1 / S) — Makefile이 실행 시점에 이미 수정되어 1순위가 성립하지 않을 때만. README 핵심 특성과 대조하여 부서 VOC·SLA·이탈 위험이 구현된 사실만 쓰고 ERP/KMS 미래 일정은 변경하지 않는다.

구현 순서 및 체크포인트(사람 승인 불필요, 각 단계는 구현자가 자체 검증):
1. [pending] 기준 `make -n test`로 누락을 재확인하고 Makefile:test의 체인 한 줄만 수정한다. 같은 명령으로 npm test가 포함되는지 확인한다.
2. [pending] README 개발 절 한 문장을 보완한다. `git diff --check` 후 `make test`로 기존 검증과 프런트 4파일의 성공을 확인한다.
3. [pending] 위 일회성 실패 주입으로 종료 코드 전파를 확인·원복하고 프런트 테스트를 다시 실행한다. 최종 diff 두 파일 확인 후 각 단계 상태와 실제 실행 결과를 회차 노트에 기록한다. 설치가 안 되면 원인과 미검증 항목을 기록하고 범위를 바꾸지 않는다.

근거와 탐색 결과:
- main@b4cdc5f에서 Makefile:test와 README:개발, web/package.json:scripts.test, .github/workflows/ci.yml의 Frontend decision tests를 직접 읽었다. `make -n test`는 go test, go vet, npm ci/typecheck만 출력했다.
- 실제 열어 본 테스트는 web/test/login.test.ts(실제 Login.tsx를 esbuild로 번들링·React 서버 렌더링), silentSso.test.ts, memberList.test.ts, dateTime.test.ts다. 타입 검사만으로 대체할 수 없는 동작 검증이다.
- 대안 A(선택): 기존 npm test를 표준 타깃에 연결 — 새 도구나 API 계약 없이 해결한다. 대안 B: README에 별도 명령만 안내 — 명령을 잊는 공백이 남는다. 대안 C: CI·release·Makefile 검증 전체 통합 — 보호 워크플로까지 번져 이 문제에 비해 범위가 크다. 현상 유지도 가능하지만 README 명령과 실제 검증 범위의 차이가 남는다.
- 핵심 가정: README의 make test가 로컬 기본 검증 진입점이라는 점. CI의 Node 24 및 npm test 실행이 그 테스트를 기본 검증에 포함할 근거다.

작업량과 여유:
- 추정 주체는 정찰 에이전트. bottom-up 기준 수정·문서 5~8분, 정상 검증 5~10분, 실패 전달 확인·정리 5~7분으로 기본 15~25분. 이미 있는 4개 테스트 파일과 명령을 재사용하며 새 테스트 설계·DB·빌드·마이그레이션은 제외한다.
- 알려진 불확실성은 npm 설치·캐시 속도: 대응 여유 5~10분을 별도로 두어 총 20~35분 예상, 환경 준비 시 45분 내 완료 신뢰도는 중간(통계적 보장 아님). 과거 S 회차의 실측 소요 시간이 없어 유사 추정으로 숫자를 검증하지는 못했다.
- 관리 예비는 이 회차에 배정하지 않는다. 네트워크 복구·도구체인 설치·워크플로 통합 같은 새 범위는 더하지 않는다.

정찰 검증 결과와 한계:
- `go test ./...`, `go vet ./...` 통과. `make -n test`로 누락 재현.
- `node --test web/test/dateTime.test.ts web/test/memberList.test.ts web/test/silentSso.test.ts`는 Node v22.23.1에서 15/15 통과.
- 저장소 변경을 피하려고 지정 run 디렉터리 assets/web-check에 web 사본을 만들어 npm ci를 실행했으나 ETIMEDOUT으로 실패했다. 따라서 로그인 렌더링을 포함한 npm test 전체·typecheck·make test 전체 성공은 이번 정찰에서 미확인이다. 구현자는 README/CI의 Node 24 환경에서 검증할 것.
- 전용 Skill 도구는 노출되지 않아 로컬 headcount의 estimating-and-contingency, implementation-planning, solution-exploration SKILL.md를 직접 읽었다. 위 대안 비교·단계/증명/체크포인트·분해 추정/여유 분리로 절차를 반영했다.
