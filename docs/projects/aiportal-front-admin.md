---
title: "aiportal-front-admin — 자율 개선 이력"
description: "aiportal-front-admin: 자율 개선 회차 8회, 릴리즈 0건. 최근 릴리즈 없음."
last_modified_at: 2026-09-05 10:00:32 +0900
---
{% raw %}
<script type="application/ld+json">
{
 "@context": "https://schema.org",
 "@type": "SoftwareSourceCode",
 "name": "aiportal-front-admin",
 "codeRepository": "https://github.com/hkjang/aiportal-front-admin",
 "url": "https://hkjang.github.io/aidev/projects/aiportal-front-admin/",
 "description": "aiportal-front-admin: 자율 개선 회차 8회, 릴리즈 0건. 최근 릴리즈 없음.",
 "inLanguage": "ko",
 "maintainer": {
  "@type": "Person",
  "name": "hkjang",
  "url": "https://github.com/hkjang"
 },
 "dateModified": "2026-09-05T10:00:32+09:00"
}
</script>

# aiportal-front-admin

<p class="tldr"><strong>요약.</strong> aiportal-front-admin: 자율 개선 회차 8회, 릴리즈 0건. 최근 릴리즈 없음.</p>

<ul class="stats"><li><b>8</b><span>회차</span></li><li><b>1</b><span>프로젝트</span></li><li><b>0</b><span>릴리즈</span></li><li><b>8</b><span>머지(릴리즈 없음)</span></li><li><b>0</b><span>변경 없음</span></li><li><b>0</b><span>실패</span></li></ul>

## 현황

<dl class="kv">
<dt>저장소</dt><dd><a href="https://github.com/hkjang/aiportal-front-admin">https://github.com/hkjang/aiportal-front-admin</a></dd>
<dt>마지막 회차</dt><dd>2026-09-05 00:10 KST — <span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/8">PR #8</a>, release skipped</dd>
<dt>최근 릴리즈</dt><dd>skipped — skipped</dd>
<dt>사유</dt><dd>이 저장소에는 따라갈 릴리즈 관례가 전혀 없다. git tag 0개(로컬·origin 모두, `git tag | wc -l`=0, `git ls-remote --tags origin` 결과 없음), GitHub Release 0개(`gh release list` 빈 결과, exit 0으로 접근 자체는 정상), CHANGELOG.md·docs/RELEASE*.md·릴리즈 노트 파일 없음, `.github/workflows/` 디렉터리 자체가 없어 태그 푸시에 반응하는 워크플로도 없음. 릴리즈 커밋 양식도 없다(전체 20개 커밋이 first commit/docs/feat/fix + PR merge 뿐). 버전 필드는 두 곳에 있으나 릴리즈용으로 쓰인 적이 없다: 루트 package.json은 first commit(e7bded2) 이후 0.0.0 그대로이고, upgrade/admin-v2/package.json은 workspace 생성 커밋(81b150f)에서 0.1.0으로 만들어진 뒤 한 번도 증가하지 않았다(둘 다 private: true, 레지스트리 배포 없음). 배포는 태그가 아니라 branch 기반이다 — .gitlab-ci.yml이 `$CI_COMMIT_BRANCH == &quot;main&quot;`에서 build 후 dist를 PVC로 cp 하며, upgrade/admin-v2/deploy/README.md도 git push -&gt; runner build -&gt; rsync 흐름만 기술한다. 릴리즈 자산을 만드는 스크립트도 없다(scripts/*는 runtime-config 검증·offline 검사·integrity manifest 뿐, Makefile·release*.sh 없음). docs/INDEX.md:168은 &quot;날짜와 version history는 실제 release 근거가 있을 때만 기록한다&quot;고 명시하는데 그 근거가 없다. 따라서 태그 형식·버전 증가 단위·릴리즈 노트 위치를 지금 새로 정하는 것은 사람의 결정이라 판단해 아무 커밋·태그·파일도 만들지 않았다. worktree는 clean 상태 그대로다.</dd>
</dl>

## 회차 이력

<div class="table-wrap"><table class="rt"><thead><tr><th class="primary">일시</th><th>결과</th></tr></thead><tbody><tr data-status="merged"><td data-label="일시" class="primary">2026-09-05 00:10</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/8">PR #8</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-04 05:30</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/7">PR #7</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 21:58</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/6">PR #6</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 13:35</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/5">PR #5</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 07:35</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/4">PR #4</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-03 01:47</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/3">PR #3</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-02 19:26</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/2">PR #2</a>, release skipped</td></tr><tr data-status="merged"><td data-label="일시" class="primary">2026-09-02 13:24</td><td data-label="결과"><span class="pill pill-merged">✅ 머지</span> merged <a href="https://github.com/hkjang/aiportal-front-admin/pull/1">PR #1</a>, release skipped</td></tr></tbody></table></div>

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


[← 대시보드](https://hkjang.github.io/aidev/)

{% endraw %}
