---
title: "jikim — 자율 개선 이력"
description: "jikim: 자율 개선 회차 2회, 릴리즈 1건. 최근 릴리즈 v0.2.2."
last_modified_at: 2026-09-06 01:30:07 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "jikim",
 "codeRepository": "https://github.com/hkjang/jikim",
 "url": "https://hkjang.github.io/aidev/projects/jikim/",
 "description": "jikim: 자율 개선 회차 2회, 릴리즈 1건. 최근 릴리즈 v0.2.2.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-06T01:30:07+09:00",
 "version": "0.2.2"
}
</script>

# jikim

<p class="tldr"><strong>요약.</strong> jikim: 자율 개선 회차 2회, 릴리즈 1건. 최근 릴리즈 v0.2.2. <span class="pill pill-released" title="14일: 릴리즈 1, 실패 0, 경고 0, 회귀 0">건강 A</span> <span class="meta">14일: 릴리즈 1, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>2</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>1</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>1</b><span>실행 오류</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/jikim">https://github.com/hkjang/jikim</a></dd>
<dt>마지막 회차</dt><dd>2026-09-05 19:50 KST — <span class="pill pill-other">• 기타</span> hold: budget</dd>
<dt>최근 릴리즈</dt><dd><a href="https://github.com/hkjang/jikim/releases/tag/v0.2.2">v0.2.2</a> — released <a href="https://github.com/hkjang/jikim/releases">전체 릴리즈 →</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-05 19:50</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=error">실행 오류</span> hold: budget</td></tr><tr data-status="released"><td data-label="일시">2026-09-05 03:52</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/jikim/">jikim</a></td><td data-label="결과"><span class="pill pill-released" title="outcome=release-ready">배포 준비 완료</span> merged <a href="https://github.com/hkjang/jikim/pull/14">PR #14</a>, released <a href="https://github.com/hkjang/jikim/releases/tag/v0.2.2">v0.2.2</a></td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-05
- 선택: SQL LIKE 와일드카드 이스케이프로 OpenBao LIST prefix 범위 이탈 수정 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `ListSecretChildren`이 caller prefix를 그대로 `LIKE` 패턴에 넣어 `_`, `%`가 와일드카드로 해석됐고, `kvlike_x` prefix LIST가 `kvlikeXx` 하위 key 이름까지 반환했습니다(감사 로그·Secret 검색어도 동일 문제). `internal/store/store.go`에 `escapeLike`, `secrets.go`에 `secretChildrenPattern`을 추가해 세 호출지점 모두 문자 그대로 대조하도록 고쳤습니다. 검증은 `logic_test.go` 단위 테스트 추가 + 일회용 postgres:17-alpine 컨테이너에 `JIKIM_TEST_POSTGRES_DSN`을 걸어 통합 테스트를 실행해 수정 전 FAIL / 수정 후 PASS를 확인했고, `./scripts/verify.sh` 전체(Go test·vet·gofmt, React test·lint·build, docs, compose)가 통과했습니다. 커밋 `6f66501`.
- 보류 아이디어:
  - `security` 설정의 잘못된 타입 값(예: `allow_local_login: "false"`)이 오류 없이 저장되고 조용히 무시되는 문제를 검증으로 막기 (가치 3 / 위험 2 / S)
  - 신뢰 프록시 목록 기반 `X-Forwarded-For` 처리로 감사 로그·로그인 rate limit의 IP가 리버스 프록시 주소로 고정되는 문제 해결 (가치 4 / 위험 3 / M)
  - Transit `batch_input`/`batch_results` 지원 추가로 OpenBao 호환 범위 확대 (가치 3 / 위험 3 / M)
  - `baoUserpassLogin`에서 `AllowLocalLogin` 확인 전에 rate limiter를 `succeeded`로 초기화하는 순서 정리 (가치 2 / 위험 1 / S)
  - `server.go`의 `var _ = fmt.Sprintf`, `auth_handlers.go`의 `var _ = store.ErrUnauthorized` 같은 데드 코드 제거 (가치 1 / 위험 1 / S)
- 릴리즈: v0.2.2 (2026-09-05)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
