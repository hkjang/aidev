import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Search, FileText, ArrowRight } from "lucide-react";
import { useApp } from "../context";
import { Modal } from "../ui";
import { documentCommand, ShortcutHelp } from "./shortcuts";
import "./style.css";
export default function CommandPalette({
  open,
  onOpenChange,
  onCreate,
  onAI,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onCreate: () => void;
  onAI: () => void;
}) {
  const { documents, createDocument, notify, user } = useApp(),
    navigate = useNavigate(),
    location = useLocation();
  const [query, setQuery] = useState(""),
    [active, setActive] = useState(0);
  const list = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (open) {
      setQuery("");
      setActive(0);
    }
  }, [open]);
  const current = /^\/app\/documents\/[a-f0-9-]+$/i.test(location.pathname);
  const commands: {
    id: string;
    title: string;
    kind: string;
    action: () => unknown;
  }[] = [
    { id: "new", title: "새 문서 만들기", kind: "문서", action: onCreate },
    {
      id: "search",
      title: "전체 검색",
      kind: "검색",
      action: () => navigate("/app/search"),
    },
    {
      id: "daily",
      title: "오늘의 노트",
      kind: "문서",
      action: async () => {
        const today = new Intl.DateTimeFormat("sv-SE", {
          timeZone: user.preferences?.timezone || "Asia/Seoul",
        }).format(new Date());
        const old = documents.find(
          (d) => d.title === today && d.owner_id === user.id,
        );
        const doc =
          old ||
          (await createDocument(
            today,
            `# ${today}\n\n## 오늘의 할 일\n\n- [ ] \n\n## 메모\n`,
            { visibility: "private", kind: "note" },
          ));
        if (doc) navigate("/app/documents/" + doc.id);
      },
    },
    {
      id: "templates",
      title: "템플릿 선택",
      kind: "문서",
      action: () => navigate("/app/templates"),
    },
    {
      id: "graph",
      title: "지식 그래프 열기",
      kind: "탐색",
      action: () => navigate("/app/graph"),
    },
    { id: "ai", title: "AI에게 물어보기", kind: "AI", action: onAI },
    {
      id: "help",
      title: "키보드 단축키 안내",
      kind: "설정",
      action: () => window.dispatchEvent(new Event("madi-shortcut-help")),
    },
    ...(current
      ? [
          ["save", "현재 문서 저장"],
          ["duplicate", "현재 문서 복제"],
          ["move", "현재 문서 이동"],
          ["export", "Markdown 내보내기"],
          ["print", "PDF 내보내기 · 인쇄"],
          ["copy", "문서 링크 복사"],
          ["focus", "집중 모드 전환"],
          ["present", "프레젠테이션 보기"],
        ].map(([id, title]) => ({
          id,
          title,
          kind: "현재 문서",
          action: () => documentCommand(id),
        }))
      : []),
    ...documents.map((doc) => ({
      id: doc.id,
      title: doc.title,
      kind: "문서 열기",
      action: () => navigate("/app/documents/" + doc.id),
    })),
  ];
  const filtered = commands
    .filter((command) =>
      (command.title + " " + command.kind)
        .toLowerCase()
        .includes(query.toLowerCase()),
    )
    .slice(0, 60);
  const selected = Math.min(active, filtered.length - 1);
  useEffect(() => {
    list.current
      ?.querySelector("[data-active=true]")
      ?.scrollIntoView({ block: "nearest" });
  }, [selected]);
  const execute = (action: () => unknown) => {
    onOpenChange(false);
    void Promise.resolve()
      .then(action)
      .catch((e) => notify(e.message, "error"));
  };
  return (
    <>
      <Modal
        open={open}
        onOpenChange={onOpenChange}
        title="어디로 연결할까요?"
        description="↑ ↓ 키로 선택하고 Enter로 실행하세요."
        wide
      >
        <div className="search-field">
          <Search size={20} />
          <input
            aria-label="문서 또는 명령 검색"
            role="combobox"
            aria-expanded={true}
            aria-controls="madi-command-options"
            aria-activedescendant={
              selected >= 0 ? "madi-command-" + selected : undefined
            }
            aria-autocomplete="list"
            autoFocus
            value={query}
            placeholder="문서 제목 또는 명령 검색"
            onChange={(e) => {
              setQuery(e.target.value);
              setActive(0);
            }}
            onKeyDown={(e) => {
              if (e.key === "ArrowDown" || e.key === "ArrowUp") {
                e.preventDefault();
                setActive((v) =>
                  Math.max(
                    0,
                    Math.min(
                      filtered.length - 1,
                      v + (e.key === "ArrowDown" ? 1 : -1),
                    ),
                  ),
                );
              }
              if (e.key === "Enter" && filtered[selected]) {
                e.preventDefault();
                execute(filtered[selected].action);
              }
            }}
          />
        </div>
        <div
          className="command-list navigation-command-list"
          id="madi-command-options"
          role="listbox"
          ref={list}
        >
          {filtered.map((command, index) => (
            <button
              key={command.id}
              id={"madi-command-" + index}
              role="option"
              aria-selected={index === selected}
              data-active={index === selected}
              onMouseMove={() => setActive(index)}
              onClick={() => execute(command.action)}
            >
              <FileText size={18} />
              <span>{command.title}</span>
              <small>{command.kind}</small>
              <ArrowRight size={16} />
            </button>
          ))}
        </div>
        {!filtered.length && (
          <p className="navigation-command-empty">
            일치하는 문서나 명령이 없습니다.
          </p>
        )}
      </Modal>
      <ShortcutHelp />
    </>
  );
}
