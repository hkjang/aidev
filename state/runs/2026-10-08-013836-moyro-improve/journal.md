# 회차 노트 2026-10-08-013836-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 01:38] base pinned — main@2adff5d
- [러너 01:38] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 1순위(postsByIDsReactions)는 09-28~10-05 다섯 회차가 연속으로 한 "httpapi 상태코드 재분류" 계열이어서 운영자 규칙상 피했다. 대신 `rows.Next()` 사용 프로덕션 파일 43개 중 `rows.Err()` 가 없는 파일을 grep 으로 추려 **정확히 1개**(compat_wave_handlers_final.go)를 찾았고, 그 안의 searchFiles/searchTeamFiles 두 함수가 결함이다 — 새 오류 id 도 새 상태 코드도 필요 없고(같은 함수가 이미 500 을 갖고 있다) 저장소의 자기 관례(files/service.go:328 등 42파일)가 근거가 된다.
- 제친 후보: metrics(223줄) 정독 — 결함 없음(ideas.json 에 rejected + 이유). postsByIDsReactions — 과제서 차선으로 남겼다. decodeCappedBody 계열 — 10-04·10-05 연속이라 쉬어야 한다.
- 추측으로 적은 것(구현자 확인 필요): **장애 주입 방법이 미확인이다.** 순회 중간 오류를 pgx v5 가 스트리밍으로 올리는지(rows.Err() 가 실제로 채워지는지) 확인하지 않았다. 과제서에 대안으로 Scan 실패 경로(수용 기준 2)로 RED 를 만드는 길을 적어 뒀다 — 실제로 RED 가 보이는 쪽을 쓸 것. DB 컨테이너(moyro-pg-improve/55433)도 이번 세션에 기동하지 않았다.
- 조심할 것: SQL 문·LIMIT 50·가시성 조건(channel_members 서브쿼리)을 건드리면 인가 변경이 된다. 같은 두 함수의 `_ = decodeCappedBody` 와 감사 details 의 terms 는 범위 밖으로 명시했다. 500 은 LogAsync 앞에서 끝나야 한다.
- 프로필은 9일 전(2026-09-29) 것이고 이번에 읽은 범위에서는 어긋난 곳이 없었다 — 예산이 다 되어 새로 쓰지 않았다(줄번호 갱신분은 ideas.json 에 남겼다).
- [러너 01:43] scout done — 파일 검색 두 핸들러가 행 순회 중 DB 장애를 200 `{"order":[],"file_infos":{}}` ("검색 결과 없음") 으로 보고하는 �

## 구현 노트
- 무엇을: `searchFiles`/`searchTeamFiles`(compat_wave_handlers_final.go) 루프 뒤에 `rows.Err()` 검사를 넣고 `rows.Scan` 오류의 `continue` 를 같은 500 반환으로 교체. 새 오류 id·새 상태 코드 없고 SQL·가시성 조건·`decodeCappedBody`·감사 details 는 미변경. 프로덕션 1파일 2함수.
- **확신 없는 곳**: Scan 분기(수용 기준 2)는 **테스트로 단독 관찰되지 않는다**. pgx 가 Scan 실패를 Rows 에 fatal 로 표시하므로, 그 분기만 `continue` 로 되돌려도 내 테스트는 그대로 통과한다(`rows.Err()` 가 한 줄 뒤에서 잡는다) — 역방향 확인에서 직접 봤고 테스트·프로덕션 주석에 적어 뒀다. 명시적 `return` 을 남긴 이유는 드라이버의 fatal 정책을 빌리지 않고 핸들러가 계약을 소유하게 하려는 것이며, 동작은 오늘 기준 동일하다. 비평가는 여기를 먼저 볼 것.
- 또 하나: 순회-중간 장애 주입이 `ORDER BY` 때문에 첫 행을 받기 **전에** 터진다(sort 가 입력을 모두 소비). 그래도 `Pool.Query` 는 성공으로 돌아오고 오류는 `rows.Err()` 에 얹히므로 결함 모양은 동일하다 — "행을 몇 개 받다가 끊기는" 진짜 연결 단절은 재현하지 않았다.
- 일부러 안 한 것: 과제서가 범위 밖으로 지정한 `_ = decodeCappedBody`(:765/:821) 와 감사 details 의 `terms`(:802/:860), 차선 후보 `postsByIDsReactions`. `gofmt -l internal/httpapi` 의 기존 9파일도 손대지 않았다(변경·신규 2파일만 clean 확인).
- 다음 역할이 조심할 것: **새 테스트는 DB 가 있어야 돈다**. `MOYRO_TEST_POSTGRES_DSN` 없으면 skip 되고 httpapi 가 0.17s 에 `ok` 를 내므로 통과로 오인 금지(DB 포함 시 60.8s). 컨테이너 `moyro-pg-improve`/호스트 55433 사용(`docker start` 로 기동). 테스트는 격리 스키마 안에서 `file_infos` 를 VIEW 로 바꿨다 되돌리므로 중간에 죽으면 그 스키마만 오염되고 다른 테스트에는 영향 없다.
- [러너 01:55] brief accepted — 채택 — 근거가 현재 코드와 정확히 일치했다(`:793-800`/`:851-858` 의 `rows.Err()` 부재, `:796`/`:854` 의 `continue`, 같은 함수가 이�
