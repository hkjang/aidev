# 회차 노트 2026-09-28-031232-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:12] base pinned — main@22549f9
- [러너 03:12] autonomy release — 

## 정찰 노트
- 골랐다: `updateMyWebhook` 의 DB 실패를 404 대신 500 으로. 같은 파일 `createMyWebhook` 이 이미 500 이라 비대칭이 코드 안에서 자명하고, 지난 3회차가 모두 이 형태(저장 실패 오분류)로 채택·릴리즈됐다.
- 제친 것: `updateAdminUser`(계정 상태·세션 폐기에 붙어 위험 한 단계 위 → 차선), `deleteCoupon`·`deleteMyWebhook` 단독(HTTP 입력으로 DB 오류를 만들 경로가 없어 증거 불가), 프런트 과제 3건(dist 재빌드·브라우저 논점), 26곳 일괄 수정(파일 10개 초과).
- 추측으로 적은 것: `webhooks.name` 의 NUL 이 SQLSTATE 22021 을 낸다는 것 — 2026-09-27 회차의 `reports.details`(같은 text) 실측에서 유추했을 뿐 이번에 실행하지 않았다. `docker version` 은 승인 거부로 막혀 docker 가용성도 재확인 못 했다.
- 조심할 것: `validateWebhookInput` 에 NUL 거부를 추가하면 증거 입력이 400 으로 막혀 과제 자체가 증명 불가가 된다. `webhooks` 에는 UNIQUE 가 없으니 `isUniqueViolation` 23505 분기를 만들지 말 것(도달 불가 코드).
- [러너 03:18] scout done — 웹훅 수정 저장의 DB 실패를 404 "웹훅을 찾을 수 없습니다." 가 아니라 500 으로 알리기 (가치 3 / 위험 1 / 작�

## 구현 노트
- 무엇을 왜: `updateMyWebhook` 의 `err != nil || tag.RowsAffected() == 0` 을 두 갈래로 갈라 저장 실패를 500 `webhook_save_failed` "웹훅을 저장하지 못했습니다."(`createMyWebhook` 문구와 동일)로, 행 없음만 404 `webhook_not_found` 로 남겼다. 같은 파일 `deleteMyWebhook` 의 동일 패턴도 함께 갈랐다(500 `webhook_delete_failed`). 프로덕션 파일 1개.
- 확신 없는 곳: `deleteMyWebhook` 의 새 500 갈래는 **도달을 증명하지 못했다** — 파라미터가 uuid 두 개뿐이라 HTTP 입력으로 DB 오류를 만들 수 없다. 회귀만 확인했다(없는 id → 404, 정상 → 204). `webhook_delete_failed` 는 이 파일에 짝이 없어 새로 만든 코드다(`webhook_save_failed` 는 `create_failed` 와 문구를 맞췄지만 코드 이름은 과제서 지정대로 `webhook_save_failed`).
- 확인한 것: 실패 경로의 오류가 `*pgconn.PgError{Code:"22021", Where:"unnamed portal parameter $3"}` 이고 행이 존재하는데도 `RowsAffected()==0` 이라는 것(= 문장 미실행)을 임시 프로브로 눈으로 확인한 뒤 프로브를 제거했다. 프로브 흔적(`fmt` import 포함) 없음 — `gofmt -l cmd internal` 무출력, `go vet` 무결.
- 일부러 하지 않은 것: `validateWebhookInput` 에 NUL 거부를 넣지 않았다(넣으면 증거 입력이 400 으로 막힌다). 남은 24곳의 같은 패턴도 손대지 않았다. `web/`·`internal/ui/dist` 미변경 — 프런트는 `error.message` 를 그대로 띄운다.
- 다음 역할이 조심할 것: 새 테스트 `TestIntegrationWebhookUpdateSeparatesSaveFailureFromNotFound`(integration_test.go 끝)는 **KKIIT_TEST_DSN 없이는 SKIP** 된다 — DSN 없는 통과는 검증이 아니다. `integrationServer` 는 전역 `apiUnderTest` 를 쓰므로 병렬 실행 금지. 검증은 버릴 PostgreSQL 16(docker 29.7.2), DB DROP/CREATE 후 전체 82.9초.
- [러너 03:24] brief accepted — 채택 — docker(29.7.2)가 가용해 수용 기준 1~3 을 모두 실제 HTTP→실제 DB 로 증명했고, 과제서가 미확인으로 남긴 두 가지(`we
- [러너 03:24] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 확인: diff 3파일 전부를 읽고 webhooks.go 의 두 갈래(err→500, RowsAffected==0→404)가 createMyWebhook 과 대칭인지, 404 유지가 소유자 격리를 깨지 않는지(`owner_id=$2` 로 오라클 없음), gofmt·vet·대상 테스트 컴파일, openapi_test 통과, 프런트가 `error.message` 만 쓰는지를 대조했다.
- 못 봤음: DSN 이 없어 통합 테스트를 직접 실행하지 못했다 — 원장의 실패 재현 출력(자기 웹훅 PUT status=404 want=500)과 양방향 변이 기록을 증거로 받아들였다.
- 거절 사유는 하나: `docs/openapi.yaml:272-276` delete 블록에 새 500(과 기존 404)이 없다. 수리가 먼저 볼 파일은 docs/openapi.yaml 뿐이고 코드는 그대로 두면 된다.
- 남는 우려: integration_test.go:5427-5433 의 "이름이 안 바뀌었다" 검사는 성공 저장 뒤에 놓여 항상 참 — 다음 회차가 이 단언을 믿지 말 것. deleteMyWebhook 의 500 은 무테스트(도달 불가).
- 릴리즈 노트: 새 오류 코드 `webhook_save_failed`·`webhook_delete_failed` 는 API 소비자에게 보이는 변경이다.
- [러너 03:27] review rejected — 리뷰 거절: docs/openapi.yaml:272-276 이번 변경이 새로 만든 응답 코드를 문서에 빠뜨렸다. 커밋 메시지는 "웹훅 수정·삭제" 둘을 다룬다고 말하고 webhooks.go:2

## 수리 노트
- 지적은 맞았다: webhooks.go:202 의 새 500 `webhook_delete_failed` 도, 기존 404 도 openapi.yaml delete 블록에 없었다(코드를 직접 읽어 확인).
- 고친 방법: `docs/openapi.yaml` delete.responses 에 '404'·'500' 두 줄만 추가(0d219d1). 코드·테스트는 손대지 않았다.
- 검증: YAML 파싱으로 delete.responses = 204/404/500 확인, gofmt 무출력, go vet 통과, `go test ./cmd/... ./internal/...` 전부 ok(TestOpenAPIDocumentsEveryRoute 포함).
- 확신 없는 곳: DELETE 도 invalid_id 400 을 낼 수 있지만 모든 `{id}` 경로에 공통인 기존 누락이라 범위 밖으로 뒀다(PUT 만 이번에 400 이 생겨 둘이 비대칭).
- 여전히 미해소: DSN 이 없어 통합은 SKIP 이고, deleteMyWebhook 의 500 은 무테스트라는 앞 회차 우려도 그대로다(문서 수리로는 해결 불가).
- [러너 03:28] repair done — # 수리 요약 (시도 1 거절 대응)  - 문제: 26ed72f 가 `deleteMyWebhook`(webhooks.go:202)에 500 `webhook_delete_failed` 를 새로 만들었는데 docs/openapi.yaml 의 delete 블록은 '204

## 비평 노트
- 확인: diff 3파일 전부와 webhooks.go 의 create/update/delete 전문을 읽었다. err→500 / RowsAffected()==0→404 순서가 NUL 케이스(둘 다 참)에서 유일하게 맞는 순서임을, 404 가 '없음·남의 것' 을 계속 함께 덮어 존재 오라클이 없음을(owner_id=$2), 오류 본문에 DB 메시지 유출이 없음을 대조했다. gofmt·vet 무결, TestOpenAPIDocumentsEveryRoute 통과, openapi.yaml 파싱으로 PUT=200/400/404/500·DELETE=204/404/500 확인. 수리(0d219d1)가 앞 회차 거절 사유를 정확히 닫았다.
- 못 봤음: DSN 이 없어 새 통합 테스트를 직접 실행하지 못했다(SKIP). 원장의 실패 재현 `status=404 want=500`(요청자 본인 소유 웹훅)과 양방향 변이 기록을 증거로 받아들였다 — 증상과 수정이 일치한다.
- 승인이어도 남는 우려: integration_test.go:5427-5433 의 '실패한 저장이 이름을 바꿨습니다' 검사는 성공 저장 뒤라 항상 참이다(실패-무영향의 증거가 아님). deleteMyWebhook 의 500 은 HTTP 도달 불가로 여전히 무테스트. DELETE 는 400 invalid_id 를 낼 수 있는데 문서에 없어 PUT 과 비대칭(모든 {id} 경로 공통 누락, 다음 회차 후보).
- 릴리즈 노트: 새 오류 코드 `webhook_save_failed`·`webhook_delete_failed` 와 두 경로의 500 은 API 소비자에게 보이는 변경. 프런트·dist 미변경.
- 차단 없음(security·legal 소견 없음): 인가 경로·비밀값·개인정보 처리 변화 없음, 권한 확대 없음.
- [러너 03:31] review approved — 리뷰 승인 (risk=low)
- [러너 03:31] pr created — https://github.com/hkjang/Kkiit/pull/14
- [러너 03:31] ci passed — 검사 없음 — 정책으로 허용
- [러너 03:31] merge done — 0d219d1
- [러너 03:40] release published — v0.4.9
- [러너 03:41] assets verified — v0.4.9 자산 1개 (이전 v0.4.8: 1)
