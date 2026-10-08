- 과제: Node StaticFiles의 실제 파일 기반 캐시·HEAD·SPA·경로 격리 회귀 테스트 (가치 3 / 위험 1 / 작업량 S)
- 왜: `apps/server/src/static.ts`의 `StaticFiles.serve`·`shell`·`resolve`는 오프라인 배포의 정적 자산과 SPA 진입점을 맡지만, 현재 server 워크스페이스에는 테스트가 없다. 실제 파일과 Request/Response로 현재 응답 계약을 보장하면 캐시·보안 헤더·깊은 링크 회귀를 DB 없이 잡을 수 있다.
- 수용 기준:
  1) 실제 임시 디렉터리에 `index.html`, `assets/main.js`, 일반 `plain.txt`를 만들고 GET의 본문·MIME·UTF-8 바이트 기준 content-length·nosniff를 검사한다. `/assets/main.js`는 `public, max-age=31536000, immutable`, `/plain.txt`는 `public, max-age=60`이어야 한다. 두 자산의 HEAD는 GET과 같은 상태·헤더에 본문이 비어야 한다.
  2) `/`, `/index.html`, `/admin/settings`는 루트 셸 본문과 `no-store`를 반환하며 HEAD에서도 같은 헤더와 빈 본문을 유지한다. CSP의 `default-src 'self'`, `frame-ancestors 'none'`, 지정한 connectSources의 `connect-src` 포함과 DENY·nosniff·referrer-policy·permissions-policy·cross-origin-opener-policy를 검사한다. 셸 파일을 다시 쓰고 `utimes`로 확실히 다른 mtime을 지정한 뒤 같은 인스턴스의 GET이 새 본문을 반환해야 한다(시간 sleep 금지).
  3) 루트 `www` 옆의 `www-sibling/outside.txt`에 별도 sentinel을 만들고 `/%`, `/%00`, `/..%2fwww-sibling%2foutside.txt` 요청은 예외 없이 셸로 돌아가며 외부 sentinel을 노출하지 않아야 한다. 각 호출은 실제 `Request`를 만들고 생산 호출부처럼 `new URL(request.url).pathname`을 `serve`에 넘긴다. 정상 파일은 실제 반환되므로 무조건 셸만 주는 구현도 테스트를 통과하지 못해야 한다. 모든 GET 스트림은 끝까지 읽고 fixture는 afterEach/finally로 정리한다.
- 건드릴 파일: `apps/server/src/__tests__/static.test.ts`(신규) — `import { StaticFiles } from '../static.js'`로 위 계약을 검증한다. 프로덕션 파일 0개, 테스트 파일 1개. `static.ts`·`index.ts`·vitest 설정은 읽기 참고용이며 변경하지 않는다.
- 검증 명령:
  - 저장소 루트, Node >=22에서 의존성이 없으면 구현 단계에서 `npm ci` 후 진행한다.
  - `npm test -- apps/server/src/__tests__/static.test.ts`
  - `npm run check` (lint → core/server/web 및 workers/db-watch 타입 검사 → 전체 Vitest).
  - 정찰 실행 결과: Node v22.23.1의 `node --experimental-transform-types --input-type=module`로 실제 `StaticFiles`를 직접 import하여 15개 시나리오(GET/HEAD·UTF-8 길이·캐시·잘못된 인코딩/NUL/형제 디렉터리·mtime·CSP)를 실행, 모두 통과했다. fixture 생성·삭제는 이 회차 출력 디렉터리 안에서만 했다. `npm test -- packages/core/src/__tests__/validate.test.ts`는 exit 127(`vitest: not found`)이었다. `npm ci`, 전체 check, HTTP 서버/bootstrap/DB는 실행하지 않았으므로 통과를 주장하지 않는다.
