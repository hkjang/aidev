- 과제: 과제 링크가 500자를 넘으면 잘라 저장하지 말고 입력 오류로 거절하기 (가치 4 / 위험 2 / 작업량 M)
- 왜: 실제 연구 저장 경로의 `parseLinks()`가 URL을 `text(l.url, 500)`으로 먼저 자르므로 `isWebLink()`의 500자 제한을 우회하고, 긴 경로·쿼리를 다른 주소로 저장한다. 길이 초과를 저장 전에 거절하면 담당자가 링크를 고칠 수 있고 참여자에게 손상된 링크가 전달되는 것을 막는다.
- 수용 기준: 1) 정규화 후 500자인 유효한 HTTP(S) URL은 전체를 보존하고, 501자 이상은 `validation_failed`와 `fieldErrors['task_links[원래 인덱스]']`로 거절한다(사용자 메시지는 500자 제한을 설명). 2) 기존 공백 제거·제어문자 정리, 빈 값/비문자열 URL 생략, 기본 라벨 '링크', 라벨 60자 제한, 첫 8개 처리, HTTP(S)만 허용하는 계약은 유지한다. 3) 실제 PUT 연구 저장→GET 연구 상세를 지나는 e2e에서 500자 링크는 그대로 조회되고, 501자 저장 시도는 HTTP 400이며 기존 링크·연구 제목 등 저장 상태가 바뀌지 않음을 증명한다.
- 건드릴 파일: `packages/core/src/domain/eligibility.ts:parseLinks` — URL 길이를 자르기 전에 판단하고 초과 오류를 수집(프로덕션 1개); `packages/core/src/__tests__/rules.test.ts:describe('submission form')` — 공개 parseLinks의 500/501 경계·원래 오류 인덱스·정규화 회귀 테스트 추가; `scripts/e2e.mjs:연구 만들기 → 검수 → 공개` — 기존 `badLink`/`saved` 주변에 실제 저장·조회 검증 추가. 총 3개, 프로덕션 1개이며 새 의존성 없음.
- 검증 명령: 준비가 필요한 구현 환경에서는 `npm ci --no-audit --fund=false`; `npm run test -- packages/core/src/__tests__/rules.test.ts`; `npm run check`; `npm run build`; 개발 Postgres·서버를 준비한 뒤 `node scripts/e2e.mjs`; 마지막 `git diff --check`와 `git status --short`. 아래 검증 환경/관찰 결과를 함께 볼 것.
- 위험과 피할 것: 공용 `text()` 구현·`isWebLink()` 허용 정책·`judgeSubmission()`·다른 URL 파서·프런트 폼·quota·auth·migrations·workflows·vendor는 수정하지 않는다. 서로 다른 파서의 통합이나 사설망 차단을 끼워 넣지 않는다. URL 원문을 audit details/로그/에러 메시지에 넣지 않는다. npm install 대신 ci를 쓰고 lockfile 변경을 포함하지 않는다. 기존 저장된 링크의 자동 보정은 범위 밖이다.
- 차선 후보: `maskEmail`/`mailboxKey`의 표시·정규화 경계 테스트 보강(가치 2 / 위험 1 / 작업량 S) — 1순위가 다른 변경으로 이미 해결된 경우에만 `packages/core/src/__tests__/text.test.ts`를 추가하고, 실제 `util/text.ts`를 호출해 null/한 글자 로컬/점 없는 도메인/+로 시작하는 로컬을 확인한다. 기대값은 현 계약을 먼저 확인하고 정상 주소를 바꾸지 않는다.

근거와 재현

- 기준 커밋 `cbc8cb5`. `domain/eligibility.ts:159 text()`는 줄바꿈 정규화·제어문자 제거·trim 뒤 `.slice(0, max)`를 한다. `parseLinks():292`는 이 함수를 max=500으로 부른 뒤 `isWebLink():283`에 보낸다.
- `routes/team.ts`의 `PUT /api/v1/team/studies/:id`는 `parseLinks(body.task_links)`를 실제로 호출하고, 반환값을 `insight.study_briefs.task_links`에 저장한다. 같은 파일의 연구 상세는 task_links를 반환하고 `routes/studies.ts`는 선정된 참여자에게 brief.task_links를 전달한다. 이 라우트 파일들은 읽기 근거일 뿐 수정 대상이 아니다.
- Node v22.23.1의 `stripTypeScriptTypes`로 기존 eligibility/errors/crypto/catalog 모듈을 메모리에서 적재하여 공개 함수를 실행했다(소스 수정·의존성 대역 없음). `https://example.com/?token=` 뒤 a로 총 길이를 맞춘 결과: 500자 → isWebLink=true, parseLinks 출력 500자/원문 동일; 501자 → isWebLink=false지만 parseLinks 출력 500자/원문 불일치. 데이터베이스 저장·실브라우저 결과는 정찰에서 미확인이다.
- 후보 비교: 긴 링크 거절은 실제 운영 호출 경로와 재현이 있는 작은 입력 검증이다. 500자 초과 URL 허용으로 확장하는 대안은 다른 링크 길이 계약과 화면을 함께 확인해야 하므로 제외한다. 현상 유지·안내 문서만 추가는 조용한 데이터 손상을 막지 못한다. 시간 함수 테스트 후보는 kstWindowOn/overlaps/backoffMs의 운영 호출처가 없고, API 캐시 204 후보도 api.cached 사용처가 없어 이번에는 뒤로 뒀다.
- 가장 큰 가정은 기존 500자 제한이 의도된 계약이라는 점이다. `isWebLink` 및 같은 저장 라우트의 meeting_url 검증이 모두 500자로 일치하는 근거가 있다. 제한을 늘려야 한다는 별도 요구는 미확인이다.

