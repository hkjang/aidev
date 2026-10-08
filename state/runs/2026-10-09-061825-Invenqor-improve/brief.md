- 과제: MCP asset_get·asset_relations의 asset_id를 canonicalUUID로 검증·정규화 (가치 3 / 위험 1 / 작업량 M)
- 왜: 두 도구는 입력 스키마에 UUID를 선언하지만 RequiredString만 거친 asset_id를 SQL에 직접 넘겨 DB 방언별 입력 해석 차이에 노출되어 있다. REST에서 검증된 canonicalUUID를 재사용하면 비정규 입력에 수정 가능한 오류를 주고 대문자로 전달한 기존 자산·관계·병합 안내를 일관되게 찾을 수 있다.
- 수용 기준:
  1) 두 도구 모두 UUID가 아닌 문자열, urn:uuid: 접두사, 중괄호형, 하이픈 없는 32자형을 SQL 조회 전에 거절한다. 실제 인증을 거친 POST /mcp tools/call은 HTTP 200, result.isError=true, content의 텍스트에 asset_id와 허용 형식(36자 하이픈 UUID)을 담는다. SQLSTATE·DB 드라이버 오류가 사용자 응답에 나타나지 않는다. 새 HTTP 400/500 또는 JSON-RPC 최상위 오류로 바꾸지 않는다.
  2) 소문자·대문자·혼합 대소문자의 정상 36자 UUID는 같은 자산과 active inbound/outbound 관계를 반환한다. 공용 String이 이미 제거하는 앞뒤 공백은 계속 허용한다. 병합된 secondary의 대문자 ID도 동일한 merged_into를 반환한다. 정규형 미존재 ID는 asset_get의 기존 asset not found와 asset_relations의 빈 페이지를 유지한다.
  3) 실제 Runtime/DB/라우터를 쓰는 회귀 테스트를 SQLite·PostgreSQL에서 통과시킨다. 정상 입력 및 비정규 입력의 수정 전 실패를 먼저 기록하고, 수정 후 기존 병합 안내·페이지네이션·프로토콜·인수 변환 테스트까지 통과시킨다. 고정 UUID 또는 문자 a-f를 반드시 포함하는 fixture로 대문자 테스트가 무의미해지지 않게 한다.
- 건드릴 파일:
  - server/internal/httpapi/mcp.go:693 mcpAssetGet, :773 mcpAssetRelations — 기존 arguments.Err() 처리 다음, 첫 SQL 앞에서 canonicalUUID 적용. 정규화한 input.AssetID를 SQL과 mcpMergedAssetHint 양쪽에 전달. 실패 시 도구 이름·asset_id·허용 형태를 설명하는 error를 반환하여 기존 writeMCPToolError 경로 사용. 두 곳에 작은 검증만 추가하고 공용 파서를 확장하지 않는다.
  - server/internal/httpapi/mcp_asset_id_validation_test.go (신규) — 위 수용 기준의 테이블 테스트. 프로덕션 1파일 + 테스트 1파일로 제한한다.
- 검증 명령 (저장소 루트 기준, 첫 명령만 정찰에서 실행):
  - `(cd server && go test ./internal/httpapi/ -run 'MCP|Arguments' -count=1)` — 현재 main@16f2592에서 exit 0, httpapi 3.214s. 새 테스트 이름도 TestMCP로 시작시킨다.
  - `POSTGRES_CONTAINER=invenqor-20261009-061825-mcp-uuid POSTGRES_PORT=55549 ./scripts/test-postgres.sh -run 'MCP|Arguments' -count=1` — 스크립트 실제 존재와 인수 전달 확인, 이번 실행은 미확인. 55549는 정찰 ss 조회에서 listener 없음; 실행 직전 다시 확인한다.
  - `(cd server && go test ./... && go vet ./... && go build -o /dev/null ./cmd/invenqor-server)` — CI와 모듈 설정에서 확인한 명령, 이번 정찰 전체 실행은 미확인.
  - `gofmt -l server/internal/httpapi/mcp.go server/internal/httpapi/mcp_asset_id_validation_test.go` 및 `git diff --check` — 구현 완료 후 출력 없음.
- 위험과 피할 것: auth/session·API key 서비스·migrations·workflows·공용 mcp_arguments.go·프로토콜 협상·도구 스키마·문서/PDF·버전·REST 관계 핸들러는 범위 밖이다. uuid.Parse 단독 사용은 비정규형도 받으므로 금지한다. MCP가 반환하는 에러는 HTTP 500이라는 과거 REST 설명을 복사하지 말 것. PostgreSQL 스크립트는 지정 이름의 기존 컨테이너를 삭제하므로 회차 고유 이름을 유지한다. confidence=0 수정(7250899)은 성공 기록이 있지만 현재 main에 없으며 auto/2026-10-08-0438에만 포함됨을 확인했다; 이 작업에 재구현하거나 체리픽하지 않는다.
- 차선 후보: 관계 생성 confidence JSON 타입 오류 회귀 테스트 보강 (가치 2 / 위험 1 / S) — 1순위가 현재 구현에서 이미 해결됐거나 실제 실패를 재현할 수 없을 때에만 선택. 기존 asset_relation_validation_test.go의 relationValidationClient를 사용해 console/external 양쪽에서 문자열·bool·배열·객체·1e309를 400 INVALID_RELATION으로 거절하고 관계 및 relation.create 감사가 늘지 않음을 증명한다. 기존 숫자 0의 기대값은 바꾸지 않는다. 검증은 `(cd server && go test ./internal/httpapi/ -run 'Relation' -count=1)`과 동일 PostgreSQL 스크립트의 -run 'Relation'; 이번 실행은 미확인.

