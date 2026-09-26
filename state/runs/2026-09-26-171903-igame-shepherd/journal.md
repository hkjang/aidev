# PR 처리기 노트 2026-09-26-171903-igame-shepherd — igame PR #28
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-26-161600-igame-improve)
# 회차 노트 2026-09-26-161600-igame-improve — igame
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:16] base pinned — main@aca2fe8
- [러너 16:16] autonomy release — 

## 정찰 노트
- 골랐다: 관리자 설정 PUT 의 `{"value": null}` 차단(S). serviceLocation(호출처 6곳, 파일 상한 초과)과 API fixture 격리(범위 넓음)를 제쳤고, metadata:null 은 `client_info` 를 읽는 코드가 아예 없어(grep 확인) 가치가 낮아 차선으로 내렸다. 설정 null 은 play_policy·privacy 가 조용히 기본값으로 되돌아가는 경로라 값이 더 크다.
- 확인한 것: putSetting(admin.go:106-157) 의 검사 순서, validateSetting 7개 키 전부 오브젝트, oidc·ai 는 래퍼 없이 타입 구조체로 받고 재마셜하므로 안전(=통합 금지), admin_pg_test.go 헬퍼 3개 이름.
- 추측으로 남긴 것: (1) `{"value": null}` 이 실제로 jsonb `null` 로 **저장**되는지 — pgx/DB 단계 실측 미완, 수용 기준 0) 으로 구현자에게 먼저 실측하게 했다. (2) `json.Unmarshal("null", &struct)` 가 no-op 이라는 것은 encoding/json 문서 기준이며 샌드박스가 `go run` 을 막아 이번 회차엔 실행하지 못했다.
- 구현자가 조심할 것: `system_settings` 는 키가 전역이고 API PG fixture 가 기본 스키마를 공유하므로 원래 값을 `t.Cleanup` 으로 복원할 것. `scripts/smoke-settings.sh` 의 왕복(조회→PUT)이 릴리즈 게이트이므로 오브젝트 강제가 그 왕복을 깨지 않는지 읽어서 확인할 것. 에러 코드는 기존 `invalid_setting` 재사용.
- [러너 16:24] scout done — 관리자 설정 PUT 이 `{"value": null}` 을 받아들여 설정을 조용히 초기화하는 것을 막기 (가치 3 / 위험 1 / 작업�

## 구현 노트
- 무엇을 왜: `putSetting`(internal/api/admin.go) 의 `json.Valid` 직후에 "value 는 JSON 오브젝트" 검사 한 줄을 넣었다. `{"value": null}` 이 실제로 200 을 받고 `system_settings.value` 가 jsonb `null` 로 바뀌는 것을 수정 전 실제 PG 로 확인했다(Red 메시지에 행 상태 포함). 배열·숫자·문자열·불리언은 이미 키별 `validateSetting` 에서 400 이었고 `null` 만 새는 구멍이었다 — 그 계약도 테스트로 고정했다. 에러 코드는 기존 `invalid_setting` 재사용, 프로덕션 파일 1개.
- 확신 없는 곳·검증 못 한 것: (1) 이미 jsonb `null` 이 저장된 **기존 행의 복구 경로는 없다** — 새 쓰기만 막았고 마이그레이션은 체크섬·릴리즈 계약 때문에 범위 밖으로 뺐다(ideas.json 에 신규 후보로 적음). (2) `scripts/smoke-settings.sh` 는 **읽어서만** 확인했다(기동 서버가 없어 미실행) — 그 스크립트가 PUT 하는 키는 `oidc`·`ai` 둘뿐이고 둘 다 이 코드 경로를 타지 않으므로 릴리즈 게이트는 안전하다고 판단했다. (3) 프런트엔드(web/)가 `{"value": null}` 을 보내는 코드가 있는지는 확인하지 않았다 — 범위 밖이라 npm 쪽 검증도 돌리지 않았다.
- 일부러 하지 않은 것: `oidc`·`ai` 경로(래퍼 없이 타입 구조체로 받고 재마셜 — 애초에 안전하고 auth 인접), 차선 후보였던 `startGameSession` 의 `metadata: null`(같은 형태의 결함이지만 `client_info` 를 읽는 코드가 없어 별 회차로), `serviceLocation` 의 오류 삼킴(호출처 6곳).
- 다음 역할이 조심할 것: 신규 `internal/api/settings_put_pg_test.go` 는 `IGAME_TEST_DSN` 이 없으면 skip 한다 — CI 의 `make test` 로는 이 회귀가 돌지 않으니 검토 시 로컬 도커로 `make test-db` 를 돌릴 것. 이 테스트는 전역 키(`service`·`play_policy`)를 실제로 덮어쓰고 `t.Cleanup` 에서 원래 값·`updated_at` 을 복원한다(실행 후 7개 키 전부 seed 값·`object` 임을 psql 로 확인). 테스트가 중간에 죽으면 DB 에 `Settings PUT test` 값이 남을 수 있으므로 일회용 DB 에서만 돌릴 것.
- [러너 16:32] brief accepted — 채택 — 수용 기준 0) 의 전제(`{"value": null}` 이 pgx/DB 에서 거부되지 않고 jsonb `null` 로 저장됨)를 실제 PG 로 확인해 과제가 
- [러너 16:33] verify passed — 검증 4개 통과 (policy)
- [러너 16:33] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 16:33] pr created — https://github.com/hkjang/igame/pull/28

## 심사 노트
- 확인: 일회용 PG(postgres:17-alpine, 전용 확장 스키마)로 Red/Green 실측 — 수정 전 `service`·`play_policy` 의 `{"value":null}` 이 200 + jsonb `null` 저장(테스트가 행 상태로 잡아냄), 수정 후 전부 400 `invalid_setting`. `go test ./...` 전체 통과, `internal/api`+`internal/database` PG 스위트 통과, 실행 후 7개 키 전부 seed 값·`object` 로 복원됨(psql 확인). gofmt·vet·build 무결.
- 확인: `oidc`·`ai` 는 `putSetting` 앞단에서 early return 하므로 새 검사를 타지 않음 → `scripts/smoke-settings.sh`(oidc/ai 만 PUT)·`smoke-realmguard.sh`/`smoke-defense-series.sh`(approval 을 `{"value":{…}}` 로 PUT) 릴리즈 게이트 안전. web/src 의 `saveAdminSetting` 호출 8곳 전부 오브젝트 전달. 편집 가능 7키 모두 구조체=오브젝트 계약.
- 확인: 거부는 DB 쓰기·audit 이전이라 행·`updated_at`·감사 로그가 그대로(테스트가 단언). `auditSettingChange` 는 `settingSecretKeys` 로 스크럽 — 비밀 유출 없음. 새 공개 경로·권한 확대·마이그레이션 없음, revert 로 완전 복원.
- 못 본 것: 이미 jsonb `null` 이 저장된 기존 행의 복구 경로(구현자가 범위 밖으로 명시, 신규 후보로 기록됨). 실제 기동 서버로 smoke 스크립트를 돌리지는 않았다(코드 경로 분기로 판단). npm/web 테스트 미실행(web 변경 없음).
- 권고 근거: 결함 없음 → approve/merge. docs/api.md:68 의 새 문장이 `value` 래퍼를 쓰지 않는 `oidc`·`ai` 에도 적용되는 것처럼 읽힐 수 있으나 바로 아래 96행이 그 계약을 따로 규정하고 코드도 그대로라 notes 로만 남긴다.
