# 회차 노트 2026-10-06-022757-ai-admin-improve — ai-admin
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:28] base pinned — main@d1dab8e
- [러너 02:28] autonomy release — 

## 정찰 노트
- 고른 이유: 2026-10-05 가 남긴 "NUL 거부를 나머지 varchar 경로로 넓히기" 를 소스로 재평가해 실제 미검증 지점 세 곳(`users.go:278` 역할 이름 120, `keys.go:118` 키 이름 160, `providers.go:478/494/502` 공급자 이름 160·모델 240·available_models jsonb)을 확정했다. 헬퍼(`storableInVarchar`)가 이미 main 에 있어 추가 설계가 없고 프로덕션 3파일로 끝난다. 차선(chat 감사 reason 테스트)은 가치가 같은데 결함을 고치지 않아 뒤로 뒀고, 감사 CSV 문서 문구는 단독 가치가 1 이라 제쳤다.
- 확신 없는 곳: 세 경로가 실제로 500 이 되는지는 DB 로 재현하지 않았다(미확인) — 컬럼 타입·검증 부재·500 코드 위치만 소스로 확인했다. 과제서는 red 를 먼저 찍고 재현되지 않는 사례는 빼라고 적었다.
- 새로 발견: 2026-10-04 회차의 `updatePreferences` jsonb 검증(f66d25c)이 **main 에 머지되지 않았다**(헬퍼 `jsonObjectHasNUL`·`storablePreferenceObject` grep 0건). 결함은 남아 있지만 미머지 브랜치와 중복이라 이번 과제에서 제외하고 ideas.json 에 사정을 적었다.
- 구현자가 조심할 것: `storableInVarchar` 의 이름·구현을 바꾸지 말 것(기존 3개 호출부와 `profile_nul_integration_test.go` 가 걸려 있다), `workflow.go` 와 `updateKeyScope`(2026-10-01 verify-failed 이력)는 건드리지 말 것, 테스트 DB 는 새 포트(55541 권장)로 띄우고 `docker run` 이 샌드박스에 막히면 대체 DB 를 찾을 것.
- [러너 02:32] scout done — 남은 varchar/jsonb 쓰기 경로(역할 이름·API 키 이름·공급자 이름/모델)의 NUL(U+0000) 을 저장 전 400 으로 막기 (

## 구현 노트
- 무엇을 왜: `updateRole` name·`createKey` name·`validateProviderRequest` name/defaultModel/availableModels 세 자리에 기존 헬퍼 `storableInVarchar`(users.go:421, 이름·구현 그대로)를 더해, varchar/jsonb 가 담을 수 없는 NUL(U+0000) 이 INSERT/UPDATE 에서 500 으로 터지던 것을 각 핸들러의 기존 400(`name_invalid`·`key_invalid`·`provider_invalid`)으로 막았다. 프로덕션 3파일 + 테스트 1 + docs/api.md.
- 확신 없는 곳·검증 못 한 것: ① `validateProviderRequest` 가 승인 실행 경로(workflow)도 덮는다는 과제서의 주장은 **코드로만** 확인했고 테스트가 그 경로를 지나지 않는다 — `workflowRequired("ai_provider_change")` 가 기본 false 라 셋업에서 workflow_policy 를 켜야 하고, 그건 별 과제로 ideas.json 에 적었다. ② `createKey` 의 고권한 승인 분기(keys.go:162 payload 생성)도 테스트가 지나지 않는다 — 검증이 핸들러 맨 앞(121)이라 그보다 앞서는 것은 소스로만 확인. ③ 공급자 이름 오류 메시지를 "1~160자여야 하고 NUL 문자를 포함할 수 없습니다." 로 바꿨다(기존 메시지 문구 변경). 메시지 substring 을 단정하는 기존 테스트는 전체 통과로 깨지지 않음을 확인했지만 웹 UI 문구 기대가 있다면 그쪽이 영향권이다.
- 일부러 하지 않은 것: `workflow.go`·`updateKeyScope`·`updatePreferences`(미머지 브랜치 `auto/2026-10-04-1142` 와 충돌)·마이그레이션·`internal/ui/dist`·감사 CSV 문서 절·VERSION/CHANGELOG. `storableInVarchar` 의 이름·구현도 바꾸지 않았다(기존 5개 호출부가 걸려 있다).
- 다음 역할이 조심할 것: `TestVarcharNULWrites` 는 `TEST_POSTGRES_DSN` 이 있어야 돌고 없으면 조용히 SKIP 한다 — `-v` 로 PASS 를 확인할 것. `DROP SCHEMA ai_admin/aiportal CASCADE` 를 하므로 전용 폐기 DB 전용(이번에 `postgres:16-alpine` 포트 55541, 컨테이너 `ai-admin-1006-pg`). 이 환경에서 `docker run` 은 동작했다.
- [러너 02:41] brief accepted — 채택 — 결함·코드 위치(`users.go:272-282`·`keys.go:118`·`providers.go:477-505`)·컬럼 타입(`varchar(120)`/`varchar(160)`/`varchar(160)`·`varcha
- [러너 02:41] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인: main 디텍트 워크트리에 신규 테스트만 올려 red 를 직접 재현했다(원장과 동일한 500 코드·줄번호 12건), HEAD 는 18 서브테스트 PASS(SKIP 아님). 전용 폐기 DB 포트 55641 로 `go test -count=1 ./...` 전체 green, `make lint`·`gofmt -l` green. 컨테이너·워크트리 정리 완료.
- 확인: 구현자가 소스로만 봤다던 두 자리 — `createKey` 검증(keys.go:121)이 승인 분기(:162)보다 앞, `validateProviderRequest` 가 생성(:107, 승인 레코드 :113 보다 앞)·수정(:185)·실행(`workflow.go:678`)을 모두 덮음 — 을 읽어 확인했다. 승인 payload jsonb 500 경로까지 함께 막혔다.
- 못 본 것: 승인 워크플로를 실제로 켜고 지나가는 통합 경로, 웹 UI 수동 확인(웹 변경 없음).
- 남는 우려(릴리즈 노트용): 같은 핸들러의 `updateRole` `description`(text)은 여전히 NUL → 500. `updateKeyScope` name·`updatePreferences` jsonb 도 잔존(둘 다 미머지 브랜치 사정). 원장 보류 아이디어에 적혀 있으니 다음 회차 우선순위로.
- 판정: approve / risk low / blocking 없음. 공급자 이름 오류 문구 변경은 저장소 어디서도 참조되지 않음을 grep 으로 확인했다.
- [러너 02:45] review approved — 리뷰 승인 (risk=low)
- [러너 02:45] pr created — https://github.com/hkjang/ai-admin/pull/40
- [러너 02:54] ci passed — 검사 2개 모두 success
- [러너 02:54] merge done — 3dce1a1
- [러너 03:07] release published — v1.2.35
- [러너 03:08] assets verified — v1.2.35 자산 2개 (이전 v1.2.34: 2)
