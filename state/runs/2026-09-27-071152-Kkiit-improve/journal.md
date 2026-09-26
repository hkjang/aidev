# 회차 노트 2026-09-27-071152-Kkiit-improve — Kkiit
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 07:11] base pinned — main@e6e937c
- [러너 07:11] autonomy release — 

## 정찰 노트
- `createReport`(reports.go:74-79)가 INSERT 의 모든 오류를 409 "이미 접수되어 처리 중" 으로 묶는 것을 골랐다. 2026-09-26 에 `createCoupon` 에서 고친 것과 같은 결함이고 `isUniqueViolation`(coupons.go:19)이 이미 있어 프로덕션 변경이 한 함수로 끝난다. 후보였던 중복 분쟁(disputes.go:123 선검사 있음)·중복 슬러그(talents.go:461 후보 루프)는 읽어 보니 이미 처리돼 기각했고, `deleteCoupon`·나머지 10여 곳의 같은 패턴은 고유 제약이 없어 실제 HTTP 로 오류를 만들 수 없어 미뤘다.
- 추측으로 적은 것: `details` 에 `\u0000` 을 넣으면 PostgreSQL 이 23505 아닌 오류로 INSERT 를 거절한다는 것(미확인). `decodeJSON`(types.go:102)이 NUL 을 통과시키는 것과 길이 검사만 있는 것은 확인했다. docker 가용성도 이번엔 확인하지 못했다(권한 프롬프트).
- 구현자 주의: `isUniqueViolation` 을 옮기거나 시그니처를 바꾸지 말 것(쿠폰 테스트가 본다). `integration_test.go` 는 전역 `apiUnderTest` 때문에 병렬화 금지, `client.do` 는 기대 상태코드 불일치에서 Fatal 이다. NUL 입력을 400 으로 막는 쪽으로 범위를 넓히지 말 것.
- 프로필은 2026-09-22 판(5일 전)이 지금 코드와 어긋나지 않아 다시 쓰지 않았다.
- [러너 07:18] scout done — 신고 접수(`createReport`)가 모든 INSERT 실패를 409 "이미 접수되어 처리 중인 신고가 있습니다." 로 보고하는 �

## 구현 노트
- `createReport` 의 INSERT 오류를 `isUniqueViolation`(23505)만 409 `report_already_open`, 그 밖은 500 `report_failed` 로 갈랐다. 열린 신고가 없는 신고자가 "이미 접수되어 처리 중" 을 받고 접수된 줄 알고 포기하던 것이 원인. 프로덕션 파일 2개(reports.go 분기, coupons.go 주석 한 줄 일반화 — 이동·리네임·시그니처 변경 없음).
- 과제서의 미확인 1순위를 확인했다: `details` 의 NUL 은 서버 측 **SQLSTATE 22021** 로 거절된다. 핸들러에 임시 프로브를 넣어 두 경로가 실제로 `23505`(중복)와 `22021`(인코딩)로 갈리는 것을 출력으로 보고 프로브를 제거했다. docker 도 가용(29.7.2)해 3순위 단위 테스트 대안은 쓰지 않았다.
- 확신 없는 곳·검증 못 한 것: `npm --prefix web test` 를 돌리지 않았다(`web/node_modules` 부재, 프런트 미변경이라 설치하지 않음). `make check`·`make build`·브라우저도 미실행. 500 응답 문구("신고를 접수하지 못했습니다. 잠시 후 다시 시도해 주세요.")는 기존 문구를 참고해 새로 지은 것으로 디자인 승인은 받지 않았다. 22021 이 유일한 비중복 실패 경로인지는 확인하지 않았다 — 분기는 SQLSTATE 와 무관하게 "23505 가 아니면 500" 이라 동작은 같다.
- 일부러 하지 않은 것: NUL 을 400 으로 막는 입력 검증(별 과제), POST /reports 의 `'404'` openapi 누락(과제서 범위 밖 — ideas.json 에 남김), `deleteCoupon` 등 같은 패턴 10여 곳(파일 수 억제), `web/`·`internal/ui/dist`(화면은 `error.message` 를 그대로 띄운다).
- 다음 역할 주의: 새 테스트 `TestIntegrationReportSeparatesDuplicateFromStorageFailure`(integration_test.go:1110 부근)는 `KKIIT_TEST_DSN` 없이는 SKIP 된다 — DSN 없는 통과는 검증이 아니다. 새 헬퍼 `errorCode`(같은 파일 끝)는 패키지 전역이라 이름 충돌에 주의. 전역 `apiUnderTest` 때문에 병렬화 금지.
- 검증 출력: 깨끗한 DB 로 `go test ./cmd/... ./internal/...` 전체 통과(`ok internal/httpapi 101.059s`, 통합 실제 실행), `gofmt -l cmd internal` 무출력, `go vet ./cmd/... ./internal/...` 무결. 분류기 변이 2종(항상 참 / 23505→00000)에서 각각 FAIL 확인.
