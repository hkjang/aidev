
[`v1.18.0`](https://github.com/hkjang/nexabuilder/releases/tag/v1.18.0)
은 목록 내보내기 수정 릴리즈 (#27). 휴지통에 든 목록이 내보내기 URL
로는 계속 내려받아지던 구멍을 막고, 내보내기 3종에 통합 테스트를 처음
붙였습니다.

### 수정 (#27)

- **휴지통에 든 목록은 CSV/XLSX/PDF 내보내기에서도 "찾을 수 없음"**
  (#27). `ListExportController` 의 세 엔드포인트는 `findById` 만 하고
  `deletedAt` 을 보지 않아, 런타임 `/app/list/<id>` 가 "List not found"
  로 막는 목록의 데이터를 `/api/v1/lists/<id>/export.{csv,xlsx,pdf}` 로는
  계속 받을 수 있었다. 세 곳에 복붙되어 있던 조회·권한 블록을
  `resolveExportable()` 하나로 모으고, 거기서 `NexaUiService.listDefinition`
  과 똑같은 `IllegalArgumentException("List not found: …")` 을 던져 런타임
  화면과 같은 400 JSON 으로 끝나게 함. 검사는 `response.getOutputStream()`
  을 열기 전에 두어 부분 파일 바이트가 새지 않는다. 403 경로는 그대로
  (`GlobalExceptionHandler` → `AccessAuditHandler` 감사 기록).

### 테스트 (#27)

- **`ListExportIntegrationTest` 신규** (#27). 실제 H2 ·
  `ScreenPermissionService` · POI · OpenPDF 를 그대로 타는 5건 — CSV
  BOM/헤더/행, XLSX 를 POI 로 다시 열기, PDF `%PDF-` 매직 + content-type,
  soft-delete 목록 3형식 모두 400, 세션 user 없음 3형식 모두 403.
  컨트롤러 수정 전에는 soft-delete 케이스만 "expected 400 but was 200"
  으로 빨갛던 것을 확인한 뒤 초록으로 만듦. `cleanTest test` 573 건
  (568 + 5) 통과.

