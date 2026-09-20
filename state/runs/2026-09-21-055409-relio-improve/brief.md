- 과제: JSON 본문 크기 초과를 두 디코딩 단계 모두에서 413으로 응답하기 (가치 3 / 위험 1 / 작업량 S)
- 왜: `internal/platform/httpx/httpx.go:DecodeJSON`은 첫 Decode의 `*http.MaxBytesError`만 413으로 처리하여, 정상 JSON 뒤 공백으로 2MiB를 넘긴 요청은 두 번째 Decode에서 400 `invalid_json`으로 잘못 분류한다. 같은 크기 제한 위반을 413 `request_too_large`로 일관되게 알려 API 호출자가 JSON 구문과 용량 문제를 구별하게 한다.
- 수용 기준: 1) 첫 JSON 값 자체가 한도를 넘거나 정상 객체 뒤 공백을 읽다 한도를 넘으면 모두 false와 HTTP 413, `error.code=request_too_large`를 반환한다. 2) 정확히 2MiB인 유효 JSON+공백 및 그 이하 정상 객체는 허용하고, 한도 안의 잘못된 JSON·알 수 없는 필드·두 개의 JSON 값은 기존 400 `invalid_json`을 유지한다. Content-Length가 없는 스트림도 실제 읽은 바이트로 제한한다. 3) 실제 `http.MaxBytesReader`·`json.Decoder`를 거친 테스트가 위 경계값과 `requestId` 보존을 증명하고, 실제 `Server.createCustomer` 핸들러에 과대 요청을 전달하는 회귀 테스트가 DB 작업에 도달하지 않고 413을 돌려줌을 증명한다. 첫 Decode에서 발생한 초과와 두 번째 Decode에서 발생한 초과를 별도 케이스로 둔다.
- 건드릴 파일: `internal/platform/httpx/httpx.go:DecodeJSON` — 두 번째 Decode의 오류에서도 errors.As로 MaxBytesError를 우선 분류(작은 공통 오류 처리 함수는 선택); `internal/platform/httpx/decode_test.go`(신규) — 실제 요청/응답 객체로 경계값·기존 오류 계약 검증; `internal/server/decode_json_test.go`(신규) — 기존 `internal/server/crm.go:Server.createCustomer`를 호출하는 과대 요청 회귀 테스트. crm.go는 읽기 참조이며 수정 불필요.
- 검증 명령: 저장소 루트에서 `go test ./internal/platform/httpx ./internal/server`; `go test -race ./...`; `go vet ./...`; `go build ./...`; `gofmt -l internal/platform/httpx/httpx.go internal/platform/httpx/decode_test.go internal/server/decode_json_test.go`. 정찰에서 `go test ./...` 전체 통과. 프런트 변경은 없으므로 이 과제의 추가 web 빌드는 불필요.
- 위험과 피할 것: `auth/`, `oidc/`, `migrations/`, `.github/workflows/`를 수정하지 않는다. 2MiB 한도, 알려지지 않은 필드 거부, EOF 확인, 기존 400 메시지, null/배열 허용 여부 등 다른 계약을 바꾸지 않는다. Content-Length만으로 판정하거나 본문 전체를 미리 읽지 않는다. 요청/감사 로그에 본문 원문을 추가하지 않는다. 초과를 발견하기 전에 명백한 구문 오류로 거절되는 모든 요청까지 413으로 바꾸려는 범위 확장은 금지한다. 단순 소스 문자열 검사나 합성 MaxBytesError 주입을 증거로 삼지 않는다.
- 차선 후보: 감사 화면 Frame 부제를 실제 채널에 맞추기 (가치 1 / 위험 1 / S) — 1순위 재현이 현재 코드에서 더 이상 성립하지 않을 때만 `web/src/pages/AdminPages.tsx:Audit` 572행 부제의 Key를 빼고 SSO를 포함한다. 같은 컴포넌트 select의 WEB/API/MCP/ADMIN/LOGIN/SSO와 `docs/ADMIN_GUIDE.md` 5.3을 기준으로 맞추고 `(cd web && npm ci && npm run typecheck && npm run build && npm test)`로 확인한다. 두 과제를 함께 구현하지 않는다.

