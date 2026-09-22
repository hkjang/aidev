import {
  lazy,
  Suspense,
  useEffect,
  useId,
  useState,
  type ReactNode,
} from "react";
import {
  Link2,
  SlidersHorizontal,
  MessageSquare,
  Sparkles,
  History,
  PanelRightClose,
  PanelRightOpen,
} from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { usePreferenceWriter } from "../navigation/preferences";
import { Button, ErrorBox, Loading } from "../ui";
import "./document-inspector.css";
const AI = lazy(() => import("../AI"));
export type InspectorTab =
  "backlinks" | "properties" | "comments" | "ai" | "versions";
const tabs = [
  { id: "backlinks", label: "연결", icon: Link2 },
  { id: "properties", label: "속성", icon: SlidersHorizontal },
  { id: "comments", label: "댓글", icon: MessageSquare },
  { id: "ai", label: "AI", icon: Sparkles },
  { id: "versions", label: "이력", icon: History },
] as const;
function validTab(value: unknown): InspectorTab {
  return tabs.some((t) => t.id === value)
    ? (value as InspectorTab)
    : "backlinks";
}
export function useDocumentInspector() {
  const { user, notify } = useApp(),
    write = usePreferenceWriter();
  const [tab, setTab] = useState<InspectorTab>(
      validTab(user.preferences?.document_panel),
    ),
    [open, setOpen] = useState(user.preferences?.document_panel_open !== false);
  useEffect(() => {
    setTab(validTab(user.preferences?.document_panel));
    setOpen(user.preferences?.document_panel_open !== false);
  }, [user.id]);
  const change = (next: InspectorTab) => {
    setTab(next);
    setOpen(true);
    void write({ document_panel: next, document_panel_open: true }).catch((e) =>
      notify(e.message, "error"),
    );
  };
  const toggle = (next: boolean) => {
    setOpen(next);
    void write({ document_panel_open: next }).catch((e) =>
      notify(e.message, "error"),
    );
  };
  return { tab, open, change, toggle };
}
export function InspectorToggle({
  open,
  onClick,
}: {
  open: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      onClick={onClick}
      aria-expanded={open}
      aria-controls="document-inspector"
    >
      {open ? <PanelRightClose size={17} /> : <PanelRightOpen size={17} />}문서
      패널
    </Button>
  );
}
export function DocumentInspector({
  documentID,
  version,
  tab,
  open,
  onChange,
  onOpen,
  backlinks,
  properties,
  comments,
  onHistory,
}: {
  documentID: string;
  version: number;
  tab: InspectorTab;
  open: boolean;
  onChange: (tab: InspectorTab) => void;
  onOpen: (v: boolean) => void;
  backlinks: ReactNode;
  properties: ReactNode;
  comments: ReactNode;
  onHistory: () => void;
}) {
  const id = useId();
  return (
    <aside
      id="document-inspector"
      className={`document-aside document-inspector ${open ? "" : "inspector-collapsed"}`}
      aria-label="문서 패널"
    >
      <div className="inspector-header">
        <strong>문서 패널</strong>
        <button
          type="button"
          className="icon-button"
          aria-label={open ? "문서 패널 접기" : "문서 패널 펼치기"}
          onClick={() => onOpen(!open)}
        >
          {open ? <PanelRightClose size={19} /> : <PanelRightOpen size={19} />}
        </button>
      </div>
      {open && (
        <>
          <div
            className="inspector-tabs"
            role="tablist"
            aria-label="문서 패널 선택"
          >
            {tabs.map((item, index) => (
              <button
                type="button"
                key={item.id}
                id={`${id}-${item.id}`}
                role="tab"
                aria-selected={tab === item.id}
                aria-controls={`${id}-content`}
                tabIndex={tab === item.id ? 0 : -1}
                onClick={() => onChange(item.id)}
                onKeyDown={(e) => {
                  if (
                    !["ArrowLeft", "ArrowRight", "Home", "End"].includes(e.key)
                  )
                    return;
                  e.preventDefault();
                  const target =
                    e.key === "Home"
                      ? 0
                      : e.key === "End"
                        ? tabs.length - 1
                        : (index +
                            (e.key === "ArrowRight" ? 1 : -1) +
                            tabs.length) %
                          tabs.length;
                  onChange(tabs[target].id);
                  document.getElementById(`${id}-${tabs[target].id}`)?.focus();
                }}
              >
                <item.icon size={18} />
                <span>{item.label}</span>
              </button>
            ))}
          </div>
          <div
            id={`${id}-content`}
            className="inspector-content"
            role="tabpanel"
            tabIndex={0}
            aria-labelledby={`${id}-${tab}`}
          >
            {tab === "backlinks" ? (
              backlinks
            ) : tab === "properties" ? (
              properties
            ) : tab === "comments" ? (
              comments
            ) : tab === "versions" ? (
              <VersionSummary
                key={documentID}
                documentID={documentID}
                version={version}
                onHistory={onHistory}
              />
            ) : (
              <Suspense fallback={<Loading />}>
                <AI
                  key={documentID}
                  embedded
                  open
                  onClose={() => {
                    onChange("backlinks");
                    document.getElementById(`${id}-backlinks`)?.focus();
                  }}
                  documentId={documentID}
                />
              </Suspense>
            )}
          </div>
        </>
      )}
    </aside>
  );
}
function VersionSummary({
  documentID,
  version,
  onHistory,
}: {
  documentID: string;
  version: number;
  onHistory: () => void;
}) {
  const [versions, setVersions] = useState<any[] | null>(null),
    [error, setError] = useState("");
  useEffect(() => {
    let active = true;
    setVersions(null);
    setError("");
    api<any[]>(`/documents/${documentID}/versions`)
      .then((data) => {
        if (active) setVersions(data);
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [documentID, version]);
  return (
    <section>
      <h3>변경 이력</h3>
      <p className="muted">
        현재 버전 {version} · 원문은 비교할 때만 불러옵니다.
      </p>
      <ErrorBox error={error} />
      {!versions && !error ? (
        <Loading />
      ) : (
        <ol className="inspector-versions">
          {versions?.slice(0, 8).map((v) => (
            <li key={v.version}>
              <strong>버전 {v.version}</strong>
              <span>{v.user_name || "저장 기록"}</span>
              <time>{datetime(v.created_at)}</time>
            </li>
          ))}
        </ol>
      )}
      <Button onClick={onHistory}>
        <History size={17} />
        변경 비교·복원 열기
      </Button>
    </section>
  );
}
