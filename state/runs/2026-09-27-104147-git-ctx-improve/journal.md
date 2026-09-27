# 회차 노트 2026-09-27-104147-git-ctx-improve — git-ctx
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 10:41] base pinned — main@01b97e6
- [러너 10:41] autonomy release — 

## 정찰 노트
- 고른 이유: 보류 목록의 pending 은 전부 "계약 미확인"(clampResponse·Sanitize)·"반려 이력 유형"(파서 확장)·"이름 미확보"(app flaky)여서 이번에도 과제로 못 세운다. 대신 이미 세 번 채택된 계열(바이트 절단 → 룬 경계)의 남은 두 곳을 찾았다: `embedding/runtime.go:182` `clipRuntimeError`, `vectorstore/milvus.go:282` `truncate`. 형제 3개(source.truncateError·mcp.runeSafeCut·search.cutAtRuneBoundary)는 이미 옳아서 갈라진 자리가 좁고 분명하다.
- 확신 있는 것: `clipRuntimeError` → `Runtime.Snapshot().LastError`(`json:"lastError,omitempty"`) → `app/health.go:325`·`339` 관리 콘솔 JSON 경로를 소스에서 따라 확인. `internal/embedding`·`internal/vectorstore`·`internal/source` 는 지금 전부 통과(4.031s / 0.017s / 0.004s, exit 0)라 수정 전 실패가 새 테스트에서만 난다.
- 추측으로 적은 것(과제서에도 "미확인" 표기): milvus 오류 문자열이 최종 JSON 으로 직렬화되는 지점은 안 봤다. 근거가 약하면 milvus 파일을 빼고 embedding 1개만 내도 수용 기준을 채운다고 적었다.
- 구현자가 조심할 것: `source.truncateError` 는 이미 옳으니 손대면 무변경 diff. limit 상수(300/400)와 `…` 접미사(milvus 는 접미사 없음)를 바꾸지 말 것. U+FFFD 검사는 원문자가 아니라 `�` 이스케이프로 할 것. 대역 타입 금지 — 실제 `Runtime.Guard` 와 실제 `Open`+`httptest`(기존 `TestMilvusHTTPErrorCarriesStatus` 가 그 패턴을 쓴다)로 갈 것.
- [러너 10:47] scout done — 진단용 오류 문자열의 바이트 절단이 UTF-8 글자를 반으로 자르지 않게 수정 — `embedding.clipRuntimeError` 와 `ve

## 구현 노트
- `embedding.clipRuntimeError`(runtime.go:182)와 `vectorstore.truncate`(milvus.go:282)의 `value[:limit]` 를 `source.truncateError` 와 같은 연속 바이트 후퇴로 고쳤다(commit 852dddf, 프로덕션 2파일·각 3줄 + 계약 주석). limit(300/400)·embedding 의 `…`·milvus 문구는 무변경.
- 정찰이 "미확인" 으로 둔 milvus 쪽 근거를 채웠다: `Status`(milvus.go:251-253) → `TestConnection`(vectorstore.go:178-180) → `searchBackendHealth`(health.go:478-480) → `adminHealth` 의 `jsonOut`(health.go:507). embedding 과 같은 관리 콘솔 JSON 이라 범위에서 빼지 않았다.
- **확신 없는 곳**: milvus 는 `httptest` 로만 검증했다 — 실제 Milvus 서버, pgvector, OpenSearch 통합은 이 환경에서 미검증이다. 실제 Milvus 가 4KiB 오류 본문을 어떤 인코딩으로 보내는지도 확인하지 못했다(테스트는 UTF-8 본문만 가정). embedding 쪽 `CircuitOpenError.Error()`(runtime.go:49-50)가 같은 `lastError` 를 붙여 호출자에게 돌려주는 경로는 소스로만 확인했고 테스트로 통과시키지 않았다.
- **밟은 함정(다음 역할이 주의)**: 여섯 글자짜리 이스케이프 표기(백슬래시-u-f-f-f-d)를 파일에 그대로 쓰려 하면 원문자 U+FFFD 로 납작해진다 — 셸 히어독만이 아니라 Write 도구로 써도 같다(이 저장소 밖 메모 파일에서 재확인). 그래서 테스트 소스에서 JSON 단언이 조용히 무의미해졌다(첫 실행에서 UTF-8 단언만 FAIL 하고 JSON 단언은 침묵). needle 을 소스에 철자하지 않고 `` `\u` + "fffd" `` 로 조립하게 고쳐 실패를 재확인했다. **같은 치환이 이 journal.md 와 ledger-entry.md 본문에서도 일어난다** — 이 파일들에 보이는 원문자 U+FFFD 는 전부 "백슬래시-u-f-f-f-d 이스케이프 표기" 로 읽을 것. 테스트 소스에 이 표기가 필요하면 철자하지 말고 반드시 조립할 것(도구를 바꿔도 소용없다). 확인은 `grep ... | cat -A` 로 소스에 `M-oM-?M-=` 가 아닌 이스케이프가 들어갔는지 보면 된다.
- **일부러 하지 않은 것**: 공용 헬퍼 패키지 신설(2026-09-23 선례대로 각 패키지 로컬 복제 + 주석 유지), `source.truncateError`·`mcp.runeSafeCut`·`search.cutAtRuneBoundary` 손대기(이미 옳음), `clipRuntimeError`/`truncate` 직접 호출 단위 테스트(배선 증명이 안 됨 — 실제 `Guard` 와 실제 `Open`+`httptest` 만 남겼다).
- 검증: `-count=1 ./internal/embedding ./internal/vectorstore`(ok 4.035s/0.020s), `-race`(ok 5.082s/1.075s), `gofmt -l ./cmd ./internal`(빈 출력), `go vet ./...`, `go build -tags sqlite_fts5 ./...`, `go test -tags sqlite_fts5 ./...`(전 패키지 ok, internal/app 101.792s, exit 0). 새 테스트는 외부 DB 없이 돈다.
- [러너 10:55] brief accepted — 채택 — 지정한 두 줄·limit·형제 관용구가 현재 코드와 정확히 맞았고, 과제서가 예측한 pad(limit-2, limit-1)만 수정 전 실패
- [러너 10:56] verify passed — 검증 3개 통과 (auto)

## 비평 노트
- 판정 approve(risk low, blocking 없음). 원장의 실패 재현을 믿지 않고 직접 재현했다 — 프로덕션 2파일만 `git checkout main --` 로 되돌려 embedding pad=298·299 와 vectorstore pad=398·399 가 FAIL, pad=297/300·397/400 이 PASS 함을 실측(출력 문자열까지 원장과 일치). 복구 후 embedding·vectorstore·source 전부 ok, gofmt -l 빈 출력, go vet 통과.
- 구현자가 의심한 자리부터 봤다: (1) 테스트의 JSON needle 이 원문자로 납작해졌는지 `grep fffd | cat -A` 로 확인 — 조립 형태가 소스에 제대로 들어가 있고 수정 전 실제로 FAIL 하므로 무의미한 단언 아님. (2) `CircuitOpenError.Error()`(runtime.go:44-52)는 이미 잘린 문자열을 이어붙일 뿐이라 별도 절단 없음 — 미커버지만 결함 아님. (3) 호출자 grep 결과 `truncate` 는 milvus.go:102, `clipRuntimeError` 는 runtime.go:176 단 하나씩이라 회귀 표면이 좁다.
- 못 본 것: 실제 Milvus·pgvector·OpenSearch 서버, 실브라우저, 외부 DSN 필요한 CI 통합. `go test -tags sqlite_fts5 ./...` 전체도 재실행하지 않았다(구현자 보고 신뢰 + 변경 표면이 두 패키지 로컬 함수라 판단).
- 남는 우려(차단 아님): milvus 오류 본문은 상류 서버가 보내는 임의 바이트라 원래 무효 UTF-8 일 수 있고, 그때는 후퇴해도 여전히 무효다. 0x80-0xBF 연속 구간이 길면 후퇴가 커져 진단문이 비는 이론적 경로가 있으나 현실 입력에서 3바이트를 넘기 어렵고 형제 3개와 같은 관용구다.
- 다음 회차·릴리즈: 사용자 가시 변화는 health JSON 의 `embedding.circuit.lastError` 와 `search.vectorDatabase` 끝 글자가 깨지지 않는 것뿐. limit·말줄임표·milvus 문구 무변경이라 계약 파급 없음. 이 브랜치에 internal/version 변경이 없으므로 릴리즈 단계에서 버전 동기화가 따로 필요하다.
- [러너 10:59] review approved — 리뷰 승인 (risk=low)
- [러너 10:59] pr created — https://github.com/hkjang/git-ctx/pull/40
- [러너 11:06] ci passed — 검사 5개 모두 success
- [러너 11:06] merge done — 852dddf
- [러너 11:22] release published — v0.77.19
- [러너 11:34] assets verified — v0.77.19 자산 2개 (이전 v0.77.18: 2)
