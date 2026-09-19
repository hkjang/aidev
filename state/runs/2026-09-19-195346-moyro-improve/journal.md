# 회차 노트 2026-09-19-195346-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:53] base pinned — main@1e1462d
- [러너 19:53] autonomy release — 

## 정찰 노트
- 선택 이유: 메일 후속(automation_failed 등)·preferences·reminders·presence 후보는 모두 미머지 auto/* 브랜치에 막혀 있음(diff --stat 로 확인). sidebar 는 2026-09-17 회차가 merge 돼 자유롭고, Update 의 무검증 저장·ErrNoRows→400 은 코드에서 직접 확인한 결함이라 S 로 끝남.
- 확신 없는 곳: Mattermost 가 `sorting: ""` 를 IsValid 통과시키는지, 웹앱이 알 수 없는 sorting 값을 어떻게 폴백하는지 — 미확인, 과제서에 "빈 값=유지" 가정으로 적음.
- 구현자 주의: replaceChannelsTx/UpdateOrder 는 직전 회차 산물이니 손대지 말 것; 통합 테스트는 DSN 없이는 조용히 skip 되므로 postgres 컨테이너를 반드시 띄울 것; 오류 ID 문자열·응답 모양 유지.
- [러너 19:57] scout done — 사이드바 카테고리 Update/Delete 가 검증 없이 저장하고 없는 행을 400 으로 내던 것 정리 (가치 3 / 위험 1 / 작

## 구현 노트
- 과제서 그대로 채택. `sidebar.Update` 가 sorting 을 {"",alpha,recent,manual} 로 검증하고, `SELECT type … FOR UPDATE` 로 행을 먼저 잠근 뒤 custom 인데 이름이 공백이면 `ErrInvalid`, 행이 없으면 `ErrNotFound` 를 내며 UPDATE 는 `display_name = CASE WHEN type='custom' …`, `sorting = COALESCE(NULLIF($3,''), sorting)` 으로 기본 타입 이름·빈 sorting 을 보존한다. `Create`/`Delete` 오류는 `ErrInvalid` 로, `Get`/`Delete` 의 ErrNoRows 는 `ErrNotFound` 로 바꿨고, 핸들러 4곳은 새 `writeSidebarError`(404/400/그 외 500) 하나를 쓴다. 오류 ID 문자열·응답 모양 그대로.
- 확신 없는 곳: Mattermost 가 `sorting: ""` 를 유효로 취급한다는 주석(`SidebarCategorySortDefault = ""` 기억에 근거, 소스로 재확인 못 함). 또 `updateSidebarCategoriesBulk` 는 여전히 카테고리마다 별도 tx 라 두 번째 항목이 404/400 이면 첫 항목의 변경이 남은 채 오류가 나간다 — 과제서가 범위 밖으로 정한 것이라 손대지 않음.
- 일부러 하지 않은 것: `getSidebarCategory` 는 과제서대로 무조건 404 유지(DB 장애도 404 — ideas.json 에 후속으로 기록). 기본 타입 카테고리에 빈 display_name 을 보내도 200 (이름을 무시하므로 거부할 이유가 없음; Mattermost 도 비 custom 은 이름을 덮어쓰기만 함). 핸들러 쪽은 실제 HTTP 라우터 통합 테스트 대신 `writeSidebarError` 순수 표 테스트 1건.
- 검증: `go vet ./...`, docker postgres:16-alpine(포트 55432 가 다른 컨테이너에 잡혀 있어 55433 사용) 으로 `MOYRO_TEST_POSTGRES_DSN` 주고 `go test -race -p 1 ./...` (아래 결과), `bash scripts/check-source-sizes.sh` 통과. 새 테스트 2건은 수정 전 코드에서 실제로 실패(빈 이름 err=nil / "no rows in result set")했고, sorting 검증·CASE·404 분기를 각각 무력화하면 해당 테스트만 깨지는 것까지 확인.
- 다음 역할 주의: `service_postgres_test.go` 의 새 테스트는 DSN 이 없으면 조용히 skip 되므로 반드시 DB 를 띄워서 돌릴 것. `replaceChannelsTx`/`UpdateOrder` 는 손대지 않았다(호출만 그대로). 웹 변경 없음.
- [러너 20:05] brief accepted — 채택 — 과제서의 근거(무검증 UPDATE·ErrNoRows→400·Get 은 404)가 코드와 정확히 일치했고 수용 기준 1~5 를 모두 충족했다.
- [러너 20:05] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인: diff 4 파일 전부 읽음. Update 의 sorting allowlist·FOR UPDATE·CASE/COALESCE, Delete/Get 의 ErrNoRows→ErrNotFound, writeSidebarError 404/400/500 분기가 커밋 메시지와 일치. uid/team 스코프 유지, 없는 행과 남의 행이 같은 ErrNotFound 라 열거 불가 — 보안·법무 차단 사유 없음.
- 테스트 실증: postgres:16-alpine(55434) 로 HEAD 통과, main 의 service.go 로 바꾸면 새 테스트 2건이 각각 err=<nil> / "no rows in result set" 로 실패 — 구현 노트의 주장 재현됨. 웹앱은 전체 객체를 PUT 하므로 custom 빈 이름 400 신설로 회귀 없음.
- 못 본 것: 실제 HTTP 라우터를 통한 통합 테스트는 없음(표 테스트만). Mattermost 소스로 `sorting:""` 유효성은 재확인하지 않았으나 빈 값=유지라 클라이언트에 해가 없음.
- 릴리즈 노트: 없는/남의 카테고리 Update·Delete 400→404, 저장소 장애 400→500, 잘못된 sorting·custom 빈 이름 400, 기본 카테고리 이름은 무시(200).
- 다음 회차: bulk 업데이트의 항목별 tx(부분 적용 후 오류), getSidebarCategory 의 DB 장애=404, 500 본문에 DB 오류 문자열 노출 관례 — 모두 기존 결함, 이번 범위 밖.
- [러너 20:07] review approved — 리뷰 승인 (risk=low)
- [러너 20:07] pr created — https://github.com/hkjang/moyro/pull/18
- [러너 20:19] ci passed — 검사 3개 모두 success
- [러너 20:19] merge done — 0665ae3
