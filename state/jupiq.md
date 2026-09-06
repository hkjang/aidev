# jupiq 자율 개선 기록

## 2026-09-03
- 선택: OpenAPI 문서와 등록 경로 계약 검증 테스트 추가 (가치 4 / 위험 1 / 작업량 M)
- 결과: 성공
- 요약: `register*` 함수 시그니처를 `*http.ServeMux`에서 최소 `router` 인터페이스로 바꿔 테스트가 등록 경로를 수집할 수 있게 하고, `internal/api/openapi_contract_test.go`에서 openapi.yaml의 paths와 양방향(경로→문서, 문서→경로)으로 대조하도록 했다. probe·별칭·항상 405인 승인 쓰기 경로는 이유를 적은 예외 목록으로 관리하며, 이 검증으로 드러난 누락 `GET /auth/oidc/callback`을 문서에 추가했다. 임시로 가짜 경로를 등록해 테스트가 실제로 드리프트를 잡는지 확인했고, `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(16파일 49개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(boundedTimeRange로 대체됨) / `internal/collector` 커버리지 8.5% 보강 / `serveSPA`의 해시 자산에 장기 Cache-Control 부여 / 로그인 리미터 `succeeded`가 ip 키를 정리하지 않는 동작에 대한 테스트·문서화
- 릴리즈: v1.3.0 (2026-09-03)

## 2026-09-04
- 선택: SPA 정적 자산 캐시 정책과 serveSPA 테스트 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `serveSPA`가 정적 파일에 Cache-Control을 전혀 붙이지 않아 브라우저 heuristic 캐시에 맡겨져 있었다. Vite가 content hash를 붙여 내보내는 `/assets/*`는 `public, max-age=31536000, immutable`로, public/에서 이름 그대로 복사되는 favicon 같은 파일은 `public, max-age=0, must-revalidate`로 응답하게 하고(`index.html`은 기존 `no-store` 유지) 근거를 주석과 README에 남겼다. 지금까지 테스트가 없던 `serveSPA`에 대해 캐시 헤더·`/api`·`/mcp` JSON 404·dist 밖 경로 차단·빌드 산출물 부재 404를 덮는 `internal/api/spa_test.go`를 `t.Chdir` 기반으로 추가했고, `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(16파일 49개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(boundedTimeRange로 대체됨) / `internal/collector` 커버리지 8.5% 보강 / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화 / `Collector.prune`이 실패해도 `lastPrune`을 갱신해 24시간 재시도하지 않는 문제 수정
- 릴리즈: v1.4.0 (2026-09-04)

## 2026-09-05
- 선택: 보존 정책 prune 실패 재시도와 설정 읽기 실패 시 삭제 보류 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `Collector.prune`이 시도 시각(`lastPrune`)을 먼저 기록해 `PruneMetrics`가 실패해도 24시간 동안 재시도하지 않던 문제를 고쳐, 실패한 주기는 `pruneRetryInterval`(30분) 뒤 재시도하고 성공하면 하루 주기로 복귀하도록 `pruneDue`/`pruneFailed`로 분리했다. 함께 무시되던 retention 설정 읽기 오류도 처리해, `ErrNotFound`가 아닌 오류로 설정을 알 수 없을 때는 `PruneMetrics`의 기본값 30일을 적용해 운영자가 더 길게 보관하도록 설정한 샘플을 지우는 대신 삭제를 건너뛰고 재시도하게 했다. 새 순수 함수 기반 테스트 2개(`TestPruneRetriesSoonAfterFailure`, `TestRetentionReadableOnlyToleratesMissingSettings`)를 추가해 collector 커버리지가 8.5%→13.9%로 올랐고, `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 58개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(boundedTimeRange로 대체됨) / `internal/collector`의 `collectPrometheus`·`collectKubernetes` 경로 커버리지 추가 보강 / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화 / `collectHubs`가 goroutine을 제한 없이 띄우는 부분에 동시성 상한 도입
- 릴리즈: v1.4.4 (2026-09-05)
## 2026-09-06
- 선택: 응답 보안 헤더 보강(CSP 지시자 추가 + TLS 한정 HSTS)과 middleware 테스트 신설 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `internal/api/middleware.go`의 CSP가 `default-src`로는 대체되지 않는 `frame-ancestors`·`base-uri`·`form-action`·`object-src`를 지정하지 않아 clickjacking과 주입된 `<base>`·`<form>`을 통한 외부 전송이 열려 있었고, HSTS는 전혀 없었다. 네 지시자를 추가하고 평문 폐쇄망 배포에서 접속이 영구히 막히지 않도록 `auth.IsSecureRequest`가 참인 요청에만 `max-age=31536000`(includeSubDomains·preload 없음)을 붙이도록 `setSecurityHeaders`로 분리했다. 그동안 테스트가 하나도 없던 middleware에 보안 헤더·HSTS 조건·동일 출처 변경 요청 거부(Origin 호스트/스킴, Sec-Fetch-Site, GET 예외)·요청 ID 생성과 에코·panic 복구를 덮는 `middleware_test.go`(5개 테스트)를 추가했고, `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 58개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(boundedTimeRange로 대체됨) / `internal/secure`의 `EncryptString`·`DecryptString`·`Derive`·`RandomToken` 테스트 공백 보강(현재 0%) / `collectHubs`가 goroutine을 제한 없이 띄우고 종료 시 기다리지 않는 부분에 동시성 상한과 대기 도입 / `collectPrometheus`가 metric마다 features 설정을 다시 읽는 중복 조회 제거

## 2026-09-07
- 선택: 허브 수집 goroutine 동시 실행 상한과 종료 대기 도입 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: `collectHubs`가 수집 대상 허브 수만큼 goroutine을 제한 없이 띄우고 아무도 기다리지 않아, 허브가 많으면 30초 주기마다 그만큼의 아웃바운드 HTTP·DB 연결이 동시에 열리고 종료 시에는 `main`의 `defer database.Close()`가 진행 중인 상태 쓰기 아래에서 풀을 닫아 버렸다. `updateHubHealth`가 `context.WithoutCancel`로 쓰기 컨텍스트를 분리해 둔 의도가 무산되던 지점이다. `goHub`(용량 8 세마포어, 취소된 컨텍스트면 대기 중인 프로브를 버림)와 `waitForHubs`(10초 상한 대기)를 추가하고 `Run`이 반환할 때만 대기하도록 해 kubernetes·prometheus 수집이 주기마다 지연되지 않게 했으며, `main`은 HTTP 종료 후 collector 종료를 기다린 뒤 DB를 닫는다. 동시 실행 상한·종료 대기·취소 시 대기열 폐기를 덮는 테스트 3개를 추가해 collector 커버리지가 13.9%→18.7%로 올랐고, 세마포어를 제거한 변형에서 테스트가 실제로 실패하는지 확인했다. `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 58개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(boundedTimeRange로 대체됨) / `internal/secure`의 `EncryptString`·`DecryptString`·`Derive`·`RandomToken` 테스트 공백 보강(현재 39.5%) / `collectPrometheus`가 metric마다 features 설정을 다시 읽는 중복 조회 제거 / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화 / `collectHubs`의 `due`가 프로브 성공 여부와 무관하게 시각을 선기록해 실패한 허브가 전체 간격만큼 재시도되지 않는 문제 검토

