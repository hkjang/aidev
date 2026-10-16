# 수리 요약

- 비평 거절은 전부 맞았다. `legacy-admins.menu_info` 가 구멍이었다: `normalizeLegacyAdminMenuInfo` 의 `json.Marshal` 이 실제 NUL 을 6글자 `\u0000` escape 으로 바꿔 `storableInVarchar`(:447)를 통과시키고, `$2::jsonb`/`$4::jsonb` 에서 PostgreSQL 22P05(폐기 컨테이너로 재확인)로 깨져 500 `resource_update_failed` 가 났다.
- 재현: 새 유닛 테스트 `TestValidateLegacyAdminMenuInfoRejectsNUL` 5건 중 4건 red(배열·escape 문자열 양쪽에서 `err=nil`, 값에 `\u0000` 잔존), 새 HTTP 서브테스트 `legacyMenuInfoRejectsNULInNestedJSONB` 는 `status=500 want=400 resource_update_failed` 로 red.
- 고친 것: `normalizeLegacyAdminMenuInfo` 가 `menuId`(TrimSpace 전 원문)·`menuNm` 을 `storableInVarchar` 로 검사하고 sentinel `errLegacyMenuInfoNUL` 을 반환, 호출부 2곳은 `legacyMenuInfoError` 로 NUL 메시지만 통과시킨다(JSON 형식 오류 메시지는 그대로). 틀렸던 주석 `:443-446` 과 `docs/api.md:371` 문장은 jsonb escape 경로를 명시하도록 고쳤다.
- 검증: `TEST_POSTGRES_DSN=…(전용 postgres:16-alpine, 포트 55723, 사후 제거) go test -race -count=1 ./...` 전체 통과(internal/server 146s, SKIP 은 TEST_KEYCLOAK_ISSUER 없는 OIDC E2E 1건뿐), `make lint`·`go build ./...`·`git diff --check` 통과. 역검증: 프로덕션 변경만 임시로 빼면 위 두 테스트가 다시 red.
- 남는 불확실: 승인 경로(`workflow.enabled` 켠 jsonb payload)와 DELETE 문자열 rowID 는 이번에도 HTTP 로 실행하지 않았다 — 같은 `validateLegacyMutation` 앞단에서 막히지만 미실측이다. 22P05 핸들러 분기는 추가하지 않았다(이 경로로는 더 이상 도달 불가, 범위 밖).
