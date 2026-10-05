## 정찰 노트
- 고른 이유: 같은 테이블에 쓰기 계약이 둘이라는 것을 코드에서 직접 봤다 — `putSetting`(admin.go:158)은 `INSERT … ON CONFLICT(key) DO UPDATE`, `putOIDCSetting`/`putAISetting`(404/509)은 bare `UPDATE … WHERE key=`. 0행이어도 err=nil 이라 200 과 `oidc.update` 감사가 남는다. 프로덕션 1파일·S·실DB 로 결정론적 Red 가 가능해, 파일 6개에 걸리는 serviceLocation(네 회차 연속 보류)과 PG 정렬 의존 재현이 필요한 페이징 tiebreak 를 제쳤다.
- 확신 없는 곳: 도달성이 약함을 과제서에 명시했다 — 001_initial.sql:237-245 가 oidc/ai 를 seed 하고 제품에 `DELETE FROM system_settings` 경로가 없음을 확인했다(테스트 2곳만). 장애가 아니라 조용한 쓰기 유실 + 거짓 감사 수정이다. 과장 금지.
- 추측으로 적은 것: 행을 지운 뒤 PUT 이 **지금** 200 을 답한다는 것은 UPDATE 0행의 SQL 의미로 추론했고 실DB 로 실행하지 않았다. 구현자는 Red 를 먼저 실측하고, 이미 500 이면 차선(secretbox)으로 갈 것.
- 조심할 것: `secret=true` 를 INSERT 와 DO UPDATE 양쪽에 명시. auth 흐름·2026-10-03 의 `oidcUnreadable` 503 분기·마이그레이션·감사 payload 필드는 건드리지 말고, 세 쓰기를 공통 헬퍼로 통합하지 말 것(putSetting 은 RETURNING 계약이 다르다).
- 프로필: 2026-09-29 판이 5일 전이고 이번에 열어 본 것과 어긋나지 않아 새로 쓰지 않았다.
