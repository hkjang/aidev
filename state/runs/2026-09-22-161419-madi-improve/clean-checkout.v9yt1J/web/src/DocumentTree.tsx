import { useEffect, useMemo, useRef, useState } from "react";
import { NavLink, useLocation } from "react-router-dom";
import { ChevronRight, FileText, History, Pin } from "lucide-react";
import { api, type Doc, type DocSummary } from "./api";
import { useApp } from "./context";
import DocumentActions, {
  moveCandidates,
  safeIDs,
} from "./navigation/DocumentActions";
import { usePreferenceWriter } from "./navigation/preferences";

export default function DocumentTree() {
  const writePreferences = usePreferenceWriter();
  const { documents, workspace, user, setUser, reload, notify } = useApp(),
    location = useLocation();
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>(
    user.preferences?.collapsed_documents || {},
  );
  const [filter, setFilter] = useState(
    ["all", "private", "shared"].includes(user.preferences?.document_filter)
      ? user.preferences.document_filter
      : "all",
  );
  const [dragging, setDragging] = useState(false),
    [moving, setMoving] = useState(false);
  const tracked = useRef("");
  const activeID = location.pathname.match(
    /^\/app\/documents\/([a-f0-9-]+)$/i,
  )?.[1];
  // Preferences store IDs only; current ACL-filtered API results supply titles.
  const scoped = useMemo(
    () =>
      documents.filter(
        (d) => d.workspace_id === workspace?.id && !d.deleted_at,
      ),
    [documents, workspace?.id],
  );
  useEffect(() => {
    if (
      !activeID ||
      activeID === tracked.current ||
      !scoped.some((d) => d.id === activeID)
    )
      return;
    tracked.current = activeID;
    const recent = [
      activeID,
      ...safeIDs(user.preferences?.recent_documents).filter(
        (id) => id !== activeID,
      ),
    ].slice(0, 60);
    void writePreferences({ recent_documents: recent }).catch((e) =>
      notify(e.message, "error"),
    );
  }, [activeID, scoped, user.id]);
  const toggle = (id: string) => {
    const next = { ...collapsed, [id]: !collapsed[id] };
    setCollapsed(next);
    const bounded = Object.fromEntries(
      Object.entries(next)
        .filter(([, value]) => value)
        .slice(-300),
    );
    void writePreferences({ collapsed_documents: bounded }).catch((e) =>
      notify(e.message, "error"),
    );
  };
  const items = scoped.filter(
    (d) =>
      filter === "all" ||
      (filter === "private"
        ? d.visibility === "private" && d.owner_id === user.id
        : d.visibility !== "private"),
  );
  const byParent = new Map<string, DocSummary[]>();
  for (const item of items) {
    const key =
      item.parent_id && items.some((parent) => parent.id === item.parent_id)
        ? item.parent_id
        : "";
    byParent.set(key, [...(byParent.get(key) || []), item]);
  }
  const move = async (e: React.DragEvent, parentID: string | null) => {
    e.preventDefault();
    e.stopPropagation();
    setDragging(false);
    if (moving) return;
    const id = e.dataTransfer.getData("text/madi-document");
    const existing = scoped.find((d) => d.id === id);
    if (!existing || id === parentID || existing.parent_id === parentID) return;
    if (
      parentID &&
      !moveCandidates(existing, scoped).some((d) => d.id === parentID)
    ) {
      notify("문서를 자기 하위 문서 안으로 이동할 수 없습니다.", "error");
      return;
    }
    setMoving(true);
    try {
      const fresh = await api<Doc>("/documents/" + id);
      if (!fresh.can_write) throw new Error("문서를 이동할 권한이 없습니다.");
      await api("/documents/" + id, "PUT", {
        version: fresh.version,
        parent_id: parentID,
      });
      await reload();
      notify("문서를 이동했습니다. 대상 상위 문서의 접근 권한이 적용됩니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setMoving(false);
    }
  };
  const row = (doc: DocSummary, depth: number, hasChildren: boolean) => (
    <div
      className="tree-row"
      style={{ paddingLeft: Math.min(depth, 12) * 10 }}
      onContextMenu={(e) => {
        e.preventDefault();
        e.currentTarget
          .querySelector<HTMLButtonElement>("[data-doc-menu]")
          ?.dispatchEvent(new Event("madi-open-context"));
      }}
      onKeyDown={(e) => {
        if (e.key === "ContextMenu" || (e.shiftKey && e.key === "F10")) {
          e.preventDefault();
          e.currentTarget
            .querySelector<HTMLButtonElement>("[data-doc-menu]")
            ?.dispatchEvent(new Event("madi-open-context"));
        }
      }}
      onDragOver={(e) => {
        if (e.dataTransfer.types.includes("text/madi-document"))
          e.preventDefault();
      }}
      onDrop={(e) => void move(e, doc.id)}
    >
      <button
        className={`tree-toggle ${hasChildren ? "" : "leaf"}`}
        tabIndex={-1}
        aria-label={`${doc.title} ${collapsed[doc.id] ? "펼치기" : "접기"}`}
        aria-expanded={hasChildren ? !collapsed[doc.id] : undefined}
        disabled={!hasChildren}
        onClick={() => toggle(doc.id)}
      >
        <ChevronRight
          size={14}
          style={{
            transform: !collapsed[doc.id] ? "rotate(90deg)" : undefined,
          }}
        />
      </button>
      <NavLink
        data-tree-link={doc.id}
        draggable={!moving}
        onDragStart={(e) => {
          e.dataTransfer.setData("text/madi-document", doc.id);
          e.dataTransfer.effectAllowed = "move";
          setDragging(true);
        }}
        onDragEnd={() => setDragging(false)}
        to={`/app/documents/${doc.id}`}
      >
        <FileText size={16} />
        <span>{doc.title || "제목 없는 문서"}</span>
      </NavLink>
      <DocumentActions doc={doc} />
    </div>
  );
  const render = (doc: DocSummary, depth: number): React.ReactNode => {
    if (depth > 32) return null;
    const children = byParent.get(doc.id) || [];
    return (
      <div
        key={doc.id}
        role="treeitem"
        aria-level={depth + 1}
        aria-selected={activeID === doc.id}
        aria-expanded={children.length ? !collapsed[doc.id] : undefined}
        className="tree-branch"
      >
        {row(doc, depth, !!children.length)}
        {!collapsed[doc.id] && !!children.length && (
          <div role="group">
            {children.map((child) => render(child, depth + 1))}
          </div>
        )}
      </div>
    );
  };
  const quick = (ids: string[], label: string, icon: React.ReactNode) => {
    const selected = ids
      .map((id) => scoped.find((d) => d.id === id))
      .filter((d): d is DocSummary => !!d)
      .slice(0, label === "최근 문서" ? 5 : 12);
    return (
      selected.length > 0 && (
        <>
          <div className="tree-section-label">
            {icon}
            {label}
          </div>
          <div className="tree-quick-links">
            {selected.map((d) => (
              <div key={d.id}>{row(d, 0, false)}</div>
            ))}
          </div>
        </>
      )
    );
  };
  return (
    <nav
      className="document-nav document-tree"
      aria-label="문서 트리"
      onKeyDown={(e) => {
        const target = (e.target as HTMLElement).closest<HTMLAnchorElement>(
          "a[data-tree-link]",
        );
        if (!target) return;
        const links = [
            ...e.currentTarget.querySelectorAll<HTMLAnchorElement>(
              "a[data-tree-link]",
            ),
          ],
          index = links.indexOf(target),
          id = target.dataset.treeLink!;
        if (
          ![
            "ArrowDown",
            "ArrowUp",
            "Home",
            "End",
            "ArrowLeft",
            "ArrowRight",
          ].includes(e.key)
        )
          return;
        e.preventDefault();
        if (e.key === "ArrowDown")
          links[Math.min(links.length - 1, index + 1)]?.focus();
        else if (e.key === "ArrowUp") links[Math.max(0, index - 1)]?.focus();
        else if (e.key === "Home") links[0]?.focus();
        else if (e.key === "End") links.at(-1)?.focus();
        else if (e.key === "ArrowRight") {
          if (byParent.get(id)?.length && collapsed[id]) toggle(id);
          else
            target
              .closest("[role=treeitem]")
              ?.querySelector<HTMLAnchorElement>(
                "[role=group] a[data-tree-link]",
              )
              ?.focus();
        } else if (e.key === "ArrowLeft") {
          if (byParent.get(id)?.length && !collapsed[id]) toggle(id);
          else
            target
              .closest("[role=treeitem]")
              ?.parentElement?.closest("[role=treeitem]")
              ?.querySelector<HTMLAnchorElement>("a[data-tree-link]")
              ?.focus();
        }
      }}
    >
      <div className="navigation-filter">
        <select
          aria-label="문서 탐색 필터"
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value);
            void writePreferences({ document_filter: e.target.value }).catch(
              (e) => notify(e.message, "error"),
            );
          }}
        >
          <option value="all">모든 문서</option>
          <option value="private">내 개인 문서</option>
          <option value="shared">공유 문서</option>
        </select>
      </div>
      {quick(
        safeIDs(user.preferences?.pinned_documents),
        "고정 문서",
        <Pin size={13} />,
      )}
      {quick(
        safeIDs(user.preferences?.recent_documents),
        "최근 문서",
        <History size={13} />,
      )}
      <div className="tree-section-label">문서 계층</div>
      <div role="tree" aria-label="페이지 계층">
        {(byParent.get("") || []).map((d) => render(d, 0))}
      </div>
      {!items.length && (
        <p className="sidebar-empty">이 범위에 표시할 문서가 없습니다.</p>
      )}
      {dragging && (
        <div
          className="tree-drop-root"
          onDragOver={(e) => e.preventDefault()}
          onDrop={(e) => void move(e, null)}
        >
          여기에 놓아 최상위로 이동
        </div>
      )}
    </nav>
  );
}
