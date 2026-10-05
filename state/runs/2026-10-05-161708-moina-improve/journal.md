# 회차 노트 2026-10-05-161708-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 16:17] base pinned — main@bd74c83
- [러너 16:17] autonomy release — 

## 정찰 노트
- 지난 회차가 남긴 1순위(storage_error 출구 이관)를 그대로 골랐다. 다른 pending 후보는 전부 선행 조건이 막혀 있다 — Scan 후보·전용 스키마 헬퍼는 SELECT 실패 재현 설계가 선행, Axe/e2e 후보는 `make image` + Playwright 실제 실패 재현이 선행(45분에 위험), API 키 후보는 제품 결정이 선행, CI 요약 후보는 workflow 파일이라 보호 경로에 가깝다.
- 파일 선택은 `grep -c 'storage_error' *.go`로 정했다: posts.go 41이 가장 많고 message 중복이 가장 심해(같은 문구 5~6곳) 로그 추가의 가치가 가장 크며, **프로덕션 파일 1개**로 끝나 재작업 위험 구간(파일 10개 이상)에서 가장 멀다.
- 과제서에 적은 것은 전부 실제로 열어 확인했다: 41개 출구 줄번호, `grep -n '^func '`로 뽑은 함수 경계와 출구→handler 매핑, `writePostError`의 `r` 없는 서명과 호출 3곳, `cursor_error` 500 둘(:547 :656), `posts_update_postgres_integration_test.go:153`의 BEFORE UPDATE 트리거 저장 실패 케이스. **추측(미실행)**: 그 트리거의 SQLSTATE가 `P0001`이라는 것(지난 회차 admin 테스트에서 `P0001`이었던 것에서 유추 — 다르면 단언 값만 고치면 된다), 그리고 41곳이 한 세션에 들어간다는 작업량 판단.
- 구현자가 조심할 것: 줄번호는 이관하면서 밀리므로 message 문자열과 감싼 함수로 식별할 것. `handler` 오기는 컴파일로 잡히지 않는 유일한 실질 결함이다. message 한 글자라도 바뀌면 프런트 `readableError`를 타고 사용자에게 보인다.
- 프로필은 1일 전 것이고 지금 코드와 맞아 다시 쓰지 않았다. 유일한 어긋남은 "posts.go ~35"라는 추정치이며 실제는 41이다(ideas.json에 기록).
- [러너 16:21] scout done — `posts.go`의 `storage_error` 500 출구 41곳을 `writeStorageError`로 이관한다 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- `posts.go`의 `storage_error` 500 출구 41곳을 `writeStorageError`로 이관해 `handler`·`cause_type`·`pg_code`가 `request_id`와 함께 남게 했다(76010d8, 프로덕션 파일 1개 + 테스트 1개). 응답 불변은 한글 리터럴 multiset 비교로 증명했다 — HEAD에만 있는 리터럴 0개.
- `handler` 인자는 손으로 적지 않고 `^func ` 경계에서 감싼 함수 이름을 유도해 치환한 뒤, 같은 유도로 43개 호출 전부 mismatch 0을 재검증했다(정찰의 수기 매핑과 일치). 과제서가 지목한 "유일한 실질 위험"은 이것으로 닫혔다고 본다.
- **과제서 범위를 넘은 곳(비평가가 먼저 볼 곳)**: `deletePost`·`deleteRemoin`의 `Exec실패 || tx.Commit() != nil` 한 조건을 두 분기로 나눴다. 합친 채로 이관하면 Commit 실패 시 `err`가 nil이라 `cause_type:"<nil>"`을 찍어 이 회차의 목적을 배반한다. 단락 평가 순서·응답·message는 그대로이고 같은 message가 한 번 더 쓰인 것이 늘어난 리터럴 2개의 정체다. **이 두 경로의 Commit 실패는 테스트로 재현하지 않았다** — 등가성은 코드 읽기로만 확인했다.
- **확신 없는 것**: red→green을 실제 핸들러로 관측한 것은 `updatePost`의 UPDATE 실패 한 경로뿐이다. 나머지 40곳의 `handler` 문자열은 기계적 유도로만 보증되고 실행으로 확인하지 않았다(컴파일로도 안 잡힌다).
- 일부러 안 한 것: `cursor_error` 500 둘·404/409 분기·`storage_error.go` 헬퍼·OpenAPI는 손대지 않았다. 응답이 불변이라 프런트 vitest와 e2e는 돌리지 않았다(지난 두 회차와 같은 판단).
- 다음 역할이 조심할 것: `posts_update_postgres_integration_test.go`는 DB가 있어야 돈다(`MOINA_TEST_POSTGRES_DSN`, 없으면 `t.Skip`). 검증은 `postgres:16-alpine` 컨테이너로 했고 포트 55432는 다른 프로젝트가 점유 중이라 **55461**을 썼다. 전체 테스트는 2회만 돌렸다(로그인 5분/IP 5회 제한).
- [러너 16:27] brief accepted — 채택 — 근거가 현재 코드와 전부 일치했습니다(41곳의 줄번호, `grep -n '^func '` 함수 경계와 출구→handler 매핑 전부, `writePo
- [러너 16:27] verify passed — 검증 7개 통과 (auto)
