import { Node, mergeAttributes, createBlockMarkdownSpec, parseAttributes, type JSONContent } from "@tiptap/core";
import { Details, DetailsSummary, DetailsContent } from "@tiptap/extension-details";

const escape = (value: unknown) => String(value ?? "").replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;").replaceAll('"', "&quot;");
const sourceAttr = (name: string, initial = "") => ({ default: initial, parseHTML: (element: HTMLElement) => element.getAttribute(`data-${name}`) || initial, renderHTML: (attrs: Record<string, any>) => ({ [`data-${name}`]: attrs[name] }) });
const encodeAttribute = (value: unknown) => String(value ?? "").replaceAll("&", "&amp;").replaceAll('"', "&quot;").replaceAll("}", "&#125;").replaceAll("\n", "&#10;").replaceAll("\r", "&#13;");
const decodeAttribute = (value: unknown) => String(value ?? "").replaceAll("&#13;", "\r").replaceAll("&#10;", "\n").replaceAll("&#125;", "}").replaceAll("&quot;", '"').replaceAll("&amp;", "&");

export const InlineMath = Node.create({
  name: "inlineMath", inline: true, group: "inline", atom: true,
  addAttributes: () => ({ latex: sourceAttr("latex", "x^2") }),
  parseHTML: () => [{ tag: "span[data-madi-math]" }],
  renderHTML: ({ HTMLAttributes }) => ["span", mergeAttributes(HTMLAttributes, { "data-madi-math": "inline" }), HTMLAttributes["data-latex"] || ""],
  markdownTokenizer: { name: "inlineMath", level: "inline", start: (source) => source.indexOf("$"), tokenize: (source) => { const match = /^\$(?!\$)((?:\\.|[^$\n])+?)\$(?!\$)/.exec(source); return match ? { type: "inlineMath", raw: match[0], latex: match[1] } : undefined; } },
  parseMarkdown: (token, helpers) => helpers.createNode("inlineMath", { latex: token.latex }),
  renderMarkdown: (node) => !node.attrs?.latex || /[\r\n$]/.test(node.attrs.latex) ? editorNodeHTML(node) : `$${node.attrs.latex}$`,
});
export const BlockMath = Node.create({
  name: "blockMath", group: "block", atom: true, selectable: true,
  addAttributes: () => ({ latex: sourceAttr("latex", "E=mc^2") }),
  parseHTML: () => [{ tag: "div[data-madi-math]" }],
  renderHTML: ({ HTMLAttributes }) => ["div", mergeAttributes(HTMLAttributes, { "data-madi-math": "block" }), HTMLAttributes["data-latex"] || ""],
  markdownTokenizer: { name: "blockMath", level: "block", start: (source) => source.indexOf("$$"), tokenize: (source) => { const match = /^\$\$[ \t]*\n([\s\S]*?)\n\$\$[ \t]*(?:\n|$)/.exec(source); return match ? { type: "blockMath", raw: match[0], latex: match[1] } : undefined; } },
  parseMarkdown: (token, helpers) => helpers.createNode("blockMath", { latex: token.latex }),
  renderMarkdown: (node) => !node.attrs?.latex || /^\$\$\s*$/m.test(node.attrs.latex) ? editorNodeHTML(node) : `$$\n${node.attrs.latex}\n$$`,
});

