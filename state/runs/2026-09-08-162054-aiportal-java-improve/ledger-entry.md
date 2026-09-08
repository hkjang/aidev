## 2026-09-08
- 선택: 게시판·자료실 등록·수정·삭제에서도 조회수가 오르는 문제 수정 (가치 3 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `BoardServiceImpl.get()`(62행)과 `LibraryServiceImpl.get()`(61행)이 상세 조회와 `view_cnt` 증가를 한 메소드에서 겸하고 있어서, 내부적으로 상세를 다시 읽는 `put()`·`modify()`·`delete()` 경로에서도 조회수가 올랐습니다 — `modify()` 는 수정 전후로 상세를 두 번 읽으므로 `PUT` 한 번에 +2, 등록과 삭제도 각각 +1 이라, 아무도 읽지 않은 글이 작성·수정만으로 조회수 4 를 갖게 됩니다(게시판 조회수 통계가 편집이 잦은 글에서 계속 부풀려짐). 조회수 증가 여부를 인자로 받는 private `find(id, countView)` 로 분리해 컨트롤러의 상세 조회 엔드포인트(`get()`)에서만 증가시키고 나머지 내부 조회는 증가시키지 않도록 했으며, 없는 글에 대한 `BOARD_NOT_FOUND` 동작은 그대로 유지했습니다. 검증은 `BoardServiceImplViewCntTest`·`LibraryServiceImplViewCntTest`(각 5: 상세 조회 시 정확히 1회 증가 / 없는 글은 예외 + 증가 없음 / 등록 증가 없음 / 수정 증가 없음(예전 +2 회귀 가드) / 삭제 증가 없음) 를 추가한 뒤 `sh gradlew check build` 실행으로 했고 26개 테스트 클래스 123개 테스트 전부 통과했습니다. 커밋 e10459a. (참고: 이 세션 환경에도 JDK 가 없고 JRE 만 있어 Temurin 21 을 `$HOME/jdks` 에 내려받아 빌드했습니다. 저장소에는 아무 변경도 하지 않았습니다. 또한 이 워크트리는 여전히 3ffa5ca 기준이라 직전 다섯 세션의 커밋이 아직 병합되지 않아, 해당 세션들이 건드린 파일은 충돌을 피하려고 이번 범위에서 제외했습니다.)
- 보류 아이디어:
  - 컨트롤러가 `@RequestParam user_id` 를 인증 사용자와 대조 없이 신뢰 (IDOR) — `ToolsController.preview` 는 매퍼에도 user_id 필터가 없어 id 증가만으로 남의 OCR 원본을 받을 수 있다 (가치 5 / 위험 4 / 작업량 L)
  - `SystemPolicyUtil.saveExtension`/`saveNote` 의 `!isEmpty()` 가드로 빈 배열 저장이 조용히 무시되는데 API 는 success 를 반환 — `deleteExtension` 의 마지막 항목 삭제도 반영되지 않는다 (가치 3 / 위험 2 / 작업량 S)
  - 인증 필터의 요청 헤더 전량 INFO 로깅 축소 — 요청마다 모든 헤더를 남겨 운영 로그에서 실제 오류를 가린다 (가치 3 / 위험 2 / 작업량 S)
  - 남은 외부 응답 무검증 접근에 `JsonNodeUtil` 확대 적용 — 유틸을 만든 커밋 602dfc3 이 main 에 병합된 뒤 진행할 것 (가치 3 / 위험 2 / 작업량 M)
  - `ToolsServiceImpl.decFileDrm` 이 `DrmUtil` 을 `new` 로 직접 생성해 테스트 불가 — 무상태 객체이므로 `@Bean` 으로 주입하면 실패 처리를 단위 테스트할 수 있다 (가치 2 / 위험 2 / 작업량 S)
