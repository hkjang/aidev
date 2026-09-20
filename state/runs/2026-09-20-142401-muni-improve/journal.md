# 회차 노트 2026-09-20-142401-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 14:24] base pinned — main@1547faa
- [러너 14:24] autonomy release — 

## 정찰 노트
- 고른 이유: 지난 회차의 `.md` 이중 제목 고침(e8c53e9, 채택·릴리즈됨)과 같은 결함이 `.html` 에 그대로 남아 있음을 코드로 확인(export.go:359 `<h1 class="doc-title">`, import_html.go:148 이 head/title 을 버림). 기존 `dropLeadingTitle` 규칙을 확장자 하나 더에 적용하는 S 과제라 위험이 가장 낮고, 같은 유형이 직전에 통과했음.
- 메일 관련 후보(outbox 즉시 깨우기·운영 현황 실패 수)는 제침 — 메일 캠페인 커밋 7afd103(브랜치 auto/2026-09-16-0842)이 main 에 없어(notify.go 옛 형태, mail_deliveries 없음) 병합/반려 여부 미확인(gh 조회 불가). 그 영역은 충돌이나 반려 반복 위험.
- 확신 없는 곳: HTML 가져오기가 body 첫 `<h1>` 앞에 빈 블록을 끼우지 않는다는 것(코드상 그럴 이유 없으나 실행으로 미확인) — 새 테스트를 먼저 돌려 첫 블록이 heading 인지 보면 바로 드러남. PDF 가져오기 쪽의 이중 제목은 미확인.
- 구현자 주의: `<title>` 읽기·doc-title 클래스 판단을 새로 만들지 말 것(범위 밖, 같은 규칙 하나 공유). live 테스트는 MUNI_TEST_DSN 없으면 SKIP 되므로 postgres 를 띄워 SKIP 0 을 확인할 것. 픽스처는 `fullHTMLWithDrawing` 실제 출력을 쓸 것.
- 프로필은 1일 전 것이라 다시 쓰지 않음. 단 "019 mail 마이그레이션" 은 main 에 없음(018 까지) — 다음 프로필 갱신 때 반영.
- [러너 14:27] scout done — HTML 가져오기도 제목과 같은 첫 H1 을 본문에서 뺀다 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- 바꾼 것: `upload.markdown` → `titleInBody` 로 이름을 바꾸고 `parseUpload` 의 `.html/.htm` 에서도 참으로 두어, 내보낸 `.html` 을 다시 가져올 때 제목과 정확히 같은 첫 H1 을 기존 `dropLeadingTitle` 규칙 그대로 뺍니다(커밋 36dccc6). `<title>`·`doc-title` 클래스는 읽지 않고, 함수 본문·`importIntoDocument`·넘겨받기는 그대로.
- 확신 없는 곳: 없음 — live 테스트가 실제 `fullHTMLWithDrawing` 출력으로 고치기 전 "first block = heading" 으로 실패하고 고친 뒤 통과했고, `.txt` 첫 줄 유지·폼 title 다를 때 heading 유지·끼워 넣기 heading 유지까지 같은 테스트에서 확인. 브라우저로는 띄우지 않았음(서버 코드 경로가 테스트가 치는 핸들러와 같음).
- 일부러 하지 않은 것: `.docx` — 내보내기가 `Title` 스타일 문단을 쓰고 리더가 그것을 level-1 heading 으로 읽어 **같은 이중 제목이 남아 있음**(docx/export.go:141, import_blocks.go:383). 과제서 수용 기준 3 이 `.docx` 불변을 요구해 이번엔 안 켰고 ideas.json 에 1순위 후보로 적음. `<title>` 을 embeddedTitle 로 올리는 것도 과제서대로 범위 밖.
- 다음 역할 주의: 새 live 테스트 `TestAnExportedHTMLFileImportsWithoutItsTitleTwice` 는 `MUNI_TEST_DSN` 없으면 조용히 SKIP — postgres:16-alpine 컨테이너(`muni-pg-0920`, 호스트 5432)를 띄워 httpapi SKIP 0 을 확인했음. 단위 테스트 `TestHTMLRoundTripDropsTheTitleHeading` 은 고치기 전에도 통과하는 성질(규칙이 형식 중립)이라 회귀 증거는 live 쪽임.
- [러너 14:32] brief accepted — 채택 — 근거(export.go 의 `<h1 class="doc-title">`, import_html 이 head/title 을 버리고 h1 을 heading 으로 냄, 빈 블록을 끼우지 않음)가
- [러너 14:32] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 판정 reject / security: internal/httpapi/handoff.go:181의 전송 실패 로그가 internal/handoff/handoff.go:275의 URL 포함 오류를 통해 미사용 claim을 노출합니다. 수리는 이 두 파일의 오류·로그 처리부터 확인하세요.
- 실제 main=1045e13으로 pinned base=1547faa와 다릅니다. 요청한 전체 diff 기준 차단이며 결함은 이전 a8b253b부터 존재; 이번 HTML 커밋 자체에서는 차단 결함을 찾지 못했습니다.
- PostgreSQL 16 live 포함 Go PASS 259·SKIP 0, 관련 프런트 20개 통과. HTML 업로드·검색 텍스트·다른 제목·TXT 및 삽입 경로 확인; 삽입 테스트의 heading 존재 단언은 약합니다.
- 전체 프런트 빌드·브라우저 E2E·PDF 시각 검증과 수정 전 재실행은 하지 않았습니다. DOCX 이중 제목은 기존 과제로 남고 코드·기존 작업 트리 변경은 건드리지 않았습니다.
- [러너 14:34] review rejected — 리뷰 거절: internal/httpapi/handoff.go:181 [P1/security] 넘겨받기 요청의 연결·TLS·타임아웃 오류를 err.Error() 그대로 경고 로그에 기록합니다. internal/handoff/handoff
- [러너 14:34] review blocked — 검토 부서 차단 소견(security) — 수리·중재 없이 운영자의 위험 수용(risk-accepted) 필요
- [러너 14:35] pr created — https://github.com/hkjang/muni/pull/20
