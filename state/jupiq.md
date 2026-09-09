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

## 2026-09-08
- 선택: internal/secure의 문자열 암복호화·파생키·토큰 생성 테스트 공백 보강 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `EncryptString`·`DecryptString`·`Derive`·`RandomToken`·`HashToken`은 OIDC state 쿠키, 허브 자격증명, JWT 서명키, API 키 해시가 모두 의존하는데 테스트가 라운드트립 하나뿐이었다. 잘못된 base64·표준 패딩·nonce보다 짧은 blob·변조·절단 ciphertext를 `DecryptString`이 모두 거부하고 오류 시 빈 문자열을 돌려주는지, `Derive`가 label과 키로 분리되며 결정적인지, `RandomToken`이 요청한 바이트 수를 그대로 디코딩하고 재사용되지 않는지 덮는 테스트 6개를 추가해 커버리지가 39.5%→87.5%로 올랐다. 겸사겸사 `RandomToken`이 크기 0 이하에 빈 문자열을 조용히 돌려주던 계약을 오류로 바꿔(현재 호출자는 모두 상수라 동작 변화 없음) 나중에 크기를 계산해 넘기는 호출자가 빈 state·nonce를 비밀값으로 쓰지 못하게 했다. 1바이트 토큰 유일성 검사는 256개 값에서 16회 추출 시 ~37% 확률로 충돌해 플레이키하므로 12바이트 이상에만 적용했고, `-count=20`으로 반복 확인했다. `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 59개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(호출자 없음을 이번에도 재확인) / `collectPrometheus`가 metric마다 `featureEnabled`로 features 설정을 다시 읽는 중복 조회 제거 / `internal/config`의 `Load()` 테스트 신설(현재 `ParseEncryptionKey`만 덮여 있고 필수 환경변수 누락 집계·비밀번호 최소 길이·오류 우선순위는 미검증) / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화

- 릴리즈: v1.4.8 (2026-09-08, run 2026-09-08-202139-jupiq-improve)
## 2026-09-09
- 선택: 큰 페이지 번호가 OFFSET을 음수로 뒤집지 않도록 pageBounds 제한 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `queryInt`가 쿼리스트링의 `page`를 상한 없이 그대로 넘겨 `store.pageBounds`의 `(page-1)*pageSize`가 int를 넘겨 음수 OFFSET으로 감쌌고, PostgreSQL이 이를 거부해 인증된 사용자가 `?page=99999999999999999` 하나로 목록·감사로그·사용자 상세 API를 500으로 만들 수 있었다. `page`를 offset이 표현 가능한 마지막 페이지로 잘라(실제 테이블 끝을 지난 값이라 빈 페이지가 정직한 답) 모든 `pageBounds` 호출자와 자체적으로 offset을 다시 계산하는 `user_detail.go` 경로까지 한 번에 막았고, 지금까지 테스트가 없던 이 순수 함수에 클램프·오버플로 방지·경계 페이지 보존을 덮는 테스트 3개를 추가했다. 클램프를 빼면 두 테스트가 실제로 실패하는 것을 확인했으며 `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 59개) 모두 통과했다.
- 보류 아이디어: `internal/api/helpers.go`의 사용되지 않는 `parseTimeQuery` 제거(이번에도 호출자 없음 재확인) / `collectPrometheus`가 metric마다 `featureEnabled`로 features 설정을 다시 읽는 중복 조회 제거 / `internal/config`의 `Load()` 테스트 신설(현재 커버리지 23.1%, 필수 환경변수 누락 집계·비밀번호 최소 길이·오류 우선순위 미검증) / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화

