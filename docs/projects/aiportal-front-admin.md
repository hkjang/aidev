---
title: "aiportal-front-admin — 자율 개선 이력"
description: "aiportal-front-admin: 자율 개선 회차 11회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-07 11:14:43 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-front-admin",
 "codeRepository": "https://github.com/hkjang/aiportal-front-admin",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-front-admin/",
 "description": "aiportal-front-admin: 자율 개선 회차 11회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-07T11:14:43+09:00"
}
</script>

# aiportal-front-admin

<p class="tldr"><strong>요약.</strong> aiportal-front-admin: 자율 개선 회차 11회, 릴리즈 0건. 최근 릴리즈 없음. <span class="pill pill-merged" title="14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0">건강 B</span> <span class="meta">14일: 릴리즈 0, 실패 0, 경고 0, 회귀 0</span></p>

<ul class="stats"><li><b>11</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>배포 준비 완료</span></li><li><b>0</b><span>릴리즈 진행 중</span></li><li><b>9</b><span>병합 완료</span></li><li><b>0</b><span>검토 대기</span></li><li><b>2</b><span>검증 실패</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실행 오류</span></li><li><b>$10.22</b><span>비용</span></li><li><b>26분</b><span>에이전트 시간</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-front-admin">https://github.com/hkjang/aiportal-front-admin</a></dd>
<dt>마지막 회차</dt><dd>2026-09-07 08:45 KST — <span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/9">PR #9</a>, release skipped</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>이 저장소에는 릴리즈 관례가 존재하지 않는다. git tag 0개(로컬·원격 모두), GitHub Release 0개, CHANGELOG.md / docs/RELEASE*.md / 릴리즈 노트 없음, .github/workflows 디렉터리 자체가 없음, scripts/release*.sh·Makefile 등 릴리즈 스크립트도 없음. 버전 필드는 root package.json 0.0.0(private)과 upgrade/admin-v2/package.json 0.1.0 두 곳에 있으나 둘 다 최초 커밋 값 그대로이며 git 이력상 한 번도 증가한 적이 없다. 배포는 .gitlab-ci.yml이 main/develop 브랜치 push에 반응해 build 후 PVC 경로로 복사하는 방식이라 태그·버전과 무관하다. docs/IMPROVEMENTS.md OBS-001도 &#x27;release/version 표식&#x27;을 아직 없는 항목으로 적고 있고 docs/INDEX.md는 &#x27;날짜와 version history는 실제 release 근거가 있을 때만 기록한다&#x27;고 명시한다. 따라서 어떤 패키지의 버전을 올릴지, 태그 형식, 릴리즈 노트 위치·양식을 전부 새로 정해야 하므로 지침 5에 따라 아무것도 만들지 않고 skipped로 기록한다.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt" data-filter="1"><thead><tr><th>일시</th><th class="primary">프로젝트</th><th>결과</th></tr></thead><tbody><tr data-status="merged"><td data-label="일시">2026-09-07 08:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/9">PR #9</a>, release skipped<div class="meta">9파일 <span style="color:var(--good)">+1044</span>/<span style="color:var(--bad)">−13</span> · 테스트 3 — fix(admin-v2): 딥링크 query가 바뀌면 화면 조건을 다시 맞춘다</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 23:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요<div class="meta">3파일 <span style="color:var(--good)">+199</span>/<span style="color:var(--bad)">−11</span> · 테스트 1 — fix(admin-v2): 목록의 날짜 값을 시간대와 형식에 흔들리지 않게 표시한다</div></td></tr><tr data-status="other"><td data-label="일시">2026-09-06 01:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-failed" title="outcome=verify-failed">검증 실패</span> verify failed: 실행한 검증 명령이 없음 — 정책(verify) 또는 자동 감지 필요<div class="meta">7파일 <span style="color:var(--good)">+219</span>/<span style="color:var(--bad)">−12</span> · 테스트 2 — fix(admin-v2): 세션이 만료되면 라우트 이동을 기다리지 않고 복구한다</div></td></tr><tr data-status="merged"><td data-label="일시">2026-09-05 00:10</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/8">PR #8</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-04 05:30</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/7">PR #7</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 21:58</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 13:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/5">PR #5</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 07:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-03 01:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 19:26</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시">2026-09-02 13:24</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="결과"><span class="pill pill-merged" title="outcome=merged">병합 완료</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

