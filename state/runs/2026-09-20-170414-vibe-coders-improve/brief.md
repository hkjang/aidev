- 과제: 언어 추론의 신뢰도 상승 시 기존 evidence 보존 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/audit/language.go:addSignal`은 높은 신뢰도 신호를 만나면 기존 `LanguageSignal`을 덮어써 앞 메시지에서 수집한 근거를 잃는다. 최대 신뢰도는 유지하면서 근거를 누적하면 동일한 대화의 메시지 순서 때문에 감사 상세의 설명 근거가 빠지는 문제가 사라진다.
- 수용 기준:
  1) 공개 함수 `audit.InferLanguages([]string{"pytest", "worker.py", "```python\npass\n```"})`의 Python 신호는 confidence 0.95이고 evidence에 `keyword: pytest`, `file extension: .py`, `code fence: python`을 모두 포함한다. 역순 입력도 같은 근거 집합과 신뢰도를 반환한다. 문자열 순서까지 같을 필요는 없으며 기존 발견 순서와 `; ` 구분 형식을 유지한다.
  2) 동일 근거를 여러 메시지에서 다시 만나도 중복해서 추가하지 않는다. 낮거나 같은 신뢰도 신호도 기존처럼 누적하고, confidence는 최댓값이며 언어 정렬·상위 5개 제한·언어 판정 규칙은 유지한다.
  3) `internal/audit/language_test.go`에 낮음→중간→높음, 역순, 반복 입력 회귀를 추가한다. `addSignal`만 직접 호출하는 테스트에 그치지 말고 실제 `InferLanguages`를 통과시키며, 첫 사례는 수정 전 실패해야 한다.
  4) 실제 요청 JSON을 `extractAudit`로 추출한 prompts를 `auditRequestWithPrompts`에 전달한 결과와 `auditRequest`의 새 추출 결과에서 같은 언어·신뢰도·근거 집합을 확인한다. `audit_prompt_reuse_test.go:TestAuditWithoutReusedPromptsIsUnchanged`가 현재 개수만 비교하므로 이를 보강할 수 있다. 민감정보 없는 위 예제를 사용하고 redaction 정책은 바꾸지 않는다.
  5) `server_test.go:TestChatCompletionStreamingProxyAndAsyncAudit`의 fixture 방식(NewServer → Routes → 실제 HTTP 요청 → store.NewAsyncLogger → SQLite)을 활용해 다중 메시지 한 요청의 저장된 `RequestDetail.Languages`에 세 근거가 남는 것을 증명한다. 요청/감사 객체를 손으로 만들어 DB에 넣거나 소스 문자열 검사로 대체하지 않는다. 기존 스트리밍 타이밍 검증은 그대로 두고 독립 테스트를 추가하는 편이 안전하다.
- 건드릴 파일:
  - `internal/audit/language.go:addSignal` — 신규 언어 초기화와 기존 신호 갱신을 분리하고, confidence 갱신과 evidence 누적을 서로 독립적으로 처리한다. 기존 substring 기반 중복 판정 자체의 확장은 이번 범위 밖이다.
  - `internal/audit/language_test.go:TestInferLanguagesUsesStrongSignals` 주변 — 공개 함수 회귀 사례 추가.
  - `internal/proxy/audit_prompt_reuse_test.go:TestAuditWithoutReusedPromptsIsUnchanged` — 실 추출을 거친 두 감사 경로의 근거 동등성 검증.
  - `internal/proxy/server_test.go:TestChatCompletionStreamingProxyAndAsyncAudit` 주변 — 실제 HTTP·비동기 저장 통합 테스트 추가(새 전용 *_test.go로 분리해도 됨).
  - 읽기 참조만: `internal/proxy/server.go:languagesFromPrompts, extractAudit, auditRequestWithPrompts`, `internal/store/queries.go:RequestDetail`, `internal/store/types.go:RequestDetail`. 이 프로덕션 경로와 스키마는 변경할 필요 없다.
- 검증 명령:
  - `go test ./internal/audit -count=1`
  - `go test ./internal/proxy -run 'Language|Audit|ReusedPrompts' -count=1`
  - `go test -race ./internal/audit -count=1`
  - `go vet ./...` 및 `go test ./... -count=1`
  - 정찰에서 실행 확인: `go test ./internal/audit -count=1` 통과, `go test ./internal/proxy -run 'TestChatCompletionStreamingProxyAndAsyncAudit|TestBuildAuditReuses' -count=1` 통과. 후자의 TestBuildAuditReuses는 존재하지 않아 실제 실행은 첫 테스트뿐이다. 새 회귀 및 전체 suite는 아직 실행하지 않았음.
- 위험과 피할 것: auth/session·migrations·workflows·.env 계열·가격 계산·SQL timeout은 건드리지 않는다. evidence에는 기존의 정형 근거 문자열만 누적하고 프롬프트 원문이나 파일 전체 경로를 새로 넣지 않는다. 순서 비교는 근거 집합으로 하되 중복 여부는 별도로 검사한다. 이전 가격·메일·MCP·timeout 구현이 master에 없다는 이유로 재구현하거나 cherry-pick하지 않는다. `web/package.json`은 아직 직접 tsc/vite를 실행하므로 러너의 `pnpm run typecheck --silent` 문제는 과거 기록상 남아 있다(이번 러너 실제 생성 명령은 미확인); 이 과제와 섞어 래퍼를 다시 이식하지 말고 검증 인프라 문제로 구분해 기록한다.
- 차선 후보: `docs/APP_UI_ROADMAP.md`의 추적 활성 페이지 캐시 정책 설명 정정 (가치 2 / 위험 1 / 작업량 S) — 현재 문서는 index.html이 항상 no-cache라고 쓰지만 `internal/appui/handler.go:serveTrackedIndex`는 nonce 페이지를 no-store로 응답한다. 활성 추적의 no-store·ETag/Last-Modified 미사용 예외를 문서에 적고, 기존 `internal/appui/tracking_test.go`를 읽어 `go test ./internal/appui -count=1`로 확인한다. 런타임 동작은 변경하지 않는다.

실행 계획·추정: 5분 재현 및 회귀 작성 → 5분 addSignal 최소 수정 → 15분 감사 경로·HTTP 저장 테스트 → 10분 검증·정리, 예비 10분(총 상한 45분). 느린 전체 race 대신 audit race를 사용한다. 1순위 근거가 성립하지 않을 때만 차선을 택하며 시간 부족으로 통합 증거를 대역으로 바꾸지 않는다.
탐색 판단: 방문 추적 textarea는 가치 3이나 화면 저장·권한 검증과 프런트 검증 부담이 더 크고, SSO 마커 수정은 인증 경로다. 이번 선택은 미착수 보류 항목 중 코드상 결함과 저장 영향이 명확하며 보호 경로를 피한다. TODO/FIXME 검색은 internal·web/src·scripts에서 결과 없음. 최근 git log 30건과 README·로드맵·운영/보안/라우팅 문서·테스트 안내·CI를 확인했다.
스킬 제약: 요청된 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration을 제공하는 Skill/skills 도구가 현재 세션에 없다. 로컬 .codex/.claude 및 aidev·회차 home에서도 해당 SKILL.md를 찾지 못했다. 따라서 고유 절차·반환 형식 준수 여부는 미확인이고, 이 과제서는 사용자가 명시한 형식에 탐색 비교·작업 추정·예비 시간·실행 순서를 보완한 결과다.
