## 2026-10-07
- 선택: 일·월 단위 쓰기 제한의 Retry-After 가 KST 경계를 9시간 넘겨 잡히는 것 고치기 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `quota.ts:retryAfterSeconds` 가 다음 창 전환 시각을 `windowStart()`(KST 날짜의 **UTC** 자정 라벨)로 계산해 `day`·`month` 제한에 걸린 사람에게 항상 9시간을 더 기다리라고 알려 주고 있었다. 전환 시각을 `kstDayStart(at)+1일` / `Date.UTC(kstY, kstM, 1) - 9h` 로 바로 계산하도록 바꿨고(`windowStart()` 는 DB 유일 키라 본문 한 줄도 건드리지 않았다), 테스트에서 직접 부를 수 있게 `retryAfterSeconds`·`windowStart` 에 `export` 를 붙였다. `rules.test.ts` 에 `describe('write limits')` 4개를 추가해 day 2건·month 2건(10→11월, 12→1월) 고정값, `minute` 분기 불변, `windowStart()` 의 ISO 문자열 4개를 못 박았다. `npm run check`(lint+typecheck+vitest 44개) 와 `npm run build` 통과. e2e 는 실 Postgres 가 필요해 돌리지 못했지만 grep 으로 `retry-after` 값을 단정하는 e2e 가 없음을 확인했다.
- 실패 재현: `AssertionError: expected 115200 to be 82800` (day, `2026-10-07T16:00:00Z`) / `AssertionError: expected 2620800 to be 2588400` (month, `2026-10-31T16:00:00Z`) — 같은 실행에서 `minute` 과 `windowStart` 테스트 2개는 이미 통과(회귀 가드).
- 보류 아이디어: util/text.ts:isSafeLink() 가 호출처 0인 죽은 코드이고 주석의 사설망 차단 약속을 지키지 않는다 / util/time.ts 에 단위 테스트가 전혀 없다(kstWindowOn·parseClock 47시간 클램프·backoffMs 의 "캡 전에 지터") / domain/catalog.ts 와 maskEmail·mailboxKey 의 엣지케이스 테스트 공백 / quota.ts:usage() 의 "one round trip" 주석이 실제(Promise.all 로 쿼리 n개)와 어긋난다 / rules.test.ts 한 파일에 단위 테스트가 전부 모여 있다(테스트가 세 배로 늘기 전에는 손대지 말 것).
- 과제서: 채택 — 과제서의 근거·수용 기준·손으로 한 검산이 지금 코드와 전부 일치했고 수정 범위도 프로덕션 파일 1개로 끝났다.

- 릴리즈: v0.1.1 (2026-10-07, run 2026-10-07-175808-insight.yeopjari.bid-improve)
## 2026-10-08
- 선택: 과제 링크가 500자를 넘으면 잘라 저장하지 말고 입력 오류로 거절 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: parseLinks의 검증 전 500자 절단이 긴 URL을 다른 주소로 저장하는 원인이었다. 기존 text 정규화를 절단 없이 적용하고 500자 초과이면 원래 인덱스의 validation_failed 오류를 수집하도록 프로덕션 1개 파일만 수정했으며, 오류에는 URL 원문을 넣지 않았다. 단위 경계·정규화 회귀 및 실제 PUT→GET e2e로 500자 완전 보존, 501자 HTTP 400/field_errors, 실패 시 제목·링크와 연구 전체 조회 상태 보존을 검증했다.
- 실패 재현: `AssertionError: expected function to throw an error, but it didn't` (501/700자); `Tests  3 failed | 29 passed (32)` — 수정 전 실제 공개 parseLinks 테스트, 수정만 되돌려도 동일 실패 재확인.
- 보류 아이디어: util/text.ts:isSafeLink() 죽은 코드·주석 정리 (가치 3 / 위험 2 / S)
- 보류 아이디어: maskEmail·mailboxKey 경계 테스트 (가치 2 / 위험 1 / S; catalog와 별도 범위)
- 보류 아이디어: quota usage one round trip 주석 확인 (가치 2 / 위험 2 / S; 측정 없는 성능 변경 금지)
- 보류 아이디어: judgeSubmission URL 답변 절단 방지 (가치 3 / 위험 2 / S; 이번 파서와 통합하지 않고 독립 재현부터)
- 과제서: 채택 — 기준 코드에서도 절단 결함이 재현되었고 지정 3개 파일과 수용 기준 그대로 구현했다.
- 검증: `npm ci --no-audit --fund=false` 성공(기존 eslint deprecated 경고, 의존성 변경 없음); `npm run test -- packages/core/src/__tests__/rules.test.ts` 수정 전 3 실패→수정 후 32 통과; 수정 되돌림 3 실패→복구 후 `npm run check` 린트·3 workspace 타입 검사·5 파일/63 테스트 통과; `npm run build` 성공; e2e 테스트 비교 수정 후 최종 `npm run check`도 63 통과.
- 실서버 검증: `BASE=http://localhost:18789 DATABASE_URL=postgres://postgres@127.0.0.1:55482/insight APP_DATABASE_URL=postgres://insight_app@127.0.0.1:55482/insight node scripts/e2e.mjs` → `176 passed, 0 failed`. 별도 폐기용 Postgres 16 컨테이너, 4개 정식 migration, 정식 빌드 서버/irumx_gateway→insight_app 역할 전환, 기존 가짜 OAuth :18999를 사용했다. .env.example을 프로세스 메모리로 읽어 포트만 분리했으며 .env 파일 작성 없이 실행했다. 기존 개발 DB/서버는 변경하지 않았다.
- 시행착오: 테스트 DB 준비 중 잘못 추정한 insight_gateway 역할명이 없어 SQL 실패; 소스의 실제 irumx_gateway를 확인해 폐기용 DB에서 바로잡고 migration을 재실행했다. 첫 e2e는 175 통과/1 실패: 직접 저장→조회로 URL 길이 500·원문 일치·라벨 일치를 확인했고, JSONB 객체 키 순서(url,label)와 입력 순서(label,url)가 원인이었다. 문자열 비교를 isDeepStrictEqual로 고쳐 값·구조 전체 비교를 유지하고 176개 모두 재실행해 통과했다.
- 범위/정리: 변경 3개(프로덕션 1개), lockfile 무변경, 빌드 산출물은 기존 .gitignore로 제외. 검증용 서버·컨테이너 종료, e2e 기존 운영자 지정 테스트가 만든 임시 credential fixture 2개 삭제. `git diff --check` 통과 및 `git status --short`로 지정 파일만 확인. UI 스크린샷·운영 배포 검증은 범위 밖이라 미실행.
- 스킬: Skill 호출 도구가 없어 /home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/ 아래 completion-verification, systematic-debugging, test-driven-development의 SKILL.md를 직접 읽고 실패→최소 수정→성공→수정 되돌림 실패→최종 검증 절차를 적용했다.
- 커밋: `fde293d` — 과제 링크: 500자 초과 주소를 자르지 않고 입력 오류로 거절 (작성자 hkjang, 트레일러 없음). 커밋 뒤 git diff --check 통과, git status --short 출력 없음.

- 릴리즈: v0.1.4 (2026-10-08, run 2026-10-08-133829-insight.yeopjari.bid-improve)
