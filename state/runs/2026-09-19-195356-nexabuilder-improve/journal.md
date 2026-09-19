# 회차 노트 2026-09-19-195356-nexabuilder-improve — nexabuilder
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 19:54] base pinned — master@ab6990b
- [러너 19:54] autonomy release — 

## 정찰 노트
- 선택 이유: 보류 아이디어는 전부 CI/문서 손질(가치 1~3)인데, `ListExportController` 를 읽다가 세 내보내기 엔드포인트가 `deletedAt` 을 안 보고(런타임 화면은 봄) 테스트도 0건인 것을 찾았다. 실제 동작이 바뀌고(휴지통 목록 내보내기 차단) 실제 런타임(H2·POI·OpenPDF)을 타는 테스트로 증명할 수 있어 운영자 규칙에 가장 잘 맞는다.
- 확신 없는 곳: CSV `writeRow` 의 인용 규칙(380행 이후 미열람) — 헤더 기대값은 구현을 읽고 맞출 것. `sh ./gradlew` 실행은 정찰 세션 권한상 못 돌려 봄(직전 회차는 568개 통과). `listData` 의 다른 호출자 존재는 미확인이라 그쪽은 손대지 말라고 적음.
- 구현자 주의: 검사는 `response.getOutputStream()` 을 열기 전에 둘 것(스트리밍 컨트롤러). `@DirtiesContext` 금지. 예외 타입·메시지는 `NexaUiService.listDefinition` 과 똑같이 `IllegalArgumentException("List not found: …")` 로 두어 400 JSON 계약을 유지할 것. PDF 는 `%PDF-` 매직까지만 검사.
- [러너 19:58] scout done — 목록 내보내기(CSV/XLSX/PDF)가 휴지통(soft-delete)에 든 목록도 내려주는 것을 막고, 내보내기 3종에 통합 테스�

## 구현 노트
- 무엇/왜: `ListExportController` 세 엔드포인트의 "findById → 세션 user → canView" 를 `resolveExportable()` 로 모으고 `deletedAt != null` 이면 `IllegalArgumentException("List not found: …")` 를 던진다(런타임 화면과 동일 판단, 400 JSON). `ListExportIntegrationTest` 5건 신규. 커밋 3bd46b2.
- TDD 증거: 컨트롤러 수정 전 단독 실행 → `softDeletedListIsNotExportableInAnyFormat` 만 실패("Status expected:<400> but was:<200>"), 나머지 4건(살아 있는 CSV/XLSX/PDF, 세션 user 없음 403)은 수정 전에도 통과 = 기존 동작 증명. 수정 후 5/5, `cleanTest test` 573/573, `bootJar -x test` 성공.
- 확신 없는 곳: 없음에 가깝다. 굳이 꼽으면 CSV 테스트가 `writeRow` 의 CRLF·"필요할 때만 인용" 규칙(escapeCsv)을 그대로 기대값으로 삼으므로 인용 규칙을 바꾸면 이 테스트도 같이 손봐야 한다. PDF 는 `%PDF-` 매직과 content-type 까지만 검사(CJK 폰트 팩 유무로 본문이 달라짐).
- 일부러 하지 않은 것: `NexaUiService.listData` 에는 검사를 넣지 않았다 — `POST /api/v1/builder/lists/{listId}/data`(UiBuilderController:219) 가 같은 서비스를 쓰고 빌더 미리보기가 휴지통 목록에 기대는지 미확인이라 ideas 에 별도 항목으로 남겼다. `GlobalExceptionHandler`/`AccessAuditHandler`/`ScreenPermissionService` 는 손대지 않았고 400/403 매핑은 테스트로 증명만 했다. OpenPDF deprecated 경고(빌드에 기존부터 있음)도 범위 밖.
- 다음 역할이 조심할 것: 테스트는 `@SpringBootTest` 라 H2 컨텍스트가 뜬다(단독 ~20s). `@DirtiesContext` 없음, 픽스처는 UUID 접미사로 격리·정리 안 함. 이 워크트리는 `gradlew` 가 100644 라 `sh ./gradlew` 로 돌려야 한다.
- [러너 20:07] brief accepted — 채택 — 과제서의 근거(세 엔드포인트가 deletedAt 을 안 봄, 테스트 0건, 400/403 매핑 경로)가 코드와 전부 일치해 그대로 구�
- [러너 20:07] verify passed — 검증 1개 통과 (auto)

## 비평 노트
- 확인: diff 2파일(컨트롤러 + 신규 테스트) 전부 읽음. resolveExportable 이 기존 3중복을 그대로 모으고 deletedAt 검사만 추가, 예외 타입/메시지가 NexaUiService.listDefinition·GlobalExceptionHandler 400 계약과 일치, 검사 위치가 getOutputStream 이전임을 코드로 확인. NexaListRepository.findById 가 deletedAt 을 필터하지 않아 수정 전엔 새 테스트가 200 으로 실패했을 것임(TDD 증거 신뢰). 단독 실행 5/5 통과.
- 못 본 것: 전체 568+5 스위트는 재실행 안 함(구현 노트 573/573 신뢰). PDF 본문·CJK 글리프는 검사 대상 아님.
- 판정 approve, risk low, blocking 없음. 400 vs 403 순서로 인한 존재 여부 구분은 수정 전부터 있던 것이라 새 공격 경로 아님.
- 릴리즈 노트: "휴지통 목록은 export.csv/xlsx/pdf 에서 400 'List not found'" — 북마크된 내보내기 URL 을 쓰던 운영자에게 알릴 것.
- 다음 회차: UiBuilderController POST /lists/{id}/data(NexaUiService.listData) 는 여전히 휴지통 목록 데이터를 반환 — 빌더 미리보기 의존 확인 후 같은 규칙 적용 검토.
- [러너 20:09] review approved — 리뷰 승인 (risk=low)
- [러너 20:09] pr created — https://github.com/hkjang/nexabuilder/pull/27
- [러너 20:18] ci passed — 검사 2개 모두 success
- [러너 20:19] merge done — 3bd46b2
- [러너 20:37] release published — v1.18.0
- [러너 20:38] assets verified — v1.18.0 자산 1개 (이전 v1.17.0: 1)
