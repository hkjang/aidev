- 과제: OpenAPI 계약 테스트에 실제 YAML 파싱 검증을 추가해 구문 오류 재발 차단 (가치 3 / 위험 1 / 작업량 S)
- 왜: internal/api/openapi_contract_test.go의 specOperations는 들여쓰기와 문자열만 읽으므로 경로가 그대로인 잘못된 YAML도 통과할 수 있으며, 695591e가 실제로 이런 문서 오류를 수정했다. 저장소의 실제 openapi/openapi.yaml 전체를 YAML 파서로 검사하면 문서를 소비하는 도구가 읽지 못하는 릴리스를 기존 Go 테스트 단계에서 차단할 수 있다.
- 수용 기준: 1) go test의 OpenAPI 테스트가 디스크의 openapi/openapi.yaml 전체를 실제 YAML 라이브러리로 파싱하며 현재 문서는 통과한다. 2) 695591e 이전의 인용 없는 /login?sso=limited 설명과 닫히지 않은 flow mapping이 실제 파서 오류로 거부되고, 올바로 인용한 동일 설명은 통과한다. paths 밖 components에 넣은 구문 오류도 거부해 전체 문서 검사를 증명한다. 3) 기존 TestOpenAPIDocumentCoversEveryRegisteredRoute, TestOpenAPIDocumentHasNoRouteWithoutHandler, TestUndocumentedRouteExceptionsStayCurrent의 경로·예외 검사 의미가 유지되고 모두 통과한다. 4) 검증은 실제 YAML 파서와 실제 문서 읽기 경로를 거친다. 파서 대역이나 소스 문자열 포함 여부만으로 성공을 주장하지 않는다. 5) Go 모듈 정리가 재실행해도 변하지 않고, 런타임 코드·API 문서 내용·인증·워크플로 변경이 없다.
- 건드릴 파일: internal/api/openapi_contract_test.go:specOperations 및 OpenAPI 테스트 — 실제 문서 파싱을 선행시키거나 별도 TestOpenAPIDocumentIsValidYAML을 추가하고, 같은 파싱 함수를 쓰는 잘못된/정상 YAML 회귀 검증을 둔다. go.mod 및 필요시 go.sum — 테스트가 직접 import하는 gopkg.in/yaml.v3 v3.0.1을 명시한다(현재 go.sum에는 있으나 go.mod 직접 require에는 없음). openapi/openapi.yaml은 읽기 대상이며 내용 변경 대상 아님.
- 검증 명령: go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api ; go vet ./... ; go test -count=1 ./... ; go mod tidy (정리 후 한 번 더 실행하여 go.mod/go.sum 무변경 확인). 의존성 명시가 필요하면 go mod edit -require=gopkg.in/yaml.v3@v3.0.1 후 go mod tidy. DB/Node/Python은 이 과제의 테스트 실행에 필요하지 않다.
- 위험과 피할 것: internal/auth/, migrations/, .github/workflows/, 런타임 server.go를 건드리지 않는다. 기존 들여쓰기 추출기를 전면 교체하거나 OpenAPI 스키마 전체 검증·코드 생성·신규 CI 도구로 확대하지 않는다. YAML 구문 유효성과 OpenAPI 의미 유효성은 다르므로 후자를 보장한다고 쓰지 않는다. 외부 Python/yq 실행에 의존하지 않는다. 문서의 기존 정상 구문을 다시 고치는 과제가 아니라 검증 공백을 막는 과제다. 모듈 정리 시 관련 없는 업그레이드가 생기면 원인을 확인하고 범위를 복원한다.
- 차선 후보: requestList의 HTTP 오류 code 보존 — web/src/api/client.ts의 request는 envelope.error.code를 전달하지만 requestList는 버린다. 동일 실제 Response로 request/requestList 양쪽의 status·message·code, 기존 목록 평탄화와 meta 보존을 검증한다. 현재 UI가 code로 분기하는 소비자는 확인되지 않아 가치 2 / 위험 1 / S이며 1순위가 이미 구현됐거나 모듈 추가가 불가능할 때만 선택한다.

