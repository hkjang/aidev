import { defineConfig } from "@playwright/test";

/**
 * 화면 검증(E2E) 설정.
 *
 * 실행 중인 SeatOn을 그대로 브라우저로 열어 확인한다. 서버 기동은 이 설정이
 * 하지 않고 E2E_BASE_URL로 받는다. 개발자는 docker compose로, CI는 워크플로에서
 * 띄운 컨테이너를 가리키면 된다.
 */
export default defineConfig({
  testDir: "./e2e",
  globalSetup: "./e2e/global-setup.ts",
  timeout: 60_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: process.env.E2E_BASE_URL ?? "http://127.0.0.1:18781",
    viewport: { width: 1440, height: 900 },
    locale: "ko-KR",
    timezoneId: "Asia/Seoul",
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
});
