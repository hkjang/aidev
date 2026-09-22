// Collect attribution text from installed service dependencies, without a network request.
import { execFileSync } from "node:child_process";
import { readFile, readdir, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { fileURLToPath } from "node:url";
import path from "node:path";
const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const sha = (v) => createHash("sha256").update(v).digest("hex");
const inputs = {};
for (const file of ["go.mod", "go.sum", "web/package.json", "web/package-lock.json"])
  inputs[file] = sha(await readFile(path.join(root, file)));
if (process.argv.includes("--check")) {
  const manifest = JSON.parse(await readFile(path.join(root, "web/public/licenses-manifest.json"), "utf8"));
  if (JSON.stringify(inputs) !== JSON.stringify(manifest.inputs)) throw new Error("Dependency attribution is stale: run node scripts/licenses.mjs after installing dependencies.");
  if (sha(await readFile(path.join(root, "web/public/licenses.txt"))) !== manifest.text_sha256) throw new Error("Attribution text hash mismatch");
  console.log(JSON.stringify({ ok: true, components: manifest.components.length }));
  process.exit(0);
}
const lock = JSON.parse(await readFile(path.join(root, "web/package-lock.json"), "utf8"));
const components = [];
for (const [relative, item] of Object.entries(lock.packages)) {
  if (!relative || item.dev) continue;
  const dir = path.join(root, "web", relative);
  let metadata;
  try { metadata = JSON.parse(await readFile(path.join(dir, "package.json"), "utf8")); }
  catch (e) { if (item.optional) continue; throw e; }
  components.push({ ecosystem: "npm", name: metadata.name, version: metadata.version, license: metadata.license || item.license || "See attribution", dir });
}
const modules = execFileSync("go", ["list", "-deps", "-f", "{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}", "./cmd/madi"], { cwd: root, encoding: "utf8", maxBuffer: 20 * 1024 * 1024 }).split("\n");
const unique = new Set();
for (const line of modules) {
  const [name, version, dir] = line.split("|");
  if (!version || !dir || unique.has(name)) continue;
  unique.add(name);
  components.push({ ecosystem: "go", name, version, license: "See attribution", dir });
}
components.push({ ecosystem: "go", name: "Go standard library", version: execFileSync("go", ["env", "GOVERSION"], { encoding: "utf8" }).trim(), license: "BSD-3-Clause", dir: execFileSync("go", ["env", "GOROOT"], { encoding: "utf8" }).trim() });
components.sort((a, b) => `${a.ecosystem}/${a.name}/${a.version}`.localeCompare(`${b.ecosystem}/${b.name}/${b.version}`));
let text = "madi — Third-party attribution\n\nGenerated from installed dependencies used by the Go service and production web dependency tree. Build tools are not listed. This file preserves upstream license and notice text; it does not assign a license to madi itself.\n";
const missing = [], manifest = [];
for (const { dir, ...component } of components) {
  const names = (await readdir(dir)).filter((file) => /^(licen[sc]e|copying|copyright|notice)([._-].*)?$/i.test(file));
  if (component.name === "pdfjs-dist") {
    // These exact directories, including their notices, are emitted by the
    // offline Vite asset plugin. Preserve their notices in the central index too.
    for (const group of ["cmaps", "standard_fonts", "wasm"])
      for (const file of await readdir(path.join(dir, group)))
        if (/^(licen[sc]e|copying|copyright|notice)([._-].*)?$/i.test(file))
          names.push(`${group}/${file}`);
  }
  const notices = [];
  for (const file of names.sort()) {
    try {
      const content = await readFile(path.join(dir, file), "utf8");
      if (Buffer.byteLength(content) > 2 * 1024 * 1024) throw new Error("Unexpected attribution size");
      notices.push({ file, content });
    } catch (e) { if (e.code !== "EISDIR") throw e; }
  }
  if (!notices.length) {
    try {
      const readme = await readFile(path.join(dir, "README.md"), "utf8");
      const section = readme.match(/^##? Licen[cs]e\s*\n([\s\S]*?)(?=^##? |$(?![\s\S]))/im)?.[1];
      if (section && /Permission is hereby granted|Redistribution and use/.test(section))
        notices.push({ file: "README.md (license section)", content: section });
    } catch (e) { if (e.code !== "ENOENT") throw e; }
  }
  if (!notices.length) {
    // The npm platform binary packages omit LICENSE while the same-version
    // parent distributes it. Validate the exact platform/version relationship;
    // do not substitute an unrelated package's license based on a name prefix.
    if (component.name.startsWith("@napi-rs/canvas-")) {
      const parentDir = path.join(root, "web/node_modules/@napi-rs/canvas");
      const parent = JSON.parse(await readFile(path.join(parentDir, "package.json"), "utf8"));
      if (parent.version !== component.version || parent.optionalDependencies?.[component.name] !== component.version || parent.license !== component.license)
        throw new Error(`Platform attribution relationship changed: ${component.name}`);
      notices.push({ file: `@napi-rs/canvas@${parent.version}/LICENSE (same-version platform distribution)`, content: await readFile(path.join(parentDir, "LICENSE"), "utf8") });
    }
  }
  if (!notices.length) {
    const fallback = component.name === "Go standard library" ? "go-LICENSE.txt" : component.name === "react-remove-scroll-bar" ? "react-remove-scroll-bar-LICENSE.txt" : "";
    if (fallback) notices.push({ file: "upstream/" + fallback, content: await readFile(path.join(root, "third_party/licenses", fallback), "utf8") });
  }
  if (!notices.length) { missing.push(`${component.ecosystem}:${component.name}@${component.version}`); continue; }
  text += `\n${"=".repeat(76)}\n${component.ecosystem}: ${component.name} ${component.version}\nDeclared license: ${typeof component.license === "string" ? component.license : JSON.stringify(component.license)}\n`;
  for (const { file, content } of notices) text += `\n--- ${file} ---\n${content.trim()}\n`;
  manifest.push({ ...component, files: notices.map(({ file, content }) => ({ file, sha256: sha(content) })) });
}
if (missing.length) throw new Error("Missing upstream attribution text; inspect these installed packages: " + missing.join(", "));
await writeFile(path.join(root, "web/public/licenses.txt"), text);
await writeFile(path.join(root, "web/public/licenses-manifest.json"), JSON.stringify({ inputs, components: manifest, text_sha256: sha(text) }, null, 2) + "\n");
console.log(JSON.stringify({ components: manifest.length, bytes: Buffer.byteLength(text), missing: 0 }));