## 비용·사용량

<div class="table-wrap"><table class="rt"><caption class="meta">최근 30세션</caption><thead><tr><th>시각</th><th class="primary">프로젝트</th><th>단계</th><th class="num">시간</th><th class="num">턴</th><th class="num">비용</th><th class="num">토큰 입력/출력</th><th>종료</th></tr></thead><tbody><tr data-status="other"><td data-label="시각">08:45</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="단계">릴리즈</td><td data-label="시간" class="num">1분</td><td data-label="턴" class="num">12</td><td data-label="비용" class="num">$0.46</td><td data-label="토큰 입력/출력" class="num">251K / 4K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">08:44</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="단계">review</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">16</td><td data-label="비용" class="num">$1.09</td><td data-label="토큰 입력/출력" class="num">628K / 14K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">08:38</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">7분</td><td data-label="턴" class="num">56</td><td data-label="비용" class="num">$3.62</td><td data-label="토큰 입력/출력" class="num">3.8M / 33K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">23:35</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">5분</td><td data-label="턴" class="num">39</td><td data-label="비용" class="num">$1.78</td><td data-label="토큰 입력/출력" class="num">1.4M / 22K</td><td data-label="종료">success</td></tr><tr data-status="other"><td data-label="시각">01:47</td><td data-label="프로젝트" class="primary"><a href="https://hkjang.github.io/aidev/projects/aiportal-front-admin/">aiportal-front-admin</a></td><td data-label="단계">개선</td><td data-label="시간" class="num">7분</td><td data-label="턴" class="num">56</td><td data-label="비용" class="num">$3.28</td><td data-label="토큰 입력/출력" class="num">3.2M / 30K</td><td data-label="종료">success</td></tr></tbody></table></div>

## 아이디어 백로그 — 대기 5 / 전체 9

