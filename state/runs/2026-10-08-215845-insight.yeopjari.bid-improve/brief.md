- 과제: 제출 답변 URL이 500자를 넘으면 자르지 말고 입력 오류로 거절하기 (가치 4 / 위험 2 / 작업량 M)
- 왜: `judgeSubmission()`은 URL 답변을 500자로 자른 뒤 검증하므로 참여자가 제출한 주소와 다른 주소를 성공 저장할 수 있다. 정규화한 전체 URL의 길이를 먼저 검증하면 증빙 링크 훼손을 막고 참여자에게 수정할 필드를 알려 준다.
- 수용 기준:
  1) 기존 `text()` 정규화 후 정확히 500자인 http/https URL은 원문 전체가 반환·저장된다. 정규화 전 공백·제어문자로 500자를 넘더라도 정규화 후 500자이면 허용한다. 길이는 기존 계약의 JS `string.length` 기준이다.
  2) 정규화 후 501자·700자인 URL은 필수/선택 질문 모두 `validation_failed`, `submission.<질문ID>`의 500자 안내로 거절한다. 실제 제출 API는 HTTP 400과 `error.field_errors`를 반환하며 URL 원문을 오류에 포함하지 않는다. 선택 URL의 빈 값·비문자열 생략, 필수 URL의 누락 오류, http(s) 정책과 다른 질문 종류의 동작은 유지한다.
  3) 단위 테스트는 경계·정규화·여러 질문의 오류 수집과 기존 text(500)/long(4000) 제한을 증명한다. 실제 `POST /api/v1/participations/:id/submit` 후 참여자 GET 및 기업 GET으로 500자 URL의 완전 보존을 확인한다. 이어 다른 답변도 바꿔 501자 URL로 재제출하면 400이며, 재조회한 전체 submission 및 submitted_at이 이전 값과 같아야 한다.
- 건드릴 파일:
  - `packages/core/src/domain/eligibility.ts:judgeSubmission` (391행 부근): URL에만 `text(value, Infinity)`를 적용하고 빈 값 처리 뒤 500자 초과 오류를 수집한 다음 기존 `isWebLink`를 사용한다. 공통 text/long 분기는 기존 제한을 유지한다.
  - `packages/core/src/__tests__/rules.test.ts:describe('submission form')` (108행 부근): 공개 `parseForm`→`judgeSubmission`으로 위 경계를 실행 검증한다. 기존 `parseLinks` 테스트를 보존한다.
  - `scripts/e2e.mjs:studyB submission_form / 비동기 과제: 제출 → 자동 완료` (273행, 461행 부근): 선택 URL 질문 f3 추가, 양식 길이 2→3 기대값 갱신, 기존 성공 제출에 500자 f3 추가, 성공 GET과 초과 재제출→GET 비교 추가. f1의 '근거' 텍스트와 f2=4는 후속 결과 요약 검증 때문에 유지한다.
  - 전체 3개, 프로덕션 1개. 라우트·UI는 확인만 하고 수정하지 않는다.
- 검증 명령: 저장소 루트에서 `npm ci --no-audit --fund=false`; `npm run test -- packages/core/src/__tests__/rules.test.ts`; `npm run check`; `npm run build`; 격리 DB/서버를 준비한 뒤 `BASE=http://localhost:18790 DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55483/insight APP_DATABASE_URL=postgres://insight_app:devapppass@127.0.0.1:55483/insight node scripts/e2e.mjs`; `git diff --check`.
- 위험과 피할 것: 앞선 회차의 parseLinks 수정은 완료됐고 이번은 별도 제출 계약이다. 파서 통합, URL 길이 상향, 공통 `text`/`isWebLink` 변경, DB 스키마·auth·migration·workflow·쿼터 수정은 범위 밖이다. URL 원문을 감사 details/오류에 넣지 않는다. 소스 문자열 검사나 직접 주입한 가짜 라우트로 배선 검증을 대체하지 않는다. JSONB 객체 비교는 기존 `isDeepStrictEqual`을 쓰고 JSON.stringify 순서를 비교하지 않는다. e2e의 6회/분 제출 제한을 넘기지 않도록 API에는 필수 경계만 넣고 나머지는 단위 테스트로 검증한다. 기존 DB를 삭제하는 dev-reset.sh는 실행하지 않는다.
- 차선 후보: maskEmail·mailboxKey 경계 단위 테스트 — 1순위가 다른 변경으로 이미 해결됐을 때만 선택. `packages/core/src/util/text.ts`의 실제 export를 대상으로 별도 `packages/core/src/__tests__/text.test.ts`를 추가해 누락 이메일/짧은 주소 마스킹, 대소문자·공백·plus-tag·비정상 주소의 현재 계약을 검증한다. catalog 및 인증 배선 변경과 묶지 않는다.