확인 근거와 구현 순서 (사람 승인 체크포인트 없음, 각 단계 검증 후 진행)

1. [미착수] 회귀 테스트와 최소 수정까지 한 단계로 완결한다. 먼저 테스트를 작성해 기존 동작의 실패를 기록한 뒤 두 핸들러를 수정하고 좁은 SQLite 명령을 통과시킨다. 중간 실패는 의도된 재현 증거이며 단계 완료로 표시하지 않는다.
   - mcp.go:50,60의 두 InputSchema는 asset_id format: uuid. mcp_arguments.go:RequiredString/String은 필수 값 및 형 변환만 수행하고 format을 검사하지 않는다. String의 TrimSpace는 보존한다.
   - assets.go:712 canonicalUUID는 uuid.Parse + len==36 + parsed.String()으로 정규형만 수락하고 소문자로 반환한다. 이 함수를 변경 없이 호출한다.
   - mcp_protocol_test.go:performMCPRequest/performRawMCPRequest/decodeMCPResult와 modernMCPMetadata를 재사용한다. newRuntime/testServer/authenticateInitialAdmin으로 서버와 인증을 만들고 external_api_test.go:166 createAPIKeySecret에 mcp.access, assets.read, relations.read 세 scope를 준다.
   - mcpProtocolTestServer는 relations.read가 없고 기존 discovery 테스트가 그것을 전제로 하므로 공용 helper를 바꾸지 않는다. 이번 테스트 파일 안에서 필요한 키를 별도로 발급한다.
   - tools/call 본문은 jsonrpc=2.0, id, method, params.name, params.arguments. legacy는 기존 기본 호출을 사용하고 modern은 params._meta=modernMCPMetadata(), MCP-Protocol-Version=mcpModernProtocolVersion, Mcp-Method=tools/call, Mcp-Name=도구명 헤더를 넣는다. 입력 표 전체는 한 프로토콜로 검사하고 다른 프로토콜에서도 도구별 정상/비정상 smoke를 검사하면 충분하다.
   - software_inventory_test.go:insertSoftwareTestAsset는 실제 DB fixture, mcp_asset_get_merged_test.go:mergeAssetsThroughREST는 실제 병합 API helper다. 병합 대문자 회귀는 별도 서버 fixture로 만들어 일반 자산/관계 조회 테스트와 간섭하지 않게 한다. 기존 TestMCPAssetGetPointsAMergedAssetAtItsPrimary, TestMCPAssetGetFollowsOneMergeHopOnly, mcp_pagination_test.go의 TestMCPAssetRelations*를 회귀 검사에 포함한다.
2. [미착수] PostgreSQL에서 같은 표를 검증한다. 양 DB의 조회 의미가 같고 raw SQL 에러가 응답에 없어야 한다. PostgreSQL 실행 환경이 없으면 성공으로 표기하지 말고 제약을 기록한다. 예상과 다른 기존 오류·라우팅이 나오면 과제서를 수정하고 범위를 넓히지 않는다.
3. [미착수] 전체 Go test/vet/build와 포맷·diff 검사를 실행한다. 변경 파일은 위 두 개에 머물러야 한다. 신규 환경/의존성 이슈를 이 과제에 섞지 않는다.

해결안 비교와 추정 근거

- 채택: 두 핸들러가 기존 canonicalUUID를 호출. 새 구성요소 없이 프로덕션 1파일로 끝나고 기존 REST의 입력 계약을 재사용한다.
- 보류: mcpArguments에 모든 JSON Schema format 검증을 일반화. 향후 도구 확대에는 유용하지만 이번 두 입력보다 영향 범위와 회귀 위험이 크다.
- 보류: 현행 유지 후 문서로 제한 안내. 코드 변경은 없지만 DB 방언 차이와 잘못된 성공/빈 결과 가능성을 해소하지 못한다.
- 핵심 가정: 선언된 uuid의 의미를 REST와 같은 36자 하이픈형으로 통일한다. 코드·스키마 근거는 확인했지만 이번에 비정규/대문자 실제 MCP 요청의 수정 전 결과를 실행해 보지는 않았다. PostgreSQL에서 에러가 나는 정확한 입력과 응답 문구도 미확인이며 구현 1단계가 이를 실측해야 한다.
- 추정: 바닥부터 합산한 기본 작업 22–33분(인증/fixture 6–10, 입력표·병합 회귀 8–10, 두 검증 추가 3–5, 검증/정리 5–8). 알려진 변동인 PostgreSQL 기동·테스트 조정에 예비 5–10분을 별도로 두어 총 27–43분, 정성적 확신 중간. 과거 relation UUID 과제와 생산 1파일+테스트 구조가 유사하나 실제 소요시간이 기록되지 않아 유사사례 정량 추정은 미확인이다. 미지의 추가 범위용 관리 예비는 이번 회차에 배정하지 않는다. 기본 항목에 예비를 중복 가산하지 않았으며 45분 초과 전망이면 범위를 확장하지 않고 미검증 항목을 남긴다.
- 적용 스킬: Skill 호출 도구는 현재 세션에 없어 로컬 원본을 직접 읽었다. [estimating-and-contingency](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md), [implementation-planning](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/implementation-planning/SKILL.md), [solution-exploration](/home/hkjang/.claude/plugins/marketplaces/headcount/plugins/technology/skills/solution-exploration/SKILL.md). 추정은 저장소와 과거 회차 근거에 한정하며 외부 가이드의 통계적 신뢰수준을 주장하지 않는다.
