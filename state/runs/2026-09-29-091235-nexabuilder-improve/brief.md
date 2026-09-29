- 과제: XLSX 내보내기 시 목록 ID를 안전한 시트 이름으로 변환한다 (가치 3 / 위험 1 / 작업량 S)
- 왜: `ListExportController.exportXlsx`는 저장 가능한 목록 ID를 `wb.createSheet(listId)`에 그대로 전달하며, 실제 POI 5.3.0은 `sales:2026`에서 `IllegalArgumentException`을 던진다. 출력 시트 이름만 안전하게 변환하면 기존 목록 ID·URL·데이터 계약을 유지하면서 해당 목록도 XLSX로 다운로드할 수 있다.
- 수용 기준:
  1) 실제 H2에 저장된 콜론 포함 ID(예: `sales:2026_<UUID8>`)의 `GET /api/v1/lists/{listId}/export.xlsx`가 200 및 기존 XLSX Content-Type/Content-Disposition으로 응답하고, 응답 바이트를 `XSSFWorkbook`으로 열 수 있다. 시트 이름은 콜론이 공백으로 바뀐 값이며 ID·이름 헤더와 씨딩한 행 값이 보존된다.
  2) 기존 정상 ID의 시트 이름은 그대로 유지한다. 앞/뒤 작은따옴표 ID와 31자 초과 ID도 안전한 시트 이름으로 열리도록 회귀 검증한다. 긴 ID는 수정 전에도 POI가 절단하므로 실패 사례가 아니라 기존 동작 보호 사례이다.
  3) 테스트는 실제 `NexaListRepository`/`EntityService`/`SchemaDdlService`, Spring MockMvc, 실제 POI를 사용한다. 콜론 사례를 수정 전에 실행해 HTTP 결과·근본 예외를 기록하고 수정 후 성공을 증명한다. `@WithMockUser`와 `sessionAttr("nexabuilder.user", adminSessionUser())`를 둘 다 유지한다.
  4) 기존 ListExportIntegrationTest 9건이 모두 통과하고 CSV/PDF, 삭제·권한 판정, `_offset`/`_limit` 동작은 유지한다.
- 건드릴 파일:
  - `src/main/java/com/nexabuilder/api/controller/ListExportController.java:exportXlsx` — 현재 169행 부근 `wb.createSheet(listId)`의 인자만 `WorkbookUtil.createSafeSheetName(listId)`로 변환하고 import 및 이유 주석 추가. 별도 정규식·새 의존성·범용 유틸은 불필요하다. 프로덕션 파일은 1개.
  - `src/test/java/com/nexabuilder/api/ListExportIntegrationTest.java:seed, xlsxExportOpensWithPoi` — 기존 `seed()`→`seed(null,1)` 관계를 보존하면서 ID를 선택할 수 있는 최소 오버로드 추가. UUID를 포함한 실제 목록 저장 및 XLSX 요청 테스트 추가. 식별자를 바꾸려고 기존 엔티티의 JPA @Id를 수정하지 말고 처음부터 원하는 ID로 새 목록을 저장한다.
- 검증 명령:
  - `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.ListExportIntegrationTest'` — 정찰에서 실제 실행 성공(25초), 9 tests / 0 failures / 0 errors / 0 skipped. 구현 전·후 동일 명령으로 새 사례 검증.
  - `sh ./gradlew --no-daemon test` — 변경 후 전체 검증(CI와 동일 test 태스크). 이번 정찰에서는 전체 실행하지 않음.
  - `sh ./gradlew --no-daemon bootJar -x test` — 변경 후 패키징(CI의 실제 태스크). 이번 정찰에서는 실행하지 않음.
- 위험과 피할 것: 목록 ID를 저장 단계에서 금지하거나 바꾸지 말 것. `UiBuilderController.saveList`는 시트 이름 문자 제한을 두지 않으며 `NexaListDef.listId`는 길이120의 문자열이다. 내보내기 파일명/CSV/PDF/페이징/RecordSet/SqlExecutor/auth/migrations/workflows는 범위 밖. 시트는 하나이므로 중복 이름 처리 추가 금지. `@DirtiesContext`, mock/직접 주입 컨트롤러, 문자열 grep만으로 통과 판정 금지. 응답 스트림을 여는 시점 때문에 기존 HTTP 실패가 400/500/부분응답 중 무엇인지는 미확인 — 정찰의 POI 예외를 HTTP 상태 증거로 쓰지 말 것.
- 차선 후보: SQL 기반 목록의 휴지통 어댑터 회귀 테스트 — `src/test/java/com/nexabuilder/api/SqlBackedListDefinitionIntegrationTest.java`의 실제 SQL 픽스처에 `deletedAt`을 설정하고 `GET /api/v1/data/lists/{id}`가 400이며 Rows가 없는지 검증한다. `sh ./gradlew --no-daemon test --tests 'com.nexabuilder.api.SqlBackedListDefinitionIntegrationTest'`; 정찰에서 이 명령은 미실행. 1순위가 실제 배선에서 성립하지 않을 때만 전환한다.

