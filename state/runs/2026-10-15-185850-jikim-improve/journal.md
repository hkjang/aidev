# 회차 노트 2026-10-15-185850-jikim-improve — jikim
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 18:58] base pinned — main@84b8fd2
- [러너 18:58] autonomy release — 

## 정찰 노트
- Momento 프록시 경로 인코딩을 골랐다. 오류 오분류 분리 광맥(OIDC·MCP·AI)은 v0.2.29~v0.2.31 에서 소진됐고, 나머지 보류 후보(CSP 속도 제한·webhook 통합·baoKVWrite TOCTOU)는 모두 새 계약이나 DB 동시성 검증이 필요해 한 세션에 맞지 않는다. 이 과제는 실제 수집기 `r.RequestURI` 로 결함을 재현했고 프로덕션 파일 한 개로 끝난다.
- 확인한 것: 수정 전 `/momento/collect/a%2Fb` → 수집기 `/base/collect/a/b`, `/momento/collect/%2E%2E%2F%2E%2E%2Fadmin` → `/base/collect/../../admin`(`momento_url` 경로 접두사 이탈). 과제서에 적은 수정 형태를 실제로 적용해 두 경우가 escape 를 보존하고 기존 `TestMomentoProxyForwardsWithoutCredentials` 가 통과하는 것까지 확인한 뒤 트리를 원상복구했다(`git status` 비어 있음).
- 미확인: 디코드되지 않은 리터럴 `../` 요청이 ServeMux 를 통과하는지, 실제 Momento 수집기가 `%2F` 를 어떻게 해석하는지, 차선 후보의 `page[:512]` 줄. 과제서에 미확인으로 명시했다.
- 구현자가 조심할 것: 수집기 쪽 `r.URL.Path` 는 고친 뒤에도 디코드된 값으로 보이므로 반드시 `r.RequestURI` 로 단정할 것. `Path` 와 `RawPath` 를 짝으로 넣지 않으면 `EscapedPath()` 가 조용히 디코드된 경로로 되돌아간다. `httptest.NewRequest` target 에 공백을 넣으면 panic 한다.
- [러너 19:03] scout done — Momento 프록시가 인코딩된 경로(`%2F`·`%2E%2E%2F`)를 디코드해 수집기에 다른 경로로 전달하는 문제 수정 (가�

## 구현 노트
- 무엇/왜: `momentoProxy` 가 디코드된 `r.URL.Path` 로 바깥 경로를 조립하고 `RawPath` 를 비워, 브라우저가 escape 한 `%2F`·`%2E%2E%2F` 가 수집기에 다른 경로로(그리고 `momento_url` 접두사 밖으로) 도착했다. `Path`/`RawPath` 를 짝으로 넣고 조립을 `SetURL` 에 맡기도록 두 줄 교체(프로덕션 1파일 +6/-2). 테스트 1개(서브 4개) 추가.
- 과제서와 다른 점: 제시된 `url.PathUnescape` + `if err == nil` 분기를 쓰지 않았다. `TrimPrefix(r.URL.Path,…)` 는 `PathUnescape(TrimPrefix(EscapedPath(),…))` 와 항상 같은 문자열이고, `RawPath` 가 `Path` 의 유효한 인코딩이 아니면 Go 가 `Path` 를 다시 escape 해 기존 동작으로 degrade 하므로 도달 불가능한 오류 분기가 필요 없다. 비평가가 여기를 먼저 보면 좋다.
- 확신 없는 곳: 라우트 prefix 자체가 escape 된 요청(`/m%6Fmento/...`)이 ServeMux 를 통과하는지 **미확인** — 통과한다면 `RawPath` TrimPrefix 가 no-op 이 되어 수정 이전 동작으로 degrade 한다(ideas.json 에 후보로 남겼다). 리터럴 `../` 요청의 ServeMux 처리, 실제 Momento 수집기가 `%2F` 를 어떻게 해석하는지도 미확인.
- 일부러 하지 않은 것: 경로 정규화(`..` 제거·거절) — 새 계약이 되고 escape 보존만으로 접두사 이탈이 닫힌다. `internal/tracking` 스니펫 생성·gate·메서드 목록·헤더 제거·64KiB·10초도 손대지 않았다.
- 다음 역할 주의: 새 테스트는 DB 불필요(httptest 수집기 + `server.httpClient = http.DefaultClient`). 수집기 쪽 `r.URL.Path` 는 고친 뒤에도 `/base/collect/a/b` 로 보이므로 단정은 반드시 `r.RequestURI` 로. 검증: `bash scripts/verify.sh` exit 0(프런트 59/59), `go test ./... -count=1`·`-race -count=5`·`go vet`·`gofmt` 모두 통과. 빌드의 500kB 청크 경고는 기존 것. 이미지·브라우저 E2E 미실행.
- [러너 19:08] brief accepted — 채택 — 근거·배선·재현 방법(`r.RequestURI` 로 단정, `httptest.NewRequest` target 공백 금지, `url` 이미 import)이 모두 코드와 맞았�
- [러너 19:08] verify passed — 검증 7개 통과 (auto)

## 비평 노트
- 확인한 것: 수정 전/후 에뮬레이션과 실제 `trackingServer`→`routes`→`momentoProxy` 양쪽에서 수집기 `r.RequestURI` 를 실측했다. 주 경로(`/momento/collect/a%2Fb`·`%2E%2E%2F`)는 정말 고쳐졌고 원장의 `- 실패 재현:` 출력과 증상이 일치한다. 범위·되돌리기·경계 조건(target.Path 빈 값, `/momento/` 루트, 비ASCII, 리터럴 `../`의 307, `/momento/..%2F..%2Fadmin`)도 모두 정상.
- 거절 사유: 구현자가 "미확인"으로 남긴 바로 그 자리가 결함이었다. Go 1.22 ServeMux 가 패턴 리터럴을 디코드해 비교하므로 `/m%6Fmento/...` 가 라우트에 매칭되고, 그러면 tracking.go:235 의 `TrimPrefix(EscapedPath(), …)` 가 no-op 이 되어 `EscapedPath()` 가 Path 를 다시 escape → 수정 이전 동작으로 degrade 한다. 실측: `/m%6Fmento/collect/%2E%2E%2F%2E%2E%2Fadmin` → 수집기 `/base/collect/../../admin`(접두사 이탈 그대로), `/m%6Fmento/collect/a%2Fb` → `/base/collect/a/b`.
- 수리가 가장 먼저 볼 파일: `internal/httpapi/tracking.go:234-235`(접두사 없는 `EscapedPath()` 는 404 로 거절하거나 접두사 세그먼트를 escape 보존 방식으로 제거) 와 `internal/httpapi/tracking_test.go:462`(서브테스트 이름이 보증을 과장 — 접두사 인코딩 변형 케이스를 추가해 함께 고정).
- 남는 우려: 이 구멍은 main 에 이미 있던 것이고 이번 변경이 좁혔으므로 `blocking` 에는 올리지 않았다(부서 veto 는 신규·확대 노출에 아껴 쓴다). 다만 성격은 SSRF 성 경로 봉쇄 우회이고 폐쇄망 내부 수집기에서는 실익이 있다 — 닫기 전에는 릴리즈 노트에 "경로 접두사 봉쇄 완결"로 쓰지 말 것.
- 못 본 것: 실제 Momento 수집기가 `%2F` 를 어떻게 해석하는지, 브라우저 실환경·이미지 E2E·CI. 개인정보 신규 수집·전송 항목 변화는 없어 법무 차단 사유 없음.
- [러너 19:12] review rejected — 리뷰 거절: internal/httpapi/tracking.go:235 — `strings.TrimPrefix(r.URL.EscapedPath(), tracking.MomentoProxyPrefix)` 는 라우트 접두사 자체가 퍼센트 인코딩된 요청에서 아무�
