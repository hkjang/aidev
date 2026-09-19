# 과제서 — 2026-09-20 SecCheck

- 과제: 화면이 부르는 모든 API 경로(get/del/upload/post/put/patch)를 `server.go` 의 `s.handle` 등록과 대조하는 vitest 추가 (가치 3 / 위험 1 / 작업량 M)
- 왜: 지금 `web/src/lib/payloads.test.ts` 는 POST/PUT/PATCH 의 **본문 키**만 `internal/web/payloads.go` 표와 대조하고, `get`·`del`·`upload` 호출(44곳, 20파일)은 아무 검사도 받지 않아 서버가 라우트 이름을 바꾸거나 화면이 오타를 내면 런타임 404 로만 드러난다. 화면의 경로 문자열을 `s.handle("METHOD", "/path", …)` 등록(134건, 전부 `internal/web/server.go`)과 대조하면 그 부류의 회귀를 `npm test` 한 번으로 잡는다.

## 0단계 (판정만, 5분)
- `curl -s https://vuln.go.dev/ID/GO-2026-6452.json | jq '.modified, [.affected[].ranges[].events]'` 로 `fixed` 이벤트 유무를 기록(정찰 환경에서는 curl 승인 거부로 **미확인**; 2026-09-20 01:40 시점 없음). 있으면 CI 가 초록이 될 수 있으니 평소대로 진행. 없어도 **이 과제는 그대로 수행**한다 — 결과물은 로컬 vitest 로 완전히 검증되고 CI 의 유일한 빨강은 이 코드와 무관한 step 7 `Go vulnerability scan` 이다. 회차 노트에 판정 결과를 한 줄 적을 것. govulncheck 게이트를 풀려고 go.mod·워크플로·excelize 호출을 건드리지 말 것(5회 반복된 실패 유형).

## 수용 기준
1) `web/src/lib/routes.test.ts`(새 파일, 또는 payloads.test.ts 안의 새 `describe`)가 `internal/web/server.go` 의 `s.handle("<METHOD>", "<path>", …)` 를 전부 읽어 `METHOD path` 표를 만들고(`expect(table.size).toBeGreaterThan(100)` 같은 하한으로 스캐너 고장을 구분), `web/src` 의 `get/del/upload/post/put/patch` 호출 중 첫 인자가 리터럴(따옴표·템플릿)인 것을 모두 모아 각 호출이 등록된 라우트에 대응하는지 단언한다. 대응 규칙: 화면 경로에서 `?` 이후를 잘라내고, `${…}` 와 `{id}`·`{userID}` 류를 같은 `{*}` 로 정규화(기존 `normalise` 재사용), 메서드는 `get→GET`, `del→DELETE`, `upload→POST`, `post→POST`, `put→PUT`, `patch→PATCH`. `api(path, { method: 'X' })` 직접 호출도 method 리터럴이 있으면 포함.
2) 정상 트리에서 `cd web && npm test --silent` 가 새 테스트 포함 전부 통과한다. **불일치가 실제로 나오면** 그것이 이 과제의 발견이다 — 라우트가 실제로 없으면 화면을 고치고(서버 라우트 이름은 바꾸지 말 것), 스캐너가 못 읽는 형태(변수 경로, `${base}/x` 처럼 앞부분이 변수인 것)면 payloads.test.ts 의 "Bodies built elsewhere … are left alone" 관례대로 건너뛰되 건너뛴 호출 수를 하한 단언에 넣지 않는다.
3) 부정 검증(테스트가 무엇을 증명하나): (a) 화면 한 곳의 경로를 고의로 어긋내면(예: `Keys.tsx` 의 `del(`/api/v1/me/api-keys/${key.id}`)` → `/api/v1/me/apikeys/…`) 실패 메시지에 `파일:줄 METHOD 경로 … server.go 에 등록된 라우트가 없습니다` 형태로 파일·줄·메서드·경로가 나온다; (b) `server.go` 의 라우트 한 줄을 고의로 주석 처리해도 같은 방식으로 잡힌다. 두 부정 검증 전에 반드시 `git add -A && git commit -m "wip"` 로 WIP 커밋을 먼저 만들 것(이 저장소에서 편집을 날린 사례 3회).
4) `bash scripts/precheck.sh` 통과(DSN 없으면 통합 테스트는 조용히 건너뜀 — 그래도 gofmt·vet·tsc·vitest·build·가이드·gitleaks 는 돈다). Go 코드·가이드·PDF 는 바꾸지 않으므로 docs_test·PDF 재생성 없음.

