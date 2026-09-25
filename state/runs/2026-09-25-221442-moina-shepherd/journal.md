# PR 처리기 노트 2026-09-25-221442-moina-shepherd — moina PR #31
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-25-192055-moina-improve)
# 회차 노트 2026-09-25-192055-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:21] base pinned — main@3ba1767
- [러너 19:21] autonomy release — 

## 정찰 노트
- `followTopic`을 골랐다. `getMoim`·`getTopic`도 같은 404 흡수지만 SELECT 실패는 평 PostgreSQL에서 재현할 수 없어(2026-09-24에 부딪힌 벽) 증명 불가 분기가 남는다. `followTopic`은 쓰기라 `moim_join` 테스트에서 검증된 BEFORE INSERT 트리거를 그대로 쓸 수 있고, 같은 파일 `unfollowTopic`이 이미 500이라 근거도 파일 안에 있다. Moim 계약을 두 회차 연속 흔들었으니 이번은 Topic 경로로 옮겼다.
- 확신 없는 것(과제서에 "미확인"으로 표시함): `ON CONFLICT … DO UPDATE`가 붙은 INSERT에서 BEFORE INSERT 트리거가 신규 행에 대해 발화하는지. red 단계에서 직접 확인하고, 안 되면 `BEFORE INSERT OR UPDATE`로 바꿀 것.
- 확인한 것: httpapi 테스트에 `followTopic` 호출 0건, `user_topic_follows`는 `001_initial.sql:149` PK(user_id,topic_id), `openapi.yaml:391` post는 `responses`에 `'200'`만 있고 `description` 없음, 프런트는 `DiscoveryPages.tsx:16,69`에서 `readableError`로 서버 message를 그대로 토스트에 띄움(코드 변경 불필요).
- 구현자 주의: 404 문구 "Topic을 찾을 수 없습니다"는 `getTopic`과 공유하므로 바꾸지 말 것. 테스트 트리거는 공유 테이블에 걸리므로 `t.Cleanup`에서 반드시 DROP. `--- SKIP` 0줄을 확인해야 integration이 실제로 돈 증거가 된다.
- 프로필은 다시 쓰지 않았다(1일 전, 내용 일치). 다만 "기준" 줄만 낡았다 — 지금 base는 main@3ba1767 / VERSION v0.1.37이고 최신 migration은 여전히 `013_retention_indexes.sql`이다.
- [러너 19:23] scout done — `followTopic`이 DB 저장 오류를 404 `not_found`로 감추는 것을 500 `storage_error`로 분리 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- `followTopic`(social.go:282)의 `err != nil || RowsAffected()==0`을 두 분기로 쪼갰다 — Exec 오류는 500 `storage_error`("Topic을 Link할 수 없습니다", `unfollowTopic` 어휘 재사용), 0행은 기존 404·기존 문구 그대로. OpenAPI는 `description` 한 줄만 추가(responses 무변경, route 120 유지).
- 확신 없는 곳·검증 못 한 것: 없음에 가깝지만 굳이 적자면 (1) 500 메시지 "Topic을 Link할 수 없습니다"는 내가 고른 조어가 아니라 정찰이 제안한 것이고 사용자 향 문구라 어휘 취향은 검토 여지가 있다. (2) frontend vitest·e2e·`make lint`의 frontend eslint 구간은 실행하지 않았다 — 프런트 코드 무변경이고 `DiscoveryPages.tsx`는 `readableError`로 서버 message를 그대로 띄우기만 해서다. (3) `make test`(Go 비-race + frontend 전체)는 돌리지 않고 `go test -race`로 대신했다.
- 일부러 하지 않은 것: `getTopic`·`getMoim`의 같은 형태 404 흡수는 손대지 않았다(평 PostgreSQL에서 SELECT만 실패시킬 수단이 없어 500 분기를 실제 DB로 증명할 수 없음). `unfollowTopic`·`joinMoim`·`leaveMoim`·weight 기본값 50·범위 검사도 과제 범위 밖이라 무변경.
- 다음 역할이 조심할 것: 새 테스트 `topic_follow_postgres_integration_test.go`는 `MOINA_TEST_POSTGRES_DSN` 없으면 조용히 SKIP된다 — 검증 증거로 쓰려면 throwaway `postgres:17-alpine` DSN을 주고 `--- SKIP` 0줄을 확인할 것. 테스트는 공유 `user_topic_follows`에 트리거를 만들었다 지우므로 `t.Cleanup`이 도는 것이 전제다(이번 실행 후 pg_trigger·pg_proc·잔여 행 모두 0 확인함).
- red 증거: 수정 전 코드에서 저장 오류 1케이스만 `404 {not_found Topic을 찾을 수 없습니다}`로 실패, 나머지 4케이스(신규 200 / 가중치 변경 200 / 400 invalid_weight / 없는 slug 404) 통과. 수정 후 5/5 PASS.
- [러너 19:27] brief accepted — 채택 — 과제서의 근거(social.go:282의 통합 404, 같은 파일 unfollowTopic의 500, `user_topic_follows` PK·CHECK, openapi.yaml:391 post에 descrip
- [러너 19:27] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- approve / low: Topic 오류 분기·실제 라우팅/인가·SQL·OpenAPI·프런트 오류 표시를 확인했고 머지 차단 결함 없음.
- 임시 PostgreSQL 17에서 새 테스트 -race 5/5 PASS, SKIP 0; 트리거·함수·사용자·Topic 잔여 0, make check·diff --check 통과.
- 로컬 main=95255f3은 낡았고 고정 base/origin/main=3ba1767 대비 실제 신규 변경은 3파일; 이전 인증·추적 주요 경로도 읽었으나 77파일 전체 전수 검증은 아님.
- 수정 전 재실행·전체 backend/frontend/e2e·실제 외부 연동은 미실행; 릴리즈 시 고정 base를 유지할 것. 보안·법무 차단 없음.
- [러너 19:29] review approved — 리뷰 승인 (risk=low)
- [러너 19:29] pr created — https://github.com/hkjang/moina/pull/31
- [러너 19:37] ci failed — 성공이 아닌 검사: Image and browser smoke=failure
