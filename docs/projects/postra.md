---
title: "postra — 자율 개선 이력"
description: "postra: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-14 00:37:02 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "postra",
 "codeRepository": "https://github.com/hkjang/postra",
 "url": "https://hkjang.github.io/aidev/projects/postra/",
 "description": "postra: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-14T00:37:02+09:00"
}
</script>

# postra

<p class="tldr"><strong>요약.</strong> postra: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 1, 경고 1, 회귀 0">건강 C</span> <span class="meta">14일: 릴리즈 0, 실패 1, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>1</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$4.97</b><span>비용</span></li><li><b>13분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/postra">https://github.com/hkjang/postra</a></dd>
<dt>마지막 회차</dt><dd>2026-09-13 22:34 KST — <span class="pill pill-failed">❌ 실패</span> <strong>CI failed</strong>, PR open <a href="https://github.com/hkjang/postra/pull/3">PR #3</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="failed"><td data-label="일시">2026-09-13 22:34</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/postra/">postra</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> <strong>CI failed</strong>, PR open <a href="https://github.com/hkjang/postra/pull/3">PR #3</a><div class="meta">11파일 <span style="color:var(--good)">+631</span>/<span style="color:var(--bad)">−26</span> · 테스트 3 — feat(auth): silent SSO via OIDC prompt=none behind an auto_login setting</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">22:34</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/postra/">postra</a></td><td data-label="단계">review</td><td data-label="시간" class="num">3분</td><td data-label="턴" class="num">12</td><td data-label="비용" class="num">$0.93</td><td data-label="토큰 입력/출력" class="num">526K / 9K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">22:31</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/postra/">postra</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">10분</td><td data-label="턴" class="num">41</td><td data-label="비용" class="num">$4.04</td><td data-label="토큰 입력/출력" class="num">3.6M / 42K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">Read OIDC runtime once per login render (OIDCConfigured + OIDCAutoLoginEnabled both call SystemSettings)</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">loginPageData 가 설정을 두 번 읽고 secret 도 두 번 acquire. 단일 OIDCLoginState(ctx) 헬퍼로 합치기.</td><td data-label="갱신">2026-09-13</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">Record an incident when OIDC discovery fails in oidcStart</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">CompleteOIDC 는 recordIncident 를 남기지만 BeginOIDC 의 discovery 실패는 로그인 페이지 502 인라인만. 관리자 오류/장애 화면에서 보이도록.</td><td data-label="갱신">2026-09-13</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">Add gofmt check to CI (internal/application/incidents.go is currently unformatted)</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">gofmt -l 이 incidents.go 를 보고함. CI 에 gofmt -l 검사 단계 추가 + 해당 파일 정리.</td><td data-label="갱신">2026-09-13</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">OIDC callback: CompleteOIDC failure → redirect to /ui/login?sso=error instead of rendering on the callback URL</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">현재 콜백 URL(code=…)에 오류를 인라인 렌더. 새로고침하면 코드 재사용 시도. 마커 리다이렉트로 주소를 정리하되 error_description 은 세션/쿼리로 넘겨야 함.</td><td data-label="갱신">2026-09-13</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">Manual local login: honour return_to behind path-rewriting proxies with a relative redirect</td><td data-label="가치/위험/크기">1/2/S</td><td data-label="상태">대기</td><td data-label="메모">loginSubmit 은 return_to 가 있으면 절대 /ui/… 로 302. 공개 /login → 내부 /ui/login 프록시에서는 /ui 접두사가 외부에 노출되지 않을 수 있음. gate 도 이미 절대 /ui/login 을 쓰므로 실제 문제 여부 확인 필요.</td><td data-label="갱신">2026-09-13</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">Silent SSO (OIDC prompt=none) behind auth.oidc.auto_login with three-layer loop guard</td><td data-label="가치/위험/크기">5/2/M</td><td data-label="상태">완료</td><td data-label="메모">캠페인 silent-sso-2026-09. auto_login 기본 꺼짐, 서버가 prompt=none 다운그레이드, sessionStorage 1회 시도(읽기 실패=시도됨), 로그아웃 억제, 콜백 login_required → /ui/login?sso=none, return_to 검증. 커밋 8111af3.</td><td data-label="갱신">2026-09-13</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-13
- 선택: Silent SSO (OIDC prompt=none) — auto_login 설정 뒤에 두고 무한 루프 3중 방지 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 캠페인 "silent-sso-2026-09" 목표를 구현했다. `auth.oidc.auto_login` 설정(기본 꺼짐, 관리 화면 체크박스 + `POSTRA_OIDC_AUTO_LOGIN`)을 추가하고, 로그인 페이지가 최상위 이동으로 `/ui/auth/oidc/start?prompt=none&return_to=…` 을 한 탭 세션에 한 번만 시도하도록 했다(sessionStorage 표시, 읽기 실패는 '이미 시도'로 간주, 로그아웃 시 억제 표시, 콜백은 login_required/interaction_required/consent_required 를 받으면 `/ui/login?sso=none` 으로 보내 주소에 표시). 서버는 auto_login 이 꺼져 있으면 `?prompt=none` 을 조용히 일반 로그인으로 바꾸고, `return_to` 는 `/` 로 시작하고 `//` 로 시작하지 않는 앱 내부 경로만 받아 gate → 로그인 폼 → 서명된 flow 쿠키 → 콜백까지 전달한다. 검증: 새 테스트 7개(SafeReturnTo·login_required 판별·BeginOIDC 다운그레이드·gate 딥링크·로그인 페이지 트리거 유무·콜백 거절 처리·로그아웃 마커) 추가, `go build ./... && go vet ./... && go test -race ./...` 전부 통과. 관리자 가이드 3.3 절 추가.
- 보류 아이디어: (1) OIDC 콜백에서 CompleteOIDC 실패 시 sso=error 마커로 로그인 페이지 리다이렉트해 주소를 정리(가치 2/위험 2/S); (2) `OIDCConfigured`·`OIDCAutoLoginEnabled` 가 로그인 렌더마다 SystemSettings 를 두 번 읽음 — 한 번에 읽는 헬퍼로 합치기(가치 2/위험 1/S); (3) `oidcStart` 에서 discovery 실패 시 502 를 로그인 페이지에 인라인으로 보여주는데 incident 기록도 남기기(가치 2/위험 1/S); (4) `internal/application/incidents.go` 가 gofmt 미적용 상태 — CI 에 gofmt 검사 추가(가치 2/위험 1/S).


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
