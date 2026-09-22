import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
const allowed = {
  "/": "recorder.html",
  "/recorder.html": "recorder.html",
  "/recorder.mjs": "recorder.mjs",
  "/model.mjs": "model.mjs",
  "/recorder.css": "recorder.css",
};
const server = createServer(async (req, res) => {
  const file = allowed[req.url];
  if (req.method !== "GET" || !file) {
    res.writeHead(404);
    res.end();
    return;
  }
  try {
    const data = await readFile(fileURLToPath(new URL(file, import.meta.url)));
    res.writeHead(200, {
      "Content-Type": file.endsWith(".html")
        ? "text/html; charset=utf-8"
        : file.endsWith(".css")
          ? "text/css; charset=utf-8"
          : "text/javascript; charset=utf-8",
      "Cache-Control": "no-store",
      "X-Content-Type-Options": "nosniff",
      "Referrer-Policy": "no-referrer",
    });
    res.end(data);
  } catch {
    res.writeHead(500);
    res.end();
  }
});
server.listen(0, "127.0.0.1", () =>
  console.log(
    `로컬 관찰 기록기: http://127.0.0.1:${server.address().port}/ (외부 수신 없음, Ctrl+C 종료)`,
  ),
);
