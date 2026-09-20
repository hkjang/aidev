import { defineConfig, devices } from "@playwright/test";

const previewPort = Number(process.env.APPSTORE_PREVIEW_PORT ?? 4173);

export default defineConfig({
  testDir: "./e2e",
  outputDir: "./test-results",
  snapshotDir: "./e2e/__screenshots__",
  fullyParallel: true,
  // The default worker count saturates a single preview server on a developer
  // machine and turns the suite flaky; CI keeps its own tuning and retries.
  workers: process.env.CI ? undefined : 4,
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [["html", { open: "never" }], ["github"]] : "list",
  use: {
    baseURL: `http://127.0.0.1:${previewPort}`,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    locale: "ko-KR",
    colorScheme: "dark",
  },
  projects: [
    {
      name: "desktop",
      use: {
        ...devices["Desktop Chrome"],
        viewport: { width: 1440, height: 1000 },
      },
    },
    { name: "mobile", use: { ...devices["Pixel 7"] } },
  ],
  webServer: {
    command: `npm run preview -- --port ${previewPort}`,
    port: previewPort,
    // Another project's preview server on the default port would otherwise be
    // reused and served in place of this app.
    reuseExistingServer: false,
  },
});