<div class="table-wrap"><table class="rt" data-filter="1"><caption class="meta">에이전트가 회차마다 재평가한다. 가치 높고 위험 낮은 대기 항목이 다음 회차 후보다.</caption><thead><tr><th class="primary">아이디어</th><th>가치/위험/크기</th><th>상태</th><th>메모</th><th>갱신</th></tr></thead><tbody><tr data-status="nochange"><td data-label="아이디어" class="primary">루트 앱의 crypto-js local tarball 의존성 제거로 clean install 복구</td><td data-label="가치/위험/크기">4/4/M</td><td data-label="상태">대기</td><td data-label="메모">루트 package.json이 crypto-js를 file:crypto-js-4.2.0.tgz로 참조해 npm ci가 불가하다. admin-v2는 별도 package.json이라 영향이 없다(이번 세션에서도 admin-v2 npm ci 정상 확인). 루트 앱은 테스트가 없어 회귀 검증 비용이 크다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">monitoringParser.toPod의 restarts NaN 노출</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">대기</td><td data-label="메모">kubectl 스타일 &quot;3 (5d ago)&quot; 같은 문자열이 오면 Number()가 NaN이 되어 표에 그대로 나온다(합계는 formatNumber가 0으로 가린다). 선행 정수만 파싱하고 실패하면 0으로 떨어뜨린다. 순수 함수라 기존 vitest로 검증 가능.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">AdvancedPolicyView가 snapshot.extensions를 v-model로 직접 변형</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">저장 실패 시 체크박스는 바뀐 상태로 남고 서버는 그대로라 화면과 서버가 어긋난다. form에 복사본을 두고 저장 성공 시에만 반영해야 한다. 이제 컴포넌트 테스트 환경(jsdom, @vue/test-utils)이 있어 검증 가능하다. 다만 운영에서는 enablePolicyWrites가 false라 실사용 영향은 작다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">PaginationBar가 범위 밖 페이지에 머물 때 표시가 어긋남</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">unwrapPageEnvelope가 요청 페이지를 유지하므로 총건수가 줄면 page &gt; totalPages가 되어 &quot;31–25 / 25&quot;처럼 표시되고, page가 totalPages+2 이상이면 이전 버튼이 disabled가 아닌데도 move() 가드에 막혀 아무 동작도 하지 않는다. 이제 컴포넌트 테스트 환경이 있어 검증 가능하다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="nochange"><td data-label="아이디어" class="primary">탭이 공유하는 loading·error 상태가 다른 탭으로 새어 나감</td><td data-label="가치/위험/크기">2/2/S</td><td data-label="상태">대기</td><td data-label="메모">CatalogView(자료실/공유 앱)와 DirectoryView(사용자/부서), ContentAccessView(메뉴/접근)는 ref 하나를 두 목록이 함께 쓴다. 한 탭의 조회가 실패한 뒤 탭을 바꾸면 관계없는 오류 문구가 남고, 느린 탭의 응답이 끝나면 다른 탭의 loading이 풀린다. 탭별 상태로 분리하면 컴포넌트 테스트로 고정할 수 있다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">세션 만료를 라우트 이동 없이 즉시 복구</td><td data-label="가치/위험/크기">4/2/M</td><td data-label="상태">완료</td><td data-label="메모">2026-09-06 회차 커밋 e425b54(브랜치 auto/2026-09-06-0140, 아직 main 미병합)에서 session.ts의 setSessionExpiredHandler()와 auth/guard.ts의 resolveAdminAccess()로 구현됐다. 이번 브랜치는 main(ce38705) 기준이라 코드에는 아직 없다. 병합 후 중복 구현하지 말 것.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">CatalogView·ContentAccessView가 같은 경로의 route query 변경에 반응하지 않음</td><td data-label="가치/위험/크기">3/2/M</td><td data-label="상태">완료</td><td data-label="메모">shared/routeQuery.ts의 useRouteQuerySync()가 진입과 query 변경에서 같은 적용 함수를 부른다. jsdom·@vue/test-utils를 devDependency로 추가하고 파일 단위 // @vitest-environment jsdom으로 두 화면의 딥링크 테스트를 붙였다. 커밋 c99bca9.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="failed"><td data-label="아이디어" class="primary">세션 만료 후 refresh token으로 원요청 1회 재시도</td><td data-label="가치/위험/크기">3/3/M</td><td data-label="상태">기각</td><td data-label="메모">루트 앱의 refresh 재시도는 별도 python backend endpoint를 쓰므로 admin-v2의 backendBaseUrl 계약 밖이다. 2026-09-06 회차에서 검토 후 제외했고 docs/ARCHITECTURE.md에 &#x27;재시도하지 않는다&#x27;로 문서화돼 있다.</td><td data-label="갱신">2026-09-07</td></tr><tr data-status="released"><td data-label="아이디어" class="primary">shared/format.ts 단위 테스트 공백과 formatDateTime 입력 형식 처리</td><td data-label="가치/위험/크기">2/1/S</td><td data-label="상태">완료</td><td data-label="메모">2026-09-06 회차 커밋 ad2f001(브랜치 auto/2026-09-06-2330, 아직 main 미병합)에서 처리됐다. 병합 후 중복 구현하지 말 것.</td><td data-label="갱신">2026-09-07</td></tr></tbody></table></div>

## 원장 (에이전트가 남긴 기록)

## 2026-09-02
- 선택: admin-v2 runtime config의 scheme-relative URL 우회 차단 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `validateRuntimeConfig`와 `scripts/validate-runtime-config.mjs`가 `//evil.test`, `/\evil.test`를 "상대 경로"로 보고 통과시켜 `allowedHosts` allowlist를 우회할 수 있었다. `backendBaseUrl`은 `access_token` 헤더·`withCredentials`와 함께 쓰이고 `authSsoUrl`은 `window.location.assign` 대상이라 토큰 유출·오픈 리다이렉트로 이어진다. 상대 경로 값을 sentinel origin(`https://relative.invalid/`)에 resolve해 origin이 유지되는지 확인하도록 두 검증 지점을 동일하게 고치고 회귀 테스트 3개를 추가했다. 검증은 `npm run verify`(typecheck + vitest 24개 통과, 기존 21개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 18개 검증) 전체 통과, 그리고 악성 값을 넣은 임시 runtime.json으로 build-time script가 exit 1을 내는지 직접 확인했다. 커밋 fdb7ab2.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 수정 (가치 3 / 위험 2 / S)
  - 절대 URL allowlist를 hostname뿐 아니라 port·scheme까지 비교하도록 강화 (가치 3 / 위험 2 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)
  - `monitoringParser.findRows` 재귀에 깊이 제한·순환 참조 방어 추가 및 테스트 (가치 2 / 위험 1 / S)
  - 루트 앱 Playwright 설정만 있고 `@playwright/test` dependency와 test script가 없는 상태 정비 (가치 3 / 위험 2 / M)

