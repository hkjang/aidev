import * as Menu from "@radix-ui/react-dropdown-menu";
import {
  Eye,
  Pencil,
  MoreHorizontal,
  Code2,
  Presentation,
  Printer,
  Focus,
  Paperclip,
  Sparkles,
  Split,
} from "lucide-react";
export function DocumentModeControls({
  mode,
  canWrite,
  onMode,
  onFocus,
  onPresent,
  onPrint,
  onAttach,
  onAI,
  onSelectionAI,
  onSplit,
}: {
  mode: string;
  canWrite: boolean;
  onMode: (mode: string) => void;
  onFocus: () => void;
  onPresent: () => void;
  onPrint: () => void;
  onAttach: () => void;
  onAI: () => void;
  onSelectionAI: () => void;
  onSplit: () => void;
}) {
  return (
    <div className="editor-mode-bar">
      <div className="segmented" aria-label="문서 모드">
        <button
          type="button"
          className={mode === "preview" || !canWrite ? "active" : ""}
          aria-pressed={mode === "preview" || !canWrite}
          onClick={() => onMode("preview")}
        >
          <Eye size={16} />
          읽기
        </button>
        <button
          type="button"
          disabled={!canWrite}
          className={mode !== "preview" && canWrite ? "active" : ""}
          aria-pressed={mode !== "preview" && canWrite}
          onClick={() => onMode("edit")}
        >
          <Pencil size={16} />
          편집
        </button>
      </div>
      <div>
        <button
          type="button"
          className="button"
          onMouseDown={(event) => event.preventDefault()}
          onClick={onSelectionAI}
        >
          <Sparkles size={16} /> 선택 AI
        </button>
        <button
          type="button"
          className="icon-button"
          aria-label="선택 블록을 새 문서로 분리"
          title="선택 블록을 새 문서로 분리"
          disabled={!canWrite}
          onMouseDown={(event) => event.preventDefault()}
          onClick={onSplit}
        >
          <Split size={18} />
        </button>
        <button
          type="button"
          className="icon-button"
          aria-label="파일 첨부"
          disabled={!canWrite}
          onClick={onAttach}
        >
          <Paperclip size={18} />
        </button>
        <button
          type="button"
          className="icon-button"
          aria-label="AI 도우미"
          onClick={onAI}
        >
          <Sparkles size={18} />
        </button>
        <Menu.Root>
          <Menu.Trigger asChild>
            <button
              type="button"
              className="icon-button"
              aria-label="문서 보기 도구"
            >
              <MoreHorizontal size={20} />
            </button>
          </Menu.Trigger>
          <Menu.Portal>
            <Menu.Content className="dropdown-menu" align="end" sideOffset={8}>
              <Menu.Item
                className="dropdown-item"
                onSelect={() => onMode("source")}
              >
                <Code2 size={17} />
                Markdown 원문
              </Menu.Item>
              <Menu.Item className="dropdown-item" onSelect={onFocus}>
                <Focus size={17} />
                집중 모드
              </Menu.Item>
              <Menu.Item className="dropdown-item" onSelect={onPresent}>
                <Presentation size={17} />
                프레젠테이션 보기
              </Menu.Item>
              <Menu.Item className="dropdown-item" onSelect={onPrint}>
                <Printer size={17} />
                인쇄 / PDF 저장
              </Menu.Item>
            </Menu.Content>
          </Menu.Portal>
        </Menu.Root>
      </div>
    </div>
  );
}
