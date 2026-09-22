import StarterKit from "@tiptap/starter-kit";
import { Markdown, MarkdownManager } from "@tiptap/markdown";
import TaskList from "@tiptap/extension-task-list";
import TaskItem from "@tiptap/extension-task-item";
import { Table, TableCell, TableHeader, TableRow } from "@tiptap/extension-table";
import Image from "@tiptap/extension-image";
import TextAlign from "@tiptap/extension-text-align";
import Paragraph from "@tiptap/extension-paragraph";
import Heading from "@tiptap/extension-heading";
import { advancedNodes, editorNodeHTML } from "./nodes";
import type { JSONContent } from "@tiptap/core";

export const PortableTable = Table.extend({
  renderMarkdown(node, helpers, context) {
    // GFM cannot express rowspan/colspan or widths. HTML tables are standard
    // Markdown and preserve those properties in Obsidian/export/import.
    let requiresHTML = false;
    const inspect = (value: JSONContent) => { const a = value.attrs || {}; if (a.colspan > 1 || a.rowspan > 1 || a.colwidth?.length || a.textAlign) requiresHTML = true; if (["tableCell","tableHeader"].includes(value.type || "") && (value.content?.length !== 1 || value.content?.[0]?.type !== "paragraph")) requiresHTML = true; value.content?.forEach(inspect); };
    inspect(node);
    node.content?.forEach((row,index)=>row.content?.forEach((cell)=>{if((index===0)!==(cell.type==="tableHeader"))requiresHTML=true;}));
    if (requiresHTML) return editorNodeHTML(node);
    return Table.config.renderMarkdown?.call(this, node, helpers, context) || editorNodeHTML(node);
  },
}).configure({ resizable: true, lastColumnResizable: true });

export const PortableParagraph = Paragraph.extend({ renderMarkdown(node, helpers) { return node.attrs?.textAlign ? editorNodeHTML(node) : helpers.renderChildren(node.content || []); } });
export const PortableHeading = Heading.extend({ renderMarkdown(node, helpers) { return node.attrs?.textAlign ? editorNodeHTML(node) : `${"#".repeat(node.attrs?.level || 1)} ${helpers.renderChildren(node.content || [])}`; } });

export const commonSchema = () => [
  StarterKit.configure({ paragraph: false, heading: false }), PortableParagraph, PortableHeading, Markdown, TaskList, TaskItem.configure({ nested: true }), PortableTable, TableCell, TableHeader, TableRow, Image,
  TextAlign.configure({ types: ["heading", "paragraph"] }), ...advancedNodes,
];

let manager: MarkdownManager | undefined;
export function parseEditorMarkdown(markdown: string): JSONContent {
  manager ||= new MarkdownManager({ extensions: commonSchema() });
  return manager.parse(markdown);
}
export function serializeEditorMarkdown(document: JSONContent): string {
  manager ||= new MarkdownManager({ extensions: commonSchema() });
  return manager.serialize(document);
}

export function frontMatterParts(markdown: string): { front: string; body: string } {
  const match = markdown.match(/^(---\r?\n[\s\S]*?\r?\n---(?:\r?\n|$))/);
  return match ? { front: match[1], body: markdown.slice(match[1].length) } : { front: "", body: markdown };
}

/** Fail closed before seeding Yjs: an unrepresentable import stays untouched. */
export function unsupportedMarkdownReason(markdown: string): string {
  const lines = frontMatterParts(markdown).body.split("\n");
  let fence = "", size = 0;
  const plain: string[] = [];
  for (const line of lines) {
    const match = /^\s{0,3}(`{3,}|~{3,})/.exec(line);
    if (match) { if (!fence) { fence = match[1][0]; size = match[1].length; } else if (match[1][0] === fence && match[1].length >= size) fence = ""; continue; }
    if (!fence) plain.push(line);
  }
  const source = plain.join("\n").replace(/<(?:https?:\/\/|mailto:)[^<>\s]+>/gi, "");
  if (/<!--|<!\[CDATA\[|<!DOCTYPE|<\?/i.test(source)) return "HTML 주석과 사용자 정의 선언은 Markdown 모드에서 원문 그대로 보존합니다.";
  const allowed = new Set(["p", "h1", "h2", "h3", "h4", "h5", "h6", "strong", "b", "em", "i", "s", "del", "u", "a", "img", "br", "hr", "blockquote", "ul", "ol", "li", "pre", "code", "table", "thead", "tbody", "tr", "td", "th", "details", "summary", "div", "span", "aside", "sup"]);
  for (const match of source.matchAll(/<\/?([a-z][a-z0-9-]*)\b[^>]*>/gi)) {
    const tag = match[1].toLowerCase();
    if (!allowed.has(tag)) return `블록 편집기로 표현할 수 없는 HTML 태그(${tag})가 있습니다. 원문을 보존하기 위해 Markdown 모드에서 편집하세요.`;
    if (!match[0].startsWith("</") && ["div", "span", "aside", "sup"].includes(tag) && !/data-madi-|data-type="detailsContent"/.test(match[0])) return "사용자 정의 HTML 레이아웃은 Markdown 모드에서 원문 그대로 편집하세요.";
    for (const marker of match[0].matchAll(/\b(data-madi-[\w-]+)/gi)) if (!["data-madi-math","data-madi-callout","data-madi-columns","data-madi-column","data-madi-footnote","data-madi-footnote-definition","data-madi-embed","data-madi-bookmark"].includes(marker[1].toLowerCase())) return "알 수 없는 HTML 블록은 Markdown 모드에서 원문 그대로 편집하세요.";
    const inlineStyle = /\bstyle\s*=\s*["']([^"']*)["']/i.exec(match[0]);
    if (inlineStyle && !/^\s*text-align\s*:\s*(left|right|center|justify)\s*;?\s*$/i.test(inlineStyle[1])) return "사용자 정의 HTML 스타일은 Markdown 모드에서 원문 그대로 편집하세요.";
    if (/\bon\w+\s*=|\b(?:src|href)\s*=\s*["']?\s*(?:javascript|data):/i.test(match[0])) return "실행 가능한 HTML 속성이 포함되어 있습니다. 원문 모드에서 확인하세요.";
  }
  for (const match of source.matchAll(/^:::([a-zA-Z][\w-]*)/gm)) {
    if (!["columns", "column", "details", "detailsSummary", "detailsContent", "bookmark"].includes(match[1])) return `지원하지 않는 확장 구문(${match[1]})입니다. Markdown 모드에서 원문을 보존하세요.`;
  }
  return "";
}
