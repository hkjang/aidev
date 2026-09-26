# 회차 노트 2026-09-26-220242-pii-masker-improve — pii-masker
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 22:02] base pinned — main@101bf10
- [러너 22:02] autonomy release — 

## 정찰 노트
- 리다이렉트 스킴 강등을 골랐다: 프로덕션 파일 1개(internal/upstage/client.go:httpClient)로 끝나고, 보류 목록 상위(config 테이블 테스트 2/1, download_url 헬퍼 2/1)는 순수 테스트·리팩터라 관측 가능한 개선이 없다. 405 Allow는 라우터 전체 영향, 동기 대기열 상한과 이력 limit은 설계 선행이라 한 세션 범위 밖.
- 확신 없는 곳(과제서에 미확인으로 적음): 동일 호스트 https→http 강등에서 Go가 Authorization 헤더를 유지하는지 — GOROOT 소스 접근이 막혀 이 세션에서 확인 못 했다. 다만 본문이 *bytes.Buffer라 GetBody가 채워지고 307/308에서 멀티파트 PII 본문이 재전송되는 것은 코드로 확인했으므로 수정 근거는 헤더 여부와 무관하다.
- 구현자 주의: 검사 조건을 "최초 요청이 https일 때만"으로 한정할 것. 내장 mock은 http라 http 시작 체인을 막으면 internal/app 통합 테스트가 통째로 죽는다.
- httptest.NewTLSServer로 end-to-end https 테스트를 하지 말 것 — httpClient()가 기본 Transport를 써서 자체 서명 인증서 검증 실패로 실패 원인이 흐려진다. https 축은 분리한 순수 헬퍼의 테이블 테스트로 덮는다.
- 기준선: main@101bf10에서 `go test -count=1 ./...` 전부 통과 확인(2026-09-26). 프로필(2026-09-24)은 config.go·server.go·client.go를 직접 대조해 여전히 정확하므로 새로 쓰지 않았다.
- [러너 22:07] scout done — 업스트림 리다이렉트의 HTTPS→HTTP 강등 차단 (가치 3 / 위험 2 / 작업량 S)

## 구현 노트
- 무엇/왜: `httpClient()`의 `CheckRedirect`가 호스트만 보고 스킴을 안 봐서 https로 나간 요청이 http로 강등 리다이렉트되면 멀티파트 PII 본문이 평문으로 재전송됐다(본문이 `*bytes.Buffer`라 `GetBody`가 채워짐). 순수 헬퍼 `checkRedirectScheme(first, target *url.URL)`을 더해 `via[0].URL.Scheme`이 https(대소문자 무시)면 대상도 https여야 통과하게 했다. 프로덕션 파일 1개(`internal/upstage/client.go`) + 테스트 + README 한 문단.
- 확신 없는 곳/검증 못 한 것: (1) **동일 호스트 강등에서 Go가 `Authorization: Bearer`를 유지하는지 확인 못 했다** — GOROOT 소스 접근이 막혀 있어 정찰 노트의 '미확인'을 그대로 남겼다. 수정 근거는 본문 재전송이라 헤더 여부와 무관하다. (2) **end-to-end https 경로는 실행하지 않았다** — 정찰 지시대로 `httptest.NewTLSServer`를 피했고(기본 Transport라 자체 서명 인증서 검증 실패가 원인을 가림), 대신 `client.httpClient().CheckRedirect` 클로저를 직접 호출해 프로덕션 정책 함수를 지나게 했다. 실제 TLS 소켓 위에서 Go가 이 에러를 `Do`에 그대로 올려 주는지는 프레임워크 계약에 의존하며, 같은 계약은 기존 `TestParseDocumentRejectsRedirectToDisallowedHost`(http 체인, 대상 핸들러 미호출 단언)가 이미 증명한다. (3) 빈 스킴(`target.Scheme == ""`)은 `defaultIfEmpty`로 메시지만 처리했고, 실제로 상대 리다이렉트를 Go가 절대 URL로 해석하므로 도달 불가 경로로 보지만 단언은 하지 않았다.
- 일부러 하지 않은 것: 최초 요청 자체의 스킴은 검사하지 않았다(사용자가 http 업스트림을 명시하는 것은 허용 구성이고 내장 mock이 http다). `hostAllowed`의 bare-host 포트 무시 규칙, `allowedHosts()` 폴백, `normalizeAllowHosts`는 손대지 않았다. `classifyRequestError`에 새 코드를 추가하지 않아 강등 거부는 `network_error`로 분류된다 — 의도적이다(`blockedHostError`와 섞이지 않는 것이 목표였고, 새 오류 코드는 공개 계약 확장이라 범위 밖).
- 다음 역할이 조심할 것: 새 테스트는 외부 의존이 전혀 없고 `t.Parallel`을 쓴다(환경변수 미사용). 검사 조건을 "최초가 https일 때만"에서 넓히면 내장 mock(http)이 통째로 죽으므로 `internal/app` 통합 테스트가 먼저 터진다. 커밋 3eb0ffb.
- [러너 22:10] brief accepted — 채택 — 과제서의 근거(`client.go:459`가 호스트만 검사, 본문이 `*bytes.Buffer`라 `GetBody`가 채워짐)가 현재 코드와 정확히 일치
- [러너 22:10] verify passed — 검증 3개 통과 (policy)

