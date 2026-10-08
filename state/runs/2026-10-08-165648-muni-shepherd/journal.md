# PR 처리기 노트 2026-10-08-165648-muni-shepherd — muni PR #36
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-08-152841-muni-improve)
# 회차 노트 2026-10-08-152841-muni-improve — muni
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 15:28] base pinned — main@9d781ef
- [러너 15:28] autonomy release — 

## 정찰 노트
- HWPX 머리말·꼬리말 절단을 선택: 실제 사용자 데이터 결함이며 기존 헬퍼로 프로덕션 1파일·네 호출에 한정; ZIP 충돌 확장·Windows 호환보다 위험이 낮다.
- 초안의 DOCX 가정은 폐기: docx reader는 이미 200룬 절단. HWPX writer/parser에는 제한 없음. 201룬 실제 라우트·DB 재현은 미확인으로 과제서 첫 단계에 명시.
- 기존 DOCX/HTTP 검사 PASS 270·SKIP 72·FAIL 0, HWPX 머리말 왕복 1개 PASS. MUNI_TEST_DSN 없어 live 미검증; skip을 성공으로 보고하지 말 것.
- cutFilenameRunes 끝 공백 제거와 삽입 응답의 DB 비변경 계약을 감시하고, 공용 헬퍼·AI·auth·migrations·workflows는 변경하지 말 것. 요청 스킬 3개는 Skill 도구 부재로 로컬 SKILL.md를 직접 읽음.
- [러너 15:35] scout done — HWPX 가져오기 머리말·꼬리말의 200룬 절단에 AI 안내 문구가 붙는 것 고치기 (가치 3 / 위험 1 / 작업량 S)

## 구현 노트
- e503055: 프로덕션 import_attachments.go 네 호출만 cutFilenameRunes로 교체하고 이유 주석 추가; 테스트는 기존 import_name_length_live_test.go 한 파일에 추가.
- 실제 hwpx.Build/Parse가 201룬을 유지함을 확인. 두 POST 경로 2테스트·10행으로 DB/응답, 200룬 경계·끝 공백, 제목·본문·방향, 삽입 대상 revision/본문/머리말·꼬리말 불변 검사; 생성 직후 자기 ID cleanup 등록.
- 수정 전 FAIL 2 → 수정 후 PASS 2 → 네 호출만 원복 FAIL 2 → 복구 뒤 HTTP PASS 249 / SKIP 0 / FAIL 0. 상세 출력은 furniture-{red,green,revert}.log 및 httpapi-tests.log.
- 전체 Go 15패키지 ok, vet·gofmt·placeholder·diff 검사 exit 0, 기존 HWPX 왕복 PASS 1. all-go-tests.log, go-vet.log, hwpx-roundtrip.log 참조.
- 확신 없는 곳·검증 못 한 것: 실제 브라우저 가져오기 UI 증상과 외부 한글 앱 출력은 미검증; 테스트가 증명하는 범위는 프로덕션 writer/parser 및 HTTP·DB 계약.
- 일부러 제외: DOCX/HWP reader, AI·공용 절단기, 기존 DB 행, 프런트·npm 검사·make build·릴리즈. 지정한 작은 수정 범위를 유지했다.
- 다음 역할 주의: live 검증은 전용 MUNI_TEST_DSN 필요(SKIP은 통과 아님). 이번 전용 postgres:16-alpine 컨테이너는 검증 후 제거하며, 기존 공용 liveServer는 수정하지 않았다.
- [러너 15:40] brief accepted — 채택 — HWPX writer/parser의 한글 201룬 무손실 전달과 HTTP 네 호출에서의 안내 문구 유입을 실제 실행으로 확인해 지정 수정�
- [러너 15:41] verify passed — 검증 7개 통과 (auto)
- [러너 15:41] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 15:41] pr created — https://github.com/hkjang/muni/pull/36