export const Callout = Node.create({
  name: "callout", group: "block", content: "block+", defining: true,
  addAttributes: () => ({ type: sourceAttr("type", "note"), title: sourceAttr("title") }),
  parseHTML: () => [{ tag: "aside[data-madi-callout]" }],
  renderHTML: ({ HTMLAttributes }) => ["aside", mergeAttributes(HTMLAttributes, { "data-madi-callout": "", class: "editor-callout" }), 0],
  markdownTokenizer: { name: "callout", level: "block", start: (source) => source.search(/^> \[!/m), tokenize: (source, _tokens, lexer) => {
    const match = /^>\s*\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]([^\n]*)\n((?:>[^\n]*(?:\n|$))*)/i.exec(source);
    if (!match) return;
    return { type: "callout", raw: match[0], calloutType: match[1].toLowerCase(), title: match[2].trim(), tokens: lexer.blockTokens(match[3].replace(/^> ?/gm, "")) };
  } },
  parseMarkdown: (token, helpers) => { const children = helpers.parseChildren(token.tokens || []); return helpers.createNode("callout", { type: token.calloutType, title: token.title }, children.length ? children : [{ type: "paragraph" }]); },
  renderMarkdown: (node, helpers) => `> [!${String(node.attrs?.type || "note").toUpperCase()}]${node.attrs?.title ? ` ${node.attrs.title}` : ""}\n> ${helpers.renderChildren(node.content || [], "\n\n").replaceAll("\n", "\n> ")}`,
});

export const Columns = Node.create({
  name: "columns", group: "block", content: "column{2,3}", isolating: true, defining: true,
  addAttributes: () => ({ count: { default: 2, parseHTML: (element) => Number(element.getAttribute("data-count")) === 3 ? 3 : 2, renderHTML: (attrs) => ({ "data-count": attrs.count }) } }),
  parseHTML: () => [{ tag: "div[data-madi-columns]" }],
  renderHTML: ({ HTMLAttributes }) => ["div", mergeAttributes(HTMLAttributes, { "data-madi-columns": "", class: "editor-columns" }), 0],
  ...createBlockMarkdownSpec({ nodeName: "columns", allowedAttributes: ["count"], defaultAttributes: { count: 2 }, parseAttributes: (value) => ({ count: Number(parseAttributes(value).count) === 3 ? 3 : 2 }) }),
});
export const Column = Node.create({
  name: "column", content: "block+", isolating: true, defining: true,
  parseHTML: () => [{ tag: "div[data-madi-column]" }],
  renderHTML: ({ HTMLAttributes }) => ["div", mergeAttributes(HTMLAttributes, { "data-madi-column": "" }), 0],
  ...createBlockMarkdownSpec({ nodeName: "column", allowedAttributes: [] }),
});

export const FootnoteReference = Node.create({
  name: "footnoteReference", group: "inline", inline: true, atom: true,
  addAttributes: () => ({ label: sourceAttr("label", "1") }),
  parseHTML: () => [{ tag: "sup[data-madi-footnote]" }],
  renderHTML: ({ node, HTMLAttributes }) => ["sup", mergeAttributes(HTMLAttributes, { "data-madi-footnote": "" }), `[${node.attrs.label}]`],
  markdownTokenizer: { name: "footnoteReference", level: "inline", start: (source) => source.indexOf("[^"), tokenize: (source) => { const match = /^\[\^([^\]\s:]{1,80})\]/.exec(source); return match ? { type: "footnoteReference", raw: match[0], label: match[1] } : undefined; } },
  parseMarkdown: (token, helpers) => helpers.createNode("footnoteReference", { label: token.label }),
  renderMarkdown: (node) => `[^${node.attrs?.label || "1"}]`,
});
export const FootnoteDefinition = Node.create({
  name: "footnoteDefinition", group: "block", content: "block+", defining: true,
  addAttributes: () => ({ label: sourceAttr("label", "1") }),
  parseHTML: () => [{ tag: "aside[data-madi-footnote-definition]" }],
  renderHTML: ({ HTMLAttributes }) => ["aside", mergeAttributes(HTMLAttributes, { "data-madi-footnote-definition": "", class: "editor-footnote" }), 0],
  markdownTokenizer: { name: "footnoteDefinition", level: "block", start: (source) => source.search(/^\[\^/m), tokenize: (source, _tokens, lexer) => { const match = /^\[\^([^\]\s:]{1,80})\]:[ \t]*([^\n]*(?:\n(?: {4}|\t)[^\n]*)*)(?:\n|$)/.exec(source); return match ? { type: "footnoteDefinition", raw: match[0], label: match[1], tokens: lexer.blockTokens(match[2].replace(/\n(?: {4}|\t)/g, "\n")) } : undefined; } },
  parseMarkdown: (token, helpers) => helpers.createNode("footnoteDefinition", { label: token.label }, helpers.parseChildren(token.tokens || [])),
  renderMarkdown: (node, helpers) => `[^${node.attrs?.label || "1"}]: ${helpers.renderChildren(node.content || [], "\n\n").replaceAll("\n", "\n    ")}`,
});

