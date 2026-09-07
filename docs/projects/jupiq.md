---
title: "jupiq — 자율 개선 이력"
description: "jupiq: 자율 개선 회차 7회, 릴리즈 3건. 최근 릴리즈 없음."
last_modified_at: 2026-09-07 11:43:01 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "jupiq",
 "codeRepository": "https://github.com/hkjang/jupiq",
 "url": "https://hkjang.github.io/aidev/projects/jupiq/",
 "description": "jupiq: 자율 개선 회차 7회, 릴리즈 3건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-07T11:43:01+09:00"
}
</script>

# jupiq

<p class="tldr"><strong>요약.</strong> jupiq: 자율 개선 회차 7회, 릴리즈 3건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>7</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>2</b><span>배포 준비 완료</span></li><li><b>1</b><span>릴리즈 진행 중</span></li><li><b>2</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>1</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$7.32</b><span>비용</span></li><li><b>19분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/jupiq">https://github.com/hkjang/jupiq</a></dd>
<dt>마지막 회차</dt><dd>2026-09-07 10:02 KST — <span class="pill pill-other">• 기타</span> release-only, release skipped</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>릴리즈할 미출시 커밋이 없습니다. origin/main과 이 체크아웃의 HEAD가 모두 def3e1928d68b0d1d1a2221b95f000da85eb57ea이며, 이는 이미 태그 v1.4.6이 가리키는 커밋이고 GitHub Release jupiq v1.4.6(2026-09-07T00:38:58Z)으로 이미 배포되었습니다. git diff v1.4.6..HEAD와 git log v1.4.6..HEAD 모두 비어 있고 VERSION은 1.4.6입니다. 이번에 언급된 &#x27;허브 수집 goroutine 동시 실행 상한&#x27; 변경(e870473)은 git tag --contains 결과 v1.4.5·v1.4.6에 이미 포함되어 있습니다. 내용 변경이 전혀 없는 v1.4.7 태그를 새로 만드는 것은 빈 릴리즈이므로 수행하지 않았습니다.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-07 10:02</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> release-only, release skipped</td></tr><tr data-status="other"><td data-label="일시">2026-09-07 01:59</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> review held, PR open <a href="https://github.com/hkjang/jupiq/pull/5">PR #5</a><div class="meta">3파일 <span style="color:var(--good)">+150</span>/<span style="color:var(--bad)">−4</span> · 테스트 1 — fix: 허브 수집 goroutine에 동시 실행 상한과 종료 대기를 도입한다</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-06 16:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/jupiq/pull/4">PR #4</a>, release blocked (secrets)<div class="meta">2파일 <span style="color:var(--good)">+169</span>/<span style="color:var(--bad)">−5</span> · 테스트 1 — fix: 응답 보안 헤더에 CSP 보강 지시자와 TLS 한정 HSTS를 추가한다</div></td></tr><tr data-status="released"><td data-label="일시">2026-09-05 04:17</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-released" title="outcome=release-ready">배포 준비 완료</span> merged <a href="https://github.com/hkjang/jupiq/pull/3">PR #3</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></td></tr><tr data-status="released"><td data-label="일시">2026-09-04 08:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=releasing">릴리즈 진행 중</span> merged <a href="https://github.com/hkjang/jupiq/pull/2">PR #2</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.0">v1.4.0</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="nochange"><td data-label="일시">2026-09-03 23:04</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-nochange" title="outcome=no-change">변경 없음</span> no change</td></tr><tr data-status="released"><td data-label="일시">2026-09-03 14:58</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-released" title="outcome=release-ready">배포 준비 완료</span> merged <a href="https://github.com/hkjang/jupiq/pull/1">PR #1</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.3.0">v1.3.0</a></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">10:02</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">11</td><td data-label="비용" class="num">$0.55</td><td data-label="토큰 입력/출력" class="num">274K / 5K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">01:58</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="단계">review</td><td data-label="시간" class="num">4분</td><td data-label="턴" class="num">14</td><td data-label="비용" class="num">$0.99</td><td data-label="토큰 입력/출력" class="num">540K / 14K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">01:55</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">34</td><td data-label="비용" class="num">$1.71</td><td data-label="토큰 입력/출력" class="num">1.4M / 19K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">16:00</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">21</td><td data-label="비용" class="num">$0.90</td><td data-label="토큰 입력/출력" class="num">756K / 5K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">15:58</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="단계">review</td><td data-label="시간" class="num">2분</td><td data-label="턴" class="num">18</td><td data-label="비용" class="num">$0.68</td><td data-label="토큰 입력/출력" class="num">400K / 8K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">15:55</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">44</td><td data-label="비용" class="num">$2.50</td><td data-label="토큰 입력/출력" class="num">2.4M / 21K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 6 / 전체 7

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">internal/secure의 EncryptString·DecryptString·Derive·RandomToken 테스트 공백 보강</td><td data-label="가치/위험/크기">3/1/S</td><td data-label="상태">대기</td><td data-label="메모">커버리지는 0%가 아니라 39.5%였다(cipher_test.go가 Encrypt/Decrypt 라운드트립과 context 인증만 덮는다). base64 라운드트립, 잘못된 인코딩·잘린 blob 거부, Derive의 결정성과 label 분리, RandomToken 길이·유일성이 여전히 미검증이다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">collectHubs의 due가 실패한 프로브도 성공처럼 간격을 소비하는 문제 검토</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">due(hub)가 goroutine 시작 전에 last를 기록해, 자격증명 읽기나 허브 응답이 실패해도 다음 시도가 CollectIntervalSeconds 전체만큼 늦춰진다. 실패 시 last를 되돌리거나 짧은 재시도 간격을 쓰면 복구가 빨라지지만, 죽은 허브를 매 30초마다 두드리지 않도록 백오프 설계가 함께 필요하다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">internal/api/helpers.go의 사용되지 않는 parseTimeQuery 제거</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">boundedTimeRange가 RFC3339 검증과 기간 상한까지 담당해 완전히 대체했다. 이번 회차에도 grep으로 정의만 남고 호출자가 없음을 재확인했다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">로그인 리미터 succeeded가 ip 키를 의도적으로 유지하는 동작에 대한 테스트·문서화</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">succeeded는 pair·account 키만 지우고 ip 키는 남겨 한 IP에서의 분산 스프레이 방어를 유지한다. 의도가 코드에 적혀 있지 않아 나중에 &#x27;버그&#x27;로 오해되어 지워질 위험이 있다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">collectPrometheus가 metric마다 features 설정을 다시 읽는 중복 조회 제거</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">루프 앞에서 gpuMonitoring을 이미 구했는데 루프 안에서 metric마다 featureEnabled를 또 호출한다. collectLLMUsage처럼 주기 중 기능 OFF를 반영하려는 의도일 수 있어 의도 확인이 먼저다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">로그인 리미터 prune의 최고령 축출이 활성 차단 항목을 지울 수 있는 점 점검</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">attempts가 loginLimiterMax(10000)에 도달하면 windowStarted가 가장 오래된 항목부터 지우는데, 이는 곧 만료될 항목이 아니라 차단이 아직 유효한 항목일 수 있다. 실제 악용은 ip 한도(40) 때문에 분산 공격에 한정되므로 우선 위협 모델을 정리한 뒤 판단한다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">collectHubs의 goroutine 동시성 상한과 종료 대기 도입</td><td data-label="가치/위험/크기">4/2/M</td><td data-label="상태">완료</td><td data-label="메모">goHub(용량 8 세마포어 + 취소 시 대기열 폐기)와 waitForHubs(10초 상한)를 추가하고 Run 반환 시에만 대기해 주기 지연을 피했다. main은 HTTP 종료 후 collector 종료를 기다린 뒤 DB를 닫는다. 테스트 3개 추가, 커버리지 13.9%→18.7%.</td><td data-label="갱신">2026-09-07</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

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


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
