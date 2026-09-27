# 회차 노트 2026-09-28-043236-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 04:32] base pinned — main@6d040b0
- [러너 04:32] autonomy release — 

## 정찰 노트
- `baseUrl` 길이 검증 누락을 골랐다: 이 저장소가 v1.2.25에서 이미 채택한 "요청 검증이 DB 컬럼 상한보다 느슨해 400 대신 500" 유형이고, 프로덕션 파일 1개(`providers.go`)에 끝나며 세 호출 지점(create·update·승인 실행)이 `validateProviderRequest` 한 곳을 공유해 쪼갤 필요가 없다.
- 제친 후보: `available_models` JSON 파싱 무시(5회차 미확인)는 네 곳을 직접 읽어 `jsonb NOT NULL DEFAULT '[]'` + 저장 전 `uniqueStrings` 정규화로 실패 경로가 없음을 확인하고 **기각**했다. 사용자 목록 roles 정렬은 `jsonb_agg(DISTINCT ...)`라 무작위가 아니고 결정적이어서 가치가 낮다. 비스트리밍 응답 크기 상한은 정상 대용량 응답을 끊는 동작 변경이라 위험을 3으로 올렸다.
- 추측으로 적은 것: 2001자 `baseUrl`이 PostgreSQL `22001`을 일으켜 실제로 500이 되는 것을 **DB로 재현하지 않았다**(코드·스키마 정적 확인만). 구현자는 red를 먼저 확보하고, 재현되지 않으면 차선(비스트리밍 truncation 감사 테스트)으로 넘어가라.
- 조심할 것: 웹 폼 `maxLength`를 같이 넣지 말 것(`internal/ui/dist` 재빌드가 따라온다), 마이그레이션의 `varchar(2000)`을 바꾸지 말 것, `PUT /api/v1/providers/{id}`는 `expectedUpdatedAt` 필수.
- 기각·차선이 반복된 유형(웹까지 번지는 대시보드/폼 변경, 캡처 44개 CI와 얽힌 합성 시드, 문서 한 줄 수정)은 이번에도 고르지 않았다.
- [러너 04:37] scout done — AI 공급자 `baseUrl`의 `varchar(2000)` DB 상한을 요청 검증 단계에서 400으로 막기 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- 무엇/왜: `validateProviderRequest`에 `utf8.RuneCountInString(request.BaseURL) > 2000` 검사 한 개를 추가했다(프로덕션 파일 1개). 2001자 `baseUrl`이 `base_url varchar(2000)`에 닿아 PostgreSQL 22001이 되고 500이 나가던 것을 400 `provider_invalid`로 바꾼다. 검증은 한 곳뿐이라 생성·수정·승인 실행 세 경로가 자동으로 같은 한도를 쓴다.
- 확신 없는 곳·검증 못 한 것: 승인 워크플로 실행 경로(`workflow.go:677`)는 같은 `validateProviderRequest`를 호출하는 것을 **읽어서만** 확인했고 테스트로 통과시키지는 않았다 — 2001자가 승인 요청 생성 단계(`createProvider`의 workflow 분기 전 검증)에서 이미 걸리므로 실행 단계까지 도달하는 시나리오를 만들지 못했다. `parsed.Host`에 비ASCII가 들어간 URL은 확인하지 않았다(한글은 path에만 넣었다).
- 일부러 하지 않은 것: 웹 공급자 폼의 `maxLength`(커밋된 `internal/ui/dist` 재빌드가 따라온다), 마이그레이션 `varchar(2000)` 변경, `normalizeProviderRequest`·`mergeProviderRequest` 수정(merge가 빈 `baseUrl`을 현재 값으로 채우는 동작에 기존 테스트가 의존한다), VERSION·CHANGELOG.
- 다음 역할이 조심할 것: `internal/server/provider_base_url_integration_test.go`는 `TEST_POSTGRES_DSN`이 없으면 조용히 SKIP한다. `DROP SCHEMA ai_admin/aiportal CASCADE`를 하므로 전용 폐기 DB만 쓰고 병렬 실행하지 말 것. 이번 회차는 `postgres:16-alpine`을 포트 55481로 띄워 검증했다(세션 종료 시 컨테이너 정리). 라우트는 `PATCH /api/v1/ai/providers/{id}`이며 `expectedUpdatedAt`이 필수다(과제서에 적힌 `PUT /api/v1/providers/{id}`는 실제 라우트가 아니다).
- [러너 04:44] brief accepted — 채택 — 결함·코드 위치·`varchar(2000)`·재사용 헬퍼가 모두 현재 코드와 일치했고 지정된 방식 그대로 red→green으로 증명�
- [러너 04:45] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: 검사 3줄을 임시 제거해 red를 직접 재현했고(500 provider_create_failed / provider_update_failed / 단위 FAIL — 원장의 `- 실패 재현:` 줄과 일치) 복구 후 새 테스트 5개 서브테스트가 전용 postgres(포트 55489, 리뷰 후 제거)에서 PASS(SKIP 아님). 워킹트리는 `git checkout`으로 되돌려 깨끗하다.
- 구현자가 의심한 승인 실행 경로(workflow.go:678)는 결함이 아니었다: createProvider:107·updateProvider:185가 workflowRequired 분기(:114·:200) **앞에서** 검증하므로 상한 초과 값이 실행 단계에 도달할 수 없다 — 도달 불가 경로라 테스트가 없는 것이 정상.
- 경계도 확인: 2000자 ASCII·2000 rune 한글 저장 성공, 2001자 400. PostgreSQL varchar(n)은 코드포인트를 세므로 BMP 밖 문자에서도 rune 기준이 맞다. 후행 `/` trim이 먼저 일어나 2001자가 2000자로 저장되는 것은 저장값 기준이라 올바르다.
- 못 본 것: `-race` 전체와 다른 패키지 통합 테스트는 돌리지 않았다(웹 변경 0건이라 npm도 생략). 돌린 것 — go build·go vet·gofmt·make lint(verify-version 1.2.29)·DSN 있는 internal/server 전체(PASS 99 / SKIP 1 = Keycloak 없음).
- 승인이어도 남는 것: ① 웹 공급자 폼 baseUrl maxLength가 없다(dist 재빌드 회피로 의도적 제외, 후속 과제). ② 릴리즈 노트에 "2001자 baseUrl이 500 대신 400 provider_invalid"를 관찰 가능한 계약 변화로 적으라(docs/api.md:246 이미 반영, VERSION·CHANGELOG는 미변경). ③ ai_provider의 varchar 4개가 모두 검증에 덮였으니 유형 ③ 후속은 다른 테이블에서 찾으라.
- [러너 04:49] review approved — 리뷰 승인 (risk=low)
- [러너 04:49] pr created — https://github.com/hkjang/ai-admin/pull/34
- [러너 04:57] ci passed — 검사 2개 모두 success
- [러너 04:57] merge done — ce6ad31