- 릴리즈: v1.4.9 (2026-09-09, run 2026-09-09-011109-jupiq-improve)
## 2026-09-09
- 선택: 기동 설정 오류 일괄 보고와 config.Load() 테스트 신설 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `config.Load()`는 누락 환경변수만 모아 보고하고 `ENCRYPTION_KEY` 파싱 실패·`BOOTSTRAP_ADMIN_PASSWORD` 최소 길이는 하나씩 순차 반환해, 운영자가 첫 기동에서 문제마다 컨테이너를 다시 띄워야 알 수 있었다. 세 종류를 하나의 오류로 합쳐 한 번에 고칠 수 있게 하고(값이 비어 있으면 이미 누락 목록에 있으므로 중복 보고하지 않음), 테스트가 `ParseEncryptionKey` 하나뿐이던 이 패키지에 트리밍 규칙(비밀번호는 의도적으로 미트리밍)·누락 4개 동시 보고·12자 경계·거부 시 빈 Config 반환·미사용 값 동시 보고를 덮는 테스트 5개를 추가해 커버리지가 23.1%→100%로 올랐다. 집계를 되돌린 변형에서 새 테스트가 실제로 실패하는 것을 확인했고, README의 환경변수 표에 최소 12자 조건을 명시했다. 겸사겸사 다섯 회차 동안 보류돼 있던 호출자 없는 `parseTimeQuery`를 별도 커밋으로 제거했다(`time` 임포트도 함께 정리). `go vet ./...`, `go test -race ./...`, `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 59개) 모두 통과했다.
- 보류 아이디어: `collectPrometheus`가 metric마다 `featureEnabled`로 features 설정을 다시 읽는 중복 조회 제거 / `/api/v1/metrics`의 GPU 게이트가 `store.IsGPUMetricName`과 달리 `cuda`·`nvidia`를 빼먹어 응답 모양(`feature_enabled`)이 달라지는 불일치 정리 / 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화 / `internal/store` 커버리지 16.7% 중 DB 없이 테스트 가능한 순수 함수 경로 보강

- 릴리즈: v1.4.10 (2026-09-09, run 2026-09-09-082108-jupiq-improve)
## 2026-09-09
- 선택: GPU 기능 게이트를 store 판정으로 일원화하고 수집 주기의 중복 설정 조회 제거 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `/api/v1/metrics` 핸들러가 `gpu`·`vram`·`dcgm` 세 substring만 자체 목록으로 검사해 `store.IsGPUMetricName`이 함께 보는 `cuda`·`nvidia` 별칭이 게이트를 지나쳤고, 데이터는 store 필터가 막았지만 응답에서 `feature_enabled`가 빠져 소비자가 '표본 없음'과 '기능 꺼짐'을 구분할 수 없었다. `Store.Metrics`가 게이트 적용 여부를 함께 돌려주게 해 판정을 한 곳에 모으고 핸들러의 중복 설정 조회도 없앴으며, 판정을 순수 함수 `GPUFeatureBlocked`로 분리해 단위 테스트로 덮고 실제 PostgreSQL에서 다섯 별칭이 모두 `feature_enabled=false`를 받는지 확인하는 통합 테스트를 추가했다(수정 전 코드에서 `cuda_cores`·`nvidia_smi_temperature`로 실제 실패하는 것을 확인). 이어서 `collectPrometheus`가 지표마다 `featureEnabled`로 설정을 다시 읽던 부분을 주기당 스냅샷 하나로 바꿔(쓰기 시점의 권위 있는 게이트는 store가 트랜잭션 안에서 확인한다) 순수 함수 `prometheusQueries`로 분리하고 테스트를 추가했다. 임시 postgres:16-alpine 컨테이너를 띄워 `go vet ./...`, `go test -race ./...`, 통합 테스트(`-run Integration ./internal/store ./internal/api`), `scripts/check-version.sh`, `scripts/check-screenshots.mjs`, `npm run lint`, `npm test`(18파일 59개) 모두 통과했다.
- 보류 아이디어: 로그인 리미터 `succeeded`가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화 / `internal/store` 커버리지 16.7% 중 DB 없이 테스트 가능한 순수 함수 경로 보강 / API 계층에서 `page`·`page_size` 파싱 실패를 400으로 돌려주기 / `auth/oidc.go`가 `RandomToken` 오류를 무시하고 state·nonce·verifier를 만드는 부분 정리 / 쿼리로만 GPU로 분류되는 별칭 지표는 여전히 `feature_enabled` 없이 빈 목록을 받는 잔여 간극 검토

