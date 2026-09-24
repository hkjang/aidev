# 회차 노트 2026-09-24-005437-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:54] base pinned — main@c5b50b4
- [러너 00:54] autonomy release — 

## 정찰 노트
- 목록 search 이스케이프를 골랐다: 사용자에게 실제로 보이는 오답(`hong_gildong`이 `hong1gildong`까지 매치, `_` 한 글자면 전체 행)이고 store 4함수 + 테스트로 끝나는 S 과제다. 차선의 page_size 문서 불일치는 문서/동작 중 어느 쪽이 정본인지 확정이 필요해 뒤로 뒀고, 순수 헬퍼 테스트는 막는 장애가 없어 제쳤다.
- `requestList` code 보존 아이디어는 기각했다 — web/src에서 ApiError.code를 읽는 소비자가 없어 동작이 바뀌지 않는다(운영자 규칙에 정면으로 걸림).
- 추측으로 적은 것: `ESCAPE '\'` 절이 PostgreSQL 기본 이스케이프 문자와 같아 동작 무관하다는 판단은 문서 기억에 기댄 것이고 이번에 DB로 확인하지 못했다. 도커 실행 권한을 못 받아 통합 테스트 실제 기동도 미확인이다.
- 구현자가 조심할 것: `searchPattern`의 TrimSpace 때문에 `$n=''` 빈 판정 값과 pattern 값이 갈라지면 공백 검색이 "전체 매치"로 뒤집힌다. 그리고 4곳 중 일부만 고치면 화면마다 검색이 달라지므로 넷 다, count 쿼리까지 함께 고칠 것.
- [러너 00:58] scout done — 목록 API의 `search` 질의가 `%`·`_`·`\`를 와일드카드로 흘려보내는 것을 전역 검색과 같은 규칙(`searchPattern`)�

## 구현 노트
- internal/store의 목록 4곳(users.go:ListLocalUsers, hubs.go:ListManagedUsersWithAccess·ListServersWithAccess, settings.go:ListResources)이 검색어를 `"%"+search+"%"`로 붙여 써 `_`·`%`가 와일드카드로 샜다. 네 곳 모두 `search = strings.TrimSpace(search)` 뒤 기존 `searchPattern`을 재사용하게 바꿔(새 헬퍼 없음) 빈 판정 값과 pattern을 같은 문자열로 통일했고, ILIKE에 `ESCAPE '\'`를 네 곳 일관되게 붙였다. 인자 개수·번호와 접근제어 SQL은 손대지 않았다.
- 확신 없는 곳: 검색어 앞뒤 공백이 이제 잘린다(`" hong "` → `hong`). 전역 검색과 같은 규칙이고 프런트는 입력을 그대로 보내지만, 앞뒤 공백을 일부러 검색하던 사용자는 결과가 달라진다. 프런트(web/)·가이드·OpenAPI는 무변경이며 실제 브라우저 화면으로는 확인하지 않았다(Go 레벨 통합 테스트로만 검증).
- 일부러 하지 않은 것: `search` 길이 제한·제어문자 정규화(같은 자리를 두 번 흔들지 않으려고 보류로 남김), 문서/PDF 수정(검색 메타문자 설명이 없음), 전역 검색 동작 변경.
- 다음 역할이 조심할 것: `internal/store/list_search_integration_test.go`는 `JUPIQ_INTEGRATION_TEST_DSN` 없으면 skip된다 — DSN 없는 `go test ./...`는 이 수정의 증거가 아니다. 이 테스트는 공유 DB에 running 서버 행을 넣으므로 `t.Cleanup`에서 지운다(풀을 닫는 `t.Cleanup(database.Close)`를 먼저 등록해 삭제가 먼저 돌게 한 것이 중요; `defer database.Close()`로 두면 삭제가 닫힌 풀에서 조용히 실패해 store_integration_test.go의 LiveSnapshot 단언이 깨진다).
- 검증: postgres:16-alpine 컨테이너에서 `make test-integration` 통과(store 4.6s / api 0.65s, store 통합 SKIP 0건), `gofmt -l .` 무출력, `go vet ./...`, `go test -count=1 ./...` 통과. 이스케이프만 되돌리면 통합 테스트 네 서브테스트가 다시 실패하는 것을 확인했다. 컨테이너는 제거했다.
- [러너 01:04] brief accepted — 채택 — 과제서의 진단(네 목록이 원문을 붙여 쓰고 count/data가 pattern을 공유)이 코드와 정확히 일치했고 수용 기준 5개를 
- [러너 01:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(low). 확인: 4개 목록 함수의 count/data 쿼리 모두 searchPattern+ESCAPE로 통일됐고 AccessPredicate 인자 번호(4/6, 5/7)는 무변경, ILIKE 전수 grep에 ESCAPE 누락 0건, 호출 지점 5곳이 모두 수정 경로를 지난다. go build·vet·`go test ./internal/store/` 통과.
- 못 본 것: DB가 없어 통합 테스트를 직접 돌리지 못했다(구현자의 postgres:16 실행 보고와 코드 논리로만 재확인). 프런트·브라우저 화면 미확인.
- 남는 우려 ①: 검색어 앞뒤 공백이 이제 잘리고 공백만 입력하면 전체 목록이 나온다 — 전역 검색과 같은 규칙이지만 커밋 메시지·가이드에 없다. 릴리즈 노트에 한 줄 적을 것.
- 남는 우려 ②: search_test.go의 새 단위 테스트는 수정 전에도 통과한다(검증이 아니라 회귀 고정). 실증은 list_search_integration_test.go 뿐이고 DSN 없으면 skip이다.
- 다음 회차 참고: 같은 테스트의 ListLocalUsers 빈검색 단언은 첫 100행 가정에 기대 공유 DB에서 흔들릴 수 있다. `ESCAPE '\'`는 standard_conforming_strings=on 의존(적용 지점이 1→8곳으로 늘었다).
- [러너 01:07] review approved — 리뷰 승인 (risk=low)
- [러너 01:07] pr created — https://github.com/hkjang/jupiq/pull/23
- [러너 01:10] ci passed — 검사 3개 모두 success
- [러너 01:10] merge done — d826d63
- [러너 01:16] release published — v1.7.3
- [러너 01:19] assets verified — v1.7.3 자산 1개 (이전 v1.7.2: 1)
