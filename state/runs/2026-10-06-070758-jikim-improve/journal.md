# 회차 노트 2026-10-06-070758-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:08] base pinned — main@003aa7b
- [러너 07:08] autonomy release — 

## 정찰 노트
- 고른 이유: 프로필의 1순위 보류(oidcExchange 의 GetUser 분기)가 v0.2.29 머지로 해금됐고, 열어 보니 두 줄 위 `ConsumeOIDCLoginCode`(oidc.go:440-443)가 같은 결함을 더 선명하게 갖고 있었다 — store/oidc.go:61-62 가 "코드 만료"에만 `ErrUnauthorized` 를 주고 드라이버 실패는 원문을 주는데 핸들러가 둘을 401 하나로 접는다. 둘을 한 과제로 묶었다(프로덕션 2파일).
- 제친 후보: temperature 클램프(도달 불가로 이번에 rejected), api-guide 문서 표(관찰 가능한 변화 없음 — 4회 연속 차선), CSP 리포트 속도 제한(새 계약·M), oidcLogout(302 가 같아 새 WARN 계약 필요 → 차선으로만).
- 추측으로 적은 것: 실제 PostgreSQL 장애가 pgx 로 어떤 오류로 올라오는지는 DSN 없이 확인 못 했다 — `driverFailure` 대리 검증임을 과제서에 명시했다. `storeError` 가 ErrNotFound 를 404 로 보내는 것도 이번 설계에서는 닿지 않게 피했다(ErrNotFound 는 기존 401 유지).
- 구현자 주의: `invalid_login_code`·`inactive_user` 두 code 와 한국어 문구는 바이트 단위 유지. happy path 는 테스트에 넣지 말 것(nil store 가 `SecurityConfig` 에서 panic). `assertSingleStoreErrorLog` 는 `SQLSTATE 28P01` 을 요구하므로 장애 케이스 전용이다.
- [러너 07:14] scout done — OIDC 일회용 코드 교환(`oidcExchange`)이 저장소 장애를 "코드 만료"·"계정 비활성화"로 보고하는 두 자리 분리

## 구현 노트
- 무엇을: `oidcExchange`(oidc.go) 의 두 저장소 읽기 분기만 쪼갰다 — `ConsumeOIDCLoginCode` 의 `ErrUnauthorized` 와 `GetUser` 의 `ErrNotFound` 만 기존 401 로 남고, 나머지(드라이버 오류)는 `s.storeError` 로 가서 500 `internal_error` + ERROR 한 줄이 된다. seam 2개(`oidcCodeConsumer`·`userLoader`)와 nil-폴백 헬퍼 2개는 `openOIDCState` 와 같은 모양이다. 커밋 `18a822d`(프로덕션 2파일 + 신규 테스트 1파일).
- 과제서와 어긋난 사실 하나: `model` 은 oidc.go 에 import 되어 **있지 않았다**. `undefined: model` 빌드 실패로 바로 드러나 import 한 줄을 추가했다. 그 외 과제서의 줄번호·배선 사실은 모두 맞았다.
- 확신 없는 곳: 실제 PostgreSQL 장애 때 pgx 가 올려보내는 오류 종류는 이 환경(DSN 없음)에서 확인 못 했다 — `driverFailure`(`SQLSTATE 28P01` 문자열) 대리 검증이다. 실제 Keycloak 왕복도 미검증. `storeError` 는 `ErrConflict`·`ErrInvalid` 등도 각자의 status 로 번역하는데, `ConsumeOIDCLoginCode`·`GetUser` 가 그런 오류를 돌려주는 경로는 없다고 보고 테스트하지 않았다(`mapError` 가 23505/23503 을 접기는 하지만 이 두 쿼리에 해당 제약이 없다) — 이 가정이 비평 포인트다.
- 일부러 하지 않은 것: `ErrNotFound`(삭제된 사용자)를 새 code 로 쪼개지 않았다(사용자 존재 여부 노출 + 범위 증가). happy path 는 테스트에 넣지 않았다 — nil store 가 그 다음 `SecurityConfig` 에서 panic 한다. `oidcLogout`·`store/`·`web/`·`CHANGELOG.md`·`scripts/version.sh` 는 손대지 않았다.
- 다음 역할 주의: 새 테스트는 DB 없이 돈다(`quietServer()` + seam 2개). 거절 3개 케이스는 수정 전에도 통과하므로 "통과한다"만으로는 분기 쪼개기가 살아 있다는 증명이 못 된다 — 장애 2개(`login code read outage`·`user read outage`)가 그 증명이다. `assertSingleStoreErrorLog` 는 ERROR 줄에 `SQLSTATE 28P01` 을 요구하므로 장애 케이스 전용이고, 거절 케이스는 새로 넣은 `assertNoStoreErrorLog` 로 ERROR 0줄을 센다.
- [러너 07:20] brief accepted — 채택 — 근거가 코드와 정확히 맞았고(oidc.go:440-449 의 두 분기, store/oidc.go:51-69 의 오류 구분, `storeError` 가 500 에서만 message
- [러너 07:20] verify passed — 검증 7개 통과 (auto)
