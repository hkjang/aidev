# 회차 노트 2026-09-29-201337-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:13] base pinned — main@dc7ad1e
- [러너 20:13] autonomy release — 

## 정찰 노트
- 고른 이유: `truncateRunes` 의 AI 안내 문구가 DB 저장 값에 새는 자리가 아직 여섯 곳 남아 있고(`import_attachments.go:121,183,312,329,570`, `handoff.go:230`), 2026-09-24·09-28 두 회차가 같은 결함을 `safeFilename`·`imageAssetName` 에서 걷어내며 문구 없는 절단기 `cutFilenameRunes` 를 이미 만들어 뒀습니다 — 근거·수정·검증 방식에 선례가 있어 S 한 회차에 맞습니다. 다른 후보(`safeFilename` 이 Windows 금지 문자를 남김, 대소문자만 다른 ZIP 항목 충돌)는 공유 헬퍼 파급이나 Windows 실제 동작 확인이 필요해 제쳤고, 운영 문서 정정은 테스트로 증명할 것이 없어 이 회차 형식과 맞지 않았습니다.
- 확인한 것: `documents.title`·`attachments.name` 이 `text` 라 문구가 그대로 저장됨(`001_initial.sql:119`), `:329` 는 감사 로그가 아니라 편집기에 제목을 돌려주는 `writeData` 응답이라는 것, `:652` 가 저장된 이름을 `extValueEscape` 로 `filename*` 에 넣는다는 것, 라우트 `POST /api/v1/documents/{id}/attachments`(`server.go:143`).
- 추측으로 남긴 것(미확인): 241룬 입력이 각 자리에 실제로 도달하는지 — 이 세션에서 동적 재현을 하지 않았고, 코드상 길이 검사가 보이지 않을 뿐입니다. 구현자는 `:570`(멀티파트 `filename`)으로 먼저 재현하세요.
- 구현자가 조심할 것: `attachments.name` 세 자리를 함께 바꿀 것(한쪽만 고치면 같은 값이 경로마다 다르게 저장됨). 머리말·꼬리말 200룬 절단과 `handoff.go:257` 안내 문장, AI 호출부는 손대지 말 것. `dropLeadingTitle` 이 확정 title 과 정확 일치를 보므로 241룬 제목에서 본문 H1 제거 동작이 바뀌는지 확인해 노트에 적을 것.
- [러너 20:19] scout done — 가져오기·첨부·넘겨받기가 저장하는 제목·첨부 이름의 240룬 절단에 AI 안내 문구가 들어가는 것 고치기 

