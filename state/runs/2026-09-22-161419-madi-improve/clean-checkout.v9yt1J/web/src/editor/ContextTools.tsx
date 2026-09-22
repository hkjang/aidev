import { useEffect, useState } from "react";
import type { Editor } from "@tiptap/core";
import { Button, Field, Modal } from "../ui";
import { copyText } from "../navigation/clipboard";
import "./context-tools.css";

export default function ContextTools({ editor }: { editor: Editor }) {
  const [, update] = useState(0);
  const [link, setLink] = useState(false),
    [url, setURL] = useState(""),
    [notice, setNotice] = useState("");
  const [target, setTarget] = useState<{
    from: number;
    to: number;
    doc: Editor["state"]["doc"];
  } | null>(null);
  useEffect(() => {
    const refresh = () => update((n) => n + 1);
    editor.on("selectionUpdate", refresh).on("transaction", refresh);
    return () => {
      editor.off("selectionUpdate", refresh).off("transaction", refresh);
    };
  }, [editor]);
  if (!editor.isEditable) return null;
  const table = editor.isActive("table"),
    code = editor.isActive("codeBlock"),
    selected = !editor.state.selection.empty;
  if (!table && !code && !selected && !link) return null;
  const safeURL = !url || /^(https?:\/\/|mailto:|\/app\/|#)/i.test(url);
  const staleLink = !!target && !editor.state.doc.eq(target.doc);
  return (
    <section className="context-tools" aria-label="선택한 내용 편집 도구">
      <strong>
        {table ? "표 편집" : code ? "코드 편집" : "선택한 텍스트"}
      </strong>
      <div role="group" aria-label="문맥 편집">
        {selected && !code && (
          <>
            <Button
              aria-pressed={editor.isActive("bold")}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().toggleBold().run()}
            >
              선택 굵게
            </Button>
            <Button
              aria-pressed={editor.isActive("italic")}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().toggleItalic().run()}
            >
              선택 기울임
            </Button>
            <Button
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => {
                setURL(editor.getAttributes("link").href || "");
                setTarget({
                  from: editor.state.selection.from,
                  to: editor.state.selection.to,
                  doc: editor.state.doc,
                });
                setLink(true);
              }}
            >
              선택 링크
            </Button>
          </>
        )}
        {code && (
          <Button
            onMouseDown={(e) => e.preventDefault()}
            onClick={async () => {
              let node = editor.state.selection.$from;
              for (let depth = node.depth; depth > 0; depth--)
                if (node.node(depth).type.name === "codeBlock") {
                  try {
                    await copyText(node.node(depth).textContent);
                    setNotice("코드를 복사했습니다.");
                  } catch {
                    setNotice(
                      "복사 권한을 확인하거나 코드를 직접 선택해 복사하세요.",
                    );
                  }
                  break;
                }
            }}
          >
            코드 내용 복사
          </Button>
        )}
        {table && (
          <>
            <Button
              disabled={!editor.can().addRowAfter()}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().addRowAfter().run()}
            >
              아래에 행 추가
            </Button>
            <Button
              disabled={!editor.can().addColumnAfter()}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().addColumnAfter().run()}
            >
              오른쪽 열 추가
            </Button>
            <Button
              disabled={!editor.can().mergeCells()}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().mergeCells().run()}
            >
              선택 셀 병합
            </Button>
            <Button
              disabled={!editor.can().splitCell()}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().splitCell().run()}
            >
              셀 분할
            </Button>
            <Button
              disabled={!editor.can().deleteRow()}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().deleteRow().run()}
            >
              현재 행 삭제
            </Button>
            <Button
              disabled={!editor.can().deleteColumn()}
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => editor.chain().focus().deleteColumn().run()}
            >
              현재 열 삭제
            </Button>
          </>
        )}
      </div>
      {notice && <p role="status">{notice}</p>}
      <Modal
        open={link}
        onOpenChange={setLink}
        title="선택한 텍스트 링크"
        description="주소는 저장만 하며 미리 읽거나 외부로 전송하지 않습니다."
      >
        <Field label="링크 주소">
          <input
            value={url}
            maxLength={4096}
            onChange={(e) => setURL(e.target.value)}
            placeholder="https:// 또는 /app/ 문서 경로"
          />
        </Field>
        {!safeURL && (
          <p role="alert">
            http(s), 메일, 문서 경로 또는 문서 안 위치만 사용할 수 있습니다.
          </p>
        )}
        {staleLink && (
          <p role="alert">
            문서가 변경되었습니다. 취소하고 현재 텍스트를 다시 선택하세요.
          </p>
        )}
        <div className="modal-actions">
          <Button onClick={() => setLink(false)}>취소</Button>
          <Button
            disabled={!safeURL || staleLink}
            onClick={() => {
              if (
                !editor.isEditable ||
                !target ||
                !editor.state.doc.eq(target.doc)
              )
                return;
              editor.commands.setTextSelection({
                from: target.from,
                to: target.to,
              });
              if (url) editor.chain().focus().setLink({ href: url }).run();
              else editor.chain().focus().unsetLink().run();
              setLink(false);
            }}
          >
            {url ? "링크 적용" : "링크 제거"}
          </Button>
        </div>
      </Modal>
    </section>
  );
}