## 확인한 근거와 현재 검증 상태

- 기준 main `c6a3399`, package 0.1.4. `git log -30`은 존재하는 15개 커밋을 반환했고 이전 두 개선이 반영돼 있다. CLAUDE.md/docs/별도 roadmap/추적된 AGENTS.md는 파일 검색에서 발견되지 않았다. packages/core/src·apps·scripts·engineering·.github의 TODO/FIXME 검색도 결과가 없었다.
- Node v22.23.1에서 `node:module`의 registerHooks/stripTypeScriptTypes로 원본 TypeScript와 상대 .js→.ts import를 메모리에서 읽어 공개 함수를 실행했다(소스 복사·수정 없음). `parseForm([{text:'증빙 링크',kind:'url',required:false}])`와 `'https://example.com/?q='.padEnd(n,'a')`를 사용한 결과: 500→500/동일, 501→500/변경, 700→500/변경 모두 성공. 같은 실행의 parseLinks(501)는 validation_failed로 정상 거절했다. 이는 함수 실행 증거이며 실제 HTTP/DB 실행 증거는 아니다.
- `routes/studies.ts`의 POST submit은 judgeSubmission 후에만 submission/submitted_at을 UPDATE한다. 참여자 GET(270행 부근)은 `application.submission`/`application.submitted_at`을 반환한다. 기업 `routes/team.ts` GET application(525행 부근)은 `submission`에 question/answer 배열을 반환한다. 기업 GET에는 원본 키별 submission 객체가 없으므로 참여자 GET으로 객체 전체를 비교한다.
- `apps/web/src/routes/Participant.tsx`의 제출 폼은 URL TextInput에 maxLength를 두지 않으며 `submission.` 접두사를 제거해 서버 오류를 필드에 표시한다. 따라서 UI 수정은 필요하지 않다.
- `npm run test -- packages/core/src/__tests__/rules.test.ts`를 실행했으나 node_modules 부재로 `vitest: not found`(127). 이번 정찰의 전체 단위/빌드/e2e 통과 수는 미확인이다. 이전 회차의 63개 단위/176개 e2e 성공은 과거 기록이며 이번 실행 결과가 아니다.

## 구현 순서와 검증 지점

1. [pending] 의존성을 npm ci로 준비하고 기존 단위 테스트를 실행한다. 위 경계 테스트를 추가해 501/700자가 거절되지 않는 실패를 확인한다. 검증: 단일 rules.test.ts 명령. 체크포인트: 예상 실패만인지 구현자가 확인; 별도 사람 승인 없음.
2. [pending] judgeSubmission의 URL 처리만 최소 수정하고 단일 테스트 및 `npm run check`를 실행한다. 체크포인트: 다른 답변/parseLinks 회귀가 없으면 진행; 사람 승인 없음.
3. [pending] 기존 studyB 시나리오에 e2e를 추가한다. 500자 성공 제출→참여자 원본/기업 표시 GET→501자 실패 재제출(다른 답도 변경)→참여자 submission/시각 동일 비교를 자동 완료 cron 이전에 둔다. 기존 평점 오류 검증을 유지한다. 검증: `npm run build` 후 아래 실서버 e2e. 체크포인트: 통과 수/실패를 기록; 사람 승인 없음.
4. [pending] `npm run check`, `git diff --check`와 변경 파일 목록을 확인한다. 코드가 계획과 다르거나 환경 준비가 예비 시간을 넘으면 범위를 늘리지 말고 계획과 미검증 항목을 갱신한다. 완료를 주장하려면 실제 HTTP/DB 회귀 실행이 필요하다.

## 격리 e2e 실행 조건

`.github/workflows/check.yml`, `.env.example`, `scripts/migrate.mjs`, `scripts/harness.mjs`를 확인했다. 이 정찰에서는 Docker/포트 사용 가능 여부와 실제 DB 준비는 미확인이다. 기존 서비스는 종료하지 말고 비어 있는 격리 포트를 사용한다. 아래는 55483/18790이 비어 있을 때의 구체 예이며, fake OAuth 18999도 필요하다(하네스가 고정 포트로 listen).

