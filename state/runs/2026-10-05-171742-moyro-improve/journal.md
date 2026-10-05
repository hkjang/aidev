# 회차 노트 2026-10-05-171742-moyro-improve — moyro
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 17:17] base pinned — main@20c1daa
- [러너 17:17] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 목록의 "쓰기 배치 조용한 절단" 전수 조사를 먼저 돌렸는데 `ids[:N]` 7곳이 전부 읽기 경로여서 그 아이디어는 done 으로 닫혔다. 대신 같은 class 의 진짜 결함이 `_ = decodeCappedBody` 쪽에 남아 있었고, 그중 `final.go:1296/1349`(커스텀 프로필 값 PATCH)는 거부된 본문이 no-op 이 된 뒤 **되읽기가 이전 값을 담아 200 으로 돌려주므로 진짜 저장 응답과 구별이 안 되는** 유일한 사례다. 2026-10-04 bulk_delete(`count:0` 200)보다 증상이 나쁘고, 그 회차가 만든 `decodeOptionalCollectionBody` 로 2줄 치환이면 끝나 프로덕션 1파일이다. 09-28~10-02 네 회차가 연속으로 낸 httpapi **상태코드 재분류** 계열은 일부러 피했다(`channels.Get` 404→500, 북마크·게스트 401 위장 전부 보류 유지).
- 확신하는 것(코드로 확인): `customprofile/service.go:199-202` 의 `len(values)==0 → return nil`, 1296/1349 가 `tooManyBatchItems` 호출자가 **아님**, `decodeOptionalCollectionBody` 의 EOF/413/400 분기, 같은 두 핸들러를 이미 띄우는 테스트 `custom_profile_values_errors_postgres_test.go` 와 헬퍼 `customProfileValuesRequest`, webapp 두 호출자(`compat.ts:336/348`)가 항상 본문을 보냄.
- 추측으로 적은 것(미확인): 실제 DB 를 한 번도 띄우지 않았다 — RED 를 관찰하지 못했고 DSN·컨테이너(`moyro-pg-improve`, 55433)는 이전 회차 기록을 그대로 옮긴 것이다. 새 오류 id `api.custom_profile.values.patch.invalid_body` 는 내가 제안한 이름이며 기존 id 가 아니다. 1 MiB 초과 본문의 수정 전 실제 응답(200 + 이전 값)은 코드 추론이지 실행 확인이 아니다.
- 구현자가 조심할 것: 기존 테스트는 `h.audit` 를 **nil** 로 둔다 — 그대로 복사하면 `LogAsync` 가 아예 안 불려 "감사행 0건" 이 거짓으로 통과한다. 감사 서브테스트에는 실제 audit 서비스를 주입하고 양성 대조를 먼저 세울 것. `customProfileValuesRequest(t, "")` 는 GET 으로 바뀌는 분기가 있어 "본문 없는 PATCH" 보존 테스트에는 못 쓴다. `decodeCollectionBody`(EOF 도 400)로 치환하면 수용 기준 3 이 깨진다. `final.go` 는 크기 상한 77000 에 73k 대라 주석 여유가 적다.
- 프로필은 다시 쓰지 않았다(6일 경과, 14일 한도 미도달). 이번에 재검증한 항목은 전부 일치했다: `newOperationsTestDB` 존재, 포트 55433 기록, `gofmt -l server/internal` 의 기존 9파일 경고, `request_body.go` 헬퍼 집합.
- [러너 17:22] scout done — 커스텀 프로필 값 PATCH 두 핸들러가 거부해야 할 본문을 삼키고 200 + 이전 값 + `count:0` 감사행으로 "저장 성

