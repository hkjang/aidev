# 회차 노트 2026-09-25-214904-moina-improve — moina
정찰 → 구현 → 비평 → 수리 → 중재 → 릴리즈가 차례로 적는다. 다음 역할은 앞선 노트를 먼저 읽는다. [러너] 줄은 러너의 단계 판정이다.
- [러너 21:49] base pinned — main@3ba1767
- [러너 21:49] autonomy release — 
- [러너 21:54] scout done — CI `Image and browser smoke` 실패가 어느 화면 때문인지 알 수 없는 것을 고친다 — 실패 상세를 `$GITHUB_STEP_SUMMARY

## 구현 노트
- 무엇을 왜: image 잡 실패가 "exit code 1"만 남겨 원인을 못 찾던 것을, `scripts/e2e-failure-summary.mjs`와 `ci.yml`의 `if: failure()` "Summarize e2e failures" 단계로 고쳤다(ab36254). 세 e2e 결과 JSON에서 실패 항목만(화면 id·diff 비율·diff PNG 경로, Axe 규칙·route, smoke 마지막 route·실패 locator·스크립트 stack frame) 뽑아 job summary에 적는다. 진단 출력만 추가했고 임계값·재시도·continue-on-error·테스트 제외는 무변경.
- 검증: `make image` → 새 postgres:17-alpine → `--read-only` 앱 18080 → 공식 `playwright:v1.62.1-noble`에서 (a) 정상 52/52 통과 exit 0, (b) `MOINA_VISUAL_MAX_DIFF_RATIO=0` 강제 실패, (c) `MOINA_E2E_VERSION` 오지정 smoke 실패, (d) a11y 통과 — 전부 **실제 스크립트가 쓴 JSON**으로 요약을 돌려 `dark-desktop-admin-smtp | 0.345%`가 출력에 나오는 것을 확인. `make check` 통과(route 120).
- 확신 없는 곳: ① a11y `blocking` 표 분기는 **미검증**이다 — 시도한 `/admin/smtp`·`/admin/media`에 Serious/Critical 위반이 없어 실제 데이터로 못 채웠다(필드는 `accessibility-regression.mjs:371~373`에서 읽었다). ② smoke의 `failures.{console,page,request,response,external}` 분기도 실제 데이터 없음(error·routes 분기만 실증). ③ e2e 스크립트가 JSON 필드명을 바꾸면 요약이 조용히 비어도 CI는 통과한다 — 고정 테스트를 안 붙였다. ④ 이번 요약 단계가 실제 GitHub runner에서 도는 것은 미확인(푸시 권한 없음). 단 job summary 파일 append는 `Dependency audit`의 선례와 같고, node는 image 잡의 setup-node 이후라 존재한다.
- 일부러 안 한 것: 베이스라인 재승인(과제서 4번). 공식 noble 이미지에서 **52/52 통과**했고(최악 `dark-desktop-admin-smtp` 0.345%, 24개 화면이 0.15%↑) `VISUAL_REGRESSION.md:45`가 승인 renderer를 CI ubuntu-24.04로 못박으며 이 이미지도 0.000~0.35% 어긋나므로, 로컬 renderer로 24장을 재승인하면 릴리즈 게이트를 검증 불가능한 상태로 바꾼다. 측정값만 원장·ideas에 남겼다. fixture JSON을 손으로 만든 단위 테스트도 일부러 안 만들었다(운영자 지시: 손으로 만든 대역으로 증명 금지).
- 다음 역할 주의: 요약 스크립트 재현에는 앱+DB+docker가 필요하다. 같은 IP에서 로그인은 5분에 5회 제한이라 e2e를 연달아 돌리면 429로 엉뚱한 실패가 난다. 결과 JSON만 있으면 `node scripts/e2e-failure-summary.mjs [결과디렉터리]`로 스크립트만 따로 돌려볼 수 있다(항상 exit 0).
- [러너 22:08] brief accepted — 채택 — 과제서의 근거(`ci.yml`에 `$GITHUB_STEP_SUMMARY`가 `Dependency audit` 한 곳뿐, 세 e2e 스크립트가 `e2e/test-results` 아래 결과 JSO
- [러너 22:08] verify passed — 검증 7개 통과 (auto)
- [러너 22:09] pr created — https://github.com/hkjang/moina/pull/32
- [러너 22:09] guard held — .github/workflows/ci.yml 