실행 순서와 체크포인트 (아래 작업 모두 구현자 대기, 사람 승인 단계 없음)
1. 재현: seed 오버로드와 콜론 사례만 추가하고 위 단일 클래스 명령 실행. 예상과 다르면 구현 전에 이 과제서의 진단을 고친다. 빨간 재현은 의도된 중간 상태이며 원인 기록 후 다음 단계로 간다.
2. 최소 수정: exportXlsx의 시트 이름 변환과 정상/긴 ID/작은따옴표 회귀 검증 추가. 동일 단일 클래스 명령이 통과한 뒤 다음 단계로 간다.
3. 완료: 전체 test와 bootJar 명령 실행. 새 실패가 있으면 범위 확장 없이 원인을 분리하고 결과를 회차 노트에 적는다.

선택 근거와 대안 (technology:solution-exploration)
- 채택: 기존 POI `WorkbookUtil` 재사용. 저장 계약을 바꾸지 않는 가장 작은 수정이며 추가 라이브러리가 없다.
- 고정 시트명 `Data`로 대체: 더 단순하지만 정상 ID를 시트명으로 쓰던 출력이 모두 달라져 제외.
- 저장 API에서 엑셀 제약 강제: 기존 저장된 목록을 고치지 못하고 CSV/PDF까지 불필요하게 제약하므로 제외.
- 현상 유지/테스트만 추가: 재현 가능한 라이브러리 실패를 남겨두므로 차선보다 가치 낮음.
- 핵심 가정: 콜론 ID가 실제 HTTP 경로에서도 목록 조회까지 도달한다. saveList의 별도 문자검사 없음은 확인했지만 해당 ID로 HTTP 재현은 아직 하지 않았다.

추정 근거 (pmo:estimating-and-contingency)
- Bottom-up: 실제 배선 재현/픽스처 8~12분 + 한 곳 수정 및 경계 검증 5~8분 + 전체 테스트/패키징·기록 10~15분 = 기본 23~35분. 알려진 변동(응답 스트림 오류 형태·테스트 실행시간)에 5분 contingency를 별도로 두어 총 28~40분, 45분 회차 내 완료 가능성은 중간 신뢰의 작업 추정이며 통계적 확률은 아니다.
- 비교: 직전 내보내기 과제도 동일 프로덕션 1파일/동일 테스트 클래스의 작은 회귀 수정이었다. 실제 소요 시간 기록이 없어 유사 사례로 S 규모만 교차 확인하며 시간 생산성 수치를 지어내지 않는다.
- 관리 예비비는 배정하지 않음(0분); 새로운 요구/인프라 문제는 범위 확대 대신 기록. POI 대체, 스트리밍, 권한 변경은 추정에서 제외.

정찰 근거
- 기준 커밋 `c446d24`, v1.26.0. `CLAUDE.md`/`AGENTS.md`는 저장소에서 발견되지 않았으며 README, docs/roadmap.md, docs/developer/testing.md, CI, git log -30 및 주요 후보 코드를 확인했다. TODO/FIXME는 src/main/java와 templates 검색 범위에서 없음.
- 지정 세 스킬은 전용 Skill 도구가 없어 `/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/{pmo,technology}/skills/`의 실제 SKILL.md를 읽어 적용했다. estimating-and-contingency의 references/sources.md도 읽었으며 외부 기관의 확률·예비비 수치를 인용하지 않았다.
- 실제 POI 타입 실행 기록은 같은 회차 폴더의 `PoiSheetProbe.java`, `poi-probe.log`. `sales:2026`는 Invalid char (:) 예외, 안전 변환 후 `sales 2026` 성공. 긴 이름은 수정 전에도 31자로 절단. 이는 라이브러리 동작 확인이며 Spring HTTP 결함 재현을 대신하지 않는다.