## 구현 노트
- `final.go:1296/1349` 의 `_ = decodeCappedBody` 두 줄을 기존 `decodeOptionalCollectionBody` 로 치환했다(상한 초과 413 / 깨진 본문 400, `h.audit.LogAsync` 앞에서 거부). 되읽기가 거부된 본문에도 이전 값을 담아 200 을 돌려줘 **진짜 저장 응답과 바이트 동일**했던 것이 핵심 결함이다. 신규 오류 id 는 `api.custom_profile.values.patch.invalid_body` 하나뿐이고 두 핸들러가 공유한다. `request_body.go` 는 무변경.
- 확신 없는 곳·검증 못 한 것: (1) **새 테스트는 `MOYRO_TEST_POSTGRES_DSN` 이 없으면 통째로 skip 된다** — DSN 없이 `ok` 가 나오면 돌지 않은 것이다(실제 실행 시 httpapi 59.5s, 없을 때 0.17s). (2) 감사 "0건" 단언은 `audit.LogAsync` 의 3초 goroutine 창을 3.2초 폴링으로 덮는 타이밍 의존 단언이다(2026-10-04 선례와 동일 방식). (3) `null` 본문이 200 no-op 으로 남는 것은 **의도된 보존**이지 이상적 계약이라고 판단한 것이 아니다 — 디코드가 성공하므로 핸들러는 `{}` 와 구별할 수 없다. (4) 웹은 호출자 두 곳(`compat.ts:336/343`)이 항상 작은 정상 JSON 맵을 보내는 것을 코드로 확인했을 뿐, webapp 게이트는 돌리지 않았다(웹 변경 0).
- 일부러 하지 않은 것: 다른 `_ = decodeCappedBody` 호출자 14곳(같은 파일의 `createCustomProfileField`/`patchCustomProfileField` 포함 — 단건 쓰기라 빈 본문 계약을 먼저 정해야 한다), `request_body.go` 헬퍼 수정, `customprofile` 서비스 수정, 되읽기·감사·응답 모양 변경, 차선 후보(`postsByIDsReactions`).
- 다음 역할이 조심할 것: 이 테스트는 **실제 PostgreSQL 이 있어야** 돈다(`docker start moyro-pg-improve`, 55433). 별도 `newOperationsTestDB` 로 격리했으니 형제 테스트의 `DROP TABLE`/`RENAME` 장애 주입과 섞지 말 것. 두 치환 줄 중 하나만 되돌리면 그 핸들러의 서브테스트 3개 + 공유 감사 서브테스트만 깨지고, `Optional` 을 `decodeCollectionBody` 로 바꾸면 "본문 없는 PATCH" 두 건만 깨진다(세 방향 모두 실제로 확인했다). `final.go` 는 73797/77000 이라 주석 여유가 줄었다.
- [러너 17:34] brief accepted — 채택 — 과제서의 근거가 현재 코드와 정확히 일치했다(`:1296/:1349` 의 `_ =`, `service.go:199-202` 의 `len==0` 조기 반환, `decodeOptio
- [러너 17:34] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 승인. **RED 를 직접 관찰했다**: 프로덕션 파일만 main 으로 되돌리니 16개 중 7개 서브테스트가 깨지고, 그 출력이 주장한 증상과 정확히 같다 — `:195 status = 200, want 413 (body {"field-dept":"Support"})`, 즉 거부가 진짜 저장과 바이트 동일한 바로 그 장면. HEAD 복원 후 httpapi 전체 ok 34.9s(DB), `-race` ok, 3회 반복 안정(3.78/3.79/3.80s), build·vet·check-source-sizes 전부 exit 0, 변경 두 파일 gofmt 깨끗(`native_activity.go` 만 기존 경고).
- 구현자의 자기 의심 4개 전부 확인: (1) DSN skip 실재 — 내가 DB 를 붙여 돌렸다. (3) `null` 200 no-op 은 디코드 성공 후 `{}` 와 구별 불가이므로 이번 범위에선 보존이 맞다. (4) 웹 호출자 두 곳(`compat.ts:336/348`)은 항상 작고 정상인 맵을 직렬화 — 사내 클라이언트 파손 없음. 보존 서브테스트 6개가 **base 에서도 통과**하는 점이 그것들이 새 동작이 아닌 진짜 핀임을 증명한다.
- 남는 우려(차단 아님): 감사 0건 단언은 3.2초 창 안에서 부정을 증명하므로 **거짓 FAIL 은 불가, 거짓 PASS 는 가능**하고 패키지에 3.2초를 고정 추가한다(양성 대조는 제대로 세워져 있어 의미는 있다). 릴리즈 노트 한 줄 필요: 두 라우트에서 깨진/상한 초과 본문이 200 → 400/413 으로 바뀌고 **신규 오류 id** `api.custom_profile.values.patch.invalid_body` 는 업스트림 id 가 아니다(기존 id·200 모양 무변경, 라우트는 각 1개뿐이라 반경은 둘로 한정).
- 보안·법무 소견 모두 차단 없음: 인증·인가는 무변경이며 **새 디코드보다 먼저** 실행된다(`requireUserParamAccess` / `uid==""` 401). 새 식별자·경로·권한·비밀값·수제 비교·의존성 0, 입력을 좁히기만 한다. 400 상세는 json 디코더 메시지로 본문 내용이 아니다. 개인정보 신규 수집·보존·전송·접근통제 변경 없음, 테스트 픽스처는 합성 데이터.
- **다음 회차가 고칠 프로필 거짓**: 컨테이너 `moyro-pg-improve` 는 이 머신에 **없다**(`docker ps -a` 확인) — 정찰이 검증 없이 옮겨 적은 줄이다. 내가 `moyro-review-pg`(postgres:16, 127.0.0.1:55611)를 띄워 **켠 채로 두었으니** 릴리즈·수리가 재사용하라; 정리는 `docker rm -f moyro-review-pg`. 남은 `_ = decodeCappedBody` 15곳(같은 파일 `createCustomProfileField`/`patchCustomProfileField` 포함)은 의도된 범위 밖.
- [러너 17:40] review approved — 리뷰 승인 (risk=low)
- [러너 17:40] pr created — https://github.com/hkjang/moyro/pull/35
- [러너 17:50] ci passed — 검사 3개 모두 success
- [러너 17:50] merge done — f4d6b10
- [러너 18:26] release published — v0.2.47
- [러너 18:44] assets verified — v0.2.47 자산 1개 (이전 v0.2.46: 1)
