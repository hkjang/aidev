# 회차 노트 2026-09-19-195351-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:53] base pinned — main@65b95e5
- [러너 19:53] autonomy release — 

## 정찰 노트
- 선택: 마크다운 첫 H1 이 제목과 같으면 덜어내기 (3/1/S). e2e-in-CI(4/2/M) 가 값은 더 크지만 workflows(보호 경로)와 시드 계정 마련이 필요해 45분 안에 검증까지 끝내기 어렵고, outbox 즉시 발송(3/2/S)은 동시성(채널·틱 겹침)이 붙어 위험이 한 단계 높음. 이 과제는 Go 두 파일 + 헬퍼 하나 + 단위 테스트로 끝나고 스키마·프런트를 건드리지 않음.
- 확신 없는 곳: richdoc 의 heading 노드 타입 이름과 level attr 이름(과제서에 "미확인"으로 적음 — `richdoc.go`·`toc.go:Headings` 에서 확인할 것); 본문이 H1 하나뿐일 때 빈 content 가 유효한지; 019 마이그레이션 파일 위치(차선 후보에만 관계).
- 구현자가 조심할 것: 제목과 **정확히** 같을 때만 지울 것(TrimSpace 이상 정규화 금지), `importIntoDocument` 와 `renderMarkdown` 은 손대지 말 것, `extractDocumentText` 가 새 content 를 보도록 순서, live 테스트는 DSN 없이 skip 되므로 postgres 를 띄워 `go test ./...` 를 한 번은 실제로 돌릴 것.
- [러너 19:57] scout done — 마크다운 가져오기·넘겨받기에서 첫 H1 이 문서 제목과 같으면 본문에서 덜어내기 (가치 3 / 위험 1 / 작업�

## 구현 노트
- 무엇·왜: `dropLeadingTitle`(import_markdown.go) — 마크다운의 첫 블록이 level-1 heading 이고 글이 확정 제목과 TrimSpace 뒤 정확히 같으면 그 블록만 뺌. `upload.markdown` 플래그로 `importDocument`·`storeHandoff` 두 자리에서 제목 확정 직후·`extractDocumentText` 직전에 적용. muni 가 내보낸 .md 를 다시 가져오거나 muni→muni 넘길 때 제목이 두 번 보이던 것을 고침. 커밋 e8c53e9.
- 확신 없는 곳·검증 못 한 것: (1) 편집기 화면에서 넘겨받은 문서를 실제로 열어 보지는 않았음 — DB 의 content_json·content_text 로만 확인(live 테스트). (2) `handoff.go` 에서 `truncateRunes(title, 240)` 을 비교 전으로 옮겼는데 저장값은 이전과 같음(같은 함수·같은 값). (3) `.txt`·`.html`·`.docx` 등 다른 형식이 바뀌지 않는 것은 `markdown` 플래그가 `.md/.markdown` case 에서만 true 인 것을 코드로 확인했고 별도 테스트로 못 박지는 않았음.
- 일부러 하지 않은 것: `importIntoDocument`(끼워 넣기)에는 적용하지 않음 — 제목은 따로 돌려주고 본문은 끼워 넣는 자리라 H1 을 빼면 글이 사라짐(live 테스트가 heading 유지를 고정). `renderMarkdown` 은 그대로(GitHub 등에서 제목이 보여야 함). 대소문자·공백 정규화 없음, 첫 H1 을 무조건 제목으로 올리는 pandoc 식 설계 안 함.
- 다음 역할이 조심할 것: `import_markdown_live_test.go` 의 새 테스트와 `handoff_live_test.go` 의 추가 case 는 `MUNI_TEST_DSN` 이 있어야 돌고 없으면 조용히 skip 됨 — 이번엔 postgres:16-alpine 컨테이너(127.0.0.1:55460, 이름 muni-improve-pg, --rm)를 띄워 SKIP 0 으로 확인했고 컨테이너는 회차 끝에 내림. 프런트는 손대지 않아 npm 단계 생략.
- [러너 20:04] brief accepted — 채택 — 과제서의 근거(parseUpload 가 .md 에 embeddedTitle 을 만들지 않음, heading 타입·level attr 이름, 빈 content 를 JSON() 이 빈 문
- [러너 20:04] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인함: e8c53e9 한 커밋(base 65b95e5) 전체를 읽고 gofmt/vet 통과, postgres:16 컨테이너(55461)로 httpapi 패키지 live 테스트 SKIP 없이 통과. `dropLeadingTitle` 을 no-op 으로 바꿔 돌리니 새 테스트 6개 중 5개가 실패(negative case 인 KeepsAHeading 만 통과) — 테스트가 변경을 실제로 못 박음. richdoc Parse→JSON 왕복은 withBlockIDs 가 이미 쓰는 경로라 필드 손실 없음. 빈 문서의 ID 없는 빈 문단은 새 문서 기본값(documents.go:153)과 같은 모양.
- 못 봄: 편집기 화면에서 실제로 열어 보지 않음(구현자와 같은 공백). 프런트는 diff 에 없어 npm 단계 생략.
- 남는 우려(결함 아님): 제목에 `/`·`\` 가 있거나 100 rune 을 넘으면 내보낸 파일명(safeFilename)이 제목과 달라져 재가져오기 때 H1 이 남아 여전히 두 번 보임 — 테스트 "file name stem" 이 이 동작을 의도로 고정. 사용자가 폼에서 H1 과 같은 제목을 직접 주면 H1 이 빠지는 것도 의도.
- 릴리즈 노트에 적을 것: .md 재가져오기·muni→muni 넘겨받기에서 제목 중복 제거, 끼워 넣기(importIntoDocument)는 그대로, 내보내기 형식은 변경 없음.
- 보안·법무 소견: 새 엔드포인트·권한·개인정보 없음, 피어 콘텐츠는 기존 허용 목록 뒤에서만 닿음. 차단 없음.
- [러너 20:07] review approved — 리뷰 승인 (risk=low)
- [러너 20:07] pr created — https://github.com/hkjang/muni/pull/19
- [러너 20:13] ci passed — 검사 2개 모두 success
- [러너 20:14] merge done — e8c53e9
- [러너 20:22] release published — v0.41.0
- [러너 20:26] assets verified — v0.41.0 자산 1개 (이전 v0.40.0: 1)
