# 회차 노트 2026-10-16-144316-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:43] base pinned — main@879082e
- [러너 14:43] autonomy release — 

## 정찰 노트
- 보류 상위 3개(postsByIDsReactions / listReactions / decodeCappedBody)는 모두 "httpapi 핸들러가 장애를 정상 응답으로 접는" 같은 계열이고 10-04·10-05·10-08 세 회차가 이미 그 계열이라, 운영자 규칙(같은 접근 반복 금지)에 따라 서비스 계층 동작 결함인 북마크 `Reorder` 를 골랐다. 실제로 열어 본 것: bookmarks/service.go 전문, late.go:1234-1276, final.go:268-270, router.go:844, sidebar/service.go:570-591, sidebar/service_postgres_test.go:1-60/661-701, bookmark_scope_postgres_test.go:100-141, store/store.go:36-38.
- 확신 없는 곳(추측으로 적은 것): ① 실패 시나리오(A/B/C 고정표본의 어긋난 순서)는 코드에서 **연역**했을 뿐 테스트로 관찰하지 않았다 — 구현자가 RED 로 먼저 확정할 것. ② 인덱스 base 를 0-based 로 권고했으나 Mattermost 원본 스펙을 이 세션에서 직접 열어 확인하지는 못했다(docs/mattermost-api-compatibility.md 는 엔드포인트를 나열만 한다). `Create` 가 1-based 로 채번하는 비대칭은 실제 코드로 확인했다.
- 구현자가 조심할 것: 범위가 `Reorder` 한 함수 + 신규 테스트 1파일이다. 같은 파일의 `Patch`/`Create`, 같은 핸들러의 `ok, _ := IsMember` 403 위장을 끌어들이지 말 것. 테스트 DB 컨테이너 `moyro-pg-improve`(55433) 는 지금 `Exited (255)` 이므로 `docker start` 가 선행되어야 하고, DSN 없는 `go test` 의 빠른 ok 는 skip 이지 통과가 아니다.
- 프로필은 다시 쓰지 않았다: base 가 프로필 작성 시점과 같은 main@879082e 이고 내용이 현재 소스와 어긋나지 않았다(bookmarks/userstatus 무테스트 기술도 이번에 `[no test files]` 로 재확인).
- [러너 14:47] scout done — 채널 북마크 `Reorder` 가 주석이 약속한 dense 재배열을 하지 않고 한 행만 UPDATE 해, `sort_order` 가 중복되고 북
