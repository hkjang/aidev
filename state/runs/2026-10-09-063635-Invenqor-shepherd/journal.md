# PR 처리기 노트 2026-10-09-063635-Invenqor-shepherd — Invenqor PR #37
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.

## PR 을 연 회차의 노트 (2026-10-09-061825-Invenqor-improve)
# 회차 노트 2026-10-09-061825-Invenqor-improve — Invenqor
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 06:18] base pinned — main@16f2592
- [러너 06:18] autonomy release — 

## 정찰 노트
- MCP 두 UUID 입력은 선언·SQL·기존 helper가 모두 확인돼 프로덕션 1파일로 제한 가능; 반복 차선인 부모 관계 삭제·재현 불가 교착/rows.Err·workflow 변경보다 위험이 낮다.
- 기존 SQLite MCP/Arguments 테스트 통과(3.214s). 비정규/대문자 실제 요청 실패와 PostgreSQL 응답은 미확인으로 과제서에 명시; 구현자가 실패 재현부터 수행한다.
- String의 TrimSpace, HTTP 200/isError, 미존재·병합 안내를 보존하고 relations.read 없는 기존 MCP helper를 확장하지 말 것. 테스트용 키를 별도로 발급한다.
- Pages 964e954는 main에 병합됐지만 confidence 7250899는 별도 브랜치에만 있다. 완료 기록을 재구현하지 않고 프로필·아이디어 상태에 차이를 남겼다.
- [러너 06:24] scout done — MCP asset_get·asset_relations의 asset_id를 canonicalUUID로 검증·정규화 (가치 3 / 위험 1 / 작업량 M)

## 구현 노트
- 7b97262: mcp.go 두 핸들러에서 기존 canonicalUUID로 asset_id를 SQL 전에 검증·정규화; DB별 입력 해석과 대문자 병합 안내 누락 해결. 프로덕션 1파일 10줄 + 신규 테스트 1파일.
- 새 TestMCP 3개: 실제 인증 HTTP legacy/modern 전체 입력표·활성 양방향 관계·미존재, REST 병합 후 대문자 안내, 닫힌 실제 DB에서 SQL 이전 거부. 고정 UUID에 a-f 포함.
- 수정 전 양 DB 실패 기록, 수정 후 SQLite 성공, 수정만 원복해 재실패, 최종 양 DB MCP/Arguments 성공(6.242s/9.907s). 전체 Go test(httpapi 46.871s)/vet/build 및 gofmt·diff 검사 통과.
- 확신 없는 곳·검증 못 한 것: PostgreSQL은 MCP|Arguments 필터로 실행했으며 전체 PostgreSQL suite·web/Rust·원격 CI는 실행하지 않음; 이번 수용 기준에 미검증 항목 없음.
- 일부러 제외: 공용 파서·도구 스키마·인증·REST·마이그레이션·confidence=0 미병합 수정·버전/릴리즈. 선택 과제의 두 파일 범위 유지.
- 다음 역할 주의: PostgreSQL 실행에는 Docker가 필요하고 지정 컨테이너를 삭제함; 회차 고유 이름/빈 포트 유지. Skill 도구가 없어 요청한 3개 technology SKILL.md를 로컬에서 읽고 절차 적용. 상세 실패/통과 증거는 같은 폴더 로그·ledger-entry.md 참고.
- [러너 06:29] brief accepted — 채택 — 두 핸들러·공용 파서·canonicalUUID의 현재 코드가 과제서와 일치했고 실제 양 DB MCP 요청에서 차이를 재현했으며 �
- [러너 06:30] verify passed — 검증 8개 통과 (auto)
- [러너 06:30] review missing — 리뷰 결과 없음/손상: missing — 보류
- [러너 06:30] pr created — https://github.com/hkjang/invenqor/pull/37