## 비평 노트
- 확인한 것: `git checkout main -- internal/upstage/client.go`로 수정 전 코드에 새 테스트를 돌려 `https_to_http_on_the_same_host`·`uppercase_...` 2건이 실제로 FAIL하는 것을 직접 재현했고(원장 `실패 재현:` 줄과 일치), 복원 후 트리 clean·`go test -count=1 ./...`·`go vet`·`gofmt -l` 전부 통과. 외부로 나가는 HTTP 클라이언트는 `httpClient()` 하나뿐이라(client.go:224, 286) TestConnection·ParseDocument 양쪽에 정책이 걸린다. 본문이 `*bytes.Buffer`(client.go:215, 278)라 `GetBody` 충족도 확인. diff는 3파일 87줄, 범위 이탈 없음.
- 구현자의 미확인 (1) 해소 — GOROOT(go1.26.7) `src/net/http/client.go:688`에서 `stripSensitiveHeaders`는 `reqs[0].URL.Host != req.URL.Host`일 때만 평가되므로 **동일 호스트 https→http 강등에서는 Authorization/x-api-key가 그대로 복사된다**. 즉 수정 근거는 본문 재전송뿐 아니라 토큰 평문 노출까지 포함해 원장에 적힌 것보다 강하다. 릴리즈 노트에 이 사실을 넣으면 좋다.
- 못 본 것: 실제 TLS 소켓 위 end-to-end 경로(정찰·구현 지시대로 `httptest.NewTLSServer` 미사용). CheckRedirect가 `send` 전에 호출된다는 것은 stdlib 코드(client.go:699)로만 확인했고 실측은 안 했다.
- 승인이어도 남는 우려: (a) `http→https→http` 체인은 `via[0]`가 http라 통과한다 — 1홉이 이미 평문이라 한계 노출 증가는 없어 차단은 아니지만, 앞으로 http 기본 URL + 업그레이드를 정상 경로로 삼으면 재검토가 필요하다. (b) 강등 거부가 `network_error`(retryable=true)로 분류되는데 이 거부는 영구적이다 — `Retryable`은 보고용 필드일 뿐 자동 재시도 루프가 없어 표시상 문제에 그친다. (c) `HTTPS://` 대문자 케이스는 `url.Parse`가 스킴을 소문자화하므로 실전 경로에서는 사실상 항상 참이며 헬퍼의 `EqualFold`만 덮는다.
- 판정: approve / risk low / blocking 없음(security·legal 모두 차단 사유 없음 — 신규 수집·전송·의존성 없고 평문 전송을 좁히는 변경).
- [러너 22:14] review approved — 리뷰 승인 (risk=low)
- [러너 22:14] pr created — https://github.com/hkjang/pii-masker/pull/27
- [러너 22:14] ci passed — 검사 없음 — 정책으로 허용
- [러너 22:14] merge done — 3eb0ffb
- [러너 22:17] release published — v1.0.29
- [러너 22:17] gh-release created — GitHub Release v1.0.29
- [러너 22:17] manifest ok — pii-masker-image.tar.gz 
- [러너 22:18] assets uploaded — 1개
- [러너 22:18] assets verified — v1.0.29 자산 1개 (이전 v1.0.28: 1)
