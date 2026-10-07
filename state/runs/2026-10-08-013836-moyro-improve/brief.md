# 과제서 2026-10-08 — moyro (base main@2adff5d)

- 과제: 파일 검색 두 핸들러가 행 순회 중 DB 장애를 200 `{"order":[],"file_infos":{}}` ("검색 결과 없음") 으로 보고하는 것 차단 — `rows.Err()` 누락 (가치 3 / 위험 1 / 작업량 S)

- 왜: `server/internal/httpapi/compat_wave_handlers_final.go` 의 `searchFiles`(:756)·`searchTeamFiles`(:811) 가 `for rows.Next()` 루프를 돌고 **`rows.Err()` 를 한 번도 확인하지 않는다**(:793-800, :851-858). pgx 에서 순회 중 연결이 끊기거나 서버가 오류를 올리면 `rows.Next()` 가 조용히 `false` 를 돌려주므로, 두 핸들러는 **부분 결과 또는 빈 결과를 200 으로** 내보낸다 — 사용자는 "그 PDF 는 없다" 로 읽고, 같은 함수가 쿼리 **시작** 실패에 대해 이미 갖고 있는 500(`api.file.search.app_error` / `api.file.search.team.app_error`)은 도달하지 못한다. 같은 결함이 `rows.Scan` 오류의 `continue`(:796, :854)에도 있다 — 한 행이 깨지면 그 파일만 결과에서 조용히 사라진다. 이것은 이 저장소의 자기 관례를 깨는 유일한 지점이다: `rows.Next()` 를 쓰는 프로덕션 파일 43개 중 **42개가 `rows.Err()` 를 확인하고**(예: `internal/files/service.go:328`·`:359`, `internal/reactions/service.go:62` 가 모두 `return out, rows.Err()`), 이 파일 하나만 파일 전체에 `rows.Err()` 가 없다(grep 으로 확인).

- 수용 기준:
  1) 행 순회 중 장애가 나면 두 핸들러가 각각 **기존** 500 id (`api.file.search.app_error` / `api.file.search.team.app_error`) 로 응답한다 — **새 오류 id·새 상태 코드 없음**.
  2) `rows.Scan` 실패도 조용한 행 누락이 아니라 같은 500 으로 보고된다(`continue` 제거).
  3) 정상 경로의 응답이 **바이트 동일**하다: 일치 파일이 있을 때의 `{"order":[...],"file_infos":{...}}`, 그리고 `terms` 가 빈 문자열일 때의 조기 반환 200 `{"order":[],"file_infos":{}}`(:767-770, :823-826) 도 그대로.
  4) **진짜 0건 검색은 여전히 200 + 빈 결과**다 — 이것이 장애와 구별되어야 하는 핵심 대조다.
  5) 거부된 요청은 `audit.ActionFileSearch` 감사행을 남기지 않는다 — 500 반환이 `h.audit.LogAsync`(:801-803, :859-861) **앞**에서 끝나야 한다(2026-09-27 업로드 스텁 회차가 정한 "거부는 감사보다 앞" 순서).
  6) 테스트가 같은 실행 안에서 장애 하의 500 과 정상·0건 하의 200 을 함께 증명한다.

- 건드릴 파일:
  - `server/internal/httpapi/compat_wave_handlers_final.go:searchFiles` — 루프 뒤에 `if err := rows.Err(); err != nil { writeError(w, 500, "api.file.search.app_error", err.Error()); return }` 추가, `:796` 의 `continue` 를 같은 500 반환으로 교체.
  - `server/internal/httpapi/compat_wave_handlers_final.go:searchTeamFiles` — 동일. id 는 `api.file.search.team.app_error`.
  - 신규 테스트 1파일 (예: `server/internal/httpapi/file_search_rows_fault_postgres_test.go`).
  - **프로덕션 1파일 2함수.** 그 이상 넓히지 말 것.

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - `cd server && go test -race -p 1 -count=1 ./internal/httpapi/` — 실제 DB 가 필요하다. `MOYRO_TEST_POSTGRES_DSN` 없으면 DB 테스트가 **skip** 되고 0.17s 에 `ok` 가 나오므로 **통과로 오인 금지**. 과거 회차 기록상 컨테이너 `moyro-pg-improve` / 호스트 포트 `55433` (이번 세션 미기동 — 먼저 `docker start moyro-pg-improve`).
  - `cd server && go build ./... && go vet ./...`
  - `cd server && gofmt -l internal/httpapi` — **기존에 9개 파일이 이미 출력된다**(교훈 기록). 변경·신규 파일만 clean 한지 보고, 독립 포맷 수정은 하지 말 것.
  - `bash scripts/check-source-sizes.sh` — `compat_wave_handlers_final.go` 는 상한 77000 바이트에 대해 약 73.8k 였다(2026-10-05 기록). 몇 줄 추가는 안전하지만 실행해 확인할 것.
  - 전체: `go test -race -p 1 -count=1 ./...` (DSN 포함, 수 분).

