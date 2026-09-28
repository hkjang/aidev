# 회차 노트 2026-09-28-092233-jupiq-improve — jupiq
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 09:22] base pinned — main@944e82b
- [러너 09:22] autonomy release — 

## 정찰 노트
- 고른 이유: 최근 4회 채택된 유형(문서화된 계약 vs 코드가 조용히 다른 값을 쓰는 자리)과 같은 결이고 프로덕션 파일 2개로 끝난다. page_size 상한 건은 공용 pageBounds를 엔드포인트별로 쪼개야 해 제쳤고, 발송 기록 Total 건은 프런트가 total·status를 쓰지 않는 것을 확인해 rejected로 내렸다.
- 추측으로 적은 것: 해결 방식(`consumption_group_by` 키 추가)은 내 판단이다 — `consumption`을 빼는 대안도 있으나 읽는 화면이 없어 키 추가가 더 안전하다고 봤다. `ConsumptionGroupBy` 함수명도 제안이다.
- 미확인: 통합 테스트를 실제로 돌려 보지 않았다(DSN 없이 `go test ./internal/store ./internal/api` 통과만 확인, 통합 skip). `internal/auth/` 목록은 이번에 다시 보지 않았다.
- 구현자 주의: project 소비량을 "지원"하려 들면 migrations/013(보호 경로)과 롤업 수집기까지 번진다. `-v`로 SKIP 0건을 확인하지 않은 통합 실행은 증거가 아니다.
- 프로필은 다시 쓰지 않았다(2026-09-27판이 구조·관례 그대로 유효, 차이는 VERSION 1.8.3→1.8.4·base 944e82b뿐).
- [러너 09:28] scout done — `/usage`가 같은 `group_by`를 두 경로에 넘기면서 `consumption`만 조용히 `user` 그룹으로 떨어지는 것을 응답에 드

## 구현 노트
- 왜: `/usage?group_by=project`는 trend를 project로 묶으면서 `consumption`은 롤업에 project 열이 없어 말없이 user로 되돌렸다. 응답이 그 사실을 말하게 `consumption_group_by`를 넣었다(프로덕션 2파일: `store/resource_usage.go`의 `consumptionColumns`+`ConsumptionGroupBy`, `api/core_handlers.go:usage` 한 줄).
- 확신 없는 곳: 키 이름 `consumption_group_by`는 내 선택이며 소비자가 없다(프런트는 `consumption`을 읽지 않는다) — 이름을 바꾸려면 지금이 비용이 가장 싸다. openapi.yaml은 `/usage` 200이 `$ref: Success` 자유 형식이라 손대지 않았고, 따라서 이 키는 문서에 없다.
- 검증 못 한 것: 프런트·브라우저 E2E 미실행(변경 없음), `make release-check` 미실행(네트워크·수 분), npm 쪽 전부 미실행.
- 일부러 안 한 것: 롤업에 project 열 추가(마이그레이션·수집기까지 번짐), `/usage/consumption`의 enum 확장이나 400 좁히기(계약 유지), 프런트 변경.
- 다음 역할 주의: `internal/api/usage_group_by_integration_test.go`는 `JUPIQ_INTEGRATION_TEST_DSN` 없으면 skip된다 — DSN 없는 `go test ./...`는 증거가 아니다. 테스트는 `metric_samples`·`resource_usage_hourly`에 직접 INSERT하고 defer로 지운다(username 표식으로 격리).
- 되돌림 확인 두 갈래(핸들러 한 줄 제거 / `ConsumptionGroupBy`를 `return groupBy`로)에서 각각 빨개지는 것을 실제로 봤다.
- [러너 09:33] brief accepted — 채택 — 과제서의 진단(settings.go:1033이 project를 알고 resource_usage.go:220-229는 모른다)이 코드와 정확히 일치했고 수용 기준 5
- [러너 09:34] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 실제 postgres:16 컨테이너를 띄워 DSN을 걸고 돌렸다 — 새 통합 테스트 4개 서브테스트 PASS(skip 0), `-run Integration ./internal/store ./internal/api` ok, `go test ./...` 전 패키지 ok, build·vet 클린. 컨테이너 삭제·워킹트리 clean 확인.
- 원장에 `실패 재현:` 줄이 없어 직접 red를 봤다: 핸들러 한 줄 제거 → `consumption_group_by = <nil>`, `ConsumptionGroupBy`를 `return groupBy`로 → `= "project", want "user"`. 테스트는 바뀐 경로를 실제로 지난다. 진단(settings.go:1033은 project를 알고 consumptionColumns는 모름)도 코드와 일치.
- 승인. 단 진술이 과장됐다 — resource_usage.go:277의 각 행이 이미 `group_by`를 담아, 새 키의 실익은 빈 목록·편의에 한정된다(결함 아님). security·legal 차단 소견 없음: 열은 여전히 표에서 고르고, 새 개인정보·의존성·마이그레이션이 없어 revert로 온전히 돌아온다.
- 릴리즈 노트/다음 회차: `consumption_group_by`는 openapi·가이드에 없는 비문서 키이고 소비자가 0 — 이름 변경·철회 비용이 지금 가장 싸다. openapi.yaml:185가 `/usage` enum에 project를 문서화하는 근본 모순은 그대로 남았다.
- 못 본 것: 프런트 npm 테스트·브라우저 E2E, `make release-check`(네트워크). 프런트는 코드로만 확인했다(ConsumptionPanel.tsx:10에 project 없음, utils/usage.ts는 trend·top_users만 읽음).
- [러너 09:46] review approved — 리뷰 승인 (risk=low)
- [러너 09:46] pr created — https://github.com/hkjang/jupiq/pull/27
- [러너 09:50] ci passed — 검사 3개 모두 success
- [러너 09:50] merge done — 92d05c7
- [러너 10:01] release published — v1.8.5
- [러너 10:04] assets verified — v1.8.5 자산 1개 (이전 v1.8.4: 1)
