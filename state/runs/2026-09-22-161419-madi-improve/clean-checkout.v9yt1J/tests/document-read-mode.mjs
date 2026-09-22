import assert from "node:assert/strict";
import ts from "../web/node_modules/typescript/lib/typescript.js";
import { readFile } from "node:fs/promises";
import { chromium } from "playwright";
const code = ts.transpileModule(
  await readFile(new URL("../web/src/editor/mode.ts", import.meta.url), "utf8"),
  {
    compilerOptions: {
      module: ts.ModuleKind.ES2022,
      target: ts.ScriptTarget.ES2022,
    },
  },
).outputText;
const { resolveEditorMode } = await import(
  `data:text/javascript;base64,${Buffer.from(code).toString("base64")}`
);
assert.equal(resolveEditorMode("read", "edit"), "preview");
assert.equal(resolveEditorMode("invalid", "edit"), "preview");
assert.equal(resolveEditorMode(null, "source"), "source");
assert.equal(resolveEditorMode(null, "edit", 4), "source");
assert.equal(resolveEditorMode("preview", "edit", 4), "preview");
assert.equal(resolveEditorMode("edit", "preview"), "edit");
if (process.env.MADI_MODE_UNIT_ONLY) {
  console.log(
    "PASS safe read alias/invalid mode and preference/source-line defaults",
  );
  process.exit(0);
}
const base = process.env.MADI_BASE_URL || "http://127.0.0.1:8080";
const browser = await chromium.launch(),
  context = await browser.newContext(),
  page = await context.newPage();
const mutations = [],
  sockets = [],
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("websocket", (socket) => sockets.push(socket.url()));
page.on("request", (request) => {
  if (request.method() !== "GET" && /\/documents\//.test(request.url()))
    mutations.push({ method: request.method(), url: request.url() });
});
async function api(path, method = "GET", data) {
  const response = await context.request.fetch(base + "/api/v1" + path, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(response.ok(), await response.text());
  return response.json();
}
try {
  await api("/auth/login", "POST", {
    email: "admin@example.test",
    password: "Browser-Test-Password-2026!",
  });
  const ws = await api("/workspaces", "POST", { name: "원문 보존 읽기 검증" });
  const markdown =
    "---\ntitle: 읽기 전용 검증\n---\n\n# 저장된 원문\n\n마지막 개행을 포함해 그대로 유지합니다.\n\n";
  const doc = await api("/documents", "POST", {
    workspace_id: ws.id,
    title: "읽기만으로 변경하지 않기",
    markdown,
  });
  await context.addInitScript(
    (wid) => localStorage.setItem("madi.workspace", wid),
    ws.id,
  );
  for (const mode of ["read", "preview", "invalid", "source"]) {
    await page.goto(base + `/app/documents/${doc.id}?mode=${mode}`);
    await page.getByLabel("문서 제목", { exact: true }).waitFor();
    if (mode === "source")
      await page.getByLabel("Markdown 원문 편집", { exact: true }).waitFor();
    else await page.locator(".editor-area .markdown-content").waitFor();
    assert.equal(await page.locator(".collaboration-bar").count(), 0);
    await page.waitForTimeout(1900);
    const current = await api(`/documents/${doc.id}`);
    assert.equal(current.version, doc.version, `${mode} changed version`);
    assert.equal(
      current.markdown,
      markdown,
      `${mode} changed canonical Markdown`,
    );
  }
  assert.deepEqual(sockets, []);
  assert.deepEqual(mutations, []);
  assert.deepEqual(errors, []);
  console.log(
    "PASS read/preview/invalid/source opening has no CRDT socket, document writes, version changes or Markdown byte changes",
  );
} finally {
  await browser.close();
}
