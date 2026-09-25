# PR 처리기 노트 2026-09-26-030328-orbit-shepherd — orbit PR #11
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-09-26-020313-orbit-improve)
# 회차 노트 2026-09-26-020313-orbit-improve — orbit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 02:03] base pinned — main@e40c4cb
- [러너 02:03] autonomy release — 

## 정찰 노트
- `/orbit` 두 경로의 응답 계약이 실제로 어긋난 것을 코드에서 직접 봤다(data.go:772 에는 `earliest_at` 이 있고 timetravel.go:165~173 에는 없다). 운영자가 되풀이한 "같은 값을 읽는 경로가 둘이면 end-to-end 로 맞춰라" 에 정확히 해당하고 고침이 4줄이라 1순위로 골랐다.
- 제친 후보: `orbitAt` DB 테스트 보강(2026-09-24 에 verify 실패한 접근이라 재선택 금지), 교류 조회 LIMIT(회귀 기준이 main 에 없어 맨손), memories count GROUP BY(같은 이유로 위험 2 로 상향), 웹 lint 스크립트 제거(가치 낮음).
- **중요**: 2026-09-24 회차의 `orbitAt` 하위 시험 10개와 `seedRelationship`·`seedMemory` 헬퍼는 main 에 없다(grep 으로 확인). 회차 기록의 "성공" 을 믿지 말 것 — 지금 `orbitAt` 테스트는 0 개다. 프로필도 이 점을 고쳐 새로 썼다.
- 추측으로 적은 것(미확인): `internal/server/openapi.go` 의 `operation()` 시그니처를 열어 보지 않았다 — 질의 매개변수를 받을 자리가 없을 수 있다. 구현자는 그 함수를 먼저 읽고, 없으면 함수를 고치지 말고 `/orbit` 항목만 리터럴로 풀어 쓸 것.
- 조심할 것: `auth.go`·`silentSso.ts` 는 v0.7.0 에서 막 바뀐 인증 경로라 열지 말 것. `orbitRange` 질의를 복사하지 말고 그대로 호출할 것. DB 테스트는 CI 에서 SKIP 되므로 CI 초록을 증거로 쓰지 말고 DSN 을 준 실행 출력을 남길 것.
- [러너 02:08] scout done — `GET /orbit?at=` 이 `earliest_at` 을 빼먹어 현재/과거 두 경로의 응답 계약이 어긋나는 것을 고치고, Time Travel 질

## 구현 노트
- `writeOrbitAt` 이 현재 경로와 **같은** `s.orbitRange` 를 불러 `earliest_at` 을 담게 했다(질의 사본 없음). 다른 키·순서는 그대로 — `getOrbit` 은 한 글자도 안 건드렸다. 커밋 297552a.
- 확신 없는 곳: docs/API.md 의 "what actually travels" 문단은 `orbitAt` 코드를 읽고 쓴 것이지 시험으로 고정하지 않았다(중요도·소속·고정이 오늘 값이라는 것은 timetravel.go 머리 주석 근거). openapi.go 는 임시 시험으로 유효 JSON 렌더만 확인하고 그 시험을 남기지 않았다 — 영구 회귀가 없다.
- 검증 못 한 것: 웹을 한 줄도 안 고쳤으므로 vitest·vite build 를 돌리지 않았다. 실제 브라우저에서 `?at=` 화면이 도는 것은 안 봤다(OrbitPage.tsx:67 가드는 그대로 두었고, 새 키는 같은 값이라 화면 동작이 달라질 수 없다).
- 일부러 안 한 것: `orbitAt` 의 교류 조회 LIMIT·memory count GROUP BY 는 동등성 회귀 기준이 main 에 없어 손대지 않았다. `getOrbit` 의 `categories: null` 차이도 이번 과제 조건(현재 경로 불변)이라 남겼다.
- 다음 역할이 조심할 것: `TestOrbitAtResponseCarriesEarliestAt` 는 **실제 postgres 가 있어야 돈다** — `ORBIT_TEST_DATABASE_URL` 이 없으면 SKIP 이고 CI 에는 DB 가 없어 늘 SKIP 이다. CI 초록을 이 변경의 증거로 쓰지 말 것. 확인하려면 `docker run -d --rm -e POSTGRES_PASSWORD=orbit -p 55481:5432 postgres:16-alpine` 뒤 `ORBIT_TEST_DATABASE_URL='postgres://postgres:orbit@127.0.0.1:55481/postgres?sslmode=disable' go test -race -count=1 -v ./internal/server -run TestOrbit` (이번에 7 PASS). 컨테이너는 정리했다.
- [러너 02:12] brief accepted — 채택 — 근거(data.go:772 에는 `earliest_at` 이 있고 timetravel.go:165~173 에는 없음, `operation()` 에 매개변수 자리 없음)가 지금 코�
- [러너 02:13] verify passed — 검증 7개 통과 (auto)
- [러너 02:13] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 02:13] pr created — https://github.com/hkjang/orbit/pull/11

## 심사 노트
- 확인한 것: 실제 postgres(55487)를 띄워 `go test -race -run TestOrbit ./internal/server` 7 PASS, timetravel.go 의 earliest_at 을 임시로 되돌리면 새 시험 3개가 전부 FAIL(키 없음) — 시험이 결함을 실제로 고정한다. 되돌린 뒤 파일 복구·git status 깨끗, 컨테이너 삭제.
- 확인한 것: /orbit 리터럴이 유효 JSON 으로 렌더되고 at 매개변수가 붙는 것을 임시 시험으로 봤다(시험은 남기지 않음 — 영구 회귀 없음). gofmt/vet/build/go test ./... 모두 통과. 질의 사본 없이 data.go 와 같은 s.orbitRange 를 부른다.
- 못 본 것: 웹을 한 줄도 안 고쳐 vitest·vite build·브라우저 화면은 돌리지 않았다. openapi 렌더 회귀가 main 에 남지 않는다. node 안 categories 의 null/[] 불일치는 이 PR 범위 밖으로 남아 있다.
- 보안·법무: 새 경로·인증 분기·마이그레이션·의존성 변경 없음. orbitRange 는 인증 컨텍스트의 user_id 로 스코프되고 타사용자 격리 시험이 실행되어 통과. 차단 소견 없음.
- 권고: approve / merge, risk low. 근거는 위 두 줄(사전 코드에서 FAIL·사후 PASS, 두 경로 값 일치를 end-to-end 로 확인)이다.