범위와 대안 판단
- 선택안: 테스트 안에서 실제 파싱을 추가하고 기존 경로 비교는 유지. 수정 범위가 작고 이미 발생했던 장애를 막는다.
- 확대안: 파싱 결과로 specOperations까지 교체하면 따옴표·들여쓰기 변형도 지원하지만 경로 추출 계약까지 바뀌므로 이번에는 제외한다.
- 외부 CLI 검증안: 코드는 적지만 개발자와 CI에 별도 도구 설치가 필요해 제외한다. 현상 유지안은 695591e 유형 재발을 못 잡는다.
- 핵심 가정: 테스트 전용 YAML 의존성 명시가 허용되고 기존 모듈 정리로 v3.0.1을 안정적으로 해석할 수 있다. 정찰에서 Go YAML 도입 후 모듈 그래프는 미확인이다.

실행 순서와 체크포인트 (구현자 기록용, 모두 미착수)
1. [ ] 실제 문서 읽기와 실제 파서 검증을 테스트 파일에 추가하고 의존성을 정리한다. 현재 문서로 go test -count=1 -run 'OpenAPI|UndocumentedRoute' ./internal/api 통과 확인. 자동 체크포인트이며 사람 승인 불필요.
2. [ ] 같은 검증 경로에 정상 인용·과거 오류·components 오류 회귀 사례를 추가한다. 가능하면 현재 문서를 메모리에서 복제해 잘못된 설명으로 바꾼 입력을 검사한다. 위 명령으로 정상/거부 모두 증명하고, 파싱을 우회한 변형에서 오류 사례가 실패하는지 확인한다. 실패 변형은 반드시 원복. 자동 체크포인트.
3. [ ] go vet ./... 및 go test -count=1 ./... 실행, go mod tidy 재실행 무변경 확인, 변경 파일 범위를 검토하고 결과를 journal에 남긴다. DB DSN 없는 통합 skip을 실검증으로 세지 않는다. 계획과 현실이 어긋나면 먼저 과제서에 근거와 범위 수정 기록.

작업량 산정 근거
- bottom-up: 파서·문서 배선 8~12분, 회귀 사례 8~12분, 모듈·전체 검사 및 기록 5~8분 = 기본 21~32분.
- 알려진 변동(모듈 정리/진단 문구) 대응 contingency 3~6분 별도, 합계 24~38분. 이는 정찰자의 중간 신뢰 판단이며 통계적 80% 보장이나 확정 납기가 아니다. 별도 management reserve는 이번 45분 세션에 배정하지 않으며 범위 확대는 후속 후보로 남긴다.
- 근거는 직접 읽은 단일 테스트 파일, 이미 go.sum에 있는 파서, 정찰 Go 테스트 성공. 과거 유사 작업의 실제 소요시간은 기록이 없어 유추 수치로 꾸미지 않았다.

정찰 확인 결과
- 기준 HEAD 9bebc9a, VERSION 1.7.2. go test ./... 통과(일부 캐시; DB DSN 없음), 스크린샷 검사 30개·21경로 통과, 버전 검사 통과.
- PyYAML로 현재 실제 문서 59 paths 파싱 성공. git show 695591e^:openapi/openapi.yaml의 원문은 117행 /login?sso=limited에서 파싱 실패. Python은 정찰 재현에만 사용했으며 구현 검증의 의존성이 아니다.
- web/node_modules 없음: 프런트 테스트 미실시. DB 통합·실제 Keycloak 검증 미실시.
- 적용 스킬: /mnt/c/Users/USER/projects/headcount/plugins/pmo/skills/estimating-and-contingency/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/implementation-planning/SKILL.md, /mnt/c/Users/USER/projects/headcount/plugins/technology/skills/solution-exploration/SKILL.md. 전용 Skill 도구가 없어 로컬 원문으로 읽었다. 추정 스킬의 references/sources.md도 확인했으며 외부 비용·편익 수치나 통계적 신뢰수준은 인용하지 않았다.
