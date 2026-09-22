import { useEffect, useRef, useState } from "react";
import type { Editor, JSONContent } from "@tiptap/core";
import DOMPurify from "dompurify";
import { Button, Modal } from "../ui";
import "./context-tools.css";

type Paste = {
  text: string;
  html: string;
  url: string;
  from: number;
  to: number;
  snapshot: Editor["state"]["doc"];
};
const limits = 2 * 1024 * 1024;
function cleanHTML(html: string) {
  const fragment = DOMPurify.sanitize(html, {
    ALLOWED_TAGS: [
      "p",
      "br",
      "h1",
      "h2",
      "h3",
      "h4",
      "h5",
      "h6",
      "strong",
      "b",
      "em",
      "i",
      "s",
      "del",
      "u",
      "blockquote",
      "pre",
      "code",
      "ul",
      "ol",
      "li",
      "table",
      "thead",
      "tbody",
      "tr",
      "th",
      "td",
      "hr",
      "a",
    ],
    ALLOWED_ATTR: ["href", "title", "colspan", "rowspan"],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
    RETURN_DOM_FRAGMENT: true,
  });
  fragment.querySelectorAll("a").forEach((a) => {
    if (!/^(https?:\/\/|mailto:|\/app\/|#)/i.test(a.getAttribute("href") || ""))
      a.removeAttribute("href");
  });
  const container = document.createElement("div");
  container.append(fragment);
  return container.innerHTML;
}
function plainContent(text: string): JSONContent[] {
  return text
    .replace(/\r\n?/g, "\n")
    .split("\n")
    .map((line) => ({
      type: "paragraph",
      ...(line ? { content: [{ type: "text", text: line }] } : {}),
    }));
}

/** Clipboard data remains transient; no URL fetching, remote images or copied block IDs. */
export default function PasteReview({ editor }: { editor: Editor }) {
  const [paste, setPaste] = useState<Paste | null>(null),
    [error, setError] = useState(""),
    [, redraw] = useState(0);
  const pending = useRef(false);
  useEffect(() => {
    const changed = () => redraw((n) => n + 1);
    const capture = (event: ClipboardEvent) => {
      if (
        !editor.isEditable ||
        event.defaultPrevented ||
        !event.clipboardData ||
        event.clipboardData.files.length
      )
        return;
      const text = event.clipboardData.getData("text/plain"),
        raw = event.clipboardData.getData("text/html"),
        candidate = text.trim();
      let url = "";
      try {
        const parsed = new URL(candidate);
        if (
          ["http:", "https:"].includes(parsed.protocol) &&
          !parsed.username &&
          !parsed.password &&
          !/[\r\n]/.test(candidate)
        )
          url = candidate;
      } catch {}
      if (!raw && !url) return;
      event.preventDefault();
      event.stopImmediatePropagation();
      if (pending.current) return;
      if (
        new TextEncoder().encode(text).length > limits ||
        new TextEncoder().encode(raw).length > limits
      ) {
        setError(
          "붙여넣기 자료는 2MiB 이하여야 합니다. 파일 가져오기를 사용하거나 범위를 줄여주세요.",
        );
        return;
      }
      pending.current = true;
      setError("");
      setPaste({
        text,
        html: cleanHTML(raw),
        url,
        from: editor.state.selection.from,
        to: editor.state.selection.to,
        snapshot: editor.state.doc,
      });
    };
    const element = editor.view.dom;
    element.addEventListener("paste", capture, true);
    editor.on("transaction", changed);
    return () => {
      element.removeEventListener("paste", capture, true);
      editor.off("transaction", changed);
    };
  }, [editor]);
  const close = () => {
    pending.current = false;
    setPaste(null);
  };
  const stale =
    !!paste && (!editor.isEditable || !editor.state.doc.eq(paste.snapshot));
  const apply = (mode: "plain" | "format" | "link" | "bookmark") => {
    if (
      !paste ||
      stale ||
      !editor.isEditable ||
      !editor.state.doc.eq(paste.snapshot)
    )
      return;
    const content =
      mode === "format"
        ? paste.html
        : mode === "bookmark"
          ? [
              {
                type: "bookmark",
                attrs: {
                  url: paste.url,
                  title: paste.url,
                  description:
                    "직접 붙여넣은 링크 · 외부 내용을 수집하지 않았습니다.",
                },
              },
            ]
          : mode === "link"
            ? [
                {
                  type: "paragraph",
                  content: [
                    {
                      type: "text",
                      text: paste.url,
                      marks: [{ type: "link", attrs: { href: paste.url } }],
                    },
                  ],
                },
              ]
            : plainContent(paste.text);
    const applied = editor
      .chain()
      .focus()
      .insertContentAt({ from: paste.from, to: paste.to }, content)
      .run();
    if (applied) {
      close();
      setError("");
    } else
      setError(
        "현재 위치에 넣을 수 없습니다. 원문을 보관하고 다른 블록에서 다시 붙여넣으세요.",
      );
  };
  return (
    <>
      {error && (
        <p className="notice" role="alert">
          {error}
        </p>
      )}
      <Modal
        open={!!paste}
        onOpenChange={(open) => {
          if (!open) close();
        }}
        title="붙여넣기 방식 선택"
        description="원문·기본 서식을 비교해 선택하세요. 이미지, 스크립트, 추적 요소와 복사된 블록 ID는 가져오지 않으며 외부 URL을 요청하지 않습니다."
      >
        {paste && (
          <div className="paste-review">
            <pre aria-label="붙여넣을 원문 미리보기">
              {paste.text.slice(0, 12000) ||
                "클립보드에 일반 텍스트가 없습니다. 기본 서식만 사용할 수 있습니다."}
              {paste.text.length > 12000 &&
                "\n… 미리보기 생략 (전체 내용은 붙여넣습니다)"}
            </pre>
            {stale && (
              <p role="alert" className="notice">
                문서가 변경되었거나 편집 권한이 달라졌습니다. 원래 내용은
                변경하지 않았으니 취소 후 현재 위치에서 다시 붙여넣으세요.
              </p>
            )}
            <div role="group" aria-label="붙여넣기 선택">
              <Button
                disabled={stale || !paste.text}
                onClick={() => apply("plain")}
              >
                텍스트만 붙여넣기
              </Button>
              <Button
                disabled={stale || !paste.html}
                onClick={() => apply("format")}
              >
                기본 서식 유지
              </Button>
              {paste.url && (
                <>
                  <Button disabled={stale} onClick={() => apply("link")}>
                    링크로 붙여넣기
                  </Button>
                  <Button disabled={stale} onClick={() => apply("bookmark")}>
                    북마크로 붙여넣기
                  </Button>
                </>
              )}
              <Button onClick={close}>취소</Button>
            </div>
          </div>
        )}
      </Modal>
    </>
  );
}
