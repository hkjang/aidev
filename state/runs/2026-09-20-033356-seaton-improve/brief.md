# 과제서 (2026-09-20, seaton, 브랜치 auto/2026-09-20-0333 = main 3f3c8ab)

- 과제: 사용자 권한 화면에서 자기 계정의 권한 강등 막기 (`400 self_demotion`) (가치 3 / 위험 1 / 작업량 S)
- 왜: `PATCH /api/v1/users/{id}` 는 `system_admin` 만 부를 수 있는데(`internal/app/server.go:121` 의 라우트 그룹), 시스템 관리자가 화면의 권한 Select 로 자기 role 을 `employee` 등으로 바꾸면 그 순간부터 이 화면·이 API 를 부를 수 없어 스스로 되돌릴 길이 없다(SSO 로그인 upsert 도 `users.role='system_admin'` 일 때만 role 을 보존하므로 `auth.go:413-415`, 강등 뒤 재로그인해도 그룹 매핑이 admin 이 아니면 그대로). 09-17 회차의 `self_deactivation` 과 같은 규칙(서버 검증 + 화면에서 자기 행 비활성)을 role 에도 적용하면 마지막 관리자가 자기를 잠그는 사고가 막힌다.

## 수용 기준
1) 자기 계정(`targetID == actorID`)에 `role` 이 `system_admin` 이 아닌 값으로 오면 `validateUserPatch` 가 `("self_demotion", "자기 계정의 권한은 낮출 수 없습니다")` 를 돌려주고 `updateUser` 는 그대로 `400` 으로 응답한다(기존 `writeError(w, 400, code, message)` 경로). 자기 계정에 `role:"system_admin"`(현상 유지)은 통과. 다른 계정의 강등은 그대로 통과. `role` 없이 `email`/`active` 만 보내는 자기 계정 PATCH(09-17 의 메일 주소 편집)는 계속 통과해야 한다.
2) 화면(`/admin/users`)에서 자기 행의 권한 `Select` 는 비활성(`disabled`)이고 Tooltip 으로 "자기 계정의 권한은 낮출 수 없습니다" 를 보여 준다. 다른 행은 지금과 같다.
3) 테스트가 증명할 것 — Go: `TestValidateUserPatch` 표에 케이스 3개 추가(자기 강등 → `self_demotion`, 자기에게 `system_admin` → 통과, 타인 강등 → 통과). vitest: `web/src/lib/users.ts` 에 `canChangeRole(user, me)` 를 두고 `users.test.ts` 에 자기/타인/me===null 케이스. Playwright: `web/e2e/admin.spec.ts` 의 기존 "자기 계정의 사용 스위치" 블록 옆에 (a) 자기 행의 권한 combobox 가 disabled, (b) `page.request.patch` 로 `{role:"employee"}` 를 자기 id 에 보내면 400 + `error.code === "self_demotion"`, (c) `GET /users` 에서 자기 role 이 여전히 `system_admin` — 실제 서버·DB 를 지나는 확인(대역 금지).
4) 문서: `internal/app/openapi.go:40` 의 PATCH 설명과 `docs/API_AND_MCP.md:47`, `docs/ADMIN_GUIDE.md:242` 한 문장에 `400 self_demotion` 을 덧붙인다. `docs/ADMIN_GUIDE.html` 은 `python3 scripts/build-docs.py ADMIN_GUIDE` 로 다시 만든다(PDF 는 굽지 않음 — 미머지 브랜치와 이진 충돌 회피, 09-17·09-18 과 같은 이유).