## 2026-09-02
- 선택: runtime config allowlist를 host(port 포함) 비교로 강화하고 평문 http 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: `validateNetworkValue`가 `url.hostname`만 allowlist와 비교해 `https://api.intranet.local:8443`처럼 allowlist된 host의 다른 port로 우회할 수 있었고, `http://` 절대 URL도 그대로 통과해 `access_token` 헤더와 `withCredentials` cookie가 평문으로 나갈 수 있었다. `url.host` 비교로 바꾸고(allowedHosts 항목을 `host`/`host:port`로 정규화, 기본 port 명시는 동일 취급), loopback(`localhost`/`127.0.0.1`/`[::1]`) 외 http를 거부하며, 절대 URL이 없어도 allowedHosts 항목 형식(scheme·경로·자격증명 포함 금지)을 검사하도록 했다. 동일 규칙을 `scripts/validate-runtime-config.mjs`에도 반영했다. 검증은 `npm run verify`(typecheck + vitest 30개 통과, 기존 24개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 18개) 전체 통과, 그리고 임시 runtime.json 7종(평문 http, 다른 port, host:port 항목, loopback http, 잘못된 allowedHosts 항목 2종, 상대 경로)으로 build-time script의 exit code를 직접 대조했다. 커밋 d2796b9.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 수정 (가치 3 / 위험 2 / S)
  - 목록 화면(Catalog/Directory/ContentAccess)의 stale response가 최신 결과를 덮어쓰는 race 방지 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils)이 없어 선행 정비 필요 (가치 3 / 위험 3 / M)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)
  - `monitoringParser.findRows` 재귀에 깊이 제한 추가 및 테스트 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)

