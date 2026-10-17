## 2026-10-17
- 선택: 백엔드 요금표의 pre-5.6 OpenAI 캐시 쓰기 단가를 입력 단가로 교정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `server/internal/metrics/pricing.go` 의 gpt-5.5 / 5.4 / 5.4-mini / 5.3-codex / 5.2-codex 다섯 행만 `CacheWritePerM` 에 **캐시 읽기** 단가(90% 할인가)를 쓰고 있어, 캐시 생성 토큰이 실제 가격의 1/10 로 `llm_cost_usd` 에 기록됐다. 표의 다른 모든 행은 입력 단가를 그대로 쓰거나(별도 쓰기 과금 없는 공급자) 입력의 1.25배(5.6+, Anthropic, Alibaba, MiniMax)를 쓰고, 프런트엔드 요금표(`packages/views/runtimes/utils.ts`)도 같은 SKU 를 입력 단가로 매겨 두 경로가 10배 어긋나 있었다. 다섯 행을 입력 단가로 맞추고, 프로덕션 배선(`RecordLLMUsage` → `llm_cost_usd`)을 통과하는 테스트와 표 수준 불변식 테스트를 추가했다. 검증: `go test ./internal/metrics -count=1` 및 `-race` 통과, `go vet ./internal/metrics/...`, `go build ./...`, `gofmt -l` 모두 클린.
- 실패 재현: `business_test.go:142: cache_write cost = 0.25, want 2.5` / `pricing_test.go:160: PriceForModelAlias("gpt-5.5") CacheWritePerM = 0.5, want InputPerM 5` (다섯 행 모두 동일하게 실패)
- 보류 아이디어: ① deepseek v4-flash 단가가 백엔드 0.56/1.12 vs 프런트 0.14/0.28 로 4배 어긋남 — 어느 쪽이 공식가인지 출처 확인이 필요해 이번엔 손대지 않음 ② 백엔드 `(^|/|:)k3$` 규칙의 `^` 분기가 공급자 없는 일반 id `k3` 까지 Kimi 단가로 매김 (프런트는 `kimi/k3` 로만 매김) ③ 백엔드 Opus/Sonnet 4.x 규칙에 `claudeVersionEnd` 종결자가 없어 미래 SKU 가 조용히 등급을 물려받을 수 있음 ④ 프런트 `cursorBaseModel` 의 `-max` 변이 접미사가 `qwen3.8-max` 같은 실제 SKU 이름 끝을 깎아내 base 후보가 raw 보다 먼저 조회됨
- 과제서: 기각 — 지정된 '릴리즈 워크플로 2회 실패' 는 저장소 결함이 아니라 러너가 PR 직전에 중단된 것(`resume closed`)이고, 베이스가 동일 SHA 로 깨끗해 고칠 대상이 없었다.

