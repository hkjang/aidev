---
title: "madi — 자율 개선 이력"
description: "madi: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-14 07:59:10 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "madi",
 "codeRepository": "https://github.com/hkjang/madi",
 "url": "https://hkjang.github.io/aidev/projects/madi/",
 "description": "madi: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-14T07:59:10+09:00"
}
</script>

# madi

<p class="tldr"><strong>요약.</strong> madi: 자율 개선 회차 1회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 1, 회귀 0</span></p>

<ul class="stats"><li><b>1</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>0</b><span>병합 완료</span></li><li><b>1</b><span>검토 대기</span></li><li><b>0</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$3.79</b><span>비용</span></li><li><b>8분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/madi">https://github.com/hkjang/madi</a></dd>
<dt>마지막 회차</dt><dd>2026-09-14 02:20 KST — <span class="pill pill-other">• 기타</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/madi/pull/1">PR #1</a></dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="other"><td data-label="일시">2026-09-14 02:20</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/madi/">madi</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=review-pending">검토 대기</span> <strong>guarded files</strong>, PR open <a href="https://github.com/hkjang/madi/pull/1">PR #1</a><div class="meta">13파일 <span style="color:var(--good)">+463</span>/<span style="color:var(--bad)">−16</span> · 테스트 2 — feat: add silent SSO (OIDC prompt=none) with opt-in auto_login</div></td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">02:20</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/madi/">madi</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">8분</td><td data-label="턴" class="num">59</td><td data-label="비용" class="num">$3.79</td><td data-label="토큰 입력/출력" class="num">3.8M / 30K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 6

<div class="table-wrap"><table class="rt"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">Playwright 브라우저 시험에 silent SSO 루프 방지 시나리오 추가</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">대기</td><td data-label="메모">가짜 OIDC 제공자로 login_required 왕복 후 새로고침 반복 시 리다이렉트가 반복되지 않는지 실제 브라우저에서 확인. tests/browser.mjs 구조를 따른다.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">비조용 OIDC 제공자 오류를 /login?sso=error 리다이렉트로 안내</td><td data-label="가치/위험/크기">3/2/S</td><td data-label="상태">대기</td><td data-label="메모">현재는 401 JSON 본문이 브라우저에 그대로 보인다. releasedock처럼 로그인 화면으로 보내되 sso=error 표시로 재시도를 막는다. 기존 통합 테스트의 401 기대치 수정 필요.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">로그인 화면에 ?sso=none 도착 시 안내 문구 표시</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">조용한 시도가 거절되어 로그인 화면이 뜬 것임을 사용자에게 알려 혼란을 줄인다. ErrorBox가 아닌 부드러운 안내여야 한다.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">SAML /auth/saml/start에도 return_to 보존 적용</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">OIDC와 같은 oidcReturnTo 검증 규칙을 재사용해 깊은 링크 복귀를 통일한다.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">OpenAPI 목록에 OIDC start의 prompt/return_to 파라미터 설명 추가</td><td data-label="가치/위험/크기">1/1/S</td><td data-label="상태">대기</td><td data-label="메모">openapi.go의 경로 목록은 요약만 담고 있어 파라미터 설명 구조가 있는지 먼저 확인해야 한다.</td><td data-label="갱신">2026-09-14</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">Silent SSO(OIDC prompt=none) + oidc_auto_login 설정 (캠페인 silent-sso-2026-09)</td><td data-label="가치/위험/크기">5/2/M</td><td data-label="상태">완료</td><td data-label="메모">서버 prompt=none 게이트·/login?sso=none 거절 처리·return_to 검증, 브라우저 3겹 루프 방지 규칙 모듈, 관리자 토글·가이드. 커밋 323950c.</td><td data-label="갱신">2026-09-14</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-14
- 선택: Silent SSO(OIDC prompt=none) + auto_login 설정 [캠페인 silent-sso-2026-09] (가치 5 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: madi에는 Keycloak OIDC(`integrations_oidc.go`)가 이미 있어 SILENT-SSO-STANDARD.md 대로 구현했다. 서버는 `oidc_auto_login`(기본 꺼짐)이 켜진 경우에만 `?prompt=none`을 제공자에 전달하고, `oidc_attempts`에 silent/return_to를 기록해 거절(`login_required`)이면 `/login?sso=none`으로, 성공이면 같은 출처 경로(`/`로 시작·`//` 제외)로 돌려보낸다. 브라우저는 새 모듈 `web/src/auth/silentSso.ts`가 sessionStorage 1회 표시(저장소 예외 시 '이미 시도'로 간주), 로그아웃 억제, URL 표시, 콜백·로그인·API·MCP·헬스 경로 제외 규칙으로 최상위 이동 한 번만 시도한다. 관리자 설정 UI 토글과 docs/admin-guide.md(manuals HTML 재생성)에 설정·동작을 적었다. 검증: `npm run build`(tsc), `go build/vet/gofmt`, 새 `tests/silent-sso.mjs`(CI 목록에 등록), 임시 PostgreSQL 17 컨테이너로 `TestPostgresOIDCSilentLogin`·기존 OIDC/설정/백업 통합 테스트, 전체 비DB `go test ./...` 통과. 커밋 323950c.
- 보류 아이디어: 로그인 화면에 `?sso=none` 도착 시 "회사 계정 세션이 없어 로그인 화면을 표시" 안내 문구(가치 2/위험 1/S); Playwright 브라우저 시험에 silent SSO 루프 방지 시나리오 추가(가치 3/위험 2/M); SAML `/auth/saml/start`에도 같은 return_to 보존 적용(가치 2/위험 2/S); 비조용 OIDC 제공자 오류(401 JSON)를 `/login?sso=error`로 안내해 사용자 경험 통일(가치 3/위험 2/S); OpenAPI 목록에 `prompt`/`return_to` 쿼리 파라미터 설명 추가(가치 1/위험 1/S).


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
