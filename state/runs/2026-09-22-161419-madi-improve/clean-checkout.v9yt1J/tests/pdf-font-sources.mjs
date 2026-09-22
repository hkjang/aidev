import assert from "node:assert/strict";
import { test } from "node:test";
import { mkdtemp, mkdir, writeFile, rm, readFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { bundlePDFFontSources } from "../scripts/bundle-pdf-font-sources.mjs";

test("PDF font source collection fails closed before fetching changed artifacts", async () => {
  const root = await mkdtemp(path.join(tmpdir(), "madi-font-provenance-test-"));
  try {
    const pdf = path.join(root, "pdf"), output = path.join(root, "out");
    await mkdir(path.join(pdf, "standard_fonts"), { recursive: true });
    let fetched = 0;
    const get = async () => { fetched++; return Buffer.from("not a source"); };
    await writeFile(path.join(pdf, "package.json"), JSON.stringify({ name: "pdfjs-dist", version: "unexpected" }));
    await assert.rejects(bundlePDFFontSources(pdf, output, get), /version changed/);
    await writeFile(path.join(pdf, "package.json"), JSON.stringify({ name: "pdfjs-dist", version: "6.3.289" }));
    await writeFile(path.join(pdf, "standard_fonts/LiberationSans-Regular.ttf"), "changed font");
    await assert.rejects(bundlePDFFontSources(pdf, output, get), /font changed/);
    assert.equal(fetched, 0);
  } finally {
    // Exact test-owned mkdtemp target, never a shared cache or workspace.
    await rm(root, { recursive: true });
  }
});

test("PDF source archive hash is checked before any tar interpretation", async () => {
  const root = await mkdtemp(path.join(tmpdir(), "madi-font-archive-test-"));
  try {
    await assert.rejects(bundlePDFFontSources(new URL("../web/node_modules/pdfjs-dist", import.meta.url).pathname, root,
      async () => Buffer.from("corrupted or swapped source archive")), /source SHA-256 mismatch/);
    const fonts = await readFile(path.join(root, "pdfjs-liberation/LiberationSans-Regular.ttf"));
    assert.ok(fonts.length > 100000);
  } finally {
    await rm(root, { recursive: true });
  }
});
