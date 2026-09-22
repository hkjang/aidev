import assert from "node:assert/strict";
import { chromium, expect } from "playwright/test";
import { mkdir } from "node:fs/promises";
import { documentPanel, documentTool } from "./document-ui.mjs";
const base = process.env.MADI_BASE_URL;
const raceCase = process.env.MADI_DOC_RACE_CASE;
assert.ok(base, "isolated Go fixture URL is required");
const browser = await chromium.launch();
const owner = await browser.newContext({
  viewport: { width: 1440, height: 1050 },
});
const page = await owner.newPage(),
  errors = [];
const releaseResponses = [];
let diagnosticPage = page;
page.on("pageerror", (error) => errors.push(error.message));
if (process.env.MADI_DOC_SAVE_TRACE) {
  await page.addInitScript(() => {
    window.__saveHTTPTrace = [];
    const json = Response.prototype.json;
    Response.prototype.json = async function (...args) {
      const path = new URL(this.url, location.origin).pathname;
      const traced =
        /^\/api\/v1\/(?:documents(?:\/[a-f0-9-]+)?|workspaces)$/.test(path);
      if (traced)
        window.__saveHTTPTrace.push({
          phase: "json-start",
          path,
          time: Date.now(),
        });
      const value = await json.apply(this, args);
      if (traced)
        window.__saveHTTPTrace.push({
          phase: "json-done",
          path,
          time: Date.now(),
          version: value?.version,
        });
      return value;
    };
  });
  for (const name of [
    "request",
    "response",
    "requestfinished",
    "requestfailed",
  ])
    page.on(name, (value) => {
      const request = name === "response" ? value.request() : value;
      const path = new URL(request.url()).pathname;
      if (/^\/api\/v1\/(?:documents(?:\/[a-f0-9-]+)?|workspaces)$/.test(path))
        console.log(
          "SAVE_HTTP",
          JSON.stringify({
            time: Date.now(),
            name,
            path,
            method: request.method(),
          }),
        );
    });
}
async function api(client, url, method = "GET", data) {
  const response = await client.request.fetch(base + "/api/v1" + url, {
    method,
    data,
    headers: { "X-Madi-Request": "1" },
  });
  assert.ok(
    response.ok(),
    `${method} ${url}: ${response.status()} ${await response.text()}`,
  );
  return response.json();
}
function gate() {
  let release;
  const promise = new Promise((resolve) => {
    release = resolve;
  });
  releaseResponses.push(release);
  return { promise, release };
}
try {
  await api(owner, "/auth/login", "POST", {
    email: "admin@example.test",
    password: "Integration-Test-Password-2026!",
  });
  const ws = await api(owner, "/workspaces", "POST", {
    name: "문서 저장과 미리보기 경합",
  });
  await owner.addInitScript(
    (id) => localStorage.setItem("madi.workspace", id),
    ws.id,
  );
  if (!raceCase || raceCase === "save") {
    const doc = await api(owner, "/documents", "POST", {
      workspace_id: ws.id,
      title: "저장 대기 전 제목",
      markdown: "# 공동 편집 원문\n\n원문을 보존합니다.\n",
      visibility: "private",
    });
    let holdingACK = false;
    const acknowledgements = [];
    await page.routeWebSocket(
      (url) => url.pathname.endsWith("/collaboration"),
      (socket) => {
        const server = socket.connectToServer();
        server.onMessage((raw) => {
          const message = JSON.parse(String(raw));
          if (holdingACK && message.type === "ack")
            acknowledgements.push({ message, release: () => socket.send(raw) });
          else socket.send(raw);
        });
      },
    );
    await page.goto(`${base}/app/documents/${doc.id}?mode=edit`);
    console.log(
      "save served bundle",
      await page.locator('script[type="module"][src]').getAttribute("src"),
    );
    const editor = page.locator('.tiptap-content[contenteditable="true"]');
    await editor.waitFor();
    await page.locator('[data-save-state="confirmed"]').waitFor();
    for (const kind of ["title", "tags"]) {
      await page.locator('[data-save-state="confirmed"]').waitFor();
      const before = await api(owner, `/documents/${doc.id}`);
      const marker =
        kind === "title" ? "제목 대기 중 본문" : "태그 대기 중 본문";
      holdingACK = true;
      await editor.click();
      await page.keyboard.press("Control+End");
      await page.keyboard.insertText(marker);
      await expect.poll(() => acknowledgements.length).toBeGreaterThan(0);
      assert.ok(
        acknowledgements.some(({ message }) =>
          message.markdown.includes(marker),
        ),
        "held ACK must confirm the body edit under test",
      );
      await page.getByRole("button", { name: "저장", exact: true }).click();
      await expect(page.locator(".save-state")).toContainText("저장 중");
      const metadataGate = gate();
      let metadataWrites = 0;
      const requestPath = `${base}/api/v1/documents/${doc.id}`;
      const holdMetadata = async (route) => {
        if (route.request().method() !== "PUT") return route.continue();
        const response = await route.fetch();
        assert.equal(response.status(), 200, await response.text());
        metadataWrites++;
        await metadataGate.promise;
        await route.fulfill({ response });
      };
      await page.route(requestPath, holdMetadata);
      if (kind === "title")
        await page
          .getByLabel("문서 제목", { exact: true })
          .fill("저장 확인 중에 입력한 제목");
      else {
        await page
          .getByRole("combobox", { name: "문서 태그", exact: true })
          .fill("대기중추가태그");
        await page
          .getByRole("combobox", { name: "문서 태그", exact: true })
          .press("Enter");
      }
      // A later autosave must not conceal an earlier false 'saved' paint. The
      // server ACK remains held while the editable metadata is changed.
      assert.notEqual(
        await page.locator("[data-save-state]").getAttribute("data-save-state"),
        "confirmed",
      );
      assert.match(await page.locator(".save-state").innerText(), /저장 중/);
      await page.evaluate(() => {
        window.__saveRace = { states: [], falseSaved: false };
        const state = document.querySelector(".save-state");
        window.__saveRaceObserver = new MutationObserver(() => {
          const value = state.textContent.trim();
          window.__saveRace.states.push(value);
          if (value === "저장됨") window.__saveRace.falseSaved = true;
        });
        window.__saveRaceObserver.observe(state, {
          childList: true,
          subtree: true,
          characterData: true,
        });
      });
      holdingACK = false;
      acknowledgements.splice(0).forEach(({ release }) => release());
      await expect
        .poll(() => metadataWrites, {
          timeout: 7000,
          message: `${kind}: input made during CRDT ACK must receive its own canonical save`,
        })
        .toBeGreaterThan(0);
      await expect(page.locator(".save-state")).toContainText("저장 중");
      const saveTrace = await page.evaluate(() => window.__saveRace);
      console.log("save-state trace", kind, JSON.stringify(saveTrace));
      assert.equal(
        saveTrace.falseSaved,
        false,
        "metadata must never be shown as saved before its canonical response",
      );
      const durable = await api(owner, `/documents/${doc.id}`);
      assert.ok(
        durable.markdown.includes(marker),
        "CRDT body must remain canonical",
      );
      if (kind === "title")
        assert.equal(durable.title, "저장 확인 중에 입력한 제목");
      else assert.ok(durable.tags.includes("대기중추가태그"));
      assert.ok(durable.version > before.version);
      const response = page.waitForResponse(
        (r) => r.url() === requestPath && r.request().method() === "PUT",
      );
      assert.equal(
        await page.evaluate(() => window.__saveRace.falseSaved),
        false,
      );
      await page.evaluate(() => window.__saveRaceObserver.disconnect());
      metadataGate.release();
      if (process.env.MADI_DOC_SAVE_TRACE)
        console.log("SAVE_GATE_RELEASE", kind, Date.now());
      await response;
      if (process.env.MADI_DOC_SAVE_TRACE)
        console.log("SAVE_HEADERS", kind, Date.now());
      await page.unroute(requestPath, holdMetadata);
      await expect(page.locator(".save-state")).toHaveText("저장됨");
      if (process.env.MADI_DOC_SAVE_TRACE)
        console.log(
          "SAVE_PHASE_JSON",
          kind,
          JSON.stringify(await page.evaluate(() => window.__saveHTTPTrace)),
        );
    }
    await page.reload();
    await expect(page.getByLabel("문서 제목", { exact: true })).toHaveValue(
      "저장 확인 중에 입력한 제목",
    );
    await page
      .getByRole("button", { name: "대기중추가태그 태그 삭제", exact: true })
      .waitFor();
    console.log(
      "PASS actual CRDT ACK barrier preserves later title/tags, explicit server save state, body and reload",
    );
  }
  if (!raceCase || raceCase === "refresh") {
    const doc = await api(owner, "/documents", "POST", {
      workspace_id: ws.id,
      title: "목록 갱신과 분리된 저장",
      markdown: "# 정본 저장 확인\n",
      visibility: "private",
    });
    const path = `${base}/api/v1/documents/${doc.id}`;
    const workspacePath = `${base}/api/v1/workspaces`;
    await page.goto(`${base}/app/documents/${doc.id}?mode=source`);
    await expect(page.locator(".document-main")).toHaveAttribute(
      "data-document-id",
      doc.id,
    );
    const firstRefresh = gate(),
      firstRefreshDone = gate();
    let refreshStarted = false;
    const holdRefresh = async (route) => {
      if (refreshStarted) return route.continue();
      const response = await route.fetch();
      assert.equal(response.status(), 200);
      refreshStarted = true;
      await firstRefresh.promise;
      await route.fulfill({ response });
      firstRefreshDone.release();
    };
    await page.route(workspacePath, holdRefresh);
    const firstSave = page.waitForResponse(
      (r) => r.url() === path && r.request().method() === "PUT",
    );
    await page
      .getByLabel("문서 제목", { exact: true })
      .fill("정본 저장 완료 후 목록 대기");
    await page.getByRole("button", { name: "저장", exact: true }).click();
    assert.equal(
      (await (await firstSave).json()).title,
      "정본 저장 완료 후 목록 대기",
    );
    await expect.poll(() => refreshStarted).toBe(true);
    await expect(page.locator(".save-state")).toHaveText("저장됨");
    // The old refresh may finish while a subsequent canonical PUT is pending.
    // It must neither keep save disabled nor release the newer save's state.
    const secondReply = gate();
    let secondCommitted = false;
    const holdSecond = async (route) => {
      if (route.request().method() !== "PUT") return route.continue();
      const response = await route.fetch();
      assert.equal(response.status(), 200);
      secondCommitted = true;
      await secondReply.promise;
      await route.fulfill({ response });
    };
    await page.route(path, holdSecond);
    const secondSave = page.waitForResponse(
      (r) => r.url() === path && r.request().method() === "PUT",
    );
    await page
      .getByLabel("문서 제목", { exact: true })
      .fill("이전 목록 갱신 중 새 정본 저장");
    await expect(
      page.getByRole("button", { name: "저장", exact: true }),
    ).toBeEnabled();
    await page.getByRole("button", { name: "저장", exact: true }).click();
    await expect.poll(() => secondCommitted).toBe(true);
    await expect(page.locator(".save-state")).toContainText("저장 중");
    const oldList = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname === "/api/v1/documents" &&
        r.request().method() === "GET",
    );
    firstRefresh.release();
    await firstRefreshDone.promise;
    await (await oldList).finished();
    await page.evaluate(
      () =>
        new Promise((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(resolve)),
        ),
    );
    await expect(page.locator(".save-state")).toContainText("저장 중");
    await expect(
      page.getByRole("button", { name: "저장", exact: true }),
    ).toBeDisabled();
    const nextList = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname === "/api/v1/documents" &&
        r.request().method() === "GET",
    );
    secondReply.release();
    assert.equal(
      (await (await secondSave).json()).title,
      "이전 목록 갱신 중 새 정본 저장",
    );
    await expect(page.locator(".save-state")).toHaveText("저장됨");
    await (await nextList).finished();
    await page.unroute(path, holdSecond);
    await page.unroute(workspacePath, holdRefresh);
    // A failed background refresh is a separate warning, never a failed save.
    const rejectedRefresh = gate();
    let rejectionPending = false;
    const rejectRefresh = async (route) => {
      const response = await route.fetch();
      assert.equal(response.status(), 200);
      rejectionPending = true;
      await rejectedRefresh.promise;
      await route.fulfill({
        status: 503,
        json: { error: "목록 갱신 시험 오류" },
      });
    };
    await page.route(workspacePath, rejectRefresh);
    const thirdSave = page.waitForResponse(
      (r) => r.url() === path && r.request().method() === "PUT",
    );
    await page
      .getByLabel("문서 제목", { exact: true })
      .fill("목록 오류에도 보존된 정본");
    await page.getByRole("button", { name: "저장", exact: true }).click();
    assert.equal(
      (await (await thirdSave).json()).title,
      "목록 오류에도 보존된 정본",
    );
    await expect.poll(() => rejectionPending).toBe(true);
    await expect(page.locator(".save-state")).toHaveText("저장됨");
    rejectedRefresh.release();
    await expect(
      page.getByRole("status").filter({
        hasText: "문서 목록을 새로 읽지 못했습니다.",
      }),
    ).toBeVisible();
    await expect(page.locator(".save-state")).toHaveText("저장됨");
    await expect(page.locator(".document-page .recovery-notice")).toHaveCount(
      0,
    );
    assert.equal(
      (await api(owner, `/documents/${doc.id}`)).title,
      "목록 오류에도 보존된 정본",
    );
    await page.unroute(workspacePath, rejectRefresh);
    await page.getByRole("button", { name: "알림 닫기", exact: true }).click();
    const lateRefresh = gate(),
      lateDone = gate();
    let latePending = false;
    const rejectAfterNavigation = async (route) => {
      const response = await route.fetch();
      assert.equal(response.status(), 200);
      latePending = true;
      await lateRefresh.promise;
      await route.fulfill({
        status: 503,
        json: { error: "이전 문서의 목록 갱신 시험 오류" },
      });
      lateDone.release();
    };
    await page.route(workspacePath, rejectAfterNavigation);
    const fourthSave = page.waitForResponse(
      (r) => r.url() === path && r.request().method() === "PUT",
    );
    await page
      .getByLabel("문서 제목", { exact: true })
      .fill("이동 전에 저장된 정본");
    await page.getByRole("button", { name: "저장", exact: true }).click();
    assert.equal(
      (await (await fourthSave).json()).title,
      "이동 전에 저장된 정본",
    );
    await expect.poll(() => latePending).toBe(true);
    await expect(page.locator(".save-state")).toHaveText("저장됨");
    await page
      .locator(".document-actionbar")
      .getByRole("link", { name: "문서", exact: true })
      .click();
    await page.waitForURL("**/app/documents");
    await expect(page.locator(".document-main")).toHaveCount(0);
    lateRefresh.release();
    await lateDone.promise;
    await page.evaluate(
      () =>
        new Promise((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(resolve)),
        ),
    );
    await expect(
      page.getByRole("status").filter({
        hasText: "문서 목록을 새로 읽지 못했습니다.",
      }),
    ).toHaveCount(0);
    await page.unroute(workspacePath, rejectAfterNavigation);
    console.log(
      "PASS canonical save is independent of held/failed list refresh; old refresh cannot unlock a newer save or warn after navigation",
    );
  }
  if (!raceCase || raceCase === "preview") {
    const account = {
      email: "preview-race-reader@example.test",
      password: "Preview-Race-Password-2026!",
      name: "미리보기 검토자",
      role: "editor",
    };
    await api(owner, "/admin/users", "POST", account);
    await api(owner, `/workspaces/${ws.id}/members`, "PUT", {
      email: account.email,
      role: "editor",
    });
    const doc = await api(owner, "/documents", "POST", {
      workspace_id: ws.id,
      title: "권한을 재확인할 자료",
      markdown: "# 현재 접근 가능한 자료\n\n- [ ] 회수 후 재표시 금지 원문\n",
      visibility: "workspace",
    });
    const reader = await browser.newContext({
      viewport: { width: 1440, height: 1050 },
    });
    await api(reader, "/auth/login", "POST", {
      email: account.email,
      password: account.password,
    });
    await reader.addInitScript(
      (id) => localStorage.setItem("madi.workspace", id),
      ws.id,
    );
    const previewPage = await reader.newPage();
    diagnosticPage = previewPage;
    await previewPage.addInitScript(() => {
      const original = Response.prototype.json;
      window.__readBodies = [];
      Response.prototype.json = function (...args) {
        return original.apply(this, args).then((value) => {
          window.__readBodies.push({ url: this.url, id: value?.id });
          return value;
        });
      };
    });
    previewPage.on("pageerror", (error) => errors.push(error.message));
    await previewPage.goto(`${base}/app/tasks?document_id=${doc.id}`);
    console.log(
      "preview served bundle",
      await previewPage
        .locator('script[type="module"][src]')
        .getAttribute("src"),
    );
    const trigger = previewPage.getByRole("button", {
      name: `${doc.title} 문서 미리보기`,
      exact: true,
    });
    await trigger.waitFor();
    const bodyGate = gate();
    let bodyReady = false;
    const documentURL = `${base}/api/v1/documents/${doc.id}`;
    await previewPage.route(documentURL, async (route) => {
      const response = await route.fetch();
      assert.equal(response.status(), 200);
      assert.ok(
        (await response.json()).markdown.includes("회수 후 재표시 금지 원문"),
        "held response contains the real previously authorized body",
      );
      bodyReady = true;
      await bodyGate.promise;
      await route.fulfill({ response });
    });
    await trigger.click();
    await expect.poll(() => bodyReady).toBe(true);
    await api(owner, `/documents/${doc.id}`, "PUT", {
      version: doc.version,
      visibility: "private",
    });
    const dialog = previewPage.getByRole("dialog", {
      name: "문서 미리보기",
      exact: true,
    });
    await expect(dialog).toContainText("현재 문서를 미리 볼 수 없습니다.", {
      timeout: 7000,
    });
    await dialog.evaluate((element, title) => {
      window.__previewRace = { violations: [] };
      const record = () => {
        const text = element.textContent;
        if (
          text.includes(title) ||
          text.includes("회수 후 재표시 금지 원문") ||
          text.includes("문서 전체 열기")
        )
          window.__previewRace.violations.push(text);
      };
      window.__previewRaceObserver = new MutationObserver(record);
      window.__previewRaceObserver.observe(element, {
        childList: true,
        subtree: true,
        characterData: true,
      });
      record();
    }, doc.title);
    const bodyResponse = previewPage.waitForResponse(
      (r) => r.url() === documentURL,
    );
    bodyGate.release();
    await (await bodyResponse).finished();
    await expect
      .poll(() =>
        previewPage.evaluate(
          (id) => window.__readBodies.some((body) => body.id === id),
          doc.id,
        ),
      )
      .toBe(true);
    // Two rendering opportunities after the held HTTP body completed. No timer
    // guesses, forced clicks or fabricated source/ACL responses are involved.
    await previewPage.evaluate(
      () =>
        new Promise((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(resolve)),
        ),
    );
    const previewTrace = await previewPage.evaluate(() => window.__previewRace);
    console.log("preview mutation trace", JSON.stringify(previewTrace));
    assert.deepEqual(
      previewTrace.violations,
      [],
      "denied document must not be re-exposed, even until the next ACL poll",
    );
    assert.ok(
      (await dialog.innerText()).includes("현재 문서를 미리 볼 수 없습니다."),
    );
    await expect(dialog).not.toContainText("회수 후 재표시 금지 원문");
    await expect(
      dialog.getByRole("link", { name: "문서 전체 열기", exact: true }),
    ).toHaveCount(0);
    await previewPage.keyboard.press("Escape");
    await expect(trigger).toBeFocused();
    await reader.close();
    console.log(
      "PASS observed ACL denial cannot be undone by delayed document body; close restores preview focus",
    );
  }
  if (!raceCase || raceCase === "focus") {
    diagnosticPage = page;
    const doc = await api(owner, "/documents", "POST", {
      workspace_id: ws.id,
      title: "문서 패널 키보드 복귀",
      markdown: "# 키보드로 문서 도구 사용\n",
      visibility: "private",
    });
    await page.goto(`${base}/app/documents/${doc.id}?mode=read`);
    console.log(
      "focus served bundle",
      await page.locator('script[type="module"][src]').getAttribute("src"),
    );
    await documentPanel(page, "AI");
    const closeAI = page.getByRole("button", {
      name: "AI 도우미 닫기",
      exact: true,
    });
    await closeAI.waitFor();
    // Traverse actual tab stops from the selected AI tab; do not force DOM
    // focus onto the disappearing close control or its expected destination.
    for (let i = 0; i < 8; i++) {
      await page.keyboard.press("Tab");
      if (
        await closeAI.evaluate((element) => element === document.activeElement)
      )
        break;
    }
    await expect(closeAI).toBeFocused();
    await page.keyboard.press("Enter");
    const backlinks = page.getByRole("tab", { name: "연결", exact: true });
    await expect(backlinks).toHaveAttribute("aria-selected", "true");
    await expect(backlinks).toBeFocused();
    console.log(
      "PASS keyboard closing embedded AI restores focus to its selected document panel tab",
    );
  }
  if (!raceCase || raceCase === "navigation") {
    diagnosticPage = page;
    const previous = await api(owner, "/documents", "POST", {
      workspace_id: ws.id,
      title: "이동 전 문서 명령",
      markdown: "# 이전 문서\n",
      visibility: "private",
    });
    const target = await api(owner, "/documents", "POST", {
      workspace_id: ws.id,
      title: "이동 후 문서 명령",
      markdown: "# 목적지 문서\n",
      visibility: "private",
    });
    await page.goto(`${base}/app/documents/${previous.id}?mode=read`);
    await expect(page.locator(".document-main")).toHaveAttribute(
      "data-document-id",
      previous.id,
    );
    await page.keyboard.press("Control+p");
    const input = page.getByRole("combobox", { name: "문서 또는 명령 검색" });
    await input.fill(target.title);
    await expect(page.getByRole("option", { selected: true })).toContainText(
      target.title,
    );
    // Fault-inject the existing command at the real router's address/commit
    // boundary. Navigation still comes from Enter on the actual palette option;
    // neither document content nor the route is fabricated by this observer.
    await page.evaluate(
      ({ id }) => {
        const push = history.pushState;
        history.pushState = function (...args) {
          const result = push.apply(this, args);
          if (location.pathname !== `/app/documents/${id}`) return result;
          history.pushState = push;
          window.__navigationCommand = {
            before: location.pathname,
            document: document
              .querySelector("[data-document-id]")
              ?.getAttribute("data-document-id"),
          };
          window.dispatchEvent(
            new CustomEvent("madi-document-command", {
              detail: { action: "focus" },
            }),
          );
          window.__navigationCommand.after = location.pathname;
          return result;
        };
      },
      { id: target.id },
    );
    await input.press("Enter");
    const boundary = await page.evaluate(() => window.__navigationCommand);
    assert.equal(
      boundary?.document,
      previous.id,
      "test must hit the outgoing document listener boundary",
    );
    assert.equal(boundary.before, `/app/documents/${target.id}`);
    assert.equal(
      boundary.after,
      boundary.before,
      "old document command cannot navigate back to its own route",
    );
    await expect(page.locator(".document-main")).toHaveAttribute(
      "data-document-id",
      target.id,
    );
    await expect(page.getByLabel("문서 제목", { exact: true })).toHaveValue(
      target.title,
    );
    await page.keyboard.press("Control+k");
    await input.fill("집중 모드 전환");
    await expect(page.getByRole("option", { selected: true })).toContainText(
      "집중 모드 전환",
    );
    await input.press("Enter");
    await expect(page.locator(".document-page")).toHaveClass(/focus-mode/);
    assert.equal(new URL(page.url()).pathname, `/app/documents/${target.id}`);
    console.log(
      "PASS command at actual navigation boundary cannot act on previous document; ready document command remains usable",
    );
    // React Router already accepts a trailing slash. The actual-address guard
    // must preserve that public URL form for save and document tool commands.
    await page.goto(`${base}/app/documents/${target.id}/?mode=source`);
    await expect(page.locator(".document-main")).toHaveAttribute(
      "data-document-id",
      target.id,
    );
    const editedTitle = "끝 슬래시 주소에서 저장한 문서";
    await page.getByLabel("문서 제목", { exact: true }).fill(editedTitle);
    await page.keyboard.press("Control+Enter");
    await expect
      .poll(async () => (await api(owner, `/documents/${target.id}`)).title)
      .toBe(editedTitle);
    await documentTool(page, "프레젠테이션 보기");
    await expect(
      page.getByRole("dialog", { name: editedTitle, exact: true }),
    ).toBeVisible();
    assert.equal(
      new URL(page.url()).pathname.replace(/\/+$/, ""),
      `/app/documents/${target.id}`,
    );
    await page.getByRole("button", { name: "발표 종료", exact: true }).click();
    console.log(
      "PASS trailing-slash document URL preserves real save and presentation commands",
    );
  }
  assert.deepEqual(errors, []);
} catch (error) {
  await mkdir(new URL("./../test-results/document-races/", import.meta.url), {
    recursive: true,
  });
  await diagnosticPage
    .screenshot({
      path: new URL(
        `./../test-results/document-races/failure-${process.env.MADI_DOC_RACE_CASE || "all"}.png`,
        import.meta.url,
      ).pathname,
    })
    .catch(() => {});
  throw error;
} finally {
  for (const release of releaseResponses) release();
  if (process.env.MADI_DOC_SAVE_TRACE)
    console.log(
      "SAVE_JSON",
      JSON.stringify(
        await page.evaluate(() => window.__saveHTTPTrace).catch(() => []),
      ),
    );
  await browser.close();
}
