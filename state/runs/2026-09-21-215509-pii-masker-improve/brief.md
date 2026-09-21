- 과제: 업스트림 진단 문자열을 UTF-8 경계에서 절단 (가치 2 / 위험 1 / 작업량 S)
- 왜: `internal/upstage/client.go:truncateString`이 바이트 인덱스로 한글·이모지를 자르므로 정상 UTF-8 업스트림 응답도 API 오류 상세와 디버그 본문 끝에서 대체문자(�)로 손상된다. 공통 절단 함수에서 기존 바이트 상한을 지키며 문자 경계를 보존하면 연결 점검·마스킹 오류·성공 디버그의 진단을 모두 온전하게 읽을 수 있다.
- 수용 기준:
  1) 유효한 UTF-8 입력의 절단 결과는 유효한 UTF-8이며 양수 maxLength 바이트를 넘지 않는다. TrimSpace, maxLength<=0의 무제한 의미, 상한 이하 문자열의 원문 유지, ASCII 출력은 유지한다. 초과 시 maxLength>3이면 `...`의 3바이트를 예약하고 그 안에 들어가는 완전한 문자 접두부만 남긴다. maxLength=1~3이면 말줄임 없이 완전한 문자만 남긴다(아무 문자도 안 들어가면 빈 문자열).
  2) `summarizeResponseBody`의 280바이트와 `formatDebugBody`의 16*1024바이트 양쪽에 적용한다. `startAppServerWithUpstream`으로 실제 app.New/HTTP 서버를 띄워, 한글 장문을 반환하는 업스트림 500에 대해 POST /v1/test-connection과 POST /v1/mask의 JSON detail이 대체문자 없이 완전한 접두부+`...`인지 확인한다. 기존 응답 상태(연결 점검 HTTP 200/ok=false, mask HTTP 502), server_error, retryable 의미는 유지한다. 성공 200 응답의 디버그 body도 같은 조건을 만족해야 한다.
  3) 단위 테이블은 ASCII·한글(3바이트)·이모지(4바이트)·혼합 문자열, 경계 직전/일치/초과, 1~3/0/음수 한도를 검증한다. 통합 테스트는 UTF-8 유효성만 검사하지 말고 디코딩된 문자열의 정확한 접두부/말줄임/바이트 길이 및 입력에 없던 U+FFFD 부재를 확인한다(JSON 인코더는 손상된 바이트를 이미 U+FFFD로 바꿈). 수정 전 새 회귀 테스트가 실패하고 수정 후 통과해야 한다. 성공 응답은 빈 fields와 긴 message를 사용해 `completed`/applied_regions=0 및 정상 파일 파트가 유지됨을 확인한다.
- 건드릴 파일:
  - internal/upstage/client.go:truncateString — 바이트 예산 안에서 UTF-8 경계로 절단. summarizeResponseBody/formatDebugBody는 호출 경로 확인 대상으로 두고 한도나 추론 원문 파싱을 변경하지 않는다.
  - internal/upstage/client_test.go — 위 경계 테이블 추가. 필요 시 실제 httptest 업스트림→NewClient.ParseDocument의 ResponseDebug.Body 검증을 보완한다.
  - internal/httpapi/integration_test.go:startAppServerWithUpstream, buildMultipartBody, createBlankPNG — 기존 헬퍼를 재사용하여 연결 점검/마스킹 오류/성공 디버그 통합 회귀 추가.
- 검증 명령: 저장소 루트에서 `go test -count=1 ./internal/upstage ./internal/httpapi`, `go test -count=1 ./...`, `go vet ./...`, `go build ./...`, `git diff --check`. 이번 정찰에서 전체 테스트·vet·build 통과. 포맷 확인은 `gofmt -l internal/upstage/client.go internal/upstage/client_test.go internal/httpapi/integration_test.go`.
- 위험과 피할 것: 바이트 상한을 rune 개수 상한으로 바꾸지 않는다. debug를 추론 원문으로 재사용하지 말고 DocumentResult.RawBody/ParsePayload 경로를 유지한다. service.process 오류 경로는 빈 DocumentResult의 ResponseDebug를 직렬화하므로 오류 debug 본문 전달 개선을 끼워 넣지 않는다(오류 detail과 성공 debug로 검증). 인증·allow-host·리다이렉트·8MB 응답 제한·jobs 저장 계약·라우팅·의존성·워크플로는 건드리지 않는다. 원래부터 잘못된 UTF-8이거나 8MB/32KB 읽기 상한에서 이미 잘린 입력의 교정은 이번 범위 밖이며 이 과제를 PII 스크럽 개선으로 표현하지 않는다.
- 차선 후보: internal/config Load 경유 환경변수 정규화·기본값 테스트 (가치 2 / 위험 1 / 작업량 S) — UTF-8 과제가 이미 해결된 경우에만 선택. config_test.go는 현재 서버 timeout 테스트 3개뿐이며 t.Setenv와 t.TempDir로 Load를 호출하고, normalizeAllowHosts/normalizeEndpointURL/normalizePIILang/normalizePIISchema/envInt/envNonNegativeInt/envBool의 실제 반환 설정을 검증한다. t.Parallel 금지, 외부 환경을 명시적으로 격리하고 프로덕션 정규화 의미는 바꾸지 않는다.

근거와 구현 순서:
- 기준 HEAD 3487570. client.go:736의 value[:maxLength-3] 및 <=3 분기의 value[:maxLength]를 직접 확인했다. 요약과 debug 두 호출자도 확인했다.
- 정찰에서 수정 없는 실제 cmd/pii-masker 바이너리와 로컬 HTTP 업스트림을 실행했다. `{"message":"가" 6000회,"fields":[]}`를 반환했을 때 연결 점검은 HTTP 200/server_error/detail 끝 `가가가가�...`, mask 오류는 HTTP 502/server_error/detail에 U+FFFD, 성공 mask는 HTTP 200/completed/debug.body 끝 `가가가��...`를 관찰했다. 디버그는 engine.debug.response 문자열을 JSON으로 한 번 더 디코딩해야 body를 볼 수 있다. 업스트림 request-id 헤더는 생략하여 상세 뒤의 추가 suffix가 바이트 예산 검사를 흐리지 않게 한다.
- 40분 예상: 회귀 테스트 및 기존 실패 확인 15분 → 함수 수정 5분 → 전체 검증 10분 → 경계 사례/실패 대응 여유 10분. rune 슬라이스로 전체 문자열을 복제할 필요 없이 절단 인덱스를 문자 시작까지 뒤로 이동하는 방식이 작은 변경이다. invalid UTF-8의 새 정규화 정책은 도입하지 않는다.
- 요청 스킬 pmo:estimating-and-contingency, technology:implementation-planning, technology:solution-exploration은 호출 가능한 Skill/skills.list/read 도구가 없고 로컬 스킬 경로·SKILL.md 내용 검색에서도 발견되지 않아 해당 절차·반환 형식은 미확인이다. 위 추정·대안·구현 순서는 프롬프트를 따른 자체 작성이며 스킬 적용 완료를 뜻하지 않는다.
