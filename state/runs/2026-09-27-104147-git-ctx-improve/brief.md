# 과제서 — 2026-09-27 (base: main@01b97e6 = release v0.77.18)

- 과제: 진단용 오류 문자열의 바이트 절단이 UTF-8 글자를 반으로 자르지 않게 수정 — `embedding.clipRuntimeError` 와 `vectorstore.truncate` (가치 3 / 위험 1 / 작업량 S)

- 왜: `internal/embedding/runtime.go:182-187` 의 `clipRuntimeError` 와 `internal/vectorstore/milvus.go:282-287` 의 `truncate` 는 `len(value) > limit` 일 때 `value[:limit]` 로 **바이트 경계**에서 자른다. 오류 본문에 한국어·이모지가 섞이면(이 저장소는 사용자 노출 진단문을 한국어로 쓴다 — 예: `internal/source/transport.go:179` `"연동이 일시 중단되었습니다…"`) 마지막 글자가 반으로 잘려 invalid UTF-8 이 되고, `clipRuntimeError` 의 결과는 `Runtime.Snapshot().LastError`(`json:"lastError,omitempty"`, runtime.go:286)를 거쳐 `internal/app/health.go:325`·`339` 의 `Circuit embedding.RuntimeSnapshot` 로 관리 콘솔 JSON 에 그대로 실린다 — `encoding/json` 은 깨진 바이트를 `�` 로 써서 운영자가 보는 오류 원문 끝에 대체 문자가 남는다(`CircuitOpenError.Error()`, runtime.go:49-50 도 같은 문자열을 붙여 호출자에게 돌려준다).
  같은 저장소의 형제 절단기 3개는 이미 경계로 물러나 자른다: `internal/source/transport.go:251-261` `truncateError`(연속 바이트 `&0xC0 == 0x80` 를 물러남), `internal/mcp/budget.go` `runeSafeCut`, `internal/search/service.go` `cutAtRuneBoundary`(2026-09-22·09-23 회차에서 각각 고친 것). 이 두 곳만 계약이 갈라져 남아 있다.

- 수용 기준:
  1) `clipRuntimeError` 가 limit 을 넘는 입력을 자를 때 결과가 항상 valid UTF-8 이고, 잘려 나간 글자는 통째로 빠진다(반쪽 바이트가 남지 않는다). 기존 `…` 접미사와 `len(value) <= limit` 일 때의 무변경 동작은 그대로.
  2) `vectorstore` 의 `truncate` 도 같다. 접미사를 새로 붙이지 말 것 — 현재 `truncate` 는 `…` 을 붙이지 않으며 그 출력이 `fmt.Errorf("milvus %s: %s", …)`(milvus.go:101-102) 오류문에 들어가므로 문구를 바꾸면 계약이 바뀐다.
  3) 테스트가 증명할 것 — **수정 전에 실패하는 것을 직접 확인한 뒤** 수정할 것:
     - `internal/embedding`: 실제 `NewRuntime()` + 실제 `Runtime.Guard(identity, RuntimePolicy{}, factory)` 로 factory 가 긴 비ASCII 오류를 반환하게 해(→ `Guard` 가 `r.finish` 를 호출해 `state.lastError` 를 기록한다, runtime.go:176) `rt.Snapshot(identity).LastError` 가 valid UTF-8 임을 단언. 추가로 `json.Marshal(snapshot)` 결과에 `�`(이스케이프 문자열 `�` 로 검사할 것 — **원문자 `<?>` 검사는 항상 통과하는 무의미한 단언이다**, 2026-09-23 교훈) 가 없음을 단언.
     - `internal/vectorstore`: 기존 `TestMilvusHTTPErrorCarriesStatus`(vectorstore_test.go:47-60) 와 같은 방식으로 `httptest.NewServer` 가 503 + 400바이트를 넘는 한국어 본문을 돌려주게 하고 `Open(FromMap{"provider":"milvus","baseUrl":server.URL,"dimensions":float64(4)}, "")` → `vectorStore.Status(ctx)` 의 오류 문자열이 valid UTF-8 임을 단언. (대역 타입 금지 — 실제 `Open`·실제 HTTP.)
     - 두 테스트 모두 **경계 패딩 표를 쓸 것**: 3바이트 글자(예 `한`)가 limit 을 걸치는 pad 와 걸치지 않는 pad 를 함께 넣어, 걸치는 케이스만 수정 전 실패하고 걸치지 않는 케이스는 수정 전에도 통과함(= 동작 무변경 대조군)을 보일 것. limit=300(embedding)·400(vectorstore) 기준 예: `strings.Repeat("a", pad) + strings.Repeat("한", 10)` 에서 pad=limit-2, limit-1 은 수정 전 invalid, pad=limit, limit-3 은 수정 전에도 valid.