실제 확인 근거와 재현
- 기준 HEAD: `38c88ce`. `httpx.go` 35~55행에서 MaxBytesError 분기는 첫 Decode에만 존재한다. `crm.go:createCustomer` 44행부터 이 함수를 호출하고 false면 즉시 반환한다.
- 정찰이 저장소 밖 회차 폴더에 둔 Go overlay 테스트로 실제 핸들러를 호출했다. `{"name":"` + x 2MiB + `"}` → 413/request_too_large; `{}` + 공백 2MiB → 400/invalid_json. 두 번째 응답 메시지는 “요청에는 JSON 객체 하나만 허용됩니다.”였다. 두 요청 모두 도메인 서비스 없이 핸들러의 입력 거절 경로를 통과했다.
- 재현 명령: `go test -overlay=/mnt/c/Users/USER/projects/aidev/state/runs/2026-09-21-055409-relio-improve/repro-overlay.json ./internal/server -run TestScoutDecodeSize -v`. 이 파일은 현재 동작을 기록하는 정찰용이므로 수정 후 두 번째 케이스의 400 기대값을 회귀 테스트로 복사하지 말 것.
- 인증 미들웨어를 포함한 배포 서버 HTTP 검증은 미실시. 오류는 실제 서비스 핸들러와 공용 Decoder에서 재현됐고, DB·외부 서비스는 이 거절 경로에서 필요하지 않다.

대안 검토 (solution-exploration)
- 채택: 기존 스트리밍 두 단계 Decode를 유지하며 오류 분류만 통일. 한 공용 함수와 테스트로 끝나고 기존 호출자 계약을 보존한다.
- 보류: 공용 미들웨어에서 모든 본문을 선검증. 범위가 두 배로 커지는 장기 설계에는 검토할 수 있지만 업로드·MCP 등 별도 제한의 경계까지 바뀌므로 이번 작은 오류 수정에는 부적합하다.
- 기각: Content-Length 선검사만 추가. 길이를 모르는 요청의 실제 읽기 초과를 해결하지 못한다.
- 현상 유지: 요청은 이미 거절되므로 보안 우회는 아니다. 다만 잘못된 400 메시지로 API 운영 진단이 어긋나며 수정 비용이 작아 고친다.
- 핵심 가정: 이미 첫 단계가 사용하는 413/error code가 공용 크기 제한의 의도된 계약이라는 것. 외부 소비자가 이 특정 400에 의존하는지는 미확인이다.

실행 계획 (implementation-planning; 시작 상태 모두 미완료)
1. [ ] 위 overlay 명령으로 기준 응답 확인 후 httpx.go 수정과 새 httpx 테스트를 한 단위로 작성한다. 증명: `go test ./internal/platform/httpx`. 체크포인트: 구현자 자체 검토, 사람 승인 대기 없음.
2. [ ] server의 실제 createCustomer 경로 회귀 테스트 추가. 과대 JSON 값과 과대 후행 공백 각각 검증. 증명: `go test ./internal/server`. 체크포인트: 실제 핸들러 호출인지 확인, 사람 승인 대기 없음.
3. [ ] 전체 race·vet·build 및 지정 파일 gofmt 검사. 실패 시 범위를 넓히지 말고 과제서와 차이를 기록한다. 체크포인트: 후속 비평 에이전트에 검증 결과와 미검증 사항 인계(정찰은 구현·커밋하지 않음).

작업량 근거 (estimating-and-contingency)
- bottom-up: 재현/테스트 입력 정리 5분, 공용 분기 수정+경계값 테스트 10~15분, 실제 핸들러 회귀 테스트 5분, 전체 검증/인계 5~10분 = 기본 25~35분.
- 알려진 변동 예비 5분: Decoder의 읽기 선행/정확한 경계 입력 조정 및 race 실행 편차. 총 30~40분 예상, 신뢰 중간(통계적 확률 아님). 경영 예비는 별도 0분이며 미지의 범위 추가 권한으로 쓰지 않는다.
- 전제: Go 도구/모듈 캐시가 현재 정찰처럼 준비됨. 입력 근거는 직접 읽은 단일 함수, 실제 재현 결과, 전체 테스트 통과. 과거 회차는 실제 소요 시간이 없으므로 유사 추정의 수치 근거로 사용하지 않았다.
- 포함: 공용 동작 수정·두 패키지 테스트·검증·인계. 제외: DB 마이그레이션, API 재설계, 인증 정책, 프런트, 배포. 45분을 넘길 외부 검증 문제가 발견되면 내용을 기록하고 완료라고 주장하지 않는다.

스킬 적용: Skill 도구가 노출되지 않아 `/mnt/c/Users/USER/projects/headcount/plugins/` 아래 pmo/skills/estimating-and-contingency 및 technology/skills/{implementation-planning,solution-exploration}/SKILL.md를 직접 읽었다. 추정 스킬의 references/sources.md도 확인했으며, 외부 원문은 열지 않아 외부 기준이나 확률 모델에 근거한 추정이라고 주장하지 않는다.
