existing summary
## e2e 실패 요약

### 시각 회귀 (`npm run test:visual`)
허용치 0.000%를 넘은 화면 26개 / 비교 52개 (차이 비율 내림차순)

| 화면 id | 차이 비율 | diff 이미지 |
| --- | --- | --- |
| `dark-desktop-admin-smtp` | 0.345% | `e2e/test-results/visual/dark-desktop-admin-smtp-diff.png` |
| `light-mobile-login` | 0.238% | `e2e/test-results/visual/light-mobile-login-diff.png` |
| `dark-mobile-login` | 0.237% | `e2e/test-results/visual/dark-mobile-login-diff.png` |
| `dark-desktop-explore` | 0.221% | `e2e/test-results/visual/dark-desktop-explore-diff.png` |
| `dark-desktop-search` | 0.221% | `e2e/test-results/visual/dark-desktop-search-diff.png` |
| `dark-desktop-notifications` | 0.221% | `e2e/test-results/visual/dark-desktop-notifications-diff.png` |
| `dark-desktop-moims` | 0.221% | `e2e/test-results/visual/dark-desktop-moims-diff.png` |
| `dark-desktop-ai` | 0.221% | `e2e/test-results/visual/dark-desktop-ai-diff.png` |
| `dark-desktop-settings-accessibility` | 0.221% | `e2e/test-results/visual/dark-desktop-settings-accessibility-diff.png` |
| `dark-desktop-admin-dashboard` | 0.221% | `e2e/test-results/visual/dark-desktop-admin-dashboard-diff.png` |
| `light-desktop-explore` | 0.220% | `e2e/test-results/visual/light-desktop-explore-diff.png` |
| `light-desktop-search` | 0.220% | `e2e/test-results/visual/light-desktop-search-diff.png` |
| `light-desktop-notifications` | 0.220% | `e2e/test-results/visual/light-desktop-notifications-diff.png` |
| `light-desktop-moims` | 0.220% | `e2e/test-results/visual/light-desktop-moims-diff.png` |
| `light-desktop-ai` | 0.220% | `e2e/test-results/visual/light-desktop-ai-diff.png` |
| `light-desktop-settings-accessibility` | 0.220% | `e2e/test-results/visual/light-desktop-settings-accessibility-diff.png` |
| `light-desktop-admin-dashboard` | 0.220% | `e2e/test-results/visual/light-desktop-admin-dashboard-diff.png` |
| `light-desktop-admin-smtp` | 0.220% | `e2e/test-results/visual/light-desktop-admin-smtp-diff.png` |
| `dark-desktop-flow` | 0.202% | `e2e/test-results/visual/dark-desktop-flow-diff.png` |
| `dark-desktop-settings-profile` | 0.202% | `e2e/test-results/visual/dark-desktop-settings-profile-diff.png` |
- …외 6건 (전체는 `moina-ci-diagnostics` artifact 참조)
- 오류: `AssertionError [ERR_ASSERTION]: 시각 회귀 26건 실패`
  - `- light-desktop-login: 차이 비율 0.059%가 허용치 0.000%를 초과했습니다.`
  - `- light-desktop-flow: 차이 비율 0.202%가 허용치 0.000%를 초과했습니다.`
  - `at file:///work/e2e/visual-regression.mjs:211:10`

### 브라우저 smoke (`npm run test:smoke`)
통과한 route 0개

- 오류: `locator.waitFor: Timeout 30000ms exceeded.`
  - `Call log:`
  - `- waiting for getByText(/moina\s+v0\.0\.0-invalid|MOINA\s+v0\.0\.0-invalid/i).first() to be visible`
  - `at /work/e2e/browser-smoke.mjs:112:154`