실행 순서와 검토 지점 (구현 전, 모두 미착수)

1. 기존 단위 테스트에 500/501 경계와 빈 항목 뒤 두 번째 링크의 오류 인덱스 사례를 작성하고 위 특정 파일 테스트 명령으로 실패를 확인한다. 검토 지점: 현재 코드에서 501자가 통과하는 실패를 확인한 뒤 진행하며 사람 승인은 필요 없다.
2. parseLinks의 URL 정규화에만 자르지 않는 길이 판단을 적용한다. 예컨대 기존 text()를 충분한 상한으로 호출해 정규화한 뒤 길이를 검사할 수 있으며, 공용 text() 동작을 바꾸지 않는다. 같은 단위 테스트와 `npm run check`가 통과한 뒤 진행한다. 출력 URL에 URL.toString() 재직렬화를 추가하지 않는다.
3. e2e의 기존 pm/liveStudy/studyA를 재사용한다. 정상 500자 링크 저장→GET 상세 완전 일치→501자 링크와 다른 제목으로 PUT 시도→400 및 field_errors 확인→GET에서 이전 제목과 링크 유지 확인 순서로 검증한다. 기존 `saved = ... liveStudy()`로 기본 연구 내용을 복구하여 뒤쪽 시나리오에 긴 링크가 섞이지 않게 한다. `npm run build` 후 실서버 e2e로 검증한다. DB/핸들러를 가짜로 주입하거나 소스 문자열 검사를 성공 근거로 삼지 않는다.
4. 변경 파일과 lockfile 상태를 검사하고 실행한 명령·실패/미실행 원인을 회차 노트에 남긴다. 모든 자동 검증 결과가 인계 검토 지점이며 별도 사람 승인은 요구하지 않는다. 재현 전제가 달라지면 조용히 범위를 늘리지 말고 과제서를 정정한다.

검증 환경과 작업량 근거

- 정찰에서 node/npm 자체는 실행 가능했다(Node 22.23.1, npm 10.9.8). `npm run test`는 `vitest: not found`로 종료(127)했다. node_modules가 없으며 읽기 전용 역할이라 설치하지 않았다. 따라서 전체 테스트/빌드 통과를 주장하지 않는다.
- `vitest.config.ts`는 packages/**/*.test.ts와 apps/**/*.test.ts를 수집한다. `npm run check`는 lint→세 workspace typecheck→vitest이고 CI도 같은 명령을 쓴다.
- e2e는 실 Postgres 16(:55442), 개발 서버(:8789), 가짜 OAuth(:18999), .env.dev가 필요하다. CI에 설치·빌드·설정·마이그레이션·서버 시작 순서가 있다. `scripts/dev-reset.sh`는 로컬 DB를 삭제하므로 오직 폐기 가능한 테스트 DB에서만 사용하고 기존 개발 DB에 무작정 실행하지 않는다. 정찰에서는 DB/서버 가용성을 확인하지 않았다.
- 상향식 추정: 재현/단위 테스트 6~8분, 좁은 수정 4~6분, e2e 추가 7~10분, check/build/e2e/차이 검토 8~11분 = 기본 25~35분. 알려진 환경 준비 변동의 예비 시간 5~10분을 별도로 잡아 총 30~45분(정찰 판단 신뢰도 중간, 실측 확률 아님). DB 환경이 준비되어 있거나 기존 CI에서 검증 가능한 전제이며 신규 인프라 구축은 포함하지 않는다. 미지의 범위 확대를 위한 관리 예비는 이번 회차 0분이며 별도 후보로 남긴다. 지난 회차는 소규모 수정 성공 사례지만 시간 실측이 없어 유사 추정 수치로 사용하지 않았다.
- 적용 스킬: 로컬 `pmo:estimating-and-contingency`, `technology:implementation-planning`, `technology:solution-exploration`의 SKILL.md와 PMO sources 목록을 읽었다. 호출 가능한 Skill 도구는 제공되지 않았다. 위 산정은 저장소 관찰에 근거한 추정이며 외부 기준의 통계나 비용 수치를 인용하지 않는다.
