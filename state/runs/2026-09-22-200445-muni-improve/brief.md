# 과제서 — 2026-09-22 (base main@1547faa)

- 과제: 마크다운 표에서 끝 구분자 없는 행의 마지막 셀에 있는 이스케이프된 파이프(`\|`)를 보존하기 (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/httpapi/import_markdown.go:577` 의 `splitTableRow` 가 이스케이프를 훑기 **전에** `strings.TrimSuffix(trimmed, "|")` 로 끝 파이프를 무조건 지웁니다. 그래서 GFM 이 허용하는 "끝 구분자를 생략한 행"에서 마지막 셀이 `\|` 로 끝나면 진짜 셀 내용인 파이프가 사라지고 역슬래시만 남아(`b\| ` → `b\`) 표 내용이 조용히 깨집니다. 한 함수 안에서 끝 구분자 판정을 이스케이프 인식으로 바꾸면 바깥에서 만든 마크다운을 가져올 때 셀 글자가 그대로 들어옵니다.

- 수용 기준:
  1) `markdownDocument("| 좌 | 우 |\n| --- | --- |\n| 가 | 나\\|\n")` 에서 마지막 셀의 텍스트가 `나|` 다(지금은 `나\`).
  2) 끝 구분자가 **있는** 기존 입력의 결과가 한 글자도 바뀌지 않는다: `| a | b |`, `| a | b`(이스케이프 없음), `a | b`, `| a\| | b |`, `| a | |`(빈 마지막 셀은 여전히 2칸), 구분자 행 `|---|---|`·`| :--- | ---: |`.
  3) `renderMarkdown` → `markdownDocument` 왕복이 바뀌지 않는다 — muni 의 내보내기는 항상 끝에 ` |` 를 붙이므로(`internal/httpapi/render.go:645` `markdownWriter.table` 의 `pad`), 파이프를 담은 셀의 왕복 JSON 이 수정 전후 동일해야 한다.
  4) 테스트가 증명할 것: 고치기 전에 (1) 이 실제로 `나\` 로 실패하는 것을 먼저 보고(구현자가 확인해 기록), 고친 뒤 (1)(2)(3) 이 모두 통과.

- 건드릴 파일:
  - `internal/httpapi/import_markdown.go:577 splitTableRow` — `TrimSuffix(trimmed, "|")` 를 없애고, `TrimPrefix` 뒤 문자열 전체를 기존 이스케이프 규칙 그대로 훑되 **마지막 문자로 소비된 이스케이프되지 않은 `|`** 였을 때만 끝의 빈 셀 하나를 만들지 않는 방식(예: 루프에서 `endedWithDelimiter` 를 기록하고 마지막 `current` 를 그 경우에만 append 하지 않음). 그 외 동작(선행 `|` 제거, `\|`→`|` 치환, 빈 문자열이면 `[""]`)은 그대로 둘 것.
  - `internal/httpapi/import_text_test.go` (또는 같은 패키지의 새 `import_markdown_test.go`) — `markdownDocument` 와 `renderMarkdown` 만 쓰는 순수 단위 테스트. DB 불필요. 기존 테스트가 `markdownDocument(...)` 로 content JSON 을 받아 블록을 뒤지는 방식(`import_text_test.go:74`, `:153`)을 그대로 따를 것.

- 검증 명령:
  - `cd /home/hkjang/.cache/auto-improve-wt/muni && go test ./internal/httpapi/ -run 'Table|Markdown' -count=1 -v`
  - `cd /home/hkjang/.cache/auto-improve-wt/muni && go test ./... ` (MUNI_TEST_DSN 없으면 live 테스트는 SKIP; 가능하면 postgres:16-alpine 컨테이너에 DSN 을 주어 httpapi SKIP 0 으로)
  - `cd /home/hkjang/.cache/auto-improve-wt/muni && go vet ./... && gofmt -l internal/httpapi`
  - `cd /home/hkjang/.cache/auto-improve-wt/muni && scripts/check-webui-placeholder.sh`
  - 기준선: 이번 정찰에서 `go test ./internal/httpapi/ -count=1` 이 (DSN 없이) `ok` 였다.

- 위험과 피할 것:
  - **범위를 넓히지 말 것.** `\\|`(역슬래시 두 개 뒤 파이프)를 GFM 처럼 "이스케이프된 역슬래시 + 진짜 구분자"로 해석하거나 `\\`→`\` 로 푸는 변경은 하지 말 것 — muni 의 내보내기는 `|` 만 이스케이프하고 역슬래시는 그대로 내보내므로(`render.go:648` 의 `ReplaceAll(text, "|", "\\|")`), 역슬래시를 풀면 `a\\b` 같은 셀의 왕복이 깨집니다. 이번에는 **끝 구분자 판정 한 가지만** 고칩니다.
  - `splitTableRow` 는 `isTableDelimiterRow`(`import_markdown.go:557`)와 헤더·정렬 파싱(`:602`,`:603`,`:648`)이 모두 씁니다. 한쪽만 맞추지 말고 구분자 행·헤더·본문 행이 같은 규칙으로 잘리는지 위 (2) 의 사례로 함께 확인할 것.
  - 표 셀 수가 달라지면 열 수(`columns`)가 흔들려 표 전체가 어긋납니다. 빈 마지막 셀(`| a | |`)이 사라지지 않는지 반드시 테스트에 넣을 것.
  - 보호 경로(auth/session/migrations/.github/workflows/settings 봉인)는 건드리지 않습니다. 프런트·문서 변경도 필요 없습니다(사용자에게 보이는 계약 변경이 아니라 조용한 데이터 손실 수정).
  - 이미 한 일 재시도 금지: 마크다운/HTML/DOCX 의 "첫 H1 중복 제목 제거", 워크스페이스 ZIP 의 `목록.md` 이름 충돌은 지난 회차들에서 이미 처리했습니다.

- 차선 후보: 워크스페이스 ZIP 의 한도 초과 안내를 실제로 넘쳤을 때만 넣기 — `internal/httpapi/workspace_export.go:155` 가 `len(items) == maxWorkspaceExport` 만 보고 정확히 2000건인 워크스페이스에도 "문서가 2000건을 넘어 그만큼만 담았습니다" 를 적습니다. `LIMIT maxWorkspaceExport+1` 로 읽고 2001번째가 있을 때만 안내하며 담기는 문서는 2000건으로 자르면 됩니다(가치 2 / 위험 1 / 작업량 S). 단, 이 파일은 `목록.md` 이름 충돌 수정과 같은 함수라 병합 충돌 가능성이 있습니다.

## 옵션 검토 (왜 이것인가)
- **가장 작은 것**: 끝 파이프 판정만 이스케이프 인식으로 — 선택. 한 함수, 한 규칙, 순수 단위 테스트, 왕복 불변.
- **GFM 이스케이프 전면 준수**(`\\`, `\*` 등): 값은 크지 않고 내보내기 쪽 이스케이프 계약(파이프만)과 어긋나 왕복 회귀 위험 — 기각.
- **아무것도 안 함**: 손실이 조용해서(글자가 `\` 로 바뀜) 사용자가 원인을 못 찾음 — 기각.
- 가장 크게 기대는 가정: muni 의 마크다운 내보내기가 **항상** 행 끝에 ` |` 를 붙인다는 것(`render.go:645` 의 `pad` 에서 확인). 이것이 참이므로 이번 수정으로 muni 자신의 왕복은 한 글자도 바뀌지 않습니다.

## 추정 근거 (basis of estimate)
- 포함: `splitTableRow` 수정(약 10줄), 단위 테스트 2~3개(표 파싱 1, 보존 사례 표-주도 1, 왕복 1), 검증 명령 실행.
- 제외: 프런트, 문서, DOCX/HTML/PDF 경로, 내보내기 쪽 이스케이프.
- 범위: 30~50분(10 중 8). 여유분: live DB 컨테이너 기동이 필요하면 +15분.
- 미확인: `splitTableRow` 의 현재 오동작은 **코드를 읽어 판단**했고 이번 회차에 실행으로 재현하지는 못했습니다(샌드박스에서 `go run` 승인 거부). 구현자는 고치기 전 실패를 반드시 먼저 눈으로 확인할 것.