## 2026-09-03
- 선택: 목록 화면의 stale response가 최신 결과를 덮어쓰는 race 방지 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: CatalogView·DirectoryView·ContentAccessView의 조회 함수들이 요청 순서를 추적하지 않아, PaginationBar 버튼·탭 전환·행 선택처럼 `:disabled="loading"`이 걸리지 않은 동선에서 요청이 겹치면 늦게 도착한 이전 응답이 최신 결과를 덮어썼다(다른 페이지 목록 표시, 취소한 상세 재출현, 지나간 오류 메시지 노출). 새 `shared/async.ts`의 `createRequestGuard()`로 ticket을 발급해 최신 요청의 결과만 상태에 반영하고, 목록을 다시 읽을 때는 진행 중인 상세 요청을 `invalidate()`로 버리며, UUID 검증 실패 같은 조기 반환 경로에서도 loading을 정리하도록 했다. 검증은 `npm run verify`(typecheck + vitest 35개 통과, 기존 30개 + guard 단위 테스트 5개로 응답 역순 도착 시나리오 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 bbab374.
- 보류 아이디어:
  - `unwrapPageEnvelope`가 서버 pageInfo 누락 시 요청한 pageNum/pageSize를 잃고 1/10으로 되돌려 페이지 이동이 막히는 문제 (가치 3 / 위험 1 / S)
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - `monitoringParser.findRows` 재귀에 깊이 제한 추가 및 테스트 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-03
- 선택: 세션 확인 실패 시 SSO 무한 왕복 차단 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: router guard가 `ensureAdminSession()` 실패를 (관리자 권한 없음 메시지를 뺀) 전부 `redirectToSso()`로 처리해, backend `/sso/ssologin`이 장애이거나 네트워크가 끊기면 SSO가 앱으로 돌려보낼 때마다 다시 SSO로 나가는 무한 왕복이 생기고 사용자는 오류 화면조차 볼 수 없었다. 새 `auth/ssoRedirect.ts`에 sessionStorage 기반 사용량 기록을 두어 60초 창에서 자동 이동을 2회로 제한하고(초과 시 access-denied 화면 + 수동 SSO 버튼), 세션 확보 시 사용량을 비우며, 저장소를 못 쓰거나 예외를 던지면 왕복 감지가 불가하므로 자동 이동을 막도록 했다. 검증은 `npm run verify`(typecheck + vitest 46개 통과, 기존 35개 + 신규 11개로 한도·창 만료·시계 역행·저장소 장애 시나리오 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 74f26a1.
- 보류 아이디어:
  - `unwrapPageEnvelope`가 서버 pageInfo 누락 시 요청한 pageNum/pageSize를 잃고 1/10·totalCount 0으로 되돌려 페이지 이동 버튼이 잠기는 문제 (가치 3 / 위험 1 / S)
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - `monitoringParser.toPod`이 `restarts`에 비숫자 문자열이 오면 NaN을 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-03
- 선택: 서버가 pageInfo의 pageNum/pageSize를 생략할 때 요청 페이지 유지 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `docs/BACKEND_API_REFERENCE.md`·`docs/API_GUIDE.md`에 기록된 기존 backend 공통 목록 계약은 `pageInfo: { totalCount }`만 반환하는데, `unwrapPageEnvelope`가 pageNum/pageSize를 서버 값에서만 읽어 매 요청마다 1/10으로 되돌렸다. 그래서 실제 backend에서는 Catalog·Directory 목록이 2페이지 데이터를 받고도 1페이지로 표시되고 다음 버튼이 같은 페이지를 반복 요청해 페이지 이동이 사실상 막혔다(mock의 `paginate`는 요청 페이지를 그대로 돌려주므로 드러나지 않았다). 서버 값이 없거나 유효하지 않으면 요청에 사용한 값을 유지하도록 하고 `apiGetPage`가 요청 params에서 그 값을 읽어 넘기게 했으며, QUICK_WINS 문서에 계약을 명시했다. 검증은 `npm run verify`(typecheck + vitest 52개 통과, 기존 46개 + 신규 6개로 totalCount만 오는 응답·유효하지 않은 서버 값·서버 값 우선·params 파싱, 그리고 axios adapter로 `apiGetPage` 전체 경로 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 e05b613.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - `monitoringParser.toPod`이 `restarts`에 비숫자 문자열이 오면 NaN을 그대로 표에 노출하는 문제 (가치 2 / 위험 1 / S)
  - `monitoringParser.findRows` 재귀에 깊이 제한·순환 참조 방어 추가 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-03
- 선택: API 401 응답에서 캐시된 관리자 세션 무효화 (가치 4 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 쿠키 세션이 만료돼 API가 401을 돌려줘도 `ensureAdminSession()`이 `state.initialized && is_admin === 'Y'` 캐시를 그대로 신뢰해 라우트를 옮겨도 `/sso/ssologin`을 다시 호출하지 않았고, 사용자는 전체 새로고침 전까지 모든 조회가 "API 요청에 실패했습니다."로 끝나는 화면에 갇혔다(router guard는 navigation 때만 돌고 http layer에는 401 처리가 전혀 없었다). `http.ts`에 `setUnauthorizedHandler` 등록 지점을 두어 401에서만(403은 인가 실패이므로 제외) handler를 호출하고, `session.ts`가 localStorage와 reactive 상태를 비우는 `invalidateAdminSession()`을 module scope에서 등록하도록 했다(session→http 단방향 import라 순환 없음). 서버가 message를 주지 않는 401·403·네트워크 단절에는 다음 행동을 알 수 있는 문구를 쓰도록 `fallbackErrorMessage`를 분리했다. 검증은 `npm run verify`(typecheck + vitest 60개 통과, 기존 52개 + 신규 8개로 axios adapter가 던지는 401/403/서버 message 우선/handler 예외 격리, 그리고 `vi.mock('@/api/http')` + `vi.resetModules()`로 세션 캐시 재사용·무효화 후 재요청 시나리오 포함)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과. 커밋 0f98a0a.
- 보류 아이디어:
  - `ensureAdminSession(force=true)`가 진행 중인 비강제 요청 promise를 그대로 반환하는 재진입 버그 (가치 3 / 위험 2 / S)
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 (가치 3 / 위험 2 / M)
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 노출하는 문제 (가치 2 / 위험 1 / S)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-04
- 선택: 뒤늦게 도착한 401이 방금 확보한 세션을 지우는 문제 수정 (가치 4 / 위험 2 / 작업량 S)
- 결과: 성공
- 요약: 목록 화면은 한 번에 여러 요청을 보내므로 세션 만료 시 401도 여러 개 돌아오는데, 첫 401이 세션을 버리고 router guard가 `/sso/ssologin`으로 세션을 다시 확보한 뒤 남은 401이 도착하면 방금 저장한 user와 access token을 다시 지웠다(직전 세션에서 도입한 401 handler가 무조건 `invalidateAdminSession()`을 부른 탓). 그러면 이후 요청이 `access_token` 헤더 없이 나가 또 401을 받는 악순환이 생긴다. `http.ts`가 요청 interceptor에서 세션 세대를 config에 새겨 401 handler에 넘기고, `session.ts`가 그 세대가 현재 세대와 같을 때만 캐시를 버리도록 했다(확보·무효화·로그아웃·다른 탭 storage 변경마다 세대 증가 → 같은 세대의 중복 401도 한 번만 처리). 로드맵 P2가 요구하는 "단일 상태기계" 방향으로 `ensureAdminSession(force=true)`가 진행 중인 비강제 promise를 재사용하던 재진입 결함도 함께 고쳤다. 검증은 `npm run verify`(typecheck + vitest 64개 통과, 기존 60개 + 신규 4개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 두 수정을 각각 되돌려 신규 테스트가 실제로 실패하는지 직접 확인했다. 커밋 f02fb08.
- 보류 아이디어:
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils) 선행 정비 필요 (가치 3 / 위험 3 / M)
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)나 boolean `ready`에서 NaN·"true"를 표에 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 + `formatDateTime`이 epoch millis를 날짜로 해석하지 못하고 원시 숫자 문자열을 그대로 보여주는 문제 (가치 2 / 위험 1 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-05
- 선택: 기존 backend의 400 + 토큰 code를 세션 만료로 처리 (가치 5 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: 지난 두 세션이 고친 401 세션 정리 경로가 실환경에서는 아예 동작하지 않았다. 루트 앱 `src/api/common/interceptors.js:192-195`와 `docs/API_GUIDE.md`(오류와 재시도 표), `docs/ARCHITECTURE.md`가 모두 기록하듯 기존 backend는 토큰 문제를 HTTP 401이 아니라 **HTTP 400 + body code `401`(토큰 없음)/`403`(만료)/`405`(토큰 정보 오류)** 로 알리는데, admin-v2 `http.ts`는 `error.response?.status === 401`만 봤다. 그래서 세션이 끝나도 캐시된 관리자 세션이 계속 유효해 보이고 router guard가 `/sso/ssologin`을 다시 부르지 않아 사용자는 새로고침 전까지 모든 조회가 "API 요청에 실패했습니다."로 끝나는 화면에 갇혔다. 판정을 `isSessionExpired()` 한곳에 모아 세션 정리 handler 호출과 `fallbackErrorMessage`가 같은 기준을 쓰게 했고(HTTP 403 인가 실패와 `BZ01` 같은 400 업무 오류는 제외), admin-v2 ARCHITECTURE.md에 이 신호 계약을 문서화했다. 검증은 `npm run verify`(typecheck + vitest 70개 통과, 기존 64개 + 신규 6개로 `isSessionExpired` 단위 판정 3개와 axios adapter를 통한 400+`403` handler 호출·400+`BZ01` 미호출·fallback 문구)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 수정을 되돌려 신규 interceptor 테스트 2개가 실제로 실패하는지 직접 확인했다. 커밋 c2b6751.
- 보류 아이디어:
  - 세션 만료 판정 후 refresh token으로 원요청을 1회 재시도하는 경로 도입 — 루트 앱은 이미 하지만 admin-v2는 SSO 재확인에만 의존한다 (가치 3 / 위험 3 / M)
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils) 선행 정비 필요 (가치 3 / 위험 3 / M)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 + `formatDateTime`이 epoch millis를 날짜로 해석하지 못하는 문제 (가치 2 / 위험 1 / S)
## 2026-09-06
- 선택: 세션 만료를 라우트 이동 없이 즉시 복구 (가치 4 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 지난 세 세션이 만든 만료 감지 경로(HTTP 401과 기존 backend의 400 + 토큰 code)는 캐시된 관리자 세션을 버리기만 하고, 다시 확인하는 주체는 `router.beforeEach`뿐이었다. 그래서 한 화면에 머무르며 검색·페이지 이동·운영 현황 자동 갱신만 하는 사용자는 라우트를 옮기기 전까지 모든 조회가 "세션이 만료됐습니다. 다시 로그인하십시오."로 끝나는 화면에 갇혔다(루트 앱의 refresh 재시도는 별도 python backend endpoint라 admin-v2의 `backendBaseUrl` 계약 밖이므로 채택하지 않았다). `session.ts`에 `setSessionExpiredHandler()`를 두어 캐시를 버린 직후 복구를 알리고, guard 본문을 새 `auth/guard.ts`의 `resolveAdminAccess(force)`(`allow`/`redirected`/`denied`)로 옮겨 router guard와 만료 복구가 같은 판단과 같은 SSO 왕복 제한을 공유하게 했다. `/sso/ssologin` 자체가 만료 응답을 받는 경우는 확인이 끝나기 전에 또 확인을 시작하는 재귀가 되므로 `state.checking`으로 걸러낸다. 검증은 `npm run verify`(typecheck + vitest 80개 통과, 기존 70개 + 신규 10개로 `resolveAdminAccess` 6가지 판단과 만료 handler 호출·오래된 세대 무시·확인 중 재귀 차단·handler 예외 격리)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 만료 알림 호출과 재귀 차단 가드를 각각 되돌려 신규 테스트가 실제로 실패하는지 직접 확인했다. 커밋 e425b54.
- 보류 아이디어:
  - PaginationBar가 요청 페이지를 유지하다 총건수가 줄면 범위 밖 페이지에 머물러 "31–25 / 25" 같은 표시를 내는 문제 — 컴포넌트 테스트 환경 선행 필요 (가치 2 / 위험 2 / S)
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils) 선행 정비 필요 (가치 3 / 위험 3 / M)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - `shared/format.ts` 단위 테스트 공백 보강 + `formatDateTime`이 epoch millis·`yyyyMMddHHmmss`를 날짜로 해석하지 못하는 문제 (가치 2 / 위험 1 / S)

## 2026-09-06
- 선택: 목록의 날짜 값을 시간대·형식에 흔들리지 않게 표시 (가치 3 / 위험 1 / 작업량 S)
- 결과: 성공
- 요약: `shared/format.ts`는 단위 테스트가 전혀 없었고 `formatDateTime`이 `new Date(String(value))` 하나로 모든 입력을 처리해, 자료실·공유 앱·정책·콘텐츠 접근·운영 현황 표의 날짜 칸이 두 가지로 어긋났다. (1) 날짜만 있는 `2026-08-25`는 UTC 자정으로 해석돼 KST에서는 없는 시각 "09:00"이 붙고 UTC-8 환경에서는 전날로 밀린다(node로 직접 재현 확인). (2) Java backend가 흔히 문자열로 내려주는 `yyyyMMdd`·`yyyyMMddHHmmss`와 epoch millis는 해석하지 못해 `20260825143005` 같은 원시 숫자가 그대로 표에 남는다. offset이 없는 값은 브라우저 구현에 맡기지 않고 지역 시각으로 직접 만들고, 날짜만 있는 값은 시각 없이 날짜로만 표시하며, `Z`·`+09:00` offset은 그대로 존중하고, 자릿수로 단위를 단정할 수 없는 숫자(10자리 등)나 달력에 없는 날짜(`2026-13-01`)는 조작하지 않고 원본을 남기도록 했다. 이 계약을 admin-v2 ARCHITECTURE.md에 문서화했다. 검증은 `npm run verify`(typecheck + vitest 83개 통과, 기존 70개 + 신규 13개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 신규 테스트를 KST·UTC·America/Los_Angeles 세 시간대에서 각각 돌려 기대값이 시간대에 의존하지 않는지 확인했다. 커밋 ad2f001.
- 보류 아이디어:
  - `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 그대로 노출하는 문제 (가치 2 / 위험 1 / S)
  - CatalogView·ContentAccessView가 route query를 onMounted에서만 읽어 같은 경로의 query 변경(딥링크 재진입)에 반응하지 않는 문제 — 컴포넌트 테스트 환경(jsdom, @vue/test-utils) 선행 정비 필요 (가치 3 / 위험 3 / M)
  - AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버 상태가 어긋나는 문제 (가치 2 / 위험 2 / S)
  - PaginationBar가 범위 밖 페이지(총건수 감소)에 머물면 "31–25 / 25"로 표시되고 이전 버튼이 `move()` 가드에 막혀 아무 동작도 하지 않는 문제 (가치 2 / 위험 2 / S)
  - 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 — 현재 `npm ci` 자체가 불가해 검증 비용 큼 (가치 4 / 위험 4 / M)

## 2026-09-07
- 선택: 딥링크 query 변경에 화면이 반응하지 않는 문제 (가치 3 / 위험 2 / 작업량 M)
- 결과: 성공
- 요약: 사용자·부서 화면이 거는 `#/content?tab=access&userId=…`와 `#/catalog?tab=shared&userId=…` 링크는 경로가 같고 query만 바뀌므로 vue-router가 화면을 다시 mount하지 않는데, 두 화면 모두 `queryValue()`를 setup/onMounted에서만 읽어 이전 사용자의 조건과 결과가 그대로 남았다(사이드바로 같은 화면을 다시 눌러 query를 비워도 마찬가지). `shared/routeQuery.ts`에 `useRouteQuerySync()`를 두어 진입과 query 변경이 같은 적용 함수를 부르게 하고, 화면이 쓰지 않는 query가 바뀌거나 다른 경로로 떠나는 중(route는 컴포넌트보다 먼저 바뀐다)이면 조회하지 않도록 진입 경로를 기억해 걸렀다. 세 세션째 "컴포넌트 테스트 환경 선행 필요"로 미뤄지던 항목이라 `jsdom`·`@vue/test-utils`를 devDependency로 넣고 파일 단위 `// @vitest-environment jsdom`으로 전역 설정 변경 없이 두 화면의 딥링크 재진입 테스트를 붙였다. 검증은 `npm run verify`(typecheck + vitest 83개 통과, 기존 70개 + 신규 13개)와 `npm run build`(build → runtime-config 검증 → offline 검사 → integrity manifest 19개) 전체 통과, 그리고 query 감시를 꺼서 신규 컴포넌트 테스트 5개가 실제로 실패하는지 직접 확인했다. 커밋 c99bca9.
- 보류 아이디어: `monitoringParser.toPod`이 kubectl 스타일 `restarts`(`"3 (5d ago)"`)에서 NaN을 표에 노출 (2/1/S) · AdvancedPolicyView가 `snapshot.extensions`를 v-model로 직접 변형해 저장 실패 시 화면과 서버가 어긋남 (2/2/S) · PaginationBar가 범위 밖 페이지에서 "31–25 / 25"를 표시하고 이전 버튼이 `move()` 가드에 막힘 — 이제 컴포넌트 테스트로 검증 가능 (2/2/S) · CatalogView·DirectoryView가 두 탭이 공유하는 `loading`·`error`를 써서 한쪽 탭의 오류가 다른 탭에 남음 (2/2/S) · 루트 앱의 `crypto-js` local tarball 의존성 제거로 clean install 복구 (4/4/M)


[← 대시보드](https://hkjang.github.io/aidev/) · [교훈 모음](https://hkjang.github.io/aidev/lessons/)

{% endraw %}
