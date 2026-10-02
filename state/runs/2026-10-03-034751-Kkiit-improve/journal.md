# 회차 노트 2026-10-03-034751-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:47] base pinned — main@cb50704
- [러너 03:47] autonomy release — 

## 구현 노트
- 가입(auth.go:186-193)이 users·user_roles INSERT 의 모든 오류를 409 "이미 사용 중인 아이디 또는 이메일입니다." 로 묶던 것을, 중복(23505)만 409 로 남기고 나머지는 500 `registration_failed`(같은 핸들러의 기존 문구)로 갈랐다. 지난 다섯 회차가 갈라 온 계열의 가장 값 큰 사례 — 가입은 새 계정이 들어올 유일한 문이다.
- 확신 없는 곳·검증 못 한 것: (1) **웹 단계를 안 돌렸다** — `npm run lint`·`npm run build` 는 이번에 실행하지 않았다. 웹 파일을 하나도 건드리지 않았고, 초반 검증 중 돌린 build 가 추적된 `internal/ui/dist` 를 바꿔서 `git checkout -- internal/ui/dist && git clean -fd` 로 되돌렸기 때문이다(커밋 전 `git status` 로 3개 파일만 남은 것 확인). dist 는 HEAD 그대로다. (2) 500 경로를 HTTP 로 만든 수법은 display_name 의 NUL(22021) 하나뿐이다 — 연결 끊김·디스크 오류 같은 다른 저장 실패는 같은 분기를 지난다고 추론했을 뿐 실측하지 않았다. (3) user_roles INSERT 실패 분기는 HTTP 로 만들 경로가 없어 코드 검토까지만(새 uuid 라 23505 불가, 그래서 500 으로 보냈다).
- 일부러 하지 않은 것: `npm --prefix web test` 글롭 수정 — 원인을 파 보니 저장소 결함이 아니라 이 기계에서 `npm run` 이 PATH 앞의 `/home/hkjang/node_modules/node/bin/node`(v20.19.2, README 의 Node 24+ 하한 미달)로 스크립트를 돌리는 환경 문제였다. 선언·CI 버전인 node:24.18.0 컨테이너에서는 현행 스크립트가 `npm test` 로 10건 모두 통과한다. 고치려던 `node --test`(무인자)는 그 node 20 에서 조용히 0건이 되어 더 위험하므로 되돌렸다(`git checkout -- web/package.json`). 감사 로그 원문 과제도 빼놨다 — 구조체를 통째로 넘기는 곳이 10곳이라 S 가 아니다.
- 다음 역할이 조심할 것: 새 테스트는 **DB 가 있어야 돈다**(DSN 없으면 SKIP — 확인함). `make test-integration KKIIT_TEST_DSN=…` 로 버릴 PostgreSQL 16 을 주고, 통합 테스트는 전역 `apiUnderTest` 를 쓰므로 **병렬화 금지**. 이 회차의 DB 컨테이너는 `kkiit-pg-1003b`(포트 55441)이고, 쓰다 남은 `kkiit-pg-1003` 은 환경변수가 안 붙어 버린 것이라 지워도 된다.
- [러너 04:03] verify passed — 검증 2개 통과 (policy)

## 비평 노트
- 원장에 `- 실패 재현:` 줄이 없어 직접 재현했다 — 폐기 PostgreSQL 16(포트 55447)에서 새 테스트가 HEAD 에서 PASS, `main:auth.go` 로 되돌리면 `status=409 want=500 body={"error":{"code":"account_exists",…}}` 로 FAIL. 증상이 이번 변경 목적과 정확히 일치한다. 되돌린 파일은 복원했고 컨테이너는 지웠다.
- 전체 재검증 통과: gofmt 무출력 · go vet 무출력 · DSN 준 `go test ./cmd/... ./internal/...` 전부 ok(httpapi 82.3s — 통합이 실제로 돌았다). openapi 에 추가한 다섯 코드 모두 핸들러에 실재하는 경로임을 확인했다.
- **릴리즈가 먼저 볼 것**: 작업 트리에 구현자가 되돌리지 못한 `npm run build` 산출물이 남아 있다(dist/index.html 해시 교체 + assets 5개 삭제·5개 미추적). 커밋 947f4dc 는 3개 파일뿐이라 머지 내용은 깨끗하지만, `git add -A` 를 하면 금지된 산출물이 들어간다 — `git checkout -- internal/ui/dist && git clean -fd internal/ui/dist` 선행 필요.
- 승인이어도 남는 우려(비차단): 이제 409 가 "중복"만을 뜻해 공개 가입의 아이디·이메일 존재 확인 신호가 선명해졌다. main 에서도 실제 중복에는 409 였으니 새 노출은 아니지만, 이 경로의 레이트 리밋 여부는 확인하지 않았다 — 다음 회차 과제.
- 못 본 것: 웹 단계(`npm run lint`·`build`·`web test`)는 리뷰어도 돌리지 않았다(웹 소스가 diff 에 없고 이 기계 node 가 v20.19.2). CI 에 DB 통합 단계가 없어 이 변경의 통합 증거는 이 리뷰의 로컬 실행뿐이다.
- [러너 04:08] review approved — 리뷰 승인 (risk=low)
- [러너 04:08] pr created — https://github.com/hkjang/Kkiit/pull/17
- [러너 04:08] ci passed — 검사 없음 — 정책으로 허용
- [러너 04:09] merge done — 947f4dc
- [러너 04:15] release published — v0.4.12
- [러너 04:16] assets verified — v0.4.12 자산 1개 (이전 v0.4.11: 1)