## 건드릴 파일
- `web/src/lib/routes.test.ts` (신규) — 라우트 표 파서 + 화면 호출 스캐너 + 대조 3개 `it`(표를 찾았다 / 호출을 찾았다(하한 ≥ 40) / 모든 호출이 등록된 라우트에 간다). 스캐너는 `payloads.test.ts` 의 `skipBalanced`·`pathOf`·`sourceFiles`·`normalise`·`lineOf` 를 그대로 쓰고 싶으면 그 함수들을 `export` 해서 가져오거나(테스트 파일끼리 import 는 vitest 에서 됨) 복제 대신 `web/src/lib/sourceScan.ts` 같은 작은 공용 모듈로 빼도 된다 — 어느 쪽이든 `payloads.test.ts` 의 기존 세 단언은 그대로 통과해야 한다.
- `web/src/lib/payloads.test.ts` — 함수 export 만(동작 변화 없음). 그 외 변경 금지.
- `internal/web/server.go` — 읽기만. 라우트 이름 변경 금지.
- 화면 파일(`web/src/pages/*.tsx`, `main.tsx`, `components/Layout.tsx`) — 2) 에서 실제 불일치가 나온 경우에만 그 경로 문자열만 고침.

## 확인한 사실 (정찰이 실제로 본 것)
- `s.handle` 시그니처: `internal/web/server.go:251 func (s *Server) handle(method, path, tag, summary string, roles []string, public bool, h http.HandlerFunc)`; 등록 134건 모두 `server.go` 99~243행, 형태는 `s.handle("GET", "/api/v1/users/directory", …)`, 경로 변수는 `{id}`·`{userID}`·`{versionID}`·`{itemID}` 처럼 Go 1.22 ServeMux 패턴.
- `web/src/lib/api.ts:68-73` — `get(path)`, `post(path, data?)`, `put`, `patch`, `del(path)`, `upload(path, form)`(POST). `download(path)`(79행) 는 `fetch` 직접이라 스캐너 범위 밖(포함하면 좋지만 필수 아님).
- 화면의 get/del/upload 호출 44곳: 쿼리 문자열 붙는 것(`/api/v1/admin/jobs?${qs}`, `/api/v1/users/directory?role=REQUESTER`, `/api/v1/admin/mail/deliveries?limit=50`), 템플릿 변수 경로(`/api/v1/review-requests/${reviewID}/participants/${person.user_id}`), `Audit.tsx:48` 의 `` `/api/v1/admin/audit/verify${full ? '?full=1' : ''}` `` 처럼 **경로 뒤에 `${…}` 로 쿼리를 붙이는 것** — 이 마지막 형태는 `?` 가 `${}` 안에 있어 단순 `split('?')` 로는 안 잘림. `${…}` 를 먼저 `{*}` 로 바꾼 뒤 끝의 `{*}` 가 `/` 뒤가 아니면(즉 `verify{*}`) 잘라내는 규칙을 두거나, 그 한 호출을 명시 예외로 두고 이유를 주석에 적을 것.
- `web/package.json` `"test": "vitest run"`, vitest 는 `latest`(4.x — 미확인, 5.x 로 올리지 말 것).

## 검증 명령
- `cd web && npm ci && npx tsc --noEmit && npm test --silent && npm run build`
- 부정 검증: 위 3)(a)(b) 각각 `npx vitest run src/lib/routes.test.ts` 로 실패 메시지 확인 후 `git checkout -- <파일>` (WIP 커밋 뒤에만)
- `bash scripts/precheck.sh`
- `git diff origin/main -- .github/workflows/ go.mod go.sum` 이 비어 있음

## 위험과 피할 것
- `go.mod`·`.github/workflows/*`·`scripts/precheck.sh`·`docs/operations.md`·`docs_test.go` 는 열린 PR #9~#12 가 건드리므로 손대지 말 것(충돌 + 5회 반복된 게이트 우회 시도로 오해).
- `internal/auth/*`·`internal/store/migrations/*` 보호 경로는 무관 — 건드리지 말 것.
- 스캐너를 `payloads.test.ts` 와 두 벌로 복제하면 나중에 한쪽만 고쳐지는 "같은 값을 읽는 경로가 여럿" 문제가 된다 — 공용 함수를 한 곳에 두고 두 테스트가 같은 것을 읽게 할 것.
- 단언 하한(라우트 > 100, 호출 ≥ 40)은 스캐너 고장 감지용이지 정확한 수를 박아 두는 것이 아니다.
- 실제 출력·동작이 안 바뀌는 변경(소스 문자열 검사만 있는 테스트)이라는 지적을 받을 수 있으나, 이 테스트는 기존 payloads.test.ts 와 같은 부류(정적 계약 대조)이며 서버 표와 화면 호출 **두 경로가 같은 값을 읽는지** 를 대조하는 것이 목적이다 — 회차 노트에 이 취지를 적을 것.
- 커밋 제목은 영어 서술문(예: "Check every screen path against the routes the server registers").

## 차선 후보
- docs_test 로 가이드가 인용하는 `<row>.<key>` 설정 키가 시드 행에 실제로 있는지 대조 (가치 2 / 위험 1 / S) — `internal/web/docs_test.go` 한 파일이지만 PR #12 가 같은 파일을 건드리므로 1순위가 성립 안 할 때만.
