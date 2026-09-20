# 회차 노트 2026-09-20-130409-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 13:04] base pinned — main@efb5096
- [러너 13:04] autonomy release — 

## 정찰 노트
- 선택 이유: 사이드바 bulk PUT 의 부분 적용은 지난 회차가 남긴 후속이고, 같은 파일·같은 sentinel/writeSidebarError 패턴을 그대로 쓰므로 위험이 가장 낮으면서 실제 동작(전부/전무)이 바뀐다. getPreferenceByName·post_reminders 는 옛 auto 브랜치가 원격에서 사라졌는데 그 PR 이 반려였는지 확인하지 못해(gh 조회 미실행) 재시도 금지 규칙에 걸릴 수 있어 제쳤고, 메일 후속은 mail 패키지가 아직 main 에 없다.
- 확신 없는 곳: `sidebarRow` 에 update_at 필드가 있는지, `check-source-sizes.sh` 의 줄 수 상한에 service.go(594줄) 가 닿는지, 테스트 DSN 의 사용자/비밀번호 형식(이전 회차 기록 기준) — 모두 과제서에 "미확인" 으로 적음.
- 구현자 주의: `Get` 을 tx 안에서 부르면 미커밋 상태를 못 보므로 커밋 뒤에 되읽을 것; 중복 id·잠금 순서 정리는 하지 말 것(효과 없는 변경); 새 테스트 (a)/(b) 가 수정 전 코드에서 실제로 실패하는 것을 한 번 확인할 것.
- [러너 13:07] scout done — 사이드바 카테고리 일괄 PUT(`updateSidebarCategoriesBulk`)을 단일 트랜잭션으로 — 두 번째 항목이 거부되면 첫 �

## 구현 노트
- 무엇/왜: `sidebar.Update` 본문을 `updateTx(ctx, tx, …)` 로 뽑고 새 `UpdateMany` 가 bulk PUT 배열 전체를 한 tx 로 처리(빈 배열은 tx 미개시, 커밋 뒤 `Get` 으로 입력 순서대로 되읽기). 핸들러 `updateSidebarCategoriesBulk` 의 루프를 `UpdateMany` 한 번으로 교체해 "두 번째 항목 거부 → 첫 항목만 커밋" 부분 적용을 없앴다. 차선 후보(`getSidebarCategory` 를 `writeSidebarError` 로 404/500 분리)도 같은 커밋에 넣었다.
- 확신 없는 곳·검증 못 한 것: 실제 브라우저 드래그-드롭(Mattermost 웹앱/moyro 웹앱)으로의 왕복은 돌리지 않았다 — 응답 JSON 모양은 이전과 같은 `[]sidebar.Category` 이고 핸들러 통합 테스트로 200/404/400 과 본문을 확인했을 뿐이다. 같은 배열 안에 같은 id 가 두 번 오면 한 tx 안에서 두 번째 `SELECT … FOR UPDATE` 가 자기 tx 의 잠금을 재획득하므로 막히지 않는 것은 핸들러 테스트의 유효 배열 케이스(같은 custom id 두 번)로 실제 확인했다.
- 일부러 하지 않은 것: 중복 id dedupe·잠금 순서 정렬(과제서가 금지, 효과 없는 변경), `replaceChannelsTx`/`UpdateOrder`/`Create`/`Delete`/`Get` SQL, `sidebar/service.go` 의 pre-existing gofmt import 순서(main 에 이미 있고 이번 diff 와 무관), `updateSidebarCategoryOrder` 가 클라이언트 배열을 그대로 돌려주는 문제(아이디어로 남김).
- 다음 역할이 조심할 것: 새 테스트 4건은 전부 `MOYRO_TEST_POSTGRES_DSN` 이 있어야 돌고 없으면 조용히 skip 된다(로컬은 `docker run -d --name moyro-test-pg -e POSTGRES_PASSWORD=postgres -p 55433:5432 postgres:16-alpine`, `-p 1` 필수). httpapi 쪽 테스트는 기존 `newOperationsTestDB` 헬퍼를 재사용하고 스키마를 격리하므로 `DROP TABLE sidebar_categories` 가 다른 테스트에 영향을 주지 않는다. `Get` 을 tx 안에서 부르면 미커밋 상태를 못 보므로 `UpdateMany` 의 되읽기는 반드시 Commit 뒤에 있어야 한다.
- [러너 13:16] brief accepted — 채택 — 과제서의 근거(항목마다 별도 tx·커밋 뒤 오류 응답·이벤트 미발송)가 코드와 정확히 일치했고 수용 기준 1~5 를 
- [러너 13:16] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인함: 로컬 postgres:16-alpine 을 띄워 DSN 을 주고 새 테스트 4건 + `internal/sidebar`·`internal/httpapi` 전체를 `-race -p 1` 로 실행해 통과; go vet 통과. 테스트는 첫 항목의 커밋된 rename 을 `before==after` 로 잡으므로 수정 전 코드에서 실제로 실패하는 구조. 인가 경로(`requireUserParamAccess`)·SQL 의 user_id/team_id 스코프는 건드리지 않았음.
- 못 봄: 실제 브라우저 드래그-드롭 왕복은 구현자와 마찬가지로 돌리지 않음(응답 shape 동일하므로 저위험).
- 승인이어도 남는 우려: 한 tx 안에서 클라이언트 순서대로 `FOR UPDATE` 를 잡으므로 같은 사용자의 동시 bulk PUT(탭 2개) 이 순서가 다르면 데드락 → 한쪽이 500. 이전 항목별 tx 에는 없던 실패 모드이며 다른 사용자에게는 영향 없음. 과제서가 잠금 순서 정렬을 금지했으니 다음 회차 아이디어로.
- 500 이 `api.sidebar.get.not_found` id 와 DB 오류 문자열을 그대로 내보내는 것은 main 의 `writeSidebarError` 관례 그대로(신규 아님). gofmt import 순서 어긋남도 main 에 이미 있음.
- 판정: approve, risk low, blocking 없음.
- [러너 13:19] review approved — 리뷰 승인 (risk=low)
- [러너 13:19] pr created — https://github.com/hkjang/moyro/pull/19
- [러너 13:30] ci passed — 검사 3개 모두 success
- [러너 13:30] merge done — ad4c266
- [러너 13:37] release published — v0.2.34
- [러너 13:55] assets verified — v0.2.34 자산 1개 (이전 v0.2.33: 1)
