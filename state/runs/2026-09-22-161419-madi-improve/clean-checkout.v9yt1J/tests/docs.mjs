import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, readdir, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { chromium } from "playwright";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../docs");
const mime = { ".html": "text/html; charset=utf-8", ".css": "text/css", ".js": "application/javascript", ".json": "application/json", ".png": "image/png", ".svg": "image/svg+xml", ".woff2": "font/woff2", ".md": "text/plain; charset=utf-8", ".xml": "application/xml" };
const server = createServer(async (req, res) => {
  try {
    const url = new URL(req.url, "http://localhost");
    if (!url.pathname.startsWith("/madi/")) { res.writeHead(404).end(); return; }
    const relative = decodeURIComponent(url.pathname.slice(6)) || "index.html";
    const file = path.resolve(root, relative);
    if (!file.startsWith(root + path.sep)) { res.writeHead(403).end(); return; }
    const body = await readFile(file);
    res.writeHead(200, { "Content-Type": mime[path.extname(file)] || "application/octet-stream" }).end(body);
  } catch { res.writeHead(404).end(); }
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const base = `http://127.0.0.1:${server.address().port}/madi/`;
const browser = await chromium.launch();
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, locale: "ko-KR", reducedMotion: "reduce" });
const page = await context.newPage(), errors = [], external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("response", (response) => { if (response.status() >= 400) errors.push(`${response.status()} ${response.url()}`); });
await context.route("**/*", (route) => {
  if (new URL(route.request().url()).origin === new URL(base).origin) return route.continue();
  external.push(route.request().url()); return route.abort();
});
const pages = ["index.html", "guide.html", "api.html", "screenshots.html", "manuals.html", ...(await readdir(path.join(root, "manuals"))).filter((f) => f.endsWith(".html")).map((f) => "manuals/" + f)];
const broken = [], seen = new Set();
try {
  for (const file of pages) {
    await page.goto(base + file, { waitUntil: "networkidle" });
    assert.equal(await page.locator("h1").count(), 1, `${file}: single h1`);
    const metadata = await page.evaluate(() => ({ language: document.documentElement.lang, title: document.title, description: document.querySelector('meta[name="description"]')?.content, canonical: document.querySelector('link[rel="canonical"]')?.href, schema: [...document.querySelectorAll('script[type="application/ld+json"]')].map((script) => JSON.parse(script.textContent)), links: [...document.querySelectorAll("a[href],img[src],link[href],script[src]")].map((node) => node.getAttribute("href") || node.getAttribute("src")) }));
    assert.equal(metadata.language, "ko"); assert.ok(metadata.title.includes("madi")); assert.ok(metadata.description?.length > 20); assert.ok(metadata.canonical.startsWith("https://hkjang.github.io/madi/")); assert.ok(metadata.schema.length);
    for (const link of metadata.links) {
      const url = new URL(link, base + file);
      if (url.origin !== new URL(base).origin || seen.has(url.pathname)) continue;
      seen.add(url.pathname);
      const resolved = path.resolve(root, decodeURIComponent(url.pathname.slice(6)) || "index.html");
      try { assert.ok(resolved.startsWith(root + path.sep)); await stat(resolved); } catch { broken.push(`${file} → ${link}`); }
    }
    await page.setViewportSize({ width: 390, height: 844 });
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false, `${file}: mobile overflow`);
    await page.setViewportSize({ width: 1440, height: 1000 });
  }
  assert.deepEqual(broken, [], "broken local documentation links");
  await page.goto(base + "screenshots.html");
  await page.getByLabel("화면 이름", { exact: true }).fill("AI");
  await page.getByLabel("분류", { exact: true }).selectOption("mobile");
  const count = await page.locator("[data-gallery-card]:visible").count();
  assert.ok(count > 0);
  await page.reload();
  assert.equal(await page.getByLabel("화면 이름", { exact: true }).inputValue(), "AI");
  assert.equal(await page.getByLabel("분류", { exact: true }).inputValue(), "mobile");
  await page.locator("[data-gallery-card]:visible button").first().click();
  const dialog = page.getByRole("dialog", { name: "서비스 화면 확대" });
  await dialog.waitFor();
  await page.waitForFunction(() => document.querySelector(".lightbox img")?.naturalWidth > 0);
  await page.keyboard.press("Escape");
  await page.getByLabel("화면 이름", { exact: true }).fill("절대로일치하지않는화면");
  await page.getByRole("status").filter({ hasText: "조건에 맞는 화면이 없습니다" }).waitFor();
  for (const file of ["index.html", "guide.html", "manuals.html", "screenshots.html"]) {
    await page.goto(base + file, { waitUntil: "networkidle" });
    await page.screenshot({ path: path.join(root, "screenshots", `product-${file.replace(".html", "")}-desktop.png`), fullPage: file !== "screenshots.html" });
    await page.setViewportSize({ width: 390, height: 844 });
    await page.getByRole("button", { name: "탐색 메뉴 열기" }).click();
    assert.equal(await page.getByRole("button", { name: "탐색 메뉴 닫기" }).getAttribute("aria-expanded"), "true");
    await page.getByRole("button", { name: "탐색 메뉴 닫기" }).click();
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1), false);
    await page.screenshot({ path: path.join(root, "screenshots", `product-${file.replace(".html", "")}-mobile.png`), fullPage: file !== "screenshots.html" });
    await page.setViewportSize({ width: 1440, height: 1000 });
  }
  assert.deepEqual(errors, []); assert.deepEqual(external, []);
  console.log(JSON.stringify({ ok: true, pages: pages.length, localLinks: seen.size, mobile: 390, structuredData: true, galleryFilterRefresh: true, externalAssets: external.length }));
} finally { await browser.close(); await new Promise((resolve) => server.close(resolve)); }
