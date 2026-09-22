// Trusted declarative renderer. Installed plugin JavaScript runs only in Worker.
(() => {
  "use strict";
  const nonce = location.hash.slice(1);
  if (!/^[a-f0-9]{32}$/.test(nonce) || parent === window) return;
  let worker = null,
    context = null,
    lastPong = Date.now(),
    burstStart = Date.now(),
    burstCount = 0;
  const send = (payload) =>
    parent.postMessage({ channel: "madi-plugin-v1", nonce, ...payload }, "*");
  const relay = (payload) =>
    worker?.postMessage({ channel: "madi-plugin-v1", nonce, ...payload });
  const stop = (error) => {
    worker?.terminate();
    worker = null;
    if (error) send({ type: "plugin-error", error });
  };
  setInterval(() => {
    if (!worker) return;
    if (Date.now() - lastPong > 4000) {
      stop("플러그인이 4초 이상 응답하지 않아 실행을 중단했습니다");
      return;
    }
    relay({ type: "ping" });
  }, 1000);
  function render(tree) {
    let count = 0;
    const build = (node, depth = 0) => {
      if (++count > 500 || depth > 20 || !node || typeof node !== "object")
        throw Error("화면 요소 한도를 초과했습니다");
      const tags = {
        stack: "div",
        row: "div",
        text: "p",
        heading: "h2",
        button: "button",
        input: "input",
        textarea: "textarea",
        select: "select",
        checkbox: "input",
        code: "pre",
        image: "img",
        divider: "hr",
        badge: "span",
      };
      const tag = tags[node.type];
      if (!tag) throw Error("지원하지 않는 화면 요소입니다");
      const element = document.createElement(tag);
      element.className = "madi-plugin-" + node.type;
      if (node.type === "row")
        Object.assign(element.style, {
          display: "flex",
          gap: "12px",
          flexWrap: "wrap",
        });
      if (node.type === "stack")
        Object.assign(element.style, {
          display: "flex",
          flexDirection: "column",
          gap: "12px",
        });
      if (node.type === "image") {
        const asset = context?.assets?.[node.asset];
        if (!asset || !/^image\/(png|jpeg|gif|webp)$/.test(asset.mime))
          throw Error("허용된 이미지 에셋이 아닙니다");
        element.src = "data:" + asset.mime + ";base64," + asset.data;
        element.alt = String(node.label || "").slice(0, 500);
      } else if (node.type === "select") {
        for (const option of (Array.isArray(node.options)
          ? node.options
          : []
        ).slice(0, 100)) {
          const item = document.createElement("option");
          item.value = String(option.value ?? option).slice(0, 500);
          item.textContent = String(option.label ?? option).slice(0, 500);
          element.append(item);
        }
      } else if (!["input", "textarea", "checkbox"].includes(node.type))
        element.textContent = String(node.text || "").slice(0, 100000);
      if (node.type === "checkbox") {
        element.type = "checkbox";
        element.checked = !!node.value;
      } else if (["input", "textarea", "select"].includes(node.type)) {
        element.value = String(node.value || "").slice(0, 100000);
        if (node.type === "input") element.type = "text";
      }
      if (node.label)
        element.setAttribute("aria-label", String(node.label).slice(0, 500));
      if (node.placeholder)
        element.setAttribute(
          "placeholder",
          String(node.placeholder).slice(0, 500),
        );
      if (node.disabled) element.disabled = true;
      for (const [key, event] of [
        ["onClick", "click"],
        ["onChange", "change"],
      ])
        if (typeof node[key] === "string")
          element.addEventListener(event, () =>
            relay({
              type: "ui-event",
              id: node[key],
              value: node.type === "checkbox" ? element.checked : element.value,
            }),
          );
      if (Array.isArray(node.children) && ["stack", "row"].includes(node.type))
        for (const child of node.children)
          element.append(build(child, depth + 1));
      return element;
    };
    document.getElementById("root").replaceChildren(build(tree));
  }
  addEventListener("message", (event) => {
    const data = event.data;
    if (
      event.source !== parent ||
      !data ||
      data.channel !== "madi-plugin-v1" ||
      data.nonce !== nonce
    )
      return;
    if (data.type === "probe" && !worker) {
      send({ type: "ready" });
      return;
    }
    if (data.type === "init" && !worker) {
      if (
        typeof data.workerSource !== "string" ||
        data.workerSource.length > 5 * 1024 * 1024
      )
        return;
      context = data.context;
      if (
        typeof data.style === "string" &&
        data.style.length < 6 * 1024 * 1024
      ) {
        const link = document.createElement("link");
        link.rel = "stylesheet";
        link.href = "data:text/css;base64," + data.style;
        document.head.append(link);
      }
      const blob = URL.createObjectURL(
        new Blob([data.workerSource], { type: "text/javascript" }),
      );
      worker = new Worker(blob);
      URL.revokeObjectURL(blob);
      lastPong = Date.now();
      worker.onmessage = (event) => {
        const message = event.data;
        if (
          !worker ||
          !message ||
          message.channel !== "madi-plugin-v1" ||
          message.nonce !== nonce
        )
          return;
        if (Date.now() - burstStart > 1000) {
          burstStart = Date.now();
          burstCount = 0;
        }
        if (++burstCount > 100) {
          stop("플러그인 메시지가 초당 100회를 초과하여 실행을 중단했습니다");
          return;
        }
        if (message.type === "pong") {
          lastPong = Date.now();
          return;
        }
        if (message.type === "ready") {
          relay({ type: "init", context });
          return;
        }
        if (message.type === "ui-render") {
          try {
            render(message.tree);
          } catch (error) {
            send({ type: "plugin-error", error: error.message });
          }
          return;
        }
        if (
          [
            "registered",
            "request",
            "cancel",
            "invocation-result",
            "plugin-error",
          ].includes(message.type)
        )
          send(message);
      };
      worker.onerror = () => stop("플러그인 실행 오류가 발생했습니다");
      return;
    }
    if (["response", "chunk", "invoke"].includes(data.type)) relay(data);
  });
  addEventListener("pagehide", () => worker?.terminate());
  send({ type: "ready" });
})();