```bash
# 폐기 가능한 새 DB만 생성한다. 기존 컨테이너/DB 재사용·초기화 금지.
docker run -d --name insight-submit-url-check -e POSTGRES_PASSWORD=devpass -e POSTGRES_DB=insight -p 127.0.0.1:55483:5432 postgres:16-alpine
# pg_isready 성공 확인 후 migration 실행
docker exec insight-submit-url-check pg_isready -U postgres
DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55483/insight INSIGHT_APP_DB_PASSWORD=devapppass INSIGHT_GATEWAY_DB_PASSWORD=devgatewaypass node scripts/migrate.mjs
# 별도 터미널/관리 가능한 프로세스에서 서버 실행. .env.dev를 덮어쓰지 않는다.
(
  set -a
  . ./.env.example
  set +a
  export INSIGHT_DATABASE_URL=postgres://irumx_gateway:devgatewaypass@127.0.0.1:55483/insight
  export INSIGHT_ADMIN_DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55483/insight
  export INSIGHT_PUBLIC_URL=http://localhost:18790 PORT=18790
  node apps/server/dist/index.js
)
# 다른 터미널에서 health 성공 확인 후 실행
curl -fsS http://localhost:18790/api/v1/health
BASE=http://localhost:18790 DATABASE_URL=postgres://postgres:devpass@127.0.0.1:55483/insight APP_DATABASE_URL=postgres://insight_app:devapppass@127.0.0.1:55483/insight node scripts/e2e.mjs
```

gateway 실제 이름은 `irumx_gateway`이며 `insight_gateway`가 아니다. 하네스의 직접 RLS 검사 APP_DATABASE_URL은 insight_app을 쓴다. `.env.dev`가 이미 있고 다른 HMAC 키가 있으면 harness.envKey가 읽을 수 있으므로 서버와 하네스의 개발 키 설정을 일치시킨다. 자신이 만든 서버/컨테이너만 종료·정리한다. 브라우저 스크린샷과 운영 배포는 이번 변경의 로컬 필수 검증에 추가하지 않는다.

## 대안 비교와 작업량 근거

- 선택: 서버 URL 분기에서 길이 오류를 수집한다. 새 구성요소 없이 API 직접 호출까지 보호하고 프로덕션 1개 파일로 끝난다. 가장 중요한 가정은 500자 상한이 기존 제출 계약이라는 것인데, `isWebLink`와 현재 judgeSubmission의 500자 제한에서 확인했다.
- UI에만 maxLength를 추가하는 대안: 작지만 직접 API 호출 및 붙여넣기 주소 훼손을 해결하지 못하므로 제외한다.
- 저장 길이 상향/URL 파서 통합: 더 긴 URL 지원에는 적합할 수 있지만 정책 결정과 다른 파서 계약 검증이 필요해 이번에는 제외한다.
- 현상 유지/문서 안내: 구현 비용은 없지만 실제 제출값 훼손이 남으므로 제외한다. 테스트 공백·문서 정리는 사용자 영향이 확인된 이번 결함보다 후순위다.
- bottom-up 예상: 준비·기준 확인 5분 + 단위 회귀·수정 8분 + e2e 보강 7분 + 전체 검증·정리 10분 = 기본 30분. 알려진 환경 위험인 의존성 다운로드/격리 DB·키·포트 맞추기에 예비 5~15분을 별도로 둬 총 35~45분으로 추정한다. 통계적 80% 신뢰구간은 아니며 확신은 중간, 이미지/DB 준비에 실패하면 45분을 넘을 수 있다. 관리 예비는 이 과제에 포함하지 않으며 새 범위 추가로 소비하지 않는다.
- 유사 추정: 이전 parseLinks 회차도 프로덕션 1개+테스트 2개와 같은 검증 경로로 성공해 범위는 비교 가능하다. 실제 소요 분 기록은 없어 시간 추정의 독립 검산으로는 쓰지 않았다. 최초 기준 테스트·DB 준비 뒤 남은 시간을 재산정한다.
- 적용 스킬: Skill 도구가 노출되지 않아 로컬 [estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md)를 직접 읽었다. 범위·가정·작업 분해·위험을 기록하는 추정 방법은 스킬 참고 자료인 [GAO 가이드 개요](https://www.gao.gov/products/gao-20-195g)를 확인해 적용했고, 위 분 단위 수치는 저장소 근거에 따른 정찰자의 추정이다.
