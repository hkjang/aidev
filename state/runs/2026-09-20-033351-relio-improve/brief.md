# 과제서 2026-09-20 — relio

- 과제: 감사 로그 목록 API 와 관리자 상세 모달에 `metadata` 를 내보내기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `audit_logs.metadata`(jsonb, `migrations/001_initial.sql:565`) 에는 로컬 LOGIN 의 `bootstrap` 여부(`internal/server/public.go:82`)와 SSO LOGIN 의 `silent` 여부(`internal/oidc/service.go:575`)가 쌓이지만, `GET /api/v1/admin/audit`(`internal/server/admin_operations.go:80` `adminAudit`) 의 SELECT 는 `before_data,after_data` 까지만 읽고 metadata 열을 아예 고르지 않아 관리자 화면 어디에서도 볼 수 없습니다. 열을 응답에 더하고 상세 모달에 보여주면 "이 로그인이 silent SSO 였나 / Bootstrap 계정이었나" 를 운영자가 DB 를 열지 않고 답할 수 있고, 다른 브랜치(0eb10f5, LOGIN_FAILED reason) 가 머지되면 그 값도 같은 자리에서 바로 보입니다.
- 수용 기준:
  1) `GET /api/v1/admin/audit` 의 각 item 에 `metadata` 키가 있다 — 행의 metadata 가 NULL 이면 `null`, JSON 이면 디코딩된 객체(`before`/`after` 와 같은 방식).
  2) 관리자 화면 `/admin/audit` 상세 모달(`web/src/pages/AdminPages.tsx:575` `Audit` 컴포넌트의 `<Modal title="감사 상세">`)에서 metadata 가 있을 때만 "부가 정보" 블록이 `변경 전/변경 후` 와 같은 `<pre>{JSON.stringify(...)}</pre>` 형식으로 보이고, 없으면 블록 자체가 렌더되지 않는다(기존 두 섹션은 그대로).
  3) 테스트가 증명할 것: 목록 행 → 응답 item 변환이 metadata NULL 을 `null` 로, 유효 JSON 을 객체로, 깨진 바이트를 `null`(before/after 와 동일한 관용) 로 내보내는 것. 이를 위해 `adminAudit` 안의 행→map 변환(현재 104~114행의 Scan 뒤 부분)을 `auditItem(...)` 같은 패키지 내 순수 함수로 떼어내고 핸들러가 그 함수를 호출하게 하여 프로덕션 경로 그대로 검증한다(가짜 DB 주입 금지 — 이 저장소 server 패키지 테스트는 모두 DB 없는 단위 테스트).
  4) `internal/api/openapi.go:148` 의 `/admin/audit` 항목은 `get("…", "admin:read")` 요약만 있어 스키마가 없으므로 계약 테스트(`internal/server/openapi_contract_test.go`) 변경은 불필요 — 단, 새 쿼리 파라미터를 더하지 않으면 그대로 통과함을 실행으로 확인.
  5) `docs/ADMIN_GUIDE.md` 5.3 절(317~321행) 한 문장 추가: 상세에서 부가 정보(로그인 방식 등)를 볼 수 있다. PDF 재생성은 하지 않음(별도 보류 항목).
- 건드릴 파일:
  - `internal/server/admin_operations.go:adminAudit` — SELECT 에 `metadata` 추가, Scan 에 `var meta []byte` 추가, 항목 map 에 `"metadata": m` 추가. 행→map 변환을 순수 함수로 추출.
  - `internal/server/admin_operations_test.go`(신규 또는 기존 admin 테스트 파일 옆) — 위 3) 의 테스트 3케이스.
  - `web/src/pages/AdminPages.tsx` `Audit` 컴포넌트 575행 모달 — `selected.metadata` 가 null/undefined 가 아닐 때만 세 번째 `<section><h3>부가 정보</h3><pre>…</pre></section>` 렌더. 파일이 한 줄 압축 스타일이므로 그 관례를 그대로 따를 것.
  - `docs/ADMIN_GUIDE.md` 5.3 절 — 한 문장.
- 검증 명령(이 저장소에서 실제로 도는 것, CI 순서와 동일):
  - `go test -race ./internal/server/ ./internal/audit/` 그 다음 `go test -race ./...`
  - `go vet ./...` · `gofmt -l internal cmd` (출력 없어야 함)
  - `cd web && npm ci && npm run typecheck && npm run build && npm test`
  - `./scripts/check-env-contract.sh` · `./scripts/check-static-assets.sh` · `./scripts/previous-release-tag-test.sh`
  - 새 테스트가 실제 결함을 잡는지: `"metadata"` 키를 map 에서 빼고 돌려 실패하는 것을 한 번 확인한 뒤 되돌릴 것.
- 위험과 피할 것:
  - `internal/audit/service.go` 의 Record/INSERT 와 `nullableJSON` 은 건드리지 말 것 — 쓰는 쪽은 이미 맞고, 이번 과제는 읽는 쪽만.
  - `internal/server/keys_approval.go:65` 의 본인 활동 목록(`/me` 쪽 audit 질의)은 범위 밖 — metadata 에는 관리자용 정보(bootstrap 여부 등)가 있으므로 일반 사용자 화면에 노출하지 않는다.
  - 운영자 규칙 "감사 details 에는 스크럽 전 원문을 넣지 말 것" 은 쓰기 쪽 규칙이라 이번엔 해당 없음. 새 metadata 를 **추가로 기록**하지 말 것(효과 없는 변경·범위 확장).
  - 마이그레이션 불필요(열은 001 부터 존재). migrations/, auth/, workflows/ 는 손대지 않는다.
  - `web/src/pages/AdminPages.tsx` 는 극단적 한 줄 스타일 — 포매터를 돌리지 말 것(diff 가 파일 전체가 됨).
  - 러너 검증은 새 worktree 에서 web build 전에 `go build` 를 돌린다 — `internal/webui/dist/README` 앵커가 이미 커밋돼 있으므로 문제 없지만, web build 뒤 `git status` 가 깨끗한지 확인.
- 차선 후보: `security.allowed_origins` 시드 행 제거(가치 2 / 위험 1 / S) — 마이그레이션 `015_drop_allowed_origins.sql`(이 브랜치의 마지막은 `014_momento_provider.sql` — 확인함; 015 는 다른 미머지 브랜치가 메일 알림에 쓰고 있어 충돌 가능, 머지 순서에 따라 번호를 다시 볼 것) 로 `DELETE FROM system_settings WHERE namespace='security' AND key='allowed_origins'` 하고 `docs/ADMIN_GUIDE.md` 설정 표에서 그 행을 빼기. 코드 어디에서도 읽지 않음은 2026-09-17 정찰이 확인(이번 회차 미재확인 — 착수 전 `grep -rn allowed_origins internal web` 로 0건인지 볼 것). 착수 전 `ls migrations` 로 번호 재확인.
