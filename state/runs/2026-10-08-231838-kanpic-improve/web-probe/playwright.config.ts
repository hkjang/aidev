import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  globalSetup: './e2e/global-setup.ts',
  // 시나리오가 만든 워크북을 실행이 끝난 뒤 정리한다. 쌓이면 홈 화면을
  // 다루는 시나리오가 느려지고 흔들린다.
  globalTeardown: './e2e/global-teardown.ts',
  timeout: 30_000,
  fullyParallel: false,
  // Several specs change global settings (AI gateway, SMTP relay) that the
  // whole server shares, so files run one at a time rather than in parallel
  // workers overwriting each other's configuration.
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL: process.env.KANPIC_E2E_BASE_URL || 'http://localhost:8080',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : undefined,
  },
})
