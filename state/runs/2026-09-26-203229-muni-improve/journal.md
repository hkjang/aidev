# 회차 노트 2026-09-26-203229-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:32] base pinned — main@44ed66d
- [러너 20:32] autonomy release — 

## 정찰 노트
- 보류 목록에서 값이 가장 높고(3) 위험이 낮은 항목을 골랐다: 헤더를 내는 다섯 자리 중 `workspace_export.go:111` 만 지난 회차의 `extValueEscape` 를 못 받은 것을 코드로 확인했고(다른 넷은 모두 호출), 워크스페이스 이름이 80룬·공백제거만 검사받아 큰따옴표가 들어갈 수 있는 것도 `workspaces.go:55` 에서 확인했다. 프로덕션 파일 하나로 끝난다.
- 제친 후보: 한도 초과 안내(2000건 재현이 무거워 차선), 헤더 헬퍼 통합(재발 방지뿐이고 순서상 이번 수정 뒤), PDF 첫 줄 제거(여섯 회차째 '첫 블록이 heading 인지' 미확인 — 확인 절차를 ideas.json 에 적어 뒀다), e2e CI(보호 경로).
- 확신 없는 곳: `mime.ParseMediaType` 이 quoted-string 안의 raw 한글 UTF-8 을 실제로 거부하는지는 실행 미확인 — 이 워크트리에서도 `go` 가 승인 대기로 막혔다(2026-09-25 와 동일). 따옴표·쌍반점 쪽 깨짐이 더 단단한 근거이니 그 재현으로 시작할 것.
- 조심할 것: `safeFilename` 은 다섯 경로 공유이므로 따옴표 제거 방식으로 고치지 말 것(ZIP 항목 이름과 `목록.md` 기록이 어긋난다). 헤더는 첫 문서 렌더 전에 나가니 새 실패 경로를 만들지 말 것. 기존 `workspace_export_live_test.go` 넷은 손대지 않고 통과해야 한다.
- 프로필은 0일 전 것이 지금 코드와 맞아(그 뒤 변경은 릴리스 노트 커밋뿐) 새로 쓰지 않았다. 다만 "기준 5d33785 / 0.45.0" 은 현재 44ed66d / 0.46.0 이다.
- [러너 20:36] scout done — 워크스페이스 ZIP 내려받기의 `Content-Disposition` 에도 `filename*` 을 주고 따옴표가 헤더를 깨지 않게 하기 (가�

## 구현 노트
- 워크스페이스 ZIP 의 `Content-Disposition` 한 줄만 바꿨다(`workspace_export.go`): 이름을 `extValueEscape` 를 지난 `filename*` 로 보내고 ASCII fallback 은 `filename="workspace-YYYYMMDD.zip"` 로 날짜만 남긴다. 날짜는 한 번 계산해 두 자리에 쓴다. 이로써 헤더를 내는 다섯 자리가 같은 규칙이 됐다.
- 재현: 새 live 테스트가 고치기 전 `mime: invalid media parameter` 로 실패했다 — 정찰이 지목한 한글 raw 바이트가 아니라 **이름의 큰따옴표** 때문에 파서가 헤더 전체를 버린 것이다. 고친 뒤 통과하고, 헤더 한 줄만 되돌리면 그 테스트 하나만 다시 실패한다(233건은 계속 통과).
- 확신 없는 곳: 한글만 있고 따옴표가 없는 이름을 `mime.ParseMediaType` 이 quoted-string 에서 거부하는지는 여전히 실행으로 확인하지 않았다(따옴표 쪽이 먼저 터져서 분리 측정을 하지 않음). `filename*` 이 있는 지금은 어느 쪽이든 결과가 같아 실질 영향은 없다.
- 일부러 하지 않은 것: 과제서가 선택으로 둔 `content_disposition_test.go` 단위 표 추가(live 가 이미 프로덕션 배선을 지난다), 다섯 자리 헬퍼 통합(다음 회차 후보로 ideas.json 에 남김), `safeFilename`·ZIP 항목 이름·`목록.md`·한도 안내는 손대지 않음.
- 다음 역할이 조심할 것: 새 테스트 `TestAWorkspaceArchiveNamesItselfSoAParserReadsTheNameBack` 은 DB 가 있어야 돈다(`MUNI_TEST_DSN`, postgres:16-alpine). 워크스페이스를 slug `quote-zip` 으로 만들고 `t.Cleanup` 에서 `workspace_members`·`workspaces` 를 지우므로, 정리가 안 된 채 죽으면 다음 실행이 slug 충돌로 실패한다. 프런트 미변경이라 npm 검사는 돌리지 않았다.
- [러너 20:40] brief accepted — 채택 — 근거(다섯 자리 중 이 한 곳만 `filename*` 이 없고 `safeFilename` 결과를 quoted-string 에 원문으로 넣음, 워크스페이스 이
- [러너 20:41] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정: approve (risk low, blocking 없음). 임시 postgres:16-alpine 을 직접 띄워 `MUNI_TEST_DSN` 으로 `go test ./... -count=1` 전체 통과를 확인했고(live·Chromium 포함 실패 0), 헤더 한 줄만 이전 코드로 되돌리자 새 테스트가 주장된 증상 그대로 실패했다(`mime: invalid media parameter`). 되돌린 파일·컨테이너·테스트 행 모두 원상 복구.
- 구현 노트의 '확신 없는 곳'은 무해함을 확인했다: `mime.ParseMediaType` 은 quoted-string 값을 넣은 뒤 continuation 스티칭에서 `filename*` 디코드 값으로 덮으므로, `filename*` 이 있는 한 raw UTF-8 여부가 결과를 바꾸지 않는다.
- 남는 우려(릴리즈가 볼 것): ① ASCII fallback 에서 워크스페이스 이름이 빠져 `workspace-YYYYMMDD.zip` 이 된다 — 릴리스 노트에 한 줄. ② 이 워크트리의 커밋 안 된 `webui/dist/index.html` 때문에 `scripts/check-webui-placeholder.sh` 가 exit 1 — `make test` 전에 `git checkout --` 로 되돌릴 것.
- 다음 회차 후보: 새 테스트의 고정 slug `quote-zip` 은 자가 치유가 안 된다(잔여 행을 남겨 재현했더니 `create workspace = 409`). 생성 전 선삭제나 임의 slug 로. 그 행이 남으면 `adminWorkspace` 의 ORDER BY 없는 `LIMIT 1` 이 잔여 행을 집을 수도 있다(그래도 기존 테스트는 통과).
- 못 본 것: 프런트·e2e 는 변경이 없어 돌리지 않았고, 브라우저 실제 저장 이름은 확인하지 않았다(파서 왕복으로만 검증).
- [러너 20:46] review approved — 리뷰 승인 (risk=low)
- [러너 20:46] pr created — https://github.com/hkjang/muni/pull/27
- [러너 20:51] ci passed — 검사 2개 모두 success
- [러너 20:51] merge done — a774de9