## 구현 노트
- `documents.title`·`attachments.name` 의 240룬 절단 여섯 자리(`import_attachments.go:121,183,312,329,570`, `handoff.go:230`)를 AI 프롬프트용 `truncateRunes` 에서 문구 없는 `cutFilenameRunes` 로 바꿨습니다. 두 컬럼이 `text` 라 「[…문서 컨텍스트가 길어 일부 생략됨…]」 이 그대로 저장되고 문서 목록·검색 색인·첨부 내려받기 이름까지 나갔습니다. 프로덕션 2파일·테스트 1파일, 커밋 9e19ea8.
- 확신 없는 곳: `:183`(`storeImportedDocument`)와 `:312`(`importIntoDocument`)에 241룬 `attachment.Name` 이 실제로 도달하는지는 **단위로도 재현하지 않았습니다** — reader 가 만드는 이름이라 `imageAssetName` 이 이미 200룬으로 자릅니다. 계약 통일과 세 자리 일관성을 위해 함께 바꿨을 뿐이니 비평가는 여기부터 보세요. `handoff.go:230` 도 피어가 보내는 값이라 재현하지 않았습니다. 실제 라우트로 재현·검증한 것은 `:570`(멀티파트 업로드)과 `:121`(가져오기 제목) 둘입니다.
- 정찰의 예상과 다른 점: 줄바꿈이 `Content-Disposition` 을 깨지 **않습니다**. `extValueEscape` 가 `filename*` 에서 `%0A` 로 인코딩하므로 헤더는 정상 파싱되고, 문구와 줄바꿈이 이름에 그대로 실려 나올 뿐입니다(2026-09-24 회차와 다름).
- 일부러 하지 않은 것: 머리말·꼬리말 200룬 절단(`:174,330,331`)과 `mail.go`, `handoff.go:257` 안내 문장, AI 호출부 전부 — 과제서 범위 밖이고 201룬 도달 재현 근거가 없습니다. `cutFilenameRunes` 개명도 하지 않았습니다(호출처가 늘어 범위 밖).
- 확인해 둔 것: 241룬 제목 + 같은 241룬 H1 에서 제목은 240룬으로 잘리고 본문 H1 은 남습니다(240 ≠ 241). 고치기 전에도 같은 불일치라 `dropLeadingTitle` 동작은 **바뀌지 않았고**, 제목 중복은 기존 공백으로 남습니다 — ideas.json 에 후보로 적었습니다.
- 다음 역할이 조심할 것: 새 테스트 셋은 전부 live 라 `MUNI_TEST_DSN` 이 있어야 돕니다. 없으면 SKIP 되고 아무것도 증명하지 않습니다. 이 워크트리에는 Chromium 이 있어 `TestDevtoolsPDFHasPageNumbers` 도 통과했습니다(없는 환경이면 이 과제와 무관하게 실패합니다).
- 검증: `go test -count=1 -v ./internal/httpapi` PASS 242 / SKIP 0 / FAIL 0(직전 239 + 새 3), `go test ./...` exit 0, `go vet ./...`·`gofmt -l .` clean, `scripts/check-webui-placeholder.sh` OK. 여섯 자리를 되돌리면 새 테스트 둘만 다시 실패합니다. 프런트 미변경이라 npm 검사는 돌리지 않았습니다.
- [러너 20:27] brief accepted — 채택 — 근거(여섯 자리의 `truncateRunes(…,240)`, `documents.title`·`attachments.name` 이 `text`, `:329` 가 감사 로그가 아니라 편집기�
- [러너 20:28] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 커밋 9e19ea8 은 선언한 여섯 자리(`import_attachments.go:121,183,312,329,570`, `handoff.go:230`)만 `truncateRunes`→`cutFilenameRunes` 로 바꿨고 그 외 수정·리팩터·의존성 변경이 없습니다. 남은 `truncateRunes` 호출 13곳을 전수로 봤더니 전부 AI 프롬프트·머리말/꼬리말·mail_deliveries·에러 문장이라 커밋 메시지 설명과 코드가 일치합니다. 원장의 `실패 재현` 두 줄이 이번 증상(저장값에 붙은 문구)과 정확히 맞고, 새 테스트는 실제 라우트를 태운 뒤 DB 를 직접 읽으므로 고치기 전 통과할 수 없습니다.
- 구현자가 의심한 자리(`:183`,`:312`,`handoff.go:230` 미재현)는 결함이 아니라 도달 경로 미확인입니다 — 세 자리 모두 저장 직전의 같은 컬럼이고 계약을 좁히기만 하므로 되돌림 위험이 없습니다. 인가·마이그레이션·비밀값·워크플로 미변경, `downloadAttachment` 의 `documentRole` 검사도 그대로라 security/legal 차단 사유 없음.
- 못 본 것: live 테스트를 이 세션에서 돌리지 못했습니다(MUNI_TEST_DSN 없음). `go build ./...`·`go vet ./internal/httpapi`·`gofmt -l internal` 만 clean 확인. hwp/hwpx/pdfx reader 가 머리말/꼬리말에 201룬 이상을 넣을 수 있는지도 확인하지 않았습니다.
- 승인이어도 남는 우려 셋: (1) `uploadAttachment` 응답이 아직 절단·Base 안 한 `header.Filename` 을 돌려줘 화면값과 저장값이 다름(:605, 기존 결함·다음 회차 후보), (2) 앞 240룬이 전부 공백인 첨부 이름은 이제 빈 이름으로 저장됨(제목과 달리 TrimSpace 를 안 거침 — 퇴행은 아님), (3) 240룬 넘는 제목의 본문 H1 중복은 **고쳐지지 않았으니** 릴리스 노트에 넣지 말 것.
- [러너 20:30] review approved — 리뷰 승인 (risk=low)
- [러너 20:31] pr created — https://github.com/hkjang/muni/pull/31
- [러너 20:36] ci passed — 검사 2개 모두 success
- [러너 20:36] merge done — 9e19ea8
- [러너 20:48] release published — v0.51.0
- [러너 20:52] assets verified — v0.51.0 자산 1개 (이전 v0.50.0: 1)
