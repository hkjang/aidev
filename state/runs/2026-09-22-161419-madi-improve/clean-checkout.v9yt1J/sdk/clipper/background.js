import {
  serverOrigin,
  sourceURL,
  captureMarkdown,
  collectPage,
} from "./common.js";
const api = globalThis.chrome || globalThis.browser;
let inflight = false;
async function settings() {
  const local = await api.storage.local.get([
    "server",
    "workspace_id",
    "allowHTTP",
  ]);
  const session = await api.storage.session.get(["token"]);
  if (!local.server || !session.token)
    throw new Error(
      "먼저 연결 설정에서 서버 주소와 개인 API 키를 입력하세요. API 키는 브라우저를 닫으면 지워집니다.",
    );
  return { ...local, ...session };
}
async function call(config, path, { method = "GET", body } = {}) {
  const server = serverOrigin(config.server, config.allowHTTP);
  if (!(await api.permissions.contains({ origins: [server + "/*"] })))
    throw new Error(
      "madi 서버 연결 권한이 없습니다. 연결 설정에서 다시 허용하세요.",
    );
  const headers = {
    Authorization: "Bearer " + config.token,
    "X-Madi-Request": "1",
  };
  if (body && !(body instanceof FormData))
    headers["Content-Type"] = "application/json";
  const response = await fetch(server + "/api/v1" + path, {
    method,
    headers,
    body:
      body instanceof FormData ? body : body ? JSON.stringify(body) : undefined,
    credentials: "omit",
    redirect: "error",
    referrerPolicy: "no-referrer",
    signal: AbortSignal.timeout(45000),
  });
  const result = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new Error(
      result.error || `서버 요청에 실패했습니다 (${response.status})`,
    );
  return result;
}
async function handle(message) {
  switch (message.action) {
    case "status": {
      const local = await api.storage.local.get([
        "server",
        "workspace_id",
        "allowHTTP",
      ]);
      const session = await api.storage.session.get(["token", "draft"]);
      return {
        ...local,
        connected: !!session.token,
        draft: session.draft || null,
      };
    }
    case "configure": {
      const server = serverOrigin(message.server, message.allowHTTP);
      if (
        typeof message.token !== "string" ||
        !message.token.startsWith("madi_") ||
        message.token.length > 300
      )
        throw new Error("유효한 madi 개인 API 키를 입력하세요.");
      if (!(await api.permissions.contains({ origins: [server + "/*"] })))
        throw new Error("서버 접근 권한을 허용하세요.");
      const old = await api.storage.local.get("server");
      await api.storage.session.remove(["token", "draft"]);
      await api.storage.local.set({
        server,
        allowHTTP: !!message.allowHTTP,
        workspace_id: "",
      });
      await api.storage.session.set({ token: message.token });
      if (old.server && old.server !== server)
        await api.permissions.remove({ origins: [old.server + "/*"] });
      return { workspaces: await call(await settings(), "/workspaces") };
    }
    case "workspace": {
      if (!/^[0-9a-f-]{36}$/i.test(message.id))
        throw new Error("워크스페이스를 선택하세요.");
      const config = await settings();
      const list = await call(config, "/workspaces");
      if (!list.some((workspace) => workspace.id === message.id))
        throw new Error("접근 가능한 워크스페이스가 아닙니다.");
      await api.storage.local.set({ workspace_id: message.id });
      return { ok: true };
    }
    case "workspaces":
      return { workspaces: await call(await settings(), "/workspaces") };
    case "disconnect": {
      const old = await api.storage.local.get("server");
      await api.storage.session.clear();
      await api.storage.local.clear();
      if (old.server)
        await api.permissions.remove({ origins: [old.server + "/*"] });
      return { ok: true };
    }
    case "capture": {
      if (
        !["article", "selection", "link", "fullpage", "screenshot"].includes(
          message.mode,
        )
      )
        throw new Error("지원하지 않는 저장 방식입니다.");
      const [tab] = await api.tabs.query({ active: true, currentWindow: true });
      if (!tab?.id || !sourceURL(tab.url))
        throw new Error(
          "HTTP 또는 HTTPS 웹페이지에서 실행하세요. 브라우저 설정 페이지는 저장할 수 없습니다.",
        );
      const result = await api.scripting.executeScript({
        target: { tabId: tab.id },
        func: collectPage,
        args: [message.mode],
      });
      const capture = result[0]?.result;
      if (!capture) throw new Error("이 페이지 내용을 읽을 수 없습니다.");
      if (message.mode === "selection" && !capture.selected)
        throw new Error("먼저 웹페이지에서 저장할 텍스트를 선택하세요.");
      let image = "";
      if (message.mode === "screenshot") {
        image = await api.tabs.captureVisibleTab(tab.windowId, {
          format: "png",
        });
        if (image.length > 12_000_000)
          throw new Error(
            "스크린샷이 너무 큽니다. 브라우저 창을 줄여 다시 시도하세요.",
          );
      }
      const draft = {
        title: capture.title,
        text: captureMarkdown(capture),
        url: sourceURL(capture.url),
        image,
        mode: message.mode,
        client_request_id: crypto.randomUUID(),
        attachment_request_id: crypto.randomUUID(),
      };
      await api.storage.session.set({ draft });
      return { draft };
    }
    case "save": {
      if (inflight)
        throw new Error("기록을 저장하고 있습니다. 잠시 기다리세요.");
      inflight = true;
      try {
        const config = await settings();
        if (!config.workspace_id)
          throw new Error("연결 설정에서 워크스페이스를 선택하세요.");
        const session = await api.storage.session.get("draft");
        let draft = session.draft;
        if (!draft) throw new Error("먼저 저장할 내용을 가져오세요.");
        const title = String(message.title || "").trim();
        const text = String(message.text || "");
        if (!title || title.length > 250 || text.length > 900000)
          throw new Error("제목 또는 본문 길이를 확인하세요.");
        if (draft.document_id && (title !== draft.title || text !== draft.text))
          throw new Error(
            "문서는 이미 저장되었으므로 첨부 재시도 중에는 내용을 바꿀 수 없습니다.",
          );
        if (title !== draft.title || text !== draft.text) {
          draft = {
            ...draft,
            title,
            text,
            client_request_id: crypto.randomUUID(),
          };
          await api.storage.session.set({ draft });
        }
        const doc = await call(config, "/captures", {
          method: "POST",
          body: {
            workspace_id: config.workspace_id,
            title: draft.title,
            text: draft.text,
            url: draft.url,
            client_request_id: draft.client_request_id,
          },
        });
        draft = { ...draft, document_id: doc.id };
        await api.storage.session.set({ draft });
        if (draft.image) {
          const blob = await (await fetch(draft.image)).blob();
          const form = new FormData();
          form.append("file", blob, "web-clip.png");
          await call(
            config,
            `/attachments?document_id=${encodeURIComponent(doc.id)}&client_request_id=${encodeURIComponent(draft.attachment_request_id)}`,
            { method: "POST", body: form },
          );
        }
        await api.storage.session.remove("draft");
        return { url: config.server + "/app/documents/" + doc.id };
      } finally {
        inflight = false;
      }
    }
    default:
      throw new Error("알 수 없는 요청입니다.");
  }
}
api.runtime.onMessage.addListener((message, sender, reply) => {
  if (
    sender.id !== api.runtime.id ||
    !sender.url?.startsWith(api.runtime.getURL(""))
  )
    return false;
  if (JSON.stringify(message).length > 1_000_000) {
    reply({ error: "요청 내용이 너무 큽니다." });
    return false;
  }
  handle(message)
    .then((result) => reply({ result }))
    .catch((error) =>
      reply({ error: error.message || "요청을 처리하지 못했습니다." }),
    );
  return true;
});
