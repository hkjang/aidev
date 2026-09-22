import { Fragment, useEffect, useMemo, useRef, type ReactNode } from "react";
import { Link, useLocation } from "react-router-dom";
import type { JSONContent } from "@tiptap/core";
import type { Doc, DocSummary } from "../api";
import ReactMarkdown from "react-markdown";
import rehypeHighlight from "rehype-highlight";
import DiagramPreview from "../DiagramPreview";
import PluginBlockPreview from "../PluginBlockPreview";
import DocumentQueryBlock from "../query/DocumentQueryBlock";
import MathPreview from "./MathPreview";
import { BookmarkCard, SyncedContent } from "./SyncedContent";
import {
  frontMatterParts,
  parseEditorMarkdown,
  unsupportedMarkdownReason,
} from "./schema";

export function MarkdownContent({
  markdown,
  documents = [],
  metadata,
  documentId,
}: {
  markdown: string;
  documents?: DocSummary[];
  metadata?: Doc["block_metadata"];
  documentId?: string;
}) {
  const container = useRef<HTMLDivElement>(null),
    location = useLocation();
  useEffect(() => {
    let anchor = "";
    try {
      anchor = decodeURIComponent(location.hash.slice(1));
    } catch {
      return;
    }
    if (!anchor) return;
    const frame = requestAnimationFrame(() =>
      container.current
        ?.querySelector(`[id="${CSS.escape(anchor)}"]`)
        ?.scrollIntoView({ block: "center" }),
    );
    return () => cancelAnimationFrame(frame);
  }, [markdown, location.hash, location.pathname, metadata]);
  const parsed = useMemo(() => {
    const reason = unsupportedMarkdownReason(markdown);
    if (reason) return { reason, document: null };
    try {
      const document = parseEditorMarkdown(frontMatterParts(markdown).body);
      let index = 0;
      const text = (node: JSONContent): string =>
        node.text || node.content?.map(text).join("") || "";
      const attach = (node: JSONContent) => {
        if (
          ![
            "doc",
            "text",
            "inlineMath",
            "footnoteReference",
            "hardBreak",
          ].includes(node.type || "")
        ) {
          const stored = metadata?.blocks?.[index++];
          if (
            stored &&
            stored.type === node.type &&
            stored.text === Array.from(text(node)).slice(0, 200).join("")
          )
            node.attrs = { ...node.attrs, id: stored.id };
        }
        node.content?.forEach(attach);
      };
      if (metadata?.blocks) attach(document);
      return { reason: "", document };
    } catch {
      return {
        reason:
          "이 Markdown은 원문 형태로 표시합니다. 내용은 변경하지 않았습니다.",
        document: null,
      };
    }
  }, [markdown, metadata]);
  const wikiText = (text: string): ReactNode => {
    const parts: ReactNode[] = [];
    let offset = 0;
    for (const match of text.matchAll(/\[\[([^\]\n]+)\]\]/g)) {
      parts.push(text.slice(offset, match.index));
      const [target, label] = match[1].split("|"),
        [title, anchor] = target.split("#");
      const candidates = documents.filter(
        (d) => d.title === title || d.aliases?.includes(title),
      );
      const doc =
        documents.find((d) => d.id === title) ||
        (candidates.length === 1 ? candidates[0] : undefined);
      const id = doc?.id || (/^[0-9a-f-]{36}$/i.test(title) ? title : "");
      parts.push(
        <Link
          key={match.index}
          title={
            candidates.length > 1 && !doc
              ? "같은 제목·별칭이 여러 개입니다. 검색에서 문서를 선택하세요."
              : undefined
          }
          to={
            id
              ? `/app/documents/${id}?mode=read${anchor ? `#${encodeURIComponent(anchor)}` : ""}`
              : `/app/search?q=${encodeURIComponent(title)}`
          }
        >
          {label || doc?.title || title}
        </Link>,
      );
      offset = match.index! + match[0].length;
    }
    parts.push(text.slice(offset));
    return parts;
  };
  const safeURL = (value: string): boolean => {
    try {
      return ["http:", "https:", "mailto:", "tel:"].includes(
        new URL(value, window.location.origin).protocol,
      );
    } catch {
      return false;
    }
  };
  const render = (node: JSONContent, key: number | string): ReactNode => {
    const a = node.attrs || {},
      content = node.content?.map((child, i) => render(child, i));
    let value: ReactNode = content;
    const text = (n: JSONContent): string =>
      n.text || n.content?.map(text).join("") || "";
    const id = a.id ? `^${a.id}` : undefined;
    const style = ["left", "center", "right", "justify"].includes(a.textAlign)
      ? { textAlign: a.textAlign }
      : undefined;
    switch (node.type) {
      case "doc":
        return <Fragment key={key}>{content}</Fragment>;
      case "text":
        value = wikiText(node.text || "");
        break;
      case "paragraph":
        return (
          <p key={key} id={id} style={style}>
            {content}
          </p>
        );
      case "heading": {
        const Tag =
          `h${Math.min(6, Math.max(1, Number(a.level) || 1))}` as "h1";
        return (
          <Tag key={key} id={id || text(node)} style={style}>
            {id && <span id={text(node)} aria-hidden="true" />}
            {content}
          </Tag>
        );
      }
      case "blockquote":
        return (
          <blockquote key={key} id={id}>
            {content}
          </blockquote>
        );
      case "bulletList":
        return (
          <ul key={key} id={id}>
            {content}
          </ul>
        );
      case "orderedList":
        return (
          <ol key={key} id={id} start={Number(a.start) || 1}>
            {content}
          </ol>
        );
      case "listItem":
        return (
          <li key={key} id={id}>
            {content}
          </li>
        );
      case "taskList":
        return (
          <ul className="contains-task-list" key={key} id={id}>
            {content}
          </ul>
        );
      case "taskItem":
        return (
          <li className="task-list-item" key={key} id={id}>
            <input type="checkbox" readOnly checked={!!a.checked} />
            {content}
          </li>
        );
      case "codeBlock": {
        const source = text(node),
          fence = "`".repeat(
            Math.max(
              3,
              ...[...source.matchAll(/`+/g)].map(
                (match) => match[0].length + 1,
              ),
            ),
          );
        return (
          <div key={key} id={id}>
            {a.language === "madi-query" ? (
              <DocumentQueryBlock source={source} documentId={documentId} />
            ) : a.language === "madi-plugin" ? (
              <PluginBlockPreview source={source} />
            ) : (
              <ReactMarkdown
                rehypePlugins={[
                  [rehypeHighlight, { detect: false, ignoreMissing: true }],
                ]}
              >{`${fence}${a.language || "text"}\n${source}\n${fence}`}</ReactMarkdown>
            )}
            {a.language === "mermaid" && <DiagramPreview source={source} />}
          </div>
        );
      }
      case "horizontalRule":
        return <hr key={key} id={id} />;
      case "hardBreak":
        return <br key={key} />;
      case "image":
        return safeURL(a.src) ? (
          <img
            key={key}
            src={a.src}
            alt={a.alt || ""}
            title={a.title || undefined}
            loading="lazy"
          />
        ) : (
          <span key={key}>이미지 주소를 확인하세요.</span>
        );
      case "table":
        return (
          <div className="table-scroll" key={key} id={id}>
            <table>
              <tbody>{content}</tbody>
            </table>
          </div>
        );
      case "tableRow":
        return <tr key={key}>{content}</tr>;
      case "tableCell":
      case "tableHeader": {
        const Tag = node.type === "tableCell" ? "td" : "th";
        return (
          <Tag
            key={key}
            colSpan={Number(a.colspan) || 1}
            rowSpan={Number(a.rowspan) || 1}
            style={{
              textAlign: ["left", "right", "center"].includes(a.align)
                ? a.align
                : undefined,
              width: a.colwidth?.[0] || undefined,
            }}
          >
            {content}
          </Tag>
        );
      }
      case "inlineMath":
        return <MathPreview key={key} source={a.latex || ""} />;
      case "blockMath":
        return (
          <div key={key} id={id}>
            <MathPreview source={a.latex || ""} display />
          </div>
        );
      case "callout":
        return (
          <aside key={key} id={id} className={`editor-callout ${a.type}`}>
            <strong>
              {a.title ||
                (
                  {
                    note: "참고",
                    tip: "도움말",
                    important: "중요",
                    warning: "주의",
                    caution: "경고",
                  } as Record<string, string>
                )[a.type] ||
                "참고"}
            </strong>
            {content}
          </aside>
        );
      case "details":
        return (
          <details key={key} id={id}>
            {content}
          </details>
        );
      case "detailsSummary":
        return <summary key={key}>{content}</summary>;
      case "detailsContent":
        return <div key={key}>{content}</div>;
      case "columns":
        return (
          <div
            key={key}
            id={id}
            className="editor-columns"
            data-count={a.count || 2}
          >
            {content}
          </div>
        );
      case "column":
        return (
          <div key={key} data-madi-column="">
            {content}
          </div>
        );
      case "footnoteReference":
        return (
          <sup key={key}>
            <a href={`#fn-${encodeURIComponent(a.label)}`}>[{a.label}]</a>
          </sup>
        );
      case "footnoteDefinition":
        return (
          <aside
            key={key}
            id={`fn-${encodeURIComponent(a.label)}`}
            className="editor-footnote"
          >
            <strong>[{a.label}]</strong>
            {content}
          </aside>
        );
      case "bookmark":
        return (
          <BookmarkCard
            key={key}
            url={a.url || ""}
            title={a.title}
            description={a.description}
          />
        );
      case "syncedEmbed":
        return (
          <SyncedContent
            key={key}
            documentId={a.documentId}
            blockId={a.blockId}
            label={a.label}
            renderMarkdown={(value) => (
              <MarkdownContent markdown={value} documents={documents} />
            )}
          />
        );
      default:
        return <pre key={key}>{text(node)}</pre>;
    }
    for (const mark of node.marks || []) {
      switch (mark.type) {
        case "bold":
          value = <strong>{value}</strong>;
          break;
        case "italic":
          value = <em>{value}</em>;
          break;
        case "underline":
          value = <u>{value}</u>;
          break;
        case "strike":
          value = <del>{value}</del>;
          break;
        case "code":
          value = <code>{node.text}</code>;
          break;
        case "link": {
          const href = mark.attrs?.href || "";
          if (safeURL(href))
            value = href.startsWith("/app/") ? (
              <Link to={href}>{value}</Link>
            ) : (
              <a href={href} target="_blank" rel="noopener noreferrer">
                {value}
              </a>
            );
          break;
        }
      }
    }
    return <Fragment key={key}>{value}</Fragment>;
  };
  return (
    <div className="markdown-content" ref={container}>
      {parsed.document ? (
        render(parsed.document, "root")
      ) : (
        <>
          <p className="notice subtle">{parsed.reason}</p>
          <pre>{markdown}</pre>
        </>
      )}
    </div>
  );
}