export const SyncedEmbed = Node.create({
  name: "syncedEmbed", group: "block", atom: true, selectable: true,
  addAttributes: () => ({ documentId: sourceAttr("documentId"), blockId: sourceAttr("blockId"), label: sourceAttr("label") }),
  parseHTML: () => [{ tag: "div[data-madi-embed]" }],
  renderHTML: ({ HTMLAttributes }) => ["div", mergeAttributes(HTMLAttributes, { "data-madi-embed": "" })],
  markdownTokenizer: { name: "syncedEmbed", level: "block", start: (source) => source.indexOf("![["), tokenize: (source) => { const match = /^!\[\[([0-9a-f-]{36})(?:#\^([^\]|\s]{1,128}))?(?:\|([^\]\n]*))?\]\][ \t]*(?:\n|$)/i.exec(source); return match ? { type: "syncedEmbed", raw: match[0], documentId: match[1], blockId: match[2] || "", label: match[3] || "" } : undefined; } },
  parseMarkdown: (token, helpers) => helpers.createNode("syncedEmbed", { documentId: token.documentId, blockId: token.blockId, label: token.label }),
  renderMarkdown: (node) => `![[${node.attrs?.documentId}${node.attrs?.blockId ? `#^${node.attrs.blockId}` : ""}${node.attrs?.label ? `|${node.attrs.label}` : ""}]]`,
});

export const Bookmark = Node.create({
  name: "bookmark", group: "block", atom: true,
  addAttributes: () => ({ url: sourceAttr("url"), title: sourceAttr("title"), description: sourceAttr("description") }),
  parseHTML: () => [{ tag: "div[data-madi-bookmark]" }],
  renderHTML: ({ node, HTMLAttributes }) => ["div", mergeAttributes(HTMLAttributes, { "data-madi-bookmark": "" }), node.attrs.title || node.attrs.url],
  ...createBlockMarkdownSpec({ nodeName: "bookmark", allowedAttributes: ["url", "title", "description"],
    serializeAttributes: (attrs) => Object.entries(attrs).map(([key,value])=>`${key}="${encodeAttribute(value)}"`).join(" "),
    parseAttributes: (value) => Object.fromEntries(Object.entries(parseAttributes(value)).map(([key,value])=>[key,decodeAttribute(value)])),
  }),
  parseMarkdown: (token, helpers) => helpers.createNode("bookmark", token.attributes),
});

// Use the public Pandoc-style Markdown specs supplied by TipTap, but limit
// serialized attributes so transient IDs/CSS cannot leak into content syntax.
export const Toggle = Details.extend({ ...createBlockMarkdownSpec({ nodeName: "details", allowedAttributes: [] }) });
export const ToggleSummary = DetailsSummary.extend({ ...createBlockMarkdownSpec({ nodeName: "detailsSummary", content: "inline", allowedAttributes: [] }) });
export const ToggleContent = DetailsContent.extend({ ...createBlockMarkdownSpec({ nodeName: "detailsContent", allowedAttributes: [] }) });

export const advancedNodes = [InlineMath, BlockMath, Callout, Columns, Column, FootnoteReference, FootnoteDefinition, SyncedEmbed, Bookmark, Toggle, ToggleSummary, ToggleContent];

/** Canonical portable HTML for tables requiring spans, widths or alignment. */
export function editorNodeHTML(node: JSONContent): string {
  const attrs = node.attrs || {};
  let body = node.type === "text" ? escape(node.text) : (node.content || []).map(editorNodeHTML).join("");
  for (const mark of node.marks || []) {
    if (mark.type === "link") body = `<a href="${escape(mark.attrs?.href)}">${body}</a>`;
    else { const tag = ({ bold: "strong", italic: "em", underline: "u", strike: "s", code: "code" } as Record<string, string>)[mark.type]; if (tag) body = `<${tag}>${body}</${tag}>`; }
  }
  const tag = ({ paragraph: "p", heading: `h${attrs.level || 1}`, blockquote: "blockquote", bulletList: "ul", orderedList: "ol", listItem: "li", taskList: "ul", taskItem: "li", table: "table", tableRow: "tr", tableCell: "td", tableHeader: "th", details: "details", detailsSummary: "summary", detailsContent: "div", columns: "div", column: "div", callout: "aside", footnoteDefinition: "aside", footnoteReference: "sup", syncedEmbed: "div", bookmark: "div", inlineMath: "span", blockMath: "div" } as Record<string, string>)[node.type || ""];
  if (tag) {
    let properties = "";
    const attribute = (key: string, value: unknown) => ` ${key}="${escape(value)}"`;
    const markers: Record<string, string> = { columns: "data-madi-columns", column: "data-madi-column", callout: "data-madi-callout", footnoteDefinition: "data-madi-footnote-definition", footnoteReference: "data-madi-footnote", syncedEmbed: "data-madi-embed", bookmark: "data-madi-bookmark" };
    if (markers[node.type || ""]) properties += attribute(markers[node.type || ""], "");
    const extra: Record<string, string[]> = { columns: ["count"], callout: ["type","title"], footnoteDefinition:["label"], footnoteReference:["label"], syncedEmbed:["documentId","blockId","label"],bookmark:["url","title","description"] };
    for(const key of extra[node.type || ""] || []) properties += attribute(`data-${key}`, attrs[key]);
    if (node.type === "inlineMath" || node.type === "blockMath") properties += attribute("data-madi-math", node.type === "inlineMath" ? "inline" : "block") + attribute("data-latex", attrs.latex);
    if (["taskList","taskItem","detailsContent"].includes(node.type || "")) properties += attribute("data-type", node.type);
    if (node.type === "taskItem") properties += attribute("data-checked", !!attrs.checked);
    if (node.type === "orderedList") properties += attribute("start", attrs.start || 1);
    if (node.type === "tableCell" || node.type === "tableHeader") {
      if (attrs.colspan > 1) properties += ` colspan="${Number(attrs.colspan)}"`;
      if (attrs.rowspan > 1) properties += ` rowspan="${Number(attrs.rowspan)}"`;
      if (attrs.colwidth?.length) properties += ` colwidth="${attrs.colwidth.map(Number).join(",")}"`;
      if (["left", "center", "right"].includes(attrs.align)) properties += ` align="${attrs.align}"`;
    }
    if (["left", "center", "right", "justify"].includes(attrs.textAlign)) properties += ` style="text-align:${attrs.textAlign}"`;
    return `<${tag}${properties}>${body}</${tag}>`;
  }
  if (node.type === "hardBreak") return "<br>";
  if (node.type === "horizontalRule") return "<hr>";
  if (node.type === "image") return `<img src="${escape(attrs.src)}" alt="${escape(attrs.alt)}">`;
  if (node.type === "codeBlock") return `<pre><code class="language-${escape(attrs.language)}">${escape(node.content?.map((child)=>child.text || "").join(""))}</code></pre>`;
  if (node.type === "text") return body;
  throw new Error(`표 HTML로 표현할 수 없는 블록입니다: ${node.type}`);
}