- 건드릴 파일 (프로덕션 2개, 테스트 2개):
  - `internal/embedding/runtime.go:182` `clipRuntimeError` — `value[:limit]` 앞에 `cut := limit; for cut > 0 && value[cut]&0xC0 == 0x80 { cut-- }` 를 넣고 `value[:cut] + "…"` 를 반환. `internal/source/transport.go:251-261` `truncateError` 와 같은 관용구를 쓰고, 주석에 "형제 계약(`source.truncateError`, `mcp.runeSafeCut`, `search.cutAtRuneBoundary`)과 같다 / lastError 가 health JSON 에 실린다" 를 한 줄 남길 것.
  - `internal/vectorstore/milvus.go:282` `truncate` — 같은 방식. 접미사 없음 유지.
  - `internal/embedding/runtime_test.go` — 새 테스트 1개 추가(기존 3개 테스트 수정 금지).
  - `internal/vectorstore/vectorstore_test.go` — 새 테스트 1개 추가(기존 `TestMilvusHTTPErrorCarriesStatus` 수정 금지).
  - 공용 헬퍼 패키지를 새로 만들지 말 것. 이 저장소는 2026-09-23 회차에서 임포트 방향 때문에 의도적으로 같은 계약의 함수를 복제하고 사유를 주석에 남기는 방식을 택했다(`search.cutAtRuneBoundary` 주석 참조). 그 선례를 따를 것.

- 검증 명령 (모두 이 저장소에서 실제로 도는 것 — 이번 정찰에서 실측: `go test -tags sqlite_fts5 -count=1 ./internal/embedding ./internal/vectorstore ./internal/source` → `ok 4.031s / ok 0.017s / ok 0.004s`, exit 0):
  1) `go test -tags sqlite_fts5 -count=1 ./internal/embedding ./internal/vectorstore`
  2) `go test -tags sqlite_fts5 -race -count=1 ./internal/embedding ./internal/vectorstore`
  3) `gofmt -l ./cmd ./internal` (출력이 비어야 성공)
  4) `go vet ./...`
  5) `go build -tags sqlite_fts5 ./...`
  6) `go test -tags sqlite_fts5 ./...` (수 분. `./internal/app` 혼자 ~100초)

- 위험과 피할 것:
  - `limit` 상수(300, 400)와 `…` 접미사·오류 문구를 바꾸지 말 것. 수정은 절단 지점 계산만이다.
  - `internal/source/transport.go` `truncateError` 는 **이미 옳다** — 고치지 말 것. 근거 없이 "통일" 하려고 손대면 무변경 diff 가 된다.
  - `internal/mcp/budget.go`·`internal/search/service.go` 의 절단은 이미 고쳐진 회차 대상이다. 건드리지 말 것.
  - 보호 경로 회피: `internal/auth`, `internal/store` migration, `.github/workflows`, `scripts/release.sh`, `internal/version` 은 이번 과제와 무관하니 열지 말 것.
  - 대역(fake) 타입으로 증명하지 말 것 — `embedding` 은 실제 `Runtime.Guard`, `vectorstore` 는 실제 `Open` + `httptest` 로 갈 것. `clipRuntimeError`/`truncate` 를 직접 호출하는 단위 테스트만 남기면 배선 증명이 안 된다(직접 호출 표는 **보조**로만 허용).
  - `internal/vectorstore` 는 외부 Milvus 서버 없이 `httptest` 로만 검증된다 — 실제 Milvus·pgvector·OpenSearch 통합은 이 환경에서 미검증이므로 그렇게 보고할 것.
  - 미확인: milvus 오류 문자열이 최종적으로 JSON 으로 직렬화되는 정확한 지점은 확인하지 않았다. 수용 기준 2 는 **오류 문자열 자체가 사용자 노출 진단문**이라는 근거로만 서 있다(embedding 쪽은 `health.go:325`·`339` 까지 소스에서 확인했다). milvus 쪽 근거가 약해 보이면 그 파일만 범위에서 빼고 embedding 1개만 내도 수용 기준 1·3 으로 충분하다 — 억지로 붙이지 말 것.

- 차선 후보: `cutAtRuneBoundary`(search) / `runeSafeCut`(mcp) / `truncateError`(source) 세 형제 함수의 계약을 같은 테이블 테스트로 고정 (가치 2 / 위험 1 / S). 순수 테스트 추가라 위험 0 이지만 가치도 낮다. 1순위가 성립하지 않을 때만.
