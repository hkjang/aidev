---
title: "jupiq — 자율 개선 이력"
description: "jupiq: 자율 개선 회차 4회, 릴리즈 3건. 최근 릴리즈 v1.4.4."
last_modified_at: 2026-09-05 11:08:19 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "jupiq",
 "codeRepository": "https://github.com/hkjang/jupiq",
 "url": "https://hkjang.github.io/aidev/projects/jupiq/",
 "description": "jupiq: 자율 개선 회차 4회, 릴리즈 3건. 최근 릴리즈 v1.4.4.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T11:08:19+09:00",
 "version": "1.4.4"
}
</script>

# jupiq

<p class="tldr"><strong>요약.</strong> jupiq: 자율 개선 회차 4회, 릴리즈 3건. 최근 릴리즈 v1.4.4. <span class="pill pill-merged" title="14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 3, 실패 0, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>4</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>3</b><span>릴리즈</span></li><li><b>0</b><span>머지(릴리즈 없음)</span></li><li><b>1</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/jupiq">https://github.com/hkjang/jupiq</a></dd>
<dt>마지막 회차</dt><dd>2026-09-05 04:17 KST — <span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jupiq/pull/3">PR #3</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></dd>
<dt>최근 릴리즈</dt><dd><a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a> — released <a href="https://github.com/hkjang/jupiq/releases">전체 릴리즈 →</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="released"><td data-label="일시">2026-09-05 04:17</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jupiq/pull/3">PR #3</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.4">v1.4.4</a></td></tr><tr data-status="released"><td data-label="일시">2026-09-04 08:40</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jupiq/pull/2">PR #2</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.4.0">v1.4.0</a>, <strong>ASSETS MISSING</strong></td></tr><tr data-status="nochange"><td data-label="일시">2026-09-03 23:04</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-nochange">➖ 변경 없음</span> no change</td></tr><tr data-status="released"><td data-label="일시">2026-09-03 14:58</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jupiq/">jupiq</a></td><td data-label="결과"><span class="pill pill-released">🚀 릴리즈</span> merged <a href="https://github.com/hkjang/jupiq/pull/1">PR #1</a>, released <a href="https://github.com/hkjang/jupiq/releases/tag/v1.3.0">v1.3.0</a></td></tr></tbody></table></div>

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


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
