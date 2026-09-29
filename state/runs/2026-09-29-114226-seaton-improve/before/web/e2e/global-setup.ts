import { seed } from "./seed.mjs";

/** 검증 대상 서버에 시드 데이터가 없으면 채운다. */
export default async function globalSetup() {
  const result = await seed({
    baseURL: process.env.E2E_BASE_URL ?? "http://127.0.0.1:18781",
    username: process.env.E2E_USERNAME ?? "admin",
    password: process.env.E2E_PASSWORD ?? "e2e-verify-pass-123",
  });
  console.log("[e2e] 시드", JSON.stringify(result));
}
