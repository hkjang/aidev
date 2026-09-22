import { useEffect, useState } from "react";
import type { Editor } from "@tiptap/core";
import { yUndoPluginKey } from "@tiptap/y-tiptap";
import {
  ArrowDown,
  ArrowUp,
  Copy,
  GripVertical,
  Indent,
  Outdent,
  Redo,
  Trash2,
  Undo,
} from "lucide-react";
import { Button, CopyButton, ErrorBox, Modal } from "../ui";
import { blockEntries, organizeBlocks } from "./blockOperations";
import "./organizer.css";
const labels: Record<string, string> = {
  paragraph: "본문",
  heading: "제목",
  bulletList: "글머리 목록",
  orderedList: "번호 목록",
  listItem: "목록 항목",
  taskList: "할 일 목록",
  taskItem: "할 일",
  blockquote: "인용",
  codeBlock: "코드",
  table: "표",
  tableRow: "표 행",
  tableCell: "표 셀",
  tableHeader: "표 제목 셀",
  image: "이미지",
  horizontalRule: "구분선",
  details: "접기",
  detailsSummary: "접기 제목",
  detailsContent: "접기 본문",
  columns: "열 레이아웃",
  column: "열",
};
export default function BlockOrganizer({
  editor,
  documentID,
  open,
  onOpenChange,
}: {
  editor: Editor;
  documentID: string;
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const [, update] = useState(0),
    [selected, setSelected] = useState<string[]>([]),
    [error, setError] = useState(""),
    [placement, setPlacement] = useState("before"),
    [removing, setRemoving] = useState(false);
  useEffect(() => {
    const render = () => update((v) => v + 1);
    editor.on("transaction", render);
    return () => {
      editor.off("transaction", render);
    };
  }, [editor]);
  useEffect(() => {
    if (open) {
      setSelected([]);
      setError("");
    }
  }, [open]);
  if (!open) return null;
  const rows = blockEntries(editor.getJSON()).filter(
    (row) => editor.schema.nodes[row.node.type || ""]?.isBlock,
  );
  const act = (operation: string, ids = selected, target = "") => {
    setError("");
    if (!editor.isEditable) {
      setError("공동 편집 연결과 쓰기 권한을 확인하세요.");
      return;
    }
    try {
      const next = organizeBlocks(
        editor.getJSON(),
        ids,
        operation,
        target,
        placement,
      );
      try {
        editor.schema.nodeFromJSON(next).check();
      } catch {
        throw new Error(
          "이 위치에는 해당 블록을 놓을 수 없습니다. 표·목록·열의 구조를 유지하는 위치를 선택하세요. 원문은 변경하지 않았습니다.",
        );
      }
      const undo = yUndoPluginKey.getState(editor.state)?.undoManager;
      undo?.stopCapturing();
      editor.commands.setContent(next, { contentType: "json" });
      undo?.stopCapturing();
      if (operation === "delete") setSelected([]);
      setRemoving(false);
    } catch (e) {
      setError((e as Error).message);
    }
  };
  return (
    <Modal
      open={open}
      onOpenChange={onOpenChange}
      title="블록 정리"
      description="블록의 계층과 고유 ID를 보존합니다. 여러 블록을 선택하고 손잡이를 끌어 이동하세요."
      wide
    >
      <ErrorBox error={error} />
      <div className="organizer-toolbar">
        <label>
          <input
            type="checkbox"
            checked={
              rows.length > 0 &&
              rows.every((row) => selected.includes(row.node.attrs?.id))
            }
            onChange={(e) =>
              setSelected(
                e.target.checked ? rows.map((row) => row.node.attrs?.id) : [],
              )
            }
          />{" "}
          전체 선택
        </label>
        <span>{selected.length}개 선택</span>
        <Button disabled={!selected.length} onClick={() => act("up")}>
          <ArrowUp size={16} />
          위로
        </Button>
        <Button disabled={!selected.length} onClick={() => act("down")}>
          <ArrowDown size={16} />
          아래로
        </Button>
        <Button disabled={!selected.length} onClick={() => act("nest")}>
          <Indent size={16} />
          인용 안에 중첩
        </Button>
        <Button disabled={!selected.length} onClick={() => act("outdent")}>
          <Outdent size={16} />한 단계 밖으로
        </Button>
        <Button disabled={!selected.length} onClick={() => act("duplicate")}>
          <Copy size={16} />
          복제
        </Button>
        <Button
          variant="danger"
          disabled={!selected.length}
          onClick={() => setRemoving(true)}
        >
          <Trash2 size={16} />
          삭제
        </Button>
        <Button
          aria-label="블록 실행 취소"
          disabled={!editor.can().undo()}
          onClick={() => {
            yUndoPluginKey.getState(editor.state)?.undoManager.stopCapturing();
            editor.commands.undo();
          }}
        >
          <Undo size={16} />
        </Button>
        <Button
          aria-label="블록 다시 실행"
          disabled={!editor.can().redo()}
          onClick={() => editor.commands.redo()}
        >
          <Redo size={16} />
        </Button>
        <label>
          놓는 위치{" "}
          <select
            aria-label="블록 놓는 위치"
            value={placement}
            onChange={(e) => setPlacement(e.target.value)}
          >
            <option value="before">대상 앞</option>
            <option value="after">대상 뒤</option>
            <option value="inside">대상 안에 중첩</option>
          </select>
        </label>
      </div>
      {removing && (
        <div className="notice" role="alert">
          <span>
            선택한 블록과 포함된 하위 블록을 삭제할까요? 실행 취소로 복구할 수
            있습니다.
          </span>
          <Button onClick={() => setRemoving(false)}>취소</Button>
          <Button variant="danger" onClick={() => act("delete")}>
            블록 삭제 확인
          </Button>
        </div>
      )}
      <div className="organizer-list" role="tree" aria-label="중첩 블록 목록">
        {rows.slice(0, 10000).map((row, index) => {
          const id = row.node.attrs?.id as string;
          const text = (node: typeof row.node): string =>
            node.text || node.content?.map(text).join(" ") || "";
          return (
            <div
              key={id}
              role="treeitem"
              aria-level={row.depth + 1}
              aria-selected={selected.includes(id)}
              data-block-id={id}
              className={`organizer-row ${selected.includes(id) ? "selected" : ""}`}
              style={{ paddingLeft: Math.min(row.depth, 10) * 16 + 10 }}
              onDragOver={(e) => {
                if (e.dataTransfer.types.includes("application/x-madi-blocks"))
                  e.preventDefault();
              }}
              onDrop={(e) => {
                e.preventDefault();
                e.stopPropagation();
                try {
                  const payload = JSON.parse(
                    e.dataTransfer.getData("application/x-madi-blocks"),
                  );
                  if (
                    payload.documentID !== documentID ||
                    !Array.isArray(payload.ids) ||
                    payload.ids.length > 500
                  )
                    return;
                  act("move", payload.ids, id);
                } catch {
                  setError("블록 드래그 정보를 확인할 수 없습니다.");
                }
              }}
            >
              <button
                className="icon-button organizer-grip"
                aria-label={`블록 ${index + 1} 드래그 손잡이`}
                draggable
                onDragStart={(e) => {
                  e.dataTransfer.effectAllowed = "move";
                  e.dataTransfer.setData(
                    "application/x-madi-blocks",
                    JSON.stringify({
                      documentID,
                      ids: selected.includes(id) ? selected : [id],
                    }),
                  );
                }}
              >
                <GripVertical size={17} />
              </button>
              <input
                type="checkbox"
                aria-label={`블록 ${index + 1} 선택`}
                checked={selected.includes(id)}
                onChange={(e) =>
                  setSelected(
                    e.target.checked
                      ? [...selected, id]
                      : selected.filter((value) => value !== id),
                  )
                }
              />
              <div>
                <strong>
                  {text(row.node).slice(0, 100) ||
                    labels[row.node.type || ""] ||
                    row.node.type}
                </strong>
                <small>
                  {labels[row.node.type || ""] || row.node.type} ·{" "}
                  {row.depth + 1}단계 · {id.slice(0, 8)}
                </small>
              </div>
              <CopyButton
                value={`${window.location.origin}/app/documents/${documentID}#^${id}`}
                label="블록 링크"
              />
              <CopyButton
                value={`![[${documentID}#^${id}]]`}
                label="동기화 참조"
              />
            </div>
          );
        })}
      </div>
      <p className="muted">
        복제한 블록과 그 하위 블록에는 새 ID가 생성됩니다. 표/목록의 필수 구조를
        깨는 이동은 원문을 변경하지 않고 거부합니다.
      </p>
    </Modal>
  );
}
