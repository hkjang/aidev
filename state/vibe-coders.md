
## 2026-09-02
- 선택: 모델 가격 prefix 매칭 비결정성 수정 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `audit.lookupPrice`가 가격표 map을 range하며 첫 prefix 일치를 반환해, 내장 카탈로그에 서로의 prefix인 항목(gpt-4o vs gpt-4o-mini, claude-sonnet-4 vs claude-sonnet-4-5)이 많은 탓에 버전 접미사가 붙은 실제 모델 ID(`gpt-4o-mini-2026-06-01`)가 호출마다 최대 16.7배 다른 단가로 계산되던 버그를 고쳤다. 가장 긴(구체적인) prefix를 채택하도록 바꾸고, 같은 규칙을 손으로 복제해 동일 버그를 갖고 있던 `admin_explain.lookupModelPrice`는 새로 export한 `audit.LookupPrice`에 위임시켰다. 회귀 테스트를 추가해 옛 구현에서 실패(16250 vs 975)함을 확인했고, gofmt·go vet·go build·`go test ./...`·`go test -race ./internal/audit ./internal/proxy`·`cmd/api-surface-audit` 모두 통과했다.
- 보류 아이디어: redact.go IPv4 규칙의 "사설망 제외" 주석과 실제 동작(전부 마스킹) 불일치 정리 / internal/config 패키지 테스트 부재 보강 / `EstimateTokens`의 `[]rune(text)` 전체 복사를 `utf8.RuneCountInString`으로 교체 / 가격표 키 정규화(소문자·trim)를 적재 시점에 일원화해 조회마다 재정규화 제거 / prefix 매칭에 경계 검사 추가로 `gpt-4`가 `gpt-45`에 잘못 매칭되는 것 방지
