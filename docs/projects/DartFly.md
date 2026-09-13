---
title: "DartFly — 자율 개선 이력"
description: "DartFly: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-14 04:37:36 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "DartFly",
 "codeRepository": "https://github.com/hkjang/DartFly",
 "url": "https://hkjang.github.io/aidev/projects/DartFly/",
 "description": "DartFly: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-14T04:37:36+09:00"
}
</script>

# DartFly

<p class="tldr"><strong>요약.</strong> DartFly: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$7.84</b><span>비용</span></li><li><b>15분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/DartFly">https://github.com/hkjang/DartFly</a></dd>
<dt>마지막 회차</dt><dd>2026-09-14 00:37 KST — <span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/DartFly/pull/2">PR #2</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-14 00:37</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/DartFly/">DartFly</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/DartFly/pull/2">PR #2</a><div class="meta">23파일 <span style="color:var(--good)">+718</span>/<span style="color:var(--bad)">−33</span> · 테스트 3 — feat: IdP 에 이미 로그인한 사람은 로그인 화면 없이 들어오게 (SSO auto_login, prompt=none)</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">00:36</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/DartFly/">DartFly</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">15분</td><td data-label="턴" class="num">81</td><td data-label="비용" class="num">$7.84</td><td data-label="토큰 입력/출력" class="num">9.5M / 55K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">로그아웃 시 IdP end_session(RP-initiated logout) 호출</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">현재 로그아웃 억제는 탭(sessionStorage) 단위라 다른 탭에서는 IdP 세션으로 조용히 다시 들어옴. 표준 범위 밖이며 IdP 설정(post_logout_redirect_uri) 필요</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">OIDC env 폴백에 DARTFLY_OIDC_AUTO_LOGIN 추가</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">env 전용 배포에서는 auto_login 을 켤 수 없음. 다만 env OIDC 변수(DARTFLY_OIDC_*)가 environment-variables.md 에 문서화조차 안 되어 있어 관리 화면이 정식 경로. 문서화와 함께 할 것</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">로그인 페이지 조용한 시도 중 폼 깜빡임 제거</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">system/info 응답 전까지 로그인 카드가 잠깐 보였다가 IdP 로 이동. auto_login 켜진 경우만 결정 전까지 카드를 숨기는 소폭 UX. system/info 실패 시 반드시 다시 보여야 함</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">조용한 시도의 IdP 오류값을 서버 로그에 남기기</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">지금은 silent 흐름의 모든 error 를 거절(sso=none)로 같게 처리. login_required/interaction_required/consent_required 외의 값(예: invalid_client)은 설정 문제이므로 Warn 로그로 구분하면 진단에 도움</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">401 리다이렉트 헬퍼 일원화</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">app.js·resources.js·mcp.js·admin-*.js 가 각자 fetch 래퍼에서 401 처리(이번에 loginUrl() 로 통일했지만 코드 중복은 그대로). layout.js 의 api() 로 모으면 return_to·CSRF 처리가 한 곳에 남음</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">Silent SSO(prompt=none) — auto_login 설정과 3겹 루프 방지</td><td data-label="가치/위험/크기">5/2/M</td><td data-label="상태">완료</td><td data-label="메모">캠페인 silent-sso-2026-09. 커밋 b88a3c8. 서버 prompt=none 게이트·흐름 쿠키·/login?sso=none, silentsso.js 규칙, df_sso_config.auto_login(DDL+ensureColumn), docs/ADMIN_GUIDE.md. 실제 MariaDB·바이너리로 신규/기존 설치 모두 확인</td><td data-label="갱신">2026-09-14</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-14
- 선택: Silent SSO(prompt=none) — auto_login 설정과 3겹 루프 방지 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 캠페인 silent-sso-2026-09 를 SILENT-SSO-STANDARD.md 대로 구현했습니다(커밋 b88a3c8). 서버는 `/auth/sso/login?prompt=none&return_to=` 를 받되 관리자 설정 `auto_login`(기본 꺼짐)이 꺼져 있으면 평범한 로그인으로 바꾸고, 흐름 쿠키(`df_sso_flow`)로 조용한 시도·return_to 를 콜백까지 들고 가 `login_required` 면 `/login?sso=none` 으로, 성공하면 같은 출처 return_to 로 보냅니다. 화면은 새 모듈 `internal/webui/js/silentsso.js` 가 sessionStorage 한 번 표시·로그아웃 억제·주소 표시·저장소 예외=이미 시도 규칙을 맡고, 로그인 페이지에서만 시도합니다(DartFly 는 SPA 가 아니라 401→/login 리다이렉트 구조라 로그인 페이지가 유일한 관문이며, 콜백/오류 마커가 붙은 주소에서는 시도하지 않음). 401 리다이렉트 7곳이 return_to 를 들고 가도록 바꿨습니다. `df_sso_config.auto_login` 컬럼은 002 DDL 과 기동 시 `ensureColumn`(information_schema 확인 후 ALTER) 양쪽에 두고 일치 테스트를 추가했습니다. 검증: gofmt/vet/`go test -race ./...` 통과, 새 테스트 6개(Go: auto_login 꺼짐이면 prompt=none 무시·켜면 전달·거절→sso=none·깊은 링크 복귀·return_to 검증·system/info 노출, JS: 규칙 전부), `test/smoke/run.sh` 실제 바이너리+MariaDB 통과, 컬럼을 지우고 재기동해 기존 설치 ALTER 경로도 실제 DB 로 확인. 저장소에 docs/ADMIN_GUIDE.md 가 없어 새로 만들고 README·authentication-hardening 에서 링크했습니다.
- 보류 아이디어: (1) OIDC env 폴백(DARTFLY_OIDC_*)에 AUTO_LOGIN 변수 추가 — env 전용 배포에서 켤 수 없음, 단 env OIDC 변수 자체가 문서화되어 있지 않음 / (2) 로그아웃 시 IdP end_session(RP-initiated logout)도 호출 — 다른 탭에서 조용히 다시 들어오는 것을 막으려면 필요하나 표준 범위 밖 / (3) 로그인 페이지에서 조용한 시도 중 폼이 잠깐 보이는 깜빡임 — 결정 전까지 카드를 숨기는 소폭 UX / (4) 콜백에서 IdP 오류값(login_required 외)을 서버 로그에 남기기 — 지금은 조용한 시도의 모든 오류를 same-as-refusal 로 처리


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
