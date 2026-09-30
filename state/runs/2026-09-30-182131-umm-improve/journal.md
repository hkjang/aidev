# 회차 노트 2026-09-30-182131-umm-improve — umm
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:21] base pinned — main@5bf8edc
- [러너 18:21] autonomy release — 

## 정찰 노트
- 골랐다: outline 내보내기의 고정 파일 이름(export_handlers.go:101). 가치는 2로 낮지만 같은 패키지 안에 선례 두 곳(:241, handoff_handlers.go:260)과 이미 outline 라우트를 등록한 시험 하네스(presentation_integration_test.go:79/941)가 있어 생산 파일 1개로 끝나고 적색 확인이 쉽다.
- 제친 이유: 오프라인 큐 401/403(가치 3)은 api.ts:667 을 열어 보니 항목 단위 거부를 isTerminalOfflineRejection 으로 이미 따로 처리하고 401/403 중단에 근거 주석까지 있는 의도된 설계라 기각했다. 파일 이름 새니타이즈 3곳 통합은 계약이 실제로 달라 위험 4로 올려 기각. 차선인 Makefile -p 1 은 "시험이 무엇을 증명하나"를 세우기 어려워 2순위.
- 확신 없는 곳: 권장 stem `"umm-outline-"+spaceName` 과 그 ASCII 대체가 `umm-outline-.md` 가 된다는 것은 asciiFallback 을 눈으로 따라간 결과이지 실행해 본 값이 아니다. 공간 이름 조회를 exportOutline 안에 넣는 위치도 handoff 패턴 복제 제안일 뿐 실행 미확인.
- 구현자가 조심할 것: 이 환경에 umm-test-pg/umm-e2e-pg 가 **없다**(docker ps 확인). 다른 프로젝트 DB 를 빌리지 말고 새로 띄우고, -v 로 SKIP 이 아닌지 볼 것. 기존 백업/handoff 이름 단언은 그대로 통과해야 한다.
- 프로필은 1일 전 것이고 이번에 연 파일들(export_handlers.go, handoff_handlers.go, content_disposition.go, Makefile, api.ts)과 어긋나지 않아 다시 쓰지 않았다.
- [러너 18:25] scout done — 문서 차례(outline) 내려받기 이름에 공간 이름 담기 (가치 2 / 위험 1 / 작업량 S)

## 구현 노트
- exportOutline 만 리터럴 `umm-outline.md` 를 보내 공간을 구분할 수 없었다. CanViewSpace 뒤에 `SELECT name FROM spaces WHERE id=$1` 로 이름을 한 번 읽고 기존 `attachmentDisposition("umm-outline-"+spaceName, ".md")` 로 교체했다(생산 파일 1개, 새 헬퍼 없음, 새니타이즈 통합 없음).
- 확신 없는 곳: 이름 조회 실패를 `_ =` 로 삼킨다 — 권한은 이미 CanViewSpace 로 판정된 뒤라 의도한 선택이지만, 그 갈래(공간 행이 사라진 경우)는 시험으로 덮지 않았다. 빈 이름 경로는 공간 이름을 ""/"   " 로 넣어 덮었다.
- 확신 없는 곳: 웹 캔버스가 자기 내려받기 이름을 따로 정하므로 이 변경은 브라우저 UI 에 안 보인다 — 웹/Playwright 는 과제서대로 돌리지 않았다.
- 일부러 안 한 것: 백업(`umm-<이름>.md`)·handoff 이름, outline 본문, 감사 로그(`format: "outline"`)는 그대로 두었다. `?title=` 을 파일 이름에 반영하는 것도 일부러 뺐다(검증 표면 증가).
- 다음 역할이 조심할 것: 새 시험 3개(`TestMarkdownExportNamesTheFileIntegration`, `TestOutlineExportNamesTheFileIntegration`, `TestOutlineExportNamesAnUnnamedSpaceIntegration`)는 **POSTGRES_DSN 없으면 t.Skip** 한다. PASS 표시만 보지 말고 `-v` 로 SKIP 이 아닌지 볼 것. 이 세션은 도커 `umm-test-pg`(postgres:17, 127.0.0.1:15433)를 새로 띄워 돌렸고 컨테이너는 남겨 두었다.
- 기존 markdown 이름 시험은 새 `exportNameHarness` 로 옮겨 썼지만 단언(`umm-9월 회의.md`)은 글자 그대로 유지했고 통과를 확인했다.
- [러너 18:29] brief accepted — 채택 — 과제서의 근거(101행만 리터럴, :241·handoff:260 선례, `asciiFallback` 이 `umm-outline-.md` 를 남긴다는 예측)가 모두 지금 �
- [러너 18:30] verify passed — 검증 9개 통과 (auto)

## 비평 노트
- 확인: 생산 코드만 main 으로 되돌려 적색을 직접 재현했다(`umm-outline.md` 로 세 단언 실패) — 원장에 `- 실패 재현:` 줄이 없어 리뷰어가 대신 했다. 복원 후 `POSTGRES_DSN`(umm-test-pg:15433) 로 `-v` 실행해 SKIP 아님 확인, `go test ./internal/httpapi` 전체·vet·build 통과.
- 보안·법무 차단 없음: 새 쿼리는 `$1` 바인딩 + 이미 통과한 CanViewSpace(owner/member, store.go:844) 뒤이고 exportMarkdown:171 과 권한 범위가 같다. dispositionSafe 가 제어문자·따옴표·구분자를 제거하므로 헤더 주입 경로도 없다. 새 개인정보 수집·전송 없음.
- 못 본 것: web typecheck/lint/vitest/build, check-i18n, Playwright, `go test -p 1 ./...` 전체. 변경이 Go 1줄이고 웹은 CanvasPage.tsx:1990 에서 자기 이름을 붙이므로 값이 낮다고 판단했다.
- 승인이어도 남는 우려(릴리즈 노트용): 이름 모양이 세 곳에서 어긋난다 — 서버 `umm-outline-<이름>.md` / 백업 `umm-<이름>.md` / 웹 `umm-<이름>-차례.md`. 다른 공간 사이 충돌(`outline-A` 의 백업 vs `A` 의 outline)은 남지만 회귀는 아니다.
- 다음 회차가 알 것: `_ =` 로 삼킨 이름 조회 실패 갈래는 시험이 없다(결과는 빈 이름 경로와 동일해 안전). docs/openapi.yaml:702 은 파일 이름을 문서화하지 않아 갱신할 문서가 없다. 도커 `umm-test-pg` 는 살아 있고 그대로 두었다.
- [러너 18:33] review approved — 리뷰 승인 (risk=low)
- [러너 18:33] pr created — https://github.com/hkjang/umm/pull/162
- [러너 18:37] ci failed — 성공이 아닌 검사: verify=failure
