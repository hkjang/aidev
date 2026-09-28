# 회차 노트 2026-09-29-035528-Quantoss-improve — Quantoss
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 03:55] base pinned — main@7e52f31
- [러너 03:55] autonomy release — 

## 정찰 노트
- 고른 이유: 실제 코드의 비대칭(5xx 분기엔 `attempt < maxAttempts-1` 가드가 있고 429 분기엔 없음)이라 근거가 grep 이 아닌 구조로 남고, 프로덕션 파일 1개·테스트 1개로 끝난다. check.sh·CI 는 4회 연속 차선으로만 반복돼 제외(CI 는 workflow 권한 미확인으로 rejected), 스윙 청산/보유 한도는 매매 계약 확정이 먼저라 제외.
- 추측으로 적은 것: 토스가 429 에 `Retry-After` 를 실제로 보내는지 **미확인**(문서·실응답 확인 안 함). 다만 과제서의 수정은 `ra<=0` 이면 현행 지수 백오프를 그대로 타므로 안 보내면 무해하다.
- 429 소진이 호출부를 실제로 오작동시키는지도 미확인 — `broker/live.go:616` 은 Status=0 이어도 "재시도" 쪽으로 떨어져 의도가 우연히 보존된다. 그래서 왜 를 "오작동" 이 아니라 "오류 정체 상실 + 최대 900초 블로킹" 으로 적었다.
- 구현자 주의: `backoff` 시그니처를 바꾸지 말 것(5xx·network 호출부와 기존 테스트 2개가 대기식을 단언). `ensureToken`·토큰 캐시는 같은 파일이어도 절대 건드리지 말 것 — 재발급이 실거래 토큰을 죽인다.
- [러너 03:59] scout done — `toss.Client.request` 의 429 분기 — 마지막 시도에서 실제 429 오류를 `retry-exhausted`(Status 0) 로 덮어쓰고, 서버가

## 구현 노트
- `internal/toss/client.go:250` 429 분기에 5xx 와 같은 `attempt < maxAttempts-1` 가드를 붙여 마지막 429 가 기존 4xx 분기로 떨어지게 했고(서버 envelope 그대로 → `*APIError{Status:429}`, `retry-exhausted` 덮어쓰기 제거), `Retry-After` 가 숫자로 파싱되면 `c.backoff(0, ra, …)` 로 불러 지수 증폭(최대 ~900초 블로킹)을 없앴다. 프로덕션 1파일 + 테스트 1파일, 커밋 9def9bc8.
- 확신 없는 곳: 토스가 429 에 `Retry-After` 를 실제로 보내는지는 여전히 **미확인**(실응답·문서 확인 안 함) — 안 보내면 `ra<=0` 경로로 현행과 동일하다. 또 429 소진이 실제로 호출부를 오작동시킨 사례는 확인하지 못했다: `broker/live.go:616` 은 전에도 Status=0 이라 재시도 쪽으로 떨어졌고 지금도 `!=429` 가 거짓이라 재시도한다(결과 동일, 의도만 명시됨). 즉 이 변경의 실질 이득은 (a) 로그·doctor 에서 429 식별 (b) 대기 시간 상한이다.
- 일부러 하지 않은 것: `backoff` 시그니처·대기식·`재시도 대기` 로그 키 불변(5xx·network 호출부와 기존 테스트 2개가 의존). `ensureToken`/401 분기/토큰 캐시는 열어보기만 하고 미변경. `Retry-After` HTTP-date 파싱은 범위 밖이라 추가하지 않음(파싱 실패 → 지수 백오프 유지가 의도된 동작이고 테스트로 고정했다).
- 다음 역할이 조심할 것: 신규 테스트는 `httptest` + 실제 `NewClient` 공개 배선(`BaseURL`/`HTTP`/`Sleep`)만 쓴다 — 대역 트랜스포트 없음. 각 테스트가 429 를 5회 받으므로 `limiter`(그룹 미등록 → 3 TPS)의 실제 `time.Sleep` 으로 케이스당 ~0.7초가 든다(`Sleep` 주입으로는 안 사라짐, `internal/toss` 전체 2.8s). 실API·`internal/zz_dbg` 는 실행하지 않았다.
- [러너 04:02] brief accepted — 채택 — 과제서가 지목한 비대칭(429 에 가드 없음·Retry-After 증폭)이 코드와 정확히 일치했고 수용 기준 1~5 를 모두 실측�
- [러너 04:02] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 확인함: 신규 테스트 3개를 main 의 client.go 로 되돌려 실제 실행 → 전부 실패(`want Status 429, got 0`, `want Sleep 4 times, got 5: [30.1s 1m0.2s 2m0.3s 4m0.1s 8m0.2s]`), 원장 `- 실패 재현:` 줄과 일치. 호출부 셋(live.go:616·677, doctor.go:83) 직접 읽어 동작 변화 없음 확인. `retry-exhausted` 문자열 분기 호출부 없음. gofmt/vet/build/test 실측 통과(toss 3.055s, broker 7.776s). 워킹트리는 clean 복원.
- 못 본 것: 토스 실응답의 Retry-After 유무(여전히 미확인 — 없으면 `ra<=0` 경로로 무해), check_dashboard.sh, 실API·zz_dbg.
- 승인이어도 남는 우려 ①: `client.go:253` 의 `ra > 0` 에 상한이 없어 Retry-After: 3600 이면 4×3600초 블로킹. 모든 경우 기존(×31)보다는 짧아 회귀가 아니지만 다음 회차에서 `min(ra, 상한)` 검토 권함.
- 승인이어도 남는 우려 ②: 마지막 429 가 4xx envelope 분기를 타면서 `code` 없는 본문은 `Message = string(data)` 로 원문이 로깅된다(live.go:660 등). 기존 4xx 와 같은 경로·공격 경로 없음이라 차단 아님.
- 범위: `git diff main...HEAD` 의 live.json churn 은 로컬 main ref(d558bc0b)가 뒤처진 탓이며 구현 커밋 9def9bc8 은 코드 2파일뿐 — 범위 이탈 아님. 보안·법무 차단 사유 없음(인증·토큰 캐시 미변경, 개인정보·비밀값·의존성 변화 없음).
- [러너 04:05] review approved — 리뷰 승인 (risk=low)
- [러너 04:05] pr created — https://github.com/hkjang/Quantoss/pull/103
- [러너 04:05] base rebased — 879b1dd, 재검증 통과
- [러너 04:06] ci passed — 검사 없음 — 정책으로 허용
- [러너 04:06] merge done — b3349fd
- [러너 04:07] release skipped — 릴리즈 안 함: 릴리즈 관례가 전혀 없는 저장소. 실측: `git tag -l` 0개(refs/tags 비어 있음), 버전 파일 없음(package.json·pyproject.toml·VERSION·Cargo.toml·Chart.yaml �
