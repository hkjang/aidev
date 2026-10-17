# 회차 노트 2026-10-17-114850-insight.yeopjari.bid-improve — insight.yeopjari.bid
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 11:48] base pinned — main@601659d
- [러너 11:48] autonomy release — 

## 정찰 노트
- v0.1.5~0.1.6 로 새로 들어온 DB 차단기(db.ts:retrying/trip, errors.ts:isDatabaseUnreachable)를 읽다가, 같은 파일이 "경합일 뿐 곧 성공한다" 고 적은 풀 포화 오류(53300·58000·EMAXCONNSESSION)가 isConnectionLost 경유로 차단기를 열어 멀쩡한 DB 에서도 isolate 전체를 10~120초 503 으로 만드는 경로를 확인해 1순위로 골랐다. 문서 개수 정리·maskEmail 테스트·isSafeLink 정리는 가치가 낮아, 월 quota purge 는 운영 호출처 미확인이라 제쳤다.
- 추측으로 적은 것: (1) 포화가 2.5초 안에 재시도 4회를 실제로 소진하는지는 코드 구조상의 판단이고 운영 로그로 확인하지 않았다. (2) 가짜 Postgres ErrorResponse 서버로 "차단기 미개방" 을 배선 테스트할 수 있는지는 벤더 PATCH 5 때문에 미확인이며, 과제서에 대체 경로를 적어 두었다.
- 구현자가 조심할 것: isDatabaseUnreachable 의 반환값은 respond.ts:58 과 pipeline.ts:161 이 같이 쓰므로 바꾸지 말고 trip() 호출만 가린다. vendor/postgres·quota windowStart·migrations 는 손대지 말 것. node_modules 가 없어 이번 정찰은 테스트를 돌리지 않았다(단위 개수는 지난 회차 기록 인용).

## 정찰 노트
- 최종 과제는 지급 목록 다운로드의 DB 장애 알림 연결이다. 앞선 초안의 풀 포화 차단기 제외는 정상 DB라는 근거가 없고 의도된 장애 억제를 약화할 수 있어 철회했다. 이 최종 brief.md가 앞선 노트보다 우선한다.
- 실제 client.ts와 로컬 HTTP 서버에서 같은503의 download 알림0/post 알림1을 확인했다. 사용자 호출이 없는 캐시·월 quota 후보보다 영향과 수정 범위가 명확하며 프로덕션1파일이다.
- UI 배너 재현·ready 보상 seed·Chromium은 미확인. 단위 실행은 vitest 부재127이며 전체 통과를 주장하지 않는다. 구현자는 실제 버튼→배너 검증을 생략하지 말 것.
- 기존 auth·DB 차단기·감사·vendor·workflow를 건드리지 말고 예상export503만 브라우저 회귀에서 허용할 것. 세 요청 스킬은 Skill 도구 부재로 설치된 SKILL.md를 직접 읽어 적용했다.
- [러너 11:58] scout done — 지급 목록 다운로드의 DB 장애도 공통 점검 안내로 전달하기 (가치 3 / 위험 1 / 작업량 M)
- [러너 12:37] resume closed — PR 이전 단계에서 중단 — 에이전트 단계는 재실행하지 않음
