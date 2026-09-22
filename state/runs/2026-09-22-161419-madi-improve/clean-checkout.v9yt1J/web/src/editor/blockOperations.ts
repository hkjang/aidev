import type { JSONContent } from "@tiptap/core";
export type BlockEntry = {
  node: JSONContent;
  parent: JSONContent;
  index: number;
  ancestors: string[];
  depth: number;
};
export function blockEntries(doc: JSONContent): BlockEntry[] {
  const rows: BlockEntry[] = [];
  function visit(parent: JSONContent, ancestors: string[], depth: number) {
    (parent.content || []).forEach((node, index) => {
      if (node.type === "text") return;
      const id = node.attrs?.id;
      if (typeof id === "string")
        rows.push({ node, parent, index, ancestors, depth });
      visit(
        node,
        typeof id === "string" ? [...ancestors, id] : ancestors,
        depth + 1,
      );
    });
  }
  visit(doc, [], 0);
  return rows;
}
export function rootSelection(doc: JSONContent, ids: string[]) {
  const selected = new Set(ids);
  return blockEntries(doc).filter(
    (row) =>
      selected.has(row.node.attrs?.id) &&
      !row.ancestors.some((id) => selected.has(id)),
  );
}
const filler = new Set([
  "doc",
  "blockquote",
  "listItem",
  "taskItem",
  "detailsContent",
  "column",
  "tableCell",
  "tableHeader",
]);
function repair(node: JSONContent, uuid: () => string) {
  if (node.content) {
    node.content = node.content.filter(
      (child) =>
        !(
          ["bulletList", "orderedList", "taskList"].includes(
            child.type || "",
          ) && !child.content?.length
        ),
    );
    for (const child of node.content) repair(child, uuid);
    if (!node.content.length && filler.has(node.type || ""))
      node.content = [{ type: "paragraph", attrs: { id: uuid() } }];
  }
}
function fresh(node: JSONContent, uuid: () => string): JSONContent {
  return {
    ...node,
    ...(node.attrs
      ? { attrs: { ...node.attrs, ...(node.attrs.id ? { id: uuid() } : {}) } }
      : {}),
    ...(node.content
      ? { content: node.content.map((child) => fresh(child, uuid)) }
      : {}),
  };
}
export function organizeBlocks(
  source: JSONContent,
  ids: string[],
  operation: string,
  targetID = "",
  placement = "before",
  uuid: () => string = () => crypto.randomUUID(),
): JSONContent {
  const doc = structuredClone(source),
    rows = rootSelection(doc, ids);
  if (!rows.length) throw new Error("이동할 블록을 선택하세요.");
  if (rows.length > 500 || blockEntries(doc).length > 10000)
    throw new Error(
      "한 번에 500개, 문서당 10,000개 이하의 블록을 정리할 수 있습니다.",
    );
  const sameParent = rows.every((row) => row.parent === rows[0].parent),
    parent = rows[0].parent;
  if (operation === "delete")
    for (const row of [...rows].reverse())
      row.parent.content!.splice(row.index, 1);
  else if (operation === "duplicate")
    for (const row of [...rows].reverse())
      row.parent.content!.splice(row.index + 1, 0, fresh(row.node, uuid));
  else if (operation === "up" || operation === "down") {
    if (!sameParent)
      throw new Error("위·아래 이동은 같은 계층의 블록끼리 선택하세요.");
    const selected = new Set(rows.map((row) => row.node.attrs?.id)),
      content = parent.content!;
    if (operation === "up") {
      for (let index = 1; index < content.length; index++)
        if (
          selected.has(content[index].attrs?.id) &&
          !selected.has(content[index - 1].attrs?.id)
        )
          [content[index - 1], content[index]] = [
            content[index],
            content[index - 1],
          ];
    } else
      for (let index = content.length - 2; index >= 0; index--)
        if (
          selected.has(content[index].attrs?.id) &&
          !selected.has(content[index + 1].attrs?.id)
        )
          [content[index + 1], content[index]] = [
            content[index],
            content[index + 1],
          ];
  } else if (operation === "nest") {
    if (
      !sameParent ||
      rows.some((row, index) => row.index !== rows[0].index + index)
    )
      throw new Error("같은 계층에서 연속한 블록을 선택해 중첩하세요.");
    parent.content!.splice(rows[0].index, rows.length, {
      type: "blockquote",
      attrs: { id: uuid() },
      content: rows.map((row) => row.node),
    });
  } else if (operation === "outdent") {
    if (!sameParent || parent.type === "doc")
      throw new Error("같은 하위 계층의 블록을 선택하세요.");
    const container = blockEntries(doc).find((row) => row.node === parent);
    if (!container)
      throw new Error("이 블록은 상위 계층으로 이동할 수 없습니다.");
    for (const row of [...rows].reverse()) parent.content!.splice(row.index, 1);
    container.parent.content!.splice(
      container.index + 1,
      0,
      ...rows.map((row) => row.node),
    );
  } else if (operation === "move") {
    const selected = new Set(rows.map((row) => row.node.attrs?.id)),
      target = blockEntries(doc).find((row) => row.node.attrs?.id === targetID);
    if (
      !target ||
      selected.has(targetID) ||
      target.ancestors.some((id) => selected.has(id))
    )
      throw new Error("블록을 자기 자신이나 자기 하위에 놓을 수 없습니다.");
    for (const row of [...rows].reverse())
      row.parent.content!.splice(row.index, 1);
    const current = blockEntries(doc).find(
      (row) => row.node.attrs?.id === targetID,
    )!;
    if (placement === "inside") {
      if (!current.node.content || !filler.has(current.node.type || ""))
        throw new Error(
          "중첩 놓기는 인용·목록 항목·열·접기 본문·표 셀에서 사용할 수 있습니다.",
        );
      current.node.content.push(...rows.map((row) => row.node));
    } else
      current.parent.content!.splice(
        current.index + (placement === "after" ? 1 : 0),
        0,
        ...rows.map((row) => row.node),
      );
  } else throw new Error("지원하지 않는 블록 명령입니다.");
  repair(doc, uuid);
  return doc;
}
