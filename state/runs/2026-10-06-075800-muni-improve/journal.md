# 회차 노트 2026-10-06-075800-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:58] base pinned — main@8b418fb
- [러너 07:58] autonomy release — 

## 정찰 노트
- 보류 1순위(ZIP 문서 정렬 동률)를 골랐다. 가치를 2→3 으로 올린 근거는 `documents.go:124` 와 `NewDocumentDialog.tsx:106` 이 제목 없는 문서 전부에 같은 `제목 없는 문서` 를 준다는 것 — 동률은 엣지케이스가 아니라 상례다. 프로덕션 1파일·SQL 한 줄로 끝나고 최근 세 회차가 손댄 함수들을 전혀 건드리지 않는다.
- 제친 후보: `safeFilename` Windows 금지 문자(리눅스에서 실동작 증명 불가, 다섯 경로 공유로 파급 큼), `createFolder` 형제 중복(계약 판단 선행), e2e CI(보호 경로). 새로 조사한 두 건 `splitTableRow` 끝 파이프와 마크다운 표 파이프 이스케이프는 실제로 올바르게 처리되고 있어 기각했다(이전 프로필 서술이 결함처럼 읽혔던 것).
- 추측으로 적은 것(미확인): 수정 전 실패를 만들 수 있는지. 테스트·`go`·DB 를 이번 세션에서 한 번도 돌리지 않았다. PostgreSQL 이 같은 힙 순서면 같은 결과를 줄 수 있어 재현이 안 될 가능성이 실재하며, 과제서에 PATCH 로 힙 순서를 흔드는 1순위 방법과 실패 시 처신(회귀 감시자 + EXPLAIN 근거, 안 되면 차선)을 적어 두었다.
- 구현자가 조심할 것: 프로브가 항목 이름 **집합**을 비교하면 이 결함을 못 잡고 거짓 통과한다 — 이름→본문(또는 `목록.md` 의 작성자) **대응**을 비교할 것. 2026-10-05 의 폴더 프로브를 그대로 복사하면 여기서 걸린다.
- [러너 08:03] scout done — 워크스페이스 ZIP 문서 쿼리의 정렬 동률을 문서 ID 로 깨서 같은 이름 문서가 어느 파일에 담길지 고정하�

## 구현 노트
- `workspace_export.go:71` 의 `ORDER BY` 에 `, d.id` 하나를 더했다. 제목 없는 문서가 전부 같은 제목을 받으므로 동률이 상례이고, 그 순서가 `(2)` 접미사와 2000건 경계를 정해 바뀌지 않은 워크스페이스를 두 번 내려받으면 파일 내용과 `목록.md` 의 작성자가 서로 바뀔 수 있었다. 프로덕션 1파일, 테스트 1파일(커밋 cdffbfe).
- **확신 없는 곳**: 없다고 본다 — 비결정성을 추론이 아니라 두 단계로 증명했다. `EXPLAIN (ANALYZE)` 가 `Sort Key: d.folder_id NULLS FIRST, d.title` / `Sort Method: quicksort` 를 내놓고, 동률 네 행 중 **한 행만** 다시 쓰면 정렬 출력이 `body1,2,3,4` → `body2,3,4,1` 로 바뀐다. 그 위에서 live 테스트가 수정 전 실패 → 수정 후 통과, `, d.id` 만 되돌리면 그 테스트만 다시 실패(나머지 246 PASS / SKIP 0).
- **검증 못 한 것**: 실제 Windows/macOS 에서의 압축 해제는 리눅스 워크트리에서 재현 불가 — 증명 범위는 ZIP 항목 이름→본문 대응까지다. 프런트 미변경이라 npm 검사·`make build` 는 돌리지 않았다(`make build` 는 tracked `webui/dist/index.html` 을 덮는다).
- **일부러 하지 않은 것**: `uniqueEntryName`·`entryKey`·`folderPaths`·`safeFilename`·`LIMIT`/`truncated` 판정은 한 글자도 건드리지 않았다(최근 세 회차가 고친 자리이고 지금 상태가 맞다). 차선 후보였던 OPERATIONS 백업 문서는 손대지 않았다.
- **다음 역할이 조심할 것**: 새 테스트는 **DB 가 있어야 돈다**(`MUNI_TEST_DSN` 없으면 SKIP 이고 그것은 통과가 아니다). 그리고 이 테스트는 **정확히 한 행만** `title=title` 로 다시 쓴다 — 네 행을 전부 다시 쓰면 상대 순서가 보존되어 수정 없이도 통과한다(첫 시도가 그래서 거짓 통과했다). 이 줄을 '네 개 다 쓰는 게 더 세다' 고 바꾸면 테스트가 결함을 못 본다.
- [러너 08:10] brief accepted — 채택 — 지정한 프로덕션 1파일·SQL 한 토큰과 수용 기준 셋을 그대로 구현했고, 과제서가 미확인으로 남긴 "수정 전 실패
- [러너 08:11] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 승인. 구현 노트가 "확신 없는 곳: 없다" 고 적은 그 확신을 직접 시험했다 — 전용 postgres:16-alpine 을 띄워 새 테스트 PASS → `, d.id` 한 토큰만 되돌려 FAIL(`download 2 put 본문 가 in "제목 없는 문서 (4).md", but the first download put it in "제목 없는 문서.md"`) → 복구 후 `./internal/httpapi` PASS 247 / SKIP 0 / FAIL 0, `go test ./...` exit 0, `go vet` exit 0, `gofmt -l` clean. 원장의 실패 재현 줄과 증상이 일치한다(접미사 번호만 힙 순서에 따라 다름). 보안·법무 차단 없음: 새 입력·라우트·개인정보 없음.
- 못 본 것: 실제 Windows/macOS 압축 해제(리눅스에서 재현 불가 — 구현 노트와 같은 한계), 프런트(미변경).
- **릴리즈가 가장 먼저 할 일**: 자동 검증이 `npm run build` 를 돌려 tracked `webui/dist/index.html` 이 빌드 번들로 바뀌어 있고 `scripts/check-webui-placeholder.sh` 가 **지금 exit 1** 이다. 태깅 전에 `git checkout -- webui/dist/index.html`. 커밋에는 안 들어갔으므로 PR diff 자체는 깨끗하다.
- 다음 회차가 알 것: `verify.json` 의 `go test ./...` 는 2초·DSN 없음 = live 전부 SKIP 이었다. 'auto 검증 7개 통과' 는 이 새 테스트를 실행하지 않았다는 뜻 — 자동 게이트의 SKIP 을 통과로 읽지 말 것.
- 남는 우려(차단 아님): 테스트의 `if len(want) != len(markers)` 가드는 항상 거짓이라 주석이 말하는 일을 하지 않는다(실제 보증은 바로 뒤 `HasPrefix`). 그리고 `adminWorkspace` 의 `LIMIT 1` 에도 ORDER BY 가 없고 live 테스트가 admin 워크스페이스에 문서를 남긴다 — 항목 수를 세는 테스트를 새로 쓰면 걸린다.
- [러너 08:16] review approved — 리뷰 승인 (risk=low)
- [러너 08:16] pr created — https://github.com/hkjang/muni/pull/35
- [러너 08:22] ci passed — 검사 2개 모두 success
- [러너 08:22] merge done — cdffbfe