- 위험과 피할 것: 현재 결함을 재현한 과제가 아니라 확인된 동작의 회귀 테스트다. auth·migrations·workflows·vendor·Node 어댑터 라우팅과 기존 미머지 쿠키/사진/IndexNow/입력 가드 과제를 건드리지 않는다. 파일 시스템·Request·Response·StaticFiles를 대역으로 바꾸거나 private 메서드 접근·소스 문자열 검사로 동작 증명을 대신하지 않는다. 셸 fallback을 404로 바꾸지 않고, symlink/realpath 정책·없는 빌드의 HEAD/503·하위 디렉터리 index.html 정책은 이번 범위에서 제외한다. `Request`가 정규화하는 `/../`만 검사해서 traversal이 증명됐다고 하지 않는다. encoded slash를 포함한 위 입력과 외부 sentinel을 함께 사용한다. 실제 클래스 응답까지의 증명이며 Node 네트워크 어댑터 전체 E2E는 아니다.
- 차선 후보: 공개 문서 생성물의 버전 동기화 — `VERSION=0.1.1`인데 `docs/llms.txt`는 0.1.0이다. 1순위가 이미 구현된 경우에만 `scripts/build-docs.mjs`의 VERSION 사용과 생성 범위를 재확인하여 한 문서 산출물로 좁힐 것; 생성 전체 변경으로 확대하지 않는다.

범위·선택 근거:
- 무변경 유지도 가능한 상태이나 server 테스트 0개인 공백이 남는다. 선택안은 실제 파일 기반 테스트 1개로 실행 환경이나 프로덕션 동작을 늘리지 않는다.
- 전체 HTTP/DB E2E는 생산 배선 범위가 넓지만 bootstrap·계정·DB 준비가 필요해 45분 과제로 부적절하다. 파일 처리 추출/통합은 검증 목적에 필요하지 않아 제외했다.
- 기존 normalisePath·redactSecrets 후보에는 이미 `error-alerts.test.ts` 4개와 `redact.test.ts` 5개의 기본 계약 테스트가 있다. 추가 외부 입력 사례의 실해로움은 미확인이므로 server의 명확한 테스트 공백보다 후순위다.
- 쿠키·사진·IndexNow·입력 가드의 이전 개선 결과는 현재 c7f11ce에 반영되지 않았다(관련 신규 테스트 3개 없음, validate의 새 가드 테스트 없음). 원격 PR 상태는 미확인이고 중복 구현하지 않는다.

실행 순서·점검점 (구현자는 각 단계 완료 후 이 상태를 갱신):
1. [pending] 의존성 준비, 실제 임시 파일 fixture와 자산 GET/HEAD 사례를 새 테스트에 추가 → `npm test -- apps/server/src/__tests__/static.test.ts`. 첫 통과 후 다음 단계. 사람 검토 점검점 없음.
2. [pending] 같은 파일에 셸·CSP·mtime·잘못된 경로 사례를 추가 → 같은 명령. 실패가 확인된 현재 계약과 다르면 멋대로 프로덕션 수정하지 말고 과제서와 결과에 차이를 기록한다. 사람 검토 점검점 없음.
3. [pending] `npm run check` 통과 및 `git diff --stat`로 테스트 1개 범위 확인. 검증 불가면 명령·에러를 기록하며 통과 처리하지 않는다. 사람 검토 점검점 없음.

추정 근거 (pmo:estimating-and-contingency):
- 정찰자가 읽은 static.ts 전부와 index.ts:handle, vitest.config.ts, server tsconfig 및 위 실파일 실행을 근거로 한 bottom-up 추정이다. fixture/자산 8~10분, 셸/경계 10~15분, check/정리 5~10분: 기본 23~35분. 알려진 변동(의존성 준비·파일시각·타입 검사)에 예비 5~10분을 별도로 두어 총 28~45분을 예상한다.
- 45분 안 완료 가능성은 보통 수준의 판단이며 통계적 신뢰구간은 아니다. 이전 테스트 전용 회차는 범위 비교에만 사용했다(실제 소요 시간 자료가 없어 속도 추정에는 사용하지 않음). Node 22와 npm 설치 가능, 기존 check가 환경 문제 없이 돈다는 가정이 가장 크다. 첫 대상 테스트 통과 때 재추정한다.
- DB·빌드·배포·정책 변경·기존 전체 검사 결함 수리는 제외한다. 미지 범위에 대한 management reserve는 이 회차에 배정하지 않았으며, 새 작업을 예비 시간에 끼워 넣지 않는다.

적용 스킬: 전용 Skill 도구가 없어 로컬 원문을 읽었다: `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md`(references/sources.md 포함), 같은 headcount의 `technology/skills/implementation-planning/SKILL.md`, `technology/skills/solution-exploration/SKILL.md`. 외부 비용 추정 표준의 수치나 권위는 사용하지 않았다.
