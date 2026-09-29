import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    proxy: { "/api": "http://localhost:8080", "/mcp": "http://localhost:8080" },
  },
  // 화면 검증(e2e)은 Playwright가 돌린다. vitest가 같은 파일을 집어 들지 않게 한다.
  test: { include: ["src/**/*.test.ts"] },
  build: { sourcemap: false, target: "es2022", chunkSizeWarningLimit: 750 },
});