- 테스트 배선 (선례를 그대로 쓸 것):
  - 실제 PostgreSQL 격리 스키마 `newOperationsTestDB` + `store.Migrate`, 실제 `auth`·`audit` 서비스. **`h.audit` 를 nil 로 두면 `LogAsync` 가 아예 불리지 않아 감사 단언이 거짓 통과한다**(2026-10-05 회차가 걸린 함정) — 실제 서비스를 주입할 것.
  - 라우팅은 `router.go:970-971` 의 `r.Post("/files/search", h.searchFiles)` / `r.Post("/teams/{teamID}/files/search", h.searchTeamFiles)` 패턴을 글자 그대로 복사.
  - 장애 주입 (**미확인 — 구현자가 실제로 확인할 것**): 순회 **중간** 오류를 결정론적으로 만드는 후보는 격리 스키마 안에서 `file_infos` 를 같은 열 목록의 **뷰**로 바꿔 특정 행에서만 평가 시 터지는 식(예: `1/(CASE WHEN name='boom' THEN 0 ELSE 1 END)`)을 넣는 것이다. pgx v5 가 이 쿼리를 스트리밍하는지(그래야 `rows.Next()==false` + `rows.Err()!=nil` 가 된다) 버퍼링하는지 나는 확인하지 않았다. 이것이 안 되면 **수용 기준 2(Scan 오류)로 RED 를 만드는 쪽이 확실하다** — 격리 스키마에서 `file_infos.size` 를 Scan 불가한 타입으로 바꾸면(예: `ALTER ... TYPE text USING 'not-a-number'`) `rows.Scan` 이 실패하고 현재 코드는 그 행을 조용히 버린 200 을 낸다. 둘 중 **실제로 RED 가 관찰되는 쪽**을 쓰고, 어느 쪽을 썼는지 회차 노트에 적을 것.
  - 수정 전 RED 를 **먼저 관찰**하고, 수정 후 GREEN, 그리고 추가한 `rows.Err()` 분기를 되돌리면 그 서브테스트만 다시 깨지는 것을 **역방향으로** 확인할 것(이 저장소의 모든 성공 회차가 한 절차).

- 위험과 피할 것:
  - **SQL 문·`LIMIT 50`·`store.LikeContains`·가시성 조건(`channel_members` 서브쿼리, `c.team_id`, `delete_at=0`)을 한 글자도 바꾸지 말 것.** 이번 과제는 오류 처리만이다. 가시성 조건을 건드리면 인가 변경이 된다.
  - **`_ = decodeCappedBody`(:765, :821) 를 같이 고치지 말 것.** 10-04·10-05 두 회차가 연속으로 `decodeCappedBody` 계열이었고 보류 목록도 "당분간 쉬는 것이 맞다" 로 적고 있다. 별도 회차 감.
  - **`audit` details 로 `terms`(사용자 검색 문자열)가 넘어가는 것(:802, :860) 도 범위 밖.** 운영자 규칙("스캔 대상 문자열을 details 로 넘기지 말 것")과 닿아 보이지만 이것은 사용자 자신의 검색어이고, 고치면 감사 스키마 변경이 되어 과제가 커진다. 원장에만 적을 것.
  - `request_body.go`·`router.go`·마이그레이션·`auth`/`session` 은 건드리지 않는다. 보호 경로 미접촉.
  - 이번 과제는 **400/403/404 를 500 으로 재분류하는 계열이 아니다**(09-28~10-05 다섯 회차가 그 계열이어서 운영자 규칙상 반복 금지). 여기서는 **이미 존재하는 500 경로가 누락된 오류 검사 때문에 도달 불가**인 것을 고치는 것이고, 새 id·새 코드가 없다.
  - `gofmt -l internal/httpapi` 의 기존 9파일 출력에 놀라지 말 것. webapp TS2345 `useDraft.test.tsx` 오류도 자기 변경 탓이 아니다(교훈 기록) — 이번 과제는 webapp 무관.

- 차선 후보: `postsByIDsReactions`(`compat_wave_handlers_late.go:304`) 의 세 `continue` 중 `ListForPost` 장애만 500 `api.reaction.list.app_error`(형제 `listReactions` 가 쓰는 기존 id) 로 분리 — 이번 회차에 코드로 재확인했다(`ch, err := ...; if err != nil || ch == ""` / `ok, _ := IsMember` / `list, err := ...; if err != nil { continue }`). 없는 글(`pgx.ErrNoRows`) 과 비회원 필터는 주석이 선언한 계약이므로 보존해야 한다. 단 이쪽은 상태코드 재분류 계열에 더 가까워 2순위로 둔다.
