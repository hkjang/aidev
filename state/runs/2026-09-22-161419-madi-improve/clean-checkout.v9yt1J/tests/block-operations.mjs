import assert from "node:assert/strict";
import {
  blockEntries,
  organizeBlocks,
  rootSelection,
} from "../web/src/editor/blockOperations.ts";
const p = (id, text) => ({
  type: "paragraph",
  attrs: { id },
  content: [{ type: "text", text }],
});
const original = {
  type: "doc",
  content: [
    p("a", "첫째"),
    p("b", "둘째"),
    {
      type: "blockquote",
      attrs: { id: "quote" },
      content: [p("c", "셋째"), p("d", "넷째")],
    },
  ],
};
let index = 0;
const uuid = () => `fresh-${index++}`;
const nested = organizeBlocks(original, ["a", "b"], "nest", "", "before", uuid);
assert.equal(nested.content[0].type, "blockquote");
assert.deepEqual(
  nested.content[0].content.map((n) => n.attrs.id),
  ["a", "b"],
);
assert.deepEqual(
  original.content.map((n) => n.attrs.id),
  ["a", "b", "quote"],
);
assert.deepEqual(
  rootSelection(original, ["quote", "c"]).map((r) => r.node.attrs.id),
  ["quote"],
);
const duplicate = organizeBlocks(
  original,
  ["quote"],
  "duplicate",
  "",
  "before",
  uuid,
);
assert.equal(duplicate.content.length, 4);
const duplicateIDs = blockEntries(duplicate).map((r) => r.node.attrs.id);
assert.equal(new Set(duplicateIDs).size, duplicateIDs.length);
assert.deepEqual(
  duplicate.content[3].content.map((n) => n.content[0].text),
  ["셋째", "넷째"],
);
const inside = organizeBlocks(
  original,
  ["a", "b"],
  "move",
  "quote",
  "inside",
  uuid,
);
assert.deepEqual(
  inside.content[0].content.map((n) => n.attrs.id),
  ["c", "d", "a", "b"],
);
const out = organizeBlocks(original, ["c", "d"], "outdent", "", "before", uuid);
assert.deepEqual(
  out.content.map((n) => n.attrs.id),
  ["a", "b", "quote", "c", "d"],
);
assert.equal(out.content[2].content[0].type, "paragraph");
const before = organizeBlocks(
  original,
  ["c", "d"],
  "move",
  "a",
  "before",
  uuid,
);
assert.deepEqual(
  before.content.slice(0, 4).map((n) => n.attrs.id),
  ["c", "d", "a", "b"],
);
assert.throws(
  () => organizeBlocks(original, ["quote"], "move", "c", "inside", uuid),
  /자기/,
);
assert.throws(
  () => organizeBlocks(original, ["a", "c"], "nest", "", "before", uuid),
  /같은 계층/,
);
assert.throws(
  () => organizeBlocks(original, ["a", "c"], "up", "", "before", uuid),
  /같은 계층/,
);
assert.equal(
  organizeBlocks(original, ["a", "b", "quote"], "delete", "", "before", uuid)
    .content[0].type,
  "paragraph",
);
const multi = organizeBlocks(
  original,
  ["b", "quote"],
  "up",
  "",
  "before",
  uuid,
);
assert.deepEqual(
  multi.content.map((n) => n.attrs.id),
  ["b", "quote", "a"],
);
console.log(
  "PASS nested block grouping, cross-level move, cycle rejection, multi-select ordering, fresh cloned IDs, original immutability and valid empty-container placeholders",
);
