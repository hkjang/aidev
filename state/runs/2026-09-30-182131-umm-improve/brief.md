- 과제: 문서 차례(outline) 내려받기 이름에 공간 이름 담기 (가치 2 / 위험 1 / 작업량 S)

- 왜: `internal/httpapi/export_handlers.go:101` 의 `exportOutline` 만 리터럴 헤더 ``attachment; filename="umm-outline.md"`` 를 보낸다(직접 열어 확인). 같은 패키지의 다른 두 내보내기 경로는 이미 사람 이름을 담는다 — 백업은 `export_handlers.go:241` 이 `attachmentDisposition("umm-"+spaceName, ".md")`, handoff 수령은 `handoff_handlers.go:260` 이 `attachmentDisposition(strings.TrimSuffix(doc.Filename, ".md"), ".md")`. 그래서 여러 공간의 차례를 받으면 브라우저가 `umm-outline (1).md`, `(2)` 로 쌓아 어느 공간 것인지 파일만 보고는 알 수 없고, curl -OJ / 백업 스크립트 같은 API 클라이언트는 덮어쓴다. 고치면 이 한 경로가 나머지 두 경로와 같은 규칙을 따른다.

- 수용 기준:
  1) `GET /api/v1/spaces/{id}/export/outline` 응답의 `Content-Disposition` 이 그 공간의 이름을 담고, `mime.ParseMediaType` 으로 읽었을 때 `attachment` 이며 `params["filename"]` 에 공간 이름이 들어 있다(따옴표·한글이 든 이름도 깨지지 않음 — `attachmentDisposition` 이 RFC 6266 `filename*` 로 처리).
  2) 같은 공간의 백업 내보내기 이름(`umm-<공간이름>.md`)과 **서로 다른 문자열**이다. 둘은 내용이 다른 파일이라 같은 이름이면 안 된다. 권장 stem 은 `"umm-outline-"+spaceName` (ASCII 대체 이름은 `umm-outline-.md` 로 남아 여전히 무엇인지 알 수 있음 — `asciiFallback` 이 비ASCII 런을 대시 하나로 접고 `trimmed == extensionOf(name)` 일 때만 fallback 으로 떨어지는 것을 `content_disposition.go:83-102` 에서 확인).
  3) 이름 없는/공백뿐인 공간에서도 헤더가 읽히는 상태로 남는다(`disposition()` 이 stem 이 비면 `"umm"` 으로 채움 — 별도 코드 불필요, 단언만).
  4) 시험은 **실제 라우터 + 실제 PostgreSQL** 을 지나야 한다. 손으로 만든 대역이나 `exportOutline` 직접 호출로 증명하지 말 것.

- 건드릴 파일 (생산 코드 1개, 시험 1개):
  - `internal/httpapi/export_handlers.go:exportOutline` — (a) `CanViewSpace` 통과 뒤 `handoff_handlers.go:202-203` 과 같은 모양으로 공간 이름을 한 번 읽고(`SELECT name FROM spaces WHERE id=$1`, `s.Store.Pool.QueryRow`), (b) 101행의 리터럴 헤더를 `attachmentDisposition(...)` 호출로 교체. 새 헬퍼를 만들지 말고 같은 패키지의 기존 `attachmentDisposition` 을 쓸 것.
  - `internal/httpapi/presentation_integration_test.go` — 이미 `:79` 에서 `router.Get("/spaces/{spaceID}/export/outline", server.exportOutline)` 를 등록하고 `:941 outlineFor` 로 실제 응답을 받는 하네스가 있고 `spaceName` 변수도 그 자리에 있다. 여기에 헤더 단언을 더하는 것이 가장 짧다(`outlineFor` 가 본문 문자열만 돌려주므로 응답 레코더를 쓰는 갈래 하나를 더하거나 반환을 넓힐 것).

- 검증 명령 (이 저장소에서 실제로 도는 것):
  - 격리 PostgreSQL 17 이 **지금 이 환경에 없다**(`docker ps` 에 `umm-test-pg`/`umm-e2e-pg` 없음 — 확인함. 다른 프로젝트의 DB 를 빌려 쓰지 말고 새로 띄울 것).
  - `POSTGRES_DSN=<격리 PostgreSQL 17 DSN> go test ./internal/httpapi -run 'Outline|Disposition|NamesTheFile' -count=1 -v`
    — `POSTGRES_DSN` 이 없으면 통합 시험은 `t.Skip` 한다. **PASS 표시만 보고 돌았다고 판정하지 말고 SKIP 이 아닌지 -v 로 확인할 것.**
  - `POSTGRES_DSN=<같은 DSN> go test -p 1 ./... -count=1` (CI 표준 — `app_settings` 공유 충돌 때문에 `-p 1` 필수. `make test-go` 는 아직 `-p 1` 이 없으므로 그대로 믿지 말 것)
  - `go vet ./...`, `gofmt -l internal/httpapi`
  - 적색 확인: 고치기 전 코드에서 새 단언이 실제로 실패하는 것(`filename = "umm-outline.md"`)을 한 번 보고 기록할 것.
  - 웹/Playwright 는 이번 변경 범위 밖이다(웹 캔버스는 자기 내려받기 이름을 따로 정한다 — `content_disposition_test.go:122-123` 주석). 돌리지 말 것.

- 위험과 피할 것:
  - **파일 이름 새니타이즈 3곳(`store.safeFilename` / `httpapi.dispositionSafe` / `handoffFilename`)을 통합하려 들지 말 것.** 계약이 서로 다르고 과거에 반복해 보류된 자리다. 이번엔 `attachmentDisposition` 을 호출만 한다.
  - 백업 내보내기(`exportMarkdown`)의 이름과 `handoff` 의 이름은 **바꾸지 말 것.** 기존 단언(`content_disposition_test.go:173` 의 `"umm-9월 회의.md"`, `handoff_integration_test.go:195`)이 그대로 통과해야 한다.
  - outline 본문(`presentation.Outline` 결과)과 감사 로그(`space.export`, `format: "outline"`)는 건드리지 말 것 — 2026-09-29 회차가 막 손댄 자리다.
  - 공간 이름 조회 실패 시 500 으로 죽이지 말고 이름 없이도 내보내기는 되게 할 것(빈 이름 → `disposition()` 이 `umm` 으로 채움). 이미 `CanViewSpace` 를 지난 뒤라 권한 판정을 다시 하는 것이 아니다.
  - 보호 경로(`internal/auth/`, `migrations/`, `.github/workflows/`)는 이번 과제에서 건드릴 일이 없다.
  - 추정 근거(basis): 생산 코드 약 6줄 + 시험 단언 1갈래, 선례 두 곳(241행·260행)을 그대로 복제하는 유사 추정. 신뢰 구간은 20~40분이며, 늘어난다면 원인은 격리 PostgreSQL 을 새로 띄우는 준비 시간 한 가지다(우발분 15분).

- 차선 후보: `make test-go` 를 CI 와 같은 직렬 실행으로 맞추기 (가치 3 / 위험 1 / S) — `Makefile:8` 이 `go test ./...` 인데 `.github/workflows/ci.yml` 은 `app_settings` 공유 충돌 때문에 `-p 1` 을 쓴다. 로컬에서만 간헐 실패하는 차이를 없애는 한 줄. 1순위가 성립하지 않으면(예: outline 라우트가 이름을 얻을 수 없는 구조로 드러나면) 이것을 고를 것.
