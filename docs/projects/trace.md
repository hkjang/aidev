---
title: "trace — 자율 개선 이력"
description: "trace: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-21 04:45:19 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "trace",
 "codeRepository": "https://github.com/hkjang/trace",
 "url": "https://hkjang.github.io/aidev/projects/trace/",
 "description": "trace: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-21T04:45:19+09:00"
}
</script>

# trace

<p class="tldr"><strong>요약.</strong> trace: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$2.64</b><span>비용</span></li><li><b>6분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/trace">https://github.com/hkjang/trace</a></dd>
<dt>마지막 회차</dt><dd>2026-09-14 19:18 KST — <span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/trace/pull/1">PR #1</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-14 19:18</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/trace/">trace</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/trace/pull/1">PR #1</a><div class="meta">18파일 <span style="color:var(--good)">+854</span>/<span style="color:var(--bad)">−74</span> · 테스트 2 — feat(auth): add silent SSO auto-login via OIDC prompt=none</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">23:52</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/trace/">trace</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">0분</td><td data-label="턴" class="num">1</td><td data-label="비용" class="num">$0.00</td><td data-label="토큰 입력/출력" class="num">0 / 0</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">19:17</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/trace/">trace</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">6분</td><td data-label="턴" class="num">58</td><td data-label="비용" class="num">$2.64</td><td data-label="토큰 입력/출력" class="num">2.1M / 27K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">OIDC 시작 엔드포인트 IP 레이트리밋</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">releasedock은 oidc-ip 키로 분당 120회 제한. Trace에는 레이트리미터가 없어 간단한 메모리 토큰버킷 도입 필요.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">OIDC 핸들러 수준 테스트를 위한 Store 인터페이스 분리</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">httpapi.Server가 *store.Store 구체 타입에 의존해 start/callback을 DB 없이 테스트 불가. 지금은 순수 헬퍼만 테스트함.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">폐쇄망용 평문 HTTP issuer 허용 옵션(AllowInsecureEndpoints, 사설 호스트 한정)</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">대기</td><td data-label="메모">표준 &#x27;폐쇄망에서&#x27; 절. validateHTTPURL이 http를 이미 허용하므로 현재는 오히려 공개 호스트 평문도 막지 않음 — 정책을 명시적으로 바꾸는 작업이라 별도 세션 권장.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">LoginPage에서 ?error= / ?sso=none 쿼리를 읽어 안내 문구 표시</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">현재 LoginPage는 콜백이 붙인 error 코드를 무시함. sso=none은 안내 없이 조용히 두고 error 코드만 한국어로 매핑하면 충분.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">openapi.yaml·docs/api.md에 public config의 oidc.autoLogin과 ?prompt=none 명세 추가</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">openapi.yaml에는 OIDC 설정 스키마 자체가 없음. 문서 정합성 작업.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">Silent SSO 자동 로그인 (OIDC prompt=none, oidc.autoLogin 기본 꺼짐)</td><td data-label="가치/위험/크기">5/2/M</td><td data-label="상태">완료</td><td data-label="메모">캠페인 silent-sso-2026-09. 서버 prompt=none 게이트+oauth_states.silent, 콜백 /login?sso=none, 프런트 silentSso.ts 3겹 루프 방지, ADMIN_GUIDE.md 추가. 커밋 8b8159b.</td><td data-label="갱신">2026-09-14</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-14
- 선택: Silent SSO 자동 로그인 (OIDC prompt=none, auto_login 설정) — 캠페인 silent-sso-2026-09 (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 이 저장소에는 이미 Keycloak OIDC(PKCE, DB 저장 state)가 있어 표준에 맞춰 silent SSO를 얹었다. 서버는 `oidc.autoLogin`(기본 꺼짐)이 켜져 있을 때만 `?prompt=none`을 존중하고 그 여부를 `oauth_states.silent`(마이그레이션 0003)에 기록해, 콜백이 `login_required`를 받으면 `/login?sso=none`으로 보낸다; `returnTo`는 `/`로 시작하고 `//`·`/\`가 아닌 같은 출처 경로만 받는다. 프런트는 `frontend/src/lib/silentSso.ts`에 규칙(탭 세션당 한 번 sessionStorage 표시, 로그아웃 억제, `?sso=none` 표시, 저장소 읽기 실패 시 '이미 시도'로 간주, `/login`·`/api/`·`/mcp`·`/healthz` 제외, 최상위 이동)을 두고 `AuthProvider`의 `/api/v1/me` 실패 시 한 번 시도한다. 관리 화면에 토글, `docs/ADMIN_GUIDE.md`(신규)와 operations.md에 설명 추가. 검증: `go vet ./... && go test ./...`(새 `auth_test.go` 3개), `npm test`(vitest 신규 도입, silentSso 9개 케이스), `npm run lint`, `npm run build` 모두 통과; Makefile `test`와 CI에 `npm test` 추가.
- 보류 아이디어: (1) OIDC 시작 엔드포인트 IP 레이트리밋(releasedock처럼 분당 제한) — 가치 3/위험 2/S; (2) LoginPage가 `?error=`·`?sso=none` 쿼리를 읽어 사용자에게 안내 문구 표시 — 가치 2/위험 1/S; (3) `handleOIDCStart`/`Callback` 핸들러 수준 테스트를 위한 Store 인터페이스 분리 — 가치 3/위험 3/M; (4) api/openapi.yaml·docs/api.md에 `/api/v1/public/config`의 `oidc.autoLogin` 필드와 `?prompt=none` 파라미터 명세 추가 — 가치 2/위험 1/S; (5) 폐쇄망용 평문 HTTP issuer 허용 옵션(`AllowInsecureEndpoints`, 사설 호스트 한정) — 가치 3/위험 3/M.


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
