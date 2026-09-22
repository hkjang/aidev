import assert from "node:assert/strict";
import ts from "../web/node_modules/typescript/lib/typescript.js";
import { readFile } from "node:fs/promises";
const source = await readFile(
  new URL("../web/src/editor/saveSnapshot.ts", import.meta.url),
  "utf8",
);
const compiled = ts.transpileModule(source, {
  compilerOptions: {
    module: ts.ModuleKind.ES2022,
    target: ts.ScriptTarget.ES2022,
  },
}).outputText;
const { reconcileSavedDocument } = await import(
  `data:text/javascript;base64,${Buffer.from(compiled).toString("base64")}`
);
const submitted = {
  title: "사용자 연락처",
  markdown: "이메일: user@example.test",
  tags: "문의, 연락",
};
const canonical = {
  title: "사용자 연락처",
  markdown: "이메일: [이메일 마스킹]",
  tags: ["문의", "연락"],
};
let result = reconcileSavedDocument(submitted, { ...submitted }, canonical);
assert.equal(result.values.markdown, canonical.markdown);
assert.equal(result.dirty, false);
assert.equal(result.applyMetadata, true);
result = reconcileSavedDocument(
  submitted,
  { ...submitted, markdown: submitted.markdown + "\n저장 중 작성" },
  canonical,
);
assert.equal(result.values.markdown, submitted.markdown + "\n저장 중 작성");
assert.equal(result.dirty, true);
assert.equal(result.applyMetadata, false);
result = reconcileSavedDocument(
  submitted,
  { ...submitted, title: "새 제목", tags: "새 태그" },
  { ...canonical, title: "[마스킹된 제목]", tags: ["마스킹된 태그"] },
);
assert.equal(result.values.title, "새 제목");
assert.equal(result.values.tags, "새 태그");
assert.equal(result.values.markdown, canonical.markdown);
assert.equal(result.dirty, true);
result = reconcileSavedDocument(submitted, { ...submitted }, canonical, true);
assert.equal(result.values.markdown, submitted.markdown);
assert.equal(result.applyMetadata, false);
assert.equal(result.dirty, true);
result = reconcileSavedDocument(
  { ...submitted, tags: "문의, 연락," },
  { ...submitted, tags: "문의, 연락," },
  canonical,
);
assert.equal(result.values.tags, "문의, 연락");
assert.equal(result.dirty, false);
// A property-only PUT has not submitted the editor text. Its baseline must be
// the previous canonical document, not the current unsaved editing values.
result = reconcileSavedDocument(
  { title: canonical.title, markdown: canonical.markdown, tags: canonical.tags.join(", ") },
  { title: "편집 중 제목", markdown: canonical.markdown + "\n아직 저장하지 않은 문장", tags: "새 분류" },
  canonical,
);
assert.equal(result.values.title, "편집 중 제목");
assert.equal(result.values.markdown, canonical.markdown + "\n아직 저장하지 않은 문장");
assert.equal(result.values.tags, "새 분류");
assert.equal(result.dirty, true);
assert.equal(result.applyMetadata, false);
console.log(
  "PASS canonical masked save reconciliation, typed-during-save and property-only draft preservation, metadata gating, CRDT pending state and tag normalization",
);
