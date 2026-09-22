import { cp, mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
const root = path.dirname(fileURLToPath(import.meta.url));
const target = process.argv[2] || "chrome";
if (!["chrome", "firefox"].includes(target))
  throw new Error("chrome 또는 firefox를 지정하세요.");
const out = path.join(root, "dist", target);
await mkdir(out, { recursive: true });
for (const name of [
  "background.js",
  "common.js",
  "popup.html",
  "popup.js",
  "options.html",
  "options.js",
  "style.css",
  "icon-48.png",
  "icon-128.png",
])
  await cp(path.join(root, name), path.join(out, name));
await cp(
  path.join(root, "../../web/public/icon-192.png"),
  path.join(out, "icon-192.png"),
);
const manifest = JSON.parse(
  await readFile(path.join(root, "manifest.json"), "utf8"),
);
if (target === "firefox") {
  delete manifest.minimum_chrome_version;
  manifest.background = { scripts: ["background.js"], type: "module" };
  manifest.browser_specific_settings = {
    gecko: {
      id: "madi-clipper@madi.local",
      strict_min_version: "142.0",
      data_collection_permissions: {
        required: ["authenticationInfo", "websiteContent", "browsingActivity"],
      },
    },
  };
}
await writeFile(
  path.join(out, "manifest.json"),
  JSON.stringify(manifest, null, 2) + "\n",
);
console.log(`클리퍼 빌드 완료: ${out}`);
