# 회차 노트 2026-10-17-204834-multica-improve — multica
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 20:48] base pinned — main@18fa9f7
- [러너 20:48] autonomy release — 

## 구현 노트
- 바꾼 것: `server/internal/metrics/pricing.go` 의 pre-5.6 OpenAI 다섯 행(gpt-5.5/5.4/5.4-mini/5.3-codex/5.2-codex)이 `CacheWritePerM` 에 캐시 **읽기** 단가를 써서 캐시 생성 토큰을 1/10 로 집계했다. 입력 단가로 맞췄다. 표의 나머지 모든 행과 프런트엔드 요금표가 이미 그 규약(별도 쓰기 과금 없음 → 입력 단가)을 따르고 있어 이 다섯 행만 예외였다.
- 확신 없는 곳: 단가의 **방향**은 저장소 자체 규약(pricing.go 28~31행 주석 "earlier OpenAI SKUs ... don't bill cache writes separately" + 프런트 표 전수 일치)으로 판단했고, OpenAI 가격 페이지를 직접 확인하지는 못했다(네트워크 미사용). 또 프런트 주석은 "Codex 는 아직 cache-write 토큰을 스트리밍하지 않는다" 고 적고 있어, 실제로 이 단가가 오늘 얼마나 자주 발화하는지는 확인하지 못했다 — 다만 `PriceForModelAlias` 는 공급자를 보지 않으므로 `openai/gpt-5.4` 를 보고하는 openclaw/opencode/Hermes 경로에서도 같은 행을 쓴다.
- 일부러 하지 않은 것: deepseek v4-flash 의 4배 불일치(출처 확인 없이는 데이터 추측), 두 요금표 통합(운영자 지침: 여러 경로의 같은 종류 파서는 통합 금지), 백엔드 4.x 규칙의 종결자 추가(현재 단가가 동일해 실패를 보이는 테스트를 쓸 수 없음). 모두 ideas.json 에 pending 으로 남겼다.
- 다음 역할이 조심할 것: 검증은 `cd server && go test ./internal/metrics -count=1` 로 충분하며 DB 가 필요 없다. `make test` 는 PostgreSQL + 마이그레이션을 요구하고 이 환경에서 돌리지 않았다. 프런트엔드는 `node_modules` 가 없어 `pnpm typecheck/lint/test` 를 돌리지 않았다 — 이번 변경은 Go 파일만 건드려 프런트엔드 영향이 없다.
- [러너 20:57] verify failed — 실패한 검증: pnpm test --silent (exit 1)
