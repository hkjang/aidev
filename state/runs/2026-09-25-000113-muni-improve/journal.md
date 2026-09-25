# 회차 노트 2026-09-25-000113-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 00:01] base pinned — main@5d33785
- [러너 00:01] autonomy release — 

## 정찰 노트
- 고른 것: `Content-Disposition` 의 `filename*` RFC 8187 인코딩. 저장소가 스스로 남긴 증거가 있어서 골랐다 — `export_filename_live_test.go:88-96` 이 "이 헤더는 아직 `mime.ParseMediaType` 을 못 통과한다, 고치는 건 별도 변경" 이라고 적어 뒀고, 같은 패키지 `handoff.go:147` 은 이미 반대 규칙(`url.PathEscape` + RFC 8187 주석)을 쓴다. 같은 값을 내는 경로가 갈라져 있는 전형적인 자리다.
- 제친 것: ZIP 한도 경계(2/1/S)는 가치가 낮아 차선으로, PDF 첫 줄 제목(2/2/S)은 다섯 회차째 "첫 블록이 heading 인지" 미확인이라 또 미뤘다, CI e2e(4/2/M)는 보호 경로라 자율 회차에서 반복 기각된 유형이라 뺐다.
- 추측으로 적은 것: (a) `;` `=` `'` 가 들어간 제목에서 `url.PathEscape` 만으로는 부족하다는 것 — 논리로 적었고 실행으로 확인하지 못했다. (b) 첨부·발표자료 경로의 증상이 문서 내려받기와 동일하다는 것 — 코드는 같은 헬퍼를 쓰지만 live 로 보지 않았다.
- 구현자가 조심할 것: **기존 live 테스트의 기대값이 반드시 깨진다**(`export_filename_live_test.go:70-96`) — 회귀가 아니라 이 과제의 일부다. 그리고 `workspace_export.go:111` 은 같은 증상군이지만 이번 범위 밖으로 두었다.
- 이 세션 한계: 샌드박스에서 `go` 실행이 승인 대기로 전부 막혀 테스트를 한 번도 돌리지 못했다. 저장소 코드는 수정하지 않았다.
- [러너 00:04] scout failed — 과제서 없음 — 구현자가 직접 고른다

## 구현 노트
- 무엇을 왜: `Content-Disposition` 의 `filename*` 를 RFC 8187 ext-value 규칙으로 인코딩하는 `extValueEscape`(export.go)를 두고, 문서 내려받기·발표자료·첨부·넘겨받기 네 경로가 같은 규칙을 쓰게 했다. 한글 제목이 파싱되지 않던 것 외에 제목 속 `;` 가 매개변수를 끊어 이름을 조용히 잘라 먹던 두 번째 증상도 함께 닫혔다. 커밋 299b4dc.
- 확신 없는 곳: **실제 브라우저가 저장하는 이름은 확인하지 못했다** — 검증은 `mime.ParseMediaType`(= `internal/handoff` 의 `filenameOf` 가 쓰는 파서)으로만 했다. 브라우저는 예전에도 raw UTF-8 을 관용적으로 받아 주었을 가능성이 있어, 사용자 체감 변화가 "이름을 잃었다 → 되찾았다" 인지 "이미 됐는데 규격만 맞췄다" 인지는 미확인. 프로그램 수신자(handoff)에게는 확실히 결함이었다. 발표자료 경로(presentations.go:243)는 외부 ptium 서버가 필요해 live 로 보지 않았고 단위 테스트와 코드 동일성으로만 덮었다.
- 일부러 하지 않은 것: `workspace_export.go:111`(ZIP 내려받기)은 `filename*` 자체가 없고 `filename="…"` 안에 따옴표 포함 원문을 넣는 별개 결함이라 범위 밖으로 두고 ideas.json 에 3/2/S 로 올렸다. `admin.go:306` 의 감사 CSV 는 고정 ASCII 이름이라 손대지 않았다. `safeFilename` 과 절단 길이 100, ASCII fallback 문자열은 한 글자도 바꾸지 않았다.
- 다음 역할이 조심할 것: 새 live 테스트 3개(`content_disposition_live_test.go`)는 `MUNI_TEST_DSN` 이 있어야 돈다 — 없으면 SKIP 이므로 단위만 통과한 것을 통과로 읽지 말 것. `export_filename_live_test.go` 의 기대값을 바꿨고(escaper 문자열 비교 → 파서 왕복) 알려진 공백 주석·`t.Logf` 를 걷어냈다 — 회귀가 아니라 이 변경의 일부다. 헬퍼 `longTitledDocument` 를 `documentTitledOverTheAPI` 로 이름만 바꿨다. 프런트는 손대지 않아 `npm run lint`/`npm test` 는 돌리지 않았다(`frontend/node_modules` 없음).
- [러너 00:31] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 approve(risk low). 구현 노트의 최대 공백을 메웠다: docker 로 postgres:16-alpine 을 띄워 MUNI_TEST_DSN 을 주고 `go test ./...` 전체를 돌렸고 전부 통과 — content_disposition_live_test.go 3개와 export_filename_live_test.go 2개가 SKIP 이 아니라 PASS 로 실행됐다. go vet·gofmt 도 깨끗.
- 새 테스트가 옛 코드에서 통과하지 않는 것도 확인했다: HEAD 를 /tmp 로 복사해 extValueEscape 본문만 옛 5문자 replacer 로 되돌리니 단위 다수와 다운로드 라이브 테스트가 `mime: invalid media parameter` 로 실패. 테스트는 진짜로 고정한다.
- 못 본 것: 실제 브라우저가 저장하는 이름(회귀 위험은 없다고 판단 — percent-encoded filename* 는 현행 브라우저 공통 구현), 그리고 발표자료 경로(ptium 서버 필요, 코드 동일성으로만).
- 승인이어도 남는 우려 — 릴리스 노트가 "내려받기 이름 전부 고쳐졌다"로 읽히지 않게 할 것: workspace_export.go:111 은 여전히 filename* 가 없고 `filename="…"` 안에 따옴표 미제거 이름을 넣어 헤더 전체가 파싱 거부될 수 있다(범위 밖, ideas.json 에 있음).
- 릴리즈가 머지 전에: 작업 트리의 커밋되지 않은 `webui/dist/index.html`(빌드 번들)을 `git checkout --` 로 되돌릴 것 — 이 PR 커밋에는 없지만 scripts/check-webui-placeholder.sh 가 실패한다.
- [러너 00:39] review approved — 리뷰 승인 (risk=low)
- [러너 00:39] pr created — https://github.com/hkjang/muni/pull/26
- [러너 00:44] ci passed — 검사 2개 모두 success
- [러너 00:44] merge done — 299b4dc
