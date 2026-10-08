# 후보 재평가 — 2026-10-09

현재 HEAD 329656c. AGENTS.md·CLAUDE.md는 저장소 및 확인한 상위 경로에 없었다. TODO/FIXME 검색은 internal/scripts/web/src에서 항목이 없었다. 최근 30개 커밋과 README·architecture·ROADMAP·CI·Makefile을 읽었다. Go REST/MCP, React UI, PostgreSQL, 오프라인 단일 바이너리가 제품 구조다. 현재 test 게이트는 Go test/vet 및 npm ci/audit/typecheck/test/build다. 로드맵의 현재 버전·VOC 예정 표기는 실제 코드와 어긋난다. 과거 릴리즈 롤백과 enum 계약 테스트 이력은 반복하지 않는다.

|후보|가치|위험|작업량|판정|
|---|---:|---:|---|---|
|web/package.json 에 esbuild 를 devDependency 로 선언|3|2|S|pending|
|release.yml 의 Test source 단계에 npm --prefix web test 추가|3|2|S|pending|
|Makefile test 에 previous-release-tag-test.sh 연결|3|1|S|pending|
|감사 목록 기간 필터(from/to) 서버·OpenAPI 쪽만|3|1|M|pending|
|voice.KnowledgeStatuses 를 참조하지 않는 손 목록이 남아 있는지 확인해 같은 관용으로 정리|2|1|S|rejected|
|연결 성립 후 끊긴 질의(net.OpError·EOF)도 원문 없이 500 으로 접기|3|2|M|pending|
|httpx.ClientIP 가 X-Forwarded-For/Forwarded 를 읽지 않음|4|3|M|pending|
|serviceError 의 영어 substring 분류를 센티널 오류(errors.Is)로 옮기기|3|3|M|pending|
|enum 그물을 CHECK 가 없는 공표 enum 쪽으로 넓히기 (forecastCategory)|2|2|M|pending|
|업그레이드 검증 실패 시 cleanup 전에 단계명과 컨테이너 진단 보존|3|1|S|pending|
|로컬 npm(10.9.8)과 CI npm(Node 24 → 11.x) 버전 불일치를 저장소에서 못박기|3|2|S|pending|
|프런트 API 클라이언트가 읽을 수 없는 성공 응답을 성공 객체로 반환하지 않게 수정|4|1|S|done|
|프런트 날짜 포매터의 잘못된 날짜 입력 처리|3|1|S|pending|

새 후보 2개를 포함해 13개 평가. 실제 사용자 호출 경계에서 실패를 재현할 수 있고 프로덕션 1파일로 해결 가능한 API 응답 오류 처리를 선택했다. 과제서가 아닌 프로젝트 프로필만 제공되어 과제서 판정은 적용하지 않는다.