## 건드릴 파일
- `internal/app/auth.go:validateUserPatch` (513-529) — `in.Role != ""` 검사 뒤에 `if in.Role != "" && in.Role != "system_admin" && targetID == actorID { return "self_demotion", "자기 계정의 권한은 낮출 수 없습니다" }`. 함수 주석(511-512)의 "자기 계정은 막을 수 없다" 문장에 강등도 포함되게 한 줄 보탬.
- `internal/app/auth_test.go:TestValidateUserPatch` (95-125) — 표에 케이스 추가. `actorID` 는 `"me"` 로 고정되어 있음.
- `web/src/lib/users.ts` — `canToggleActive` 옆에 `canChangeRole(user: Pick<User,"id">, me: Pick<User,"id">|null): boolean` (`me === null || user.id !== me.id`). `web/src/lib/users.test.ts` 에 케이스.
- `web/src/pages/UsersPage.tsx` (240-252) — `Select` 에 `disabled={!canChangeRole(u, me)}`, 사용 스위치(254-276)와 같은 모양으로 `Tooltip` + `<span>` 으로 감쌈. `me` 는 이미 `useAuth()` 에서 옴(44행). Select 의 `slotProps.input["aria-label"]` 을 `${u.displayName} 권한` 으로 두면 e2e 에서 `getByRole("combobox", { name: /admin 권한/ })` 로 잡을 수 있음(현재 라벨 없음 — 미확인이므로 실제 렌더된 role 을 Playwright 로 확인할 것; MUI Select 는 `combobox` role 의 div 와 숨은 input 이 함께 있어 `toBeDisabled()` 는 `aria-disabled` 를 보는 combobox 쪽이 맞을 수 있음).
- `web/e2e/admin.spec.ts` (134-144 근처) — 위 3) 의 확인. `myID`·`me.csrfToken`·`users()` 헬퍼가 이미 그 블록에 있음.
- `internal/app/openapi.go:40`, `docs/API_AND_MCP.md:47`, `docs/ADMIN_GUIDE.md:242`, `docs/ADMIN_GUIDE.html`(생성물).

## 검증 명령 (이 저장소에서 실제로 도는 것)
- `go test ./internal/app/ -run UserPatch -count=1 -v` (정찰에서 통과 확인, 0.004s) → 전체 `go test ./... && go vet ./... && gofmt -l internal cmd`
- `cd web && npm test` (vitest run; 정찰 세션에서는 npm 실행 권한이 없어 미실행) 와 `npm run lint` (tsc -b)
- Playwright: CI 와 같은 방법 — `docker build --build-arg VERSION=e2e -t seaton:e2e .` → postgres:16 + `docker run --network host -e POSTGRES_DSN=… -e BOOTSTRAP_ADMIN=admin -e BOOTSTRAP_ADMIN_PASSWORD=…` → `cd web && E2E_BASE_URL=http://127.0.0.1:8080 E2E_USERNAME=admin E2E_PASSWORD=… npx playwright test e2e/admin.spec.ts`. 호스트 8080 이 막혀 있으면 09-18 처럼 브리지 네트워크 + `--add-host=host.docker.internal:host-gateway`.
- 캡처 `docs/assets/guide/admin-users.png` 는 Select 가 비활성으로 바뀌므로 교체하면 좋지만(`web/e2e/guide-shots.mjs` 160-163), 필수 아님.

## 위험과 피할 것
- `auth.go` 의 `authenticate`(114-170)는 미머지 MCP OAuth 브랜치(origin/auto/2026-09-18-0413)가 고치는 자리이므로 건드리지 말 것. `validateUserPatch`/`updateUser`/`UsersPage`/`auth_test.go`/`admin.spec.ts` 는 메일 브랜치(origin/auto/2026-09-16-1022)·OAuth 브랜치 어느 쪽도 손대지 않음(git diff 로 확인).
- `openapi.go`·`API_AND_MCP.md` 는 두 미머지 브랜치가 다른 hunk 를 고치므로 기존 줄(40행·47행)만 고치고 새 줄을 끼워 넣지 말 것. ADMIN_GUIDE.pdf 는 굽지 말 것.
- 강등만 막고 `system_admin` 으로의 자기 재지정(현상 유지)은 막지 말 것 — 화면 Select 가 같은 값을 다시 보내는 경우가 있음. `active`·`email` 만 있는 자기 PATCH 는 통과해야 09-17 의 메일 주소 편집 e2e(admin.spec 120-131)가 깨지지 않는다.
- e2e 는 실제 서버·DB 를 지나야 함(운영자 규칙). 화면에서 Select 를 disabled 로만 만들고 서버 검사를 빼면 API 로 여전히 강등되니 둘 다 넣을 것. 마이그레이션·세션·MCP 는 건드릴 것 없음.

## 차선 후보
- 좌석맵에서 배정 해제 단추 (3/2/S) — `DELETE /api/v1/seat-assignments/{seatID}` 를 쓰는 UI 가 없음. 프런트(`web/src/pages/SeatMapPage.tsx`)만 고치면 되지만 메일 브랜치가 `seats.go` 배정 경로를 고치고 있어 e2e 가 겹칠 수 있음 — 1순위가 성립하지 않을 때만.
