import assert from "node:assert/strict";
import ts from "../web/node_modules/typescript/lib/typescript.js";
import { readFile } from "node:fs/promises";
const code = ts.transpileModule(
  await readFile(new URL("../web/src/graph/model.ts", import.meta.url), "utf8"),
  {
    compilerOptions: {
      module: ts.ModuleKind.ES2022,
      target: ts.ScriptTarget.ES2022,
    },
  },
).outputText;
const { filterGraph } = await import(
  `data:text/javascript;base64,${Buffer.from(code).toString("base64")}`
);
const nodes = Array.from({ length: 8 }, (_, n) => ({
  id: String(n),
  title: `문서 ${n}`,
  tags: n === 0 ? ["시작"] : ["문서"],
  owner_id: n % 2 ? "a" : "b",
  space_id: n < 3 ? "s" : null,
  indexed: n !== 7,
}));
const data = {
  nodes,
  edges: nodes
    .slice(0, 5)
    .map((n, i) => ({
      source: n.id,
      target: String(i + 1),
      type: i === 0 ? "reference" : "related",
      origin: "manual",
    })),
  unresolved: [
    { source: "0", target: "존재하지 않는 문서", reason: "unresolved" },
  ],
  diagnostics: { limit: 2000, truncated: false, pending: 1 },
};
const base = {
  q: "",
  tag: "",
  space: "",
  owner: "",
  focus: "",
  depth: 2,
  type: "",
};
assert.equal(filterGraph(data, base).nodes.length, 8);
assert.deepEqual(
  filterGraph(data, base).isolated.map((n) => n.id),
  ["6"],
);
assert.deepEqual(
  filterGraph(data, { ...base, focus: "0", depth: 1 }).nodes.map((n) => n.id),
  ["0", "1"],
);
assert.equal(
  filterGraph(data, { ...base, focus: "0", depth: 5 }).nodes.length,
  6,
);
assert.equal(
  filterGraph(data, { ...base, focus: "0", depth: 999 }).nodes.length,
  6,
);
assert.deepEqual(
  filterGraph(data, {
    ...base,
    focus: "0",
    type: "reference",
    depth: 5,
  }).nodes.map((n) => n.id),
  ["0", "1"],
);
assert.deepEqual(
  filterGraph(data, { ...base, focus: "0", tag: "문서", space: "s" }).nodes.map(
    (n) => n.id,
  ),
  ["0", "1", "2"],
);
assert.equal(filterGraph(data, { ...base, q: "시작" }).unresolved.length, 1);
assert.equal(filterGraph(data, { ...base, owner: "a" }).unresolved.length, 0);
assert.equal(filterGraph(data, { ...base, focus: "unknown" }).nodes.length, 0);
assert.equal(filterGraph(data, { ...base, space: "none" }).nodes.length, 5);
const many = {
  ...data,
  nodes: Array.from({ length: 2000 }, (_, i) => ({
    ...nodes[0],
    id: String(i),
  })),
  edges: Array.from({ length: 1999 }, (_, i) => ({
    source: String(i),
    target: String(i + 1),
    type: "related",
  })),
};
assert.equal(
  filterGraph(many, { ...base, focus: "0", depth: 5 }).nodes.length,
  6,
);
console.log(
  "PASS graph ACL-input-only filters, typed edges, local depth 1–5, center preservation, pending exclusion, unresolved and bounded 2,000-node traversal",
);
