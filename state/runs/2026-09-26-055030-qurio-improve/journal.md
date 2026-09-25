# 회차 노트 2026-09-26-055030-qurio-improve — qurio
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 05:50] base pinned — main@1ad4e50
- [러너 05:50] autonomy release — 

## 정찰 노트
- 522f0e5 가 고친 `defer pool.Close()` → `t.Cleanup(DELETE…)` 순서 역전이 저장소에 아직 52곳 남아 있어, 보호 경로를 뺀 19곳(runtimeapi 11 / intelligenceapi 7 / platformapi 1)을 이어서 고르는 것이 가장 안전하고 확실하다 — 결함·수정·검증 방법이 이미 한 번 머지로 증명됐다. store DSN 통일(차선)은 관례가 셋으로 갈리는 설계 판단이 필요해 뒤로 미뤘고, Keycloak logout·OAuth 호출 한도는 인증 보호 경로라 자율 회차 범위를 넘는다.
- 추측으로 적은 것: 행 번호는 grep 으로만 확인했으니 편집 전에 재확인할 것. `database_drafts:43`·`agent_instructions:31` 두 곳은 `t.Cleanup(func…)` 이 0개라 지금 잃는 정리가 없다(관례 통일 목적으로만 포함). 잔존 카운트 쿼리의 테이블명은 각 파일 DELETE 문에서 읽었지만 실제 실행은 이 회차에서 못 했다(DB 미기동).
- 구현자가 조심할 것: `t.Cleanup(pool.Close)` 를 **원래 `defer` 가 있던 자리 그대로** 등록해야 LIFO 로 마지막에 돈다. 범위 밖 파일(agentapi·httpapi·legacyapi·store·jobs·backup·integrations·domain/legacy, `intelligenceapi/repository_test.go`)은 diff 에 넣지 말 것. `_, _ =` 를 `t.Errorf` 로 바꾸지 말 것. grep 결과를 증거로 제출하지 말고 실제 PostgreSQL 에서 red→green→되돌려 red 를 보일 것.
- 기각·상태 갱신: `internal/mail` 이 main 에 없음을 `ls internal/` 로 확인해 mail 전제 아이디어를 rejected 로 내렸다. gofmt/make lint(a432f3f), store 4곳 순서 역전(522f0e5)은 done. PR #18 딥링크 접근은 rejected 유지.
- 프로필은 기준 커밋·버전이 1.4.2→1.4.5 로 어긋나 새로 썼다(profile.md).
- [러너 05:55] scout done — `defer pool.Close()` → `t.Cleanup(pool.Close)` 순서 역전을 runtimeapi·intelligenceapi·platformapi 통합 테스트로 확장 (가�

## 구현 노트
- 무엇/왜: 과제서의 19개 `defer pool.Close()` 를 `t.Cleanup(pool.Close)` 로 등록 위치 그대로 치환(a8f0775, diff 19+/19-, 전부 동일한 한 줄). Go 가 테스트 본문 defer 를 t.Cleanup 보다 먼저 돌려 픽스처 DELETE 가 닫힌 풀 위에서 `_, _ =` 로 조용히 죽던 것을 복구.
- 확신 없는 곳: **없음에 가깝지만 하나만** — 수용 기준 3 "잔존 0" 은 달성 못 했고 기준 1 과 배타적이라 달성 불가다. 남은 13행/회차는 전부 *DELETE 를 등록조차 안 하는* 테스트 소유임을 행 단위로 특정했다(users 20·21+provider/secret/ai_usage = 범위 밖 intelligenceapi/repository_test.go, users 22·23+`agent-instruction-key` = 픽스처 cleanup 0개인 agent_instructions, user 26+`publish-*` = users/api_keys 를 안 지우는 publication). 2회 연속 실행의 증가분이 정확히 일정한 것이 "고친 19개의 누적 기여 0" 의 근거다. 비평가는 여기부터 보면 된다.
- 일부러 안 한 것: 위 세 테스트에 DELETE 추가(기준 1 의 "diff 19줄·검증 로직 무변경" 위반), `_, _ =` → `t.Logf`(과제서가 별도 과제로 못박음), 보호 경로 33곳(agentapi/httpapi/legacyapi/store 등), 범위 밖 `intelligenceapi/repository_test.go:51`.
- 다음 역할 주의: 이 테스트들은 실제 PostgreSQL 이 있어야 돈다 — **없으면 Skip 하고도 `ok` 로 통과한다**(이번에 `export A=x B="$A"` 연쇄 전개 실수로 env 가 비어 전 패키지가 조용히 Skip 되는 것을 실제로 겪었다; env 는 반드시 줄을 나눠 export 하고 Skip 여부를 `-v` 로 확인할 것). 검증은 폐기 DB bootstrap(`go test -tags=integration ./cmd/qurio -run '^TestIntegrationDatabaseMigrations$'`) → 패키지 실행 → psql 잔존 카운트 순. `-p=1` 유지 필수(dbexec 고정명 픽스처). 이번엔 포트 55432/55433/55439 가 점유 중이라 55442 를 썼다; 컨테이너는 제거했다.
- [러너 06:02] brief accepted — 채택 — 19개 지점의 파일·행 번호가 코드와 정확히 일치했고 수용 기준 1·2·4·5 를 실제 PostgreSQL 로 red→green→되돌려 red
- [러너 06:05] verify passed — 검증 9개 통과 (auto)
- [러너 06:05] pr created — https://github.com/hkjang/qurio/pull/22
- [러너 06:05] guard held — internal/intelligenceapi/credential_race_integration_test.go 
