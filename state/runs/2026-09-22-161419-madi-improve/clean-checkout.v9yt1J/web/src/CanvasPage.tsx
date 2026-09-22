import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react";
import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import {
  ArrowLeft,
  ArrowUpRight,
  Check,
  ChevronDown,
  Copy,
  Database,
  Diamond,
  Download,
  Expand,
  FileText,
  Hand,
  Image,
  LayoutDashboard,
  Link2,
  LoaderCircle,
  Lock,
  MousePointer2,
  Network,
  Palette,
  PenTool,
  Plus,
  Redo2,
  RefreshCw,
  Save,
  Share2,
  Sparkles,
  Square,
  StickyNote,
  Trash2,
  Undo2,
  Upload,
  X,
  ZoomIn,
  ZoomOut,
  Circle,
  MoveDiagonal,
  GitBranch,
} from "lucide-react";
import { api, date, type Doc } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  CopyButton,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "./ui";
import DiagramPreview from "./DiagramPreview";
import { MarkdownContent as MarkdownView } from "./editor/MarkdownContent";

type NodeKind =
  | "note"
  | "document"
  | "image"
  | "url"
  | "database"
  | "ai"
  | "diagram"
  | "rectangle"
  | "ellipse"
  | "diamond"
  | "drawing";
type NodeColor = "mint" | "lavender" | "sand" | "sky" | "rose" | "white";
type CanvasNode = {
  id: string;
  kind: NodeKind;
  x: number;
  y: number;
  width: number;
  height: number;
  color: NodeColor;
  title?: string;
  text?: string;
  ref_id?: string;
  url?: string;
  points?: [number, number][];
};
type CanvasEdge = {
  id: string;
  source: string;
  target: string;
  label?: string;
  color: NodeColor;
};
type CanvasData = { nodes: CanvasNode[]; edges: CanvasEdge[] };
type CanvasRecord = {
  id: string;
  workspace_id: string;
  space_id?: string;
  owner_id: string;
  title: string;
  visibility: "private" | "workspace" | "selected";
  data: CanvasData;
  version: number;
  updated_at: string;
  deleted_at?: string;
  can_write: boolean;
  can_manage: boolean;
  can_restore?: boolean;
  node_count?: number;
  resolved?: Record<
    string,
    {
      title: string;
      snippet?: string;
      url?: string;
      unavailable?: boolean;
      error?: string;
    }
  >;
};
const emptyData: CanvasData = { nodes: [], edges: [] };
const colors: Record<NodeColor, { fill: string; line: string }> = {
  mint: { fill: "#eaf4ee", line: "#3b8064" },
  lavender: { fill: "#f0ecfa", line: "#8f78b3" },
  sand: { fill: "#fbf0d9", line: "#ac8648" },
  sky: { fill: "#e9f3fa", line: "#628aab" },
  rose: { fill: "#faeceb", line: "#b57a75" },
  white: { fill: "#ffffff", line: "#87968c" },
};
const kindLabels: Record<NodeKind, string> = {
  note: "메모",
  document: "문서",
  image: "이미지",
  url: "링크",
  database: "데이터베이스",
  ai: "AI 질문",
  diagram: "Mermaid",
  rectangle: "사각형",
  ellipse: "타원",
  diamond: "마름모",
  drawing: "펜 드로잉",
};
const nodeIcon = (kind: NodeKind) =>
  ({
    note: StickyNote,
    document: FileText,
    image: Image,
    url: Link2,
    database: Database,
    ai: Sparkles,
    diagram: GitBranch,
    rectangle: Square,
    ellipse: Circle,
    diamond: Diamond,
    drawing: PenTool,
  })[kind];
const snapshot = (
  data: CanvasData,
  title: string,
  visibility: string,
  space: string,
) => JSON.stringify({ data, title, visibility, space_id: space });
function downloadJSON(name: string, value: any) {
  const blob = new Blob([JSON.stringify(value, null, 2)], {
      type: "application/json",
    }),
    url = URL.createObjectURL(blob),
    anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = name;
  anchor.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
const safeLink = (value?: string) => {
  try {
    const url = new URL(value || "");
    return ["http:", "https:"].includes(url.protocol) &&
      !url.username &&
      !url.password
      ? url.href
      : "";
  } catch {
    return "";
  }
};

export default function CanvasPage() {
  const { workspace, notify, user } = useApp(),
    navigate = useNavigate(),
    params = useParams();
  const [searchParams] = useSearchParams();
  const id = params["*"]?.split("/")[0] || "",
    trash = searchParams.get("trash") === "true";
  const [items, setItems] = useState<CanvasRecord[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [newOpen, setNewOpen] = useState(false),
    [title, setTitle] = useState(""),
    [visibility, setVisibility] = useState("private"),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    if (!workspace) return;
    setLoading(true);
    api<CanvasRecord[]>(`/canvases?workspace_id=${workspace.id}&trash=${trash}`)
      .then((v) => {
        if (active) {
          setItems(v);
          setError("");
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [workspace?.id, trash, id]);
  if (id)
    return (
      <>
        <CanvasStyles />
        <CanvasEditor key={id} id={id} />
      </>
    );
  return (
    <>
      <CanvasStyles />
      <PageHeading
        eyebrow="SPACE FOR CONNECTED IDEAS"
        title={trash ? "캔버스 휴지통" : "캔버스"}
        description="문서와 아이디어를 자유롭게 펼치고, 새로운 연결을 발견하세요."
        actions={
          <>
            <Link
              className="button secondary"
              to={trash ? "/app/canvases" : "/app/canvases?trash=true"}
            >
              {trash ? <LayoutDashboard size={17} /> : <Trash2 size={17} />}{" "}
              {trash ? "캔버스 목록" : "휴지통"}
            </Link>
            {!trash && user.role !== "viewer" && (
              <Button variant="primary" onClick={() => setNewOpen(true)}>
                <Plus size={18} />새 캔버스
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : (
        <div className="canvas-card-grid">
          {items.map((c) => (
            <article className="panel canvas-list-card" key={c.id}>
              <Link to={"/app/canvases/" + c.id}>
                <div className="canvas-card-cover">
                  <span />
                  <span />
                  <span />
                  <Network size={31} />
                </div>
                <div className="canvas-card-body">
                  <h2>{c.title}</h2>
                  <p>
                    {c.node_count || 0}개 노드 · {date(c.updated_at)}
                  </p>
                  <Badge>
                    {c.visibility === "private"
                      ? "나만 보기"
                      : c.visibility === "selected"
                        ? "선택한 사용자"
                        : "워크스페이스"}
                  </Badge>
                </div>
              </Link>
              {trash && c.can_restore && (
                <Button
                  onClick={async () => {
                    try {
                      await api(`/canvases/${c.id}/restore`, "POST", {});
                      setItems(items.filter((v) => v.id !== c.id));
                      notify("캔버스를 복원했습니다.");
                    } catch (e) {
                      notify((e as Error).message, "error");
                    }
                  }}
                >
                  <RefreshCw size={15} />
                  복원
                </Button>
              )}
            </article>
          ))}
          {!items.length && (
            <Empty
              title={
                trash
                  ? "휴지통이 비어 있습니다"
                  : "아이디어가 연결되는 넓은 공간"
              }
              text={
                trash
                  ? "삭제한 캔버스를 이곳에서 복원할 수 있습니다."
                  : "메모, 문서, 이미지와 다이어그램을 한 화면에 배치하세요."
              }
              action={
                !trash && (
                  <Button variant="primary" onClick={() => setNewOpen(true)}>
                    <Plus size={17} />첫 캔버스 만들기
                  </Button>
                )
              }
            />
          )}
        </div>
      )}
      <Modal open={newOpen} onOpenChange={setNewOpen} title="새 캔버스">
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (!workspace) return;
            setBusy(true);
            try {
              const c = await api<CanvasRecord>("/canvases", "POST", {
                workspace_id: workspace.id,
                title,
                visibility,
                data: emptyData,
              });
              setNewOpen(false);
              navigate("/app/canvases/" + c.id);
              notify("캔버스를 만들었습니다.");
            } catch (e) {
              notify((e as Error).message, "error");
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field label="캔버스 이름">
            <input
              required
              maxLength={150}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="예: AI 플랫폼 아이디어 맵"
            />
          </Field>
          <Field label="공개 범위">
            <select
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
            >
              <option value="private">나만 보기</option>
              <option value="workspace">워크스페이스 공유</option>
              <option value="selected">선택한 사용자와 공유</option>
            </select>
          </Field>
          <div className="modal-actions">
            <Button variant="primary" disabled={busy || !title.trim()}>
              만들기
            </Button>
          </div>
        </form>
      </Modal>
    </>
  );
}

type Tool =
  "select" | "pan" | "connect" | "pen" | "rectangle" | "ellipse" | "diamond";
type Interaction = {
  kind: "pan" | "move" | "resize" | "draw" | "shape" | "marquee";
  start: { x: number; y: number };
  before: CanvasData;
  pan?: { x: number; y: number };
  ids?: string[];
  id?: string;
  shift?: boolean;
};
function CanvasEditor({ id }: { id: string }) {
  const { notify, user, documents, workspace } = useApp(),
    navigate = useNavigate();
  const [canvas, setCanvas] = useState<CanvasRecord | null>(null),
    [data, setData] = useState<CanvasData>(emptyData),
    [title, setTitle] = useState(""),
    [visibility, setVisibility] =
      useState<CanvasRecord["visibility"]>("private"),
    [space, setSpace] = useState("");
  const [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [saving, setSaving] = useState(false),
    [dirty, setDirty] = useState(false),
    [conflict, setConflict] = useState(false),
    [draft, setDraft] = useState<any>(null);
  const [tool, setTool] = useState<Tool>("select"),
    [interacting, setInteracting] = useState(false),
    [color, setColor] = useState<NodeColor>("mint"),
    [selection, setSelection] = useState<string[]>([]),
    [edgeSelection, setEdgeSelection] = useState(""),
    [pan, setPan] = useState({ x: 80, y: 70 }),
    [scale, setScale] = useState(0.85),
    [marquee, setMarquee] = useState<{
      x: number;
      y: number;
      width: number;
      height: number;
    } | null>(null),
    [historyCount, setHistoryCount] = useState({ undo: 0, redo: 0 });
  const [addKind, setAddKind] = useState<NodeKind | null>(null),
    [editingNode, setEditingNode] = useState<CanvasNode | null>(null),
    [shareOpen, setShareOpen] = useState(false),
    [edgeEdit, setEdgeEdit] = useState<CanvasEdge | null>(null),
    [aiNode, setAINode] = useState<CanvasNode | null>(null);
  const svg = useRef<SVGSVGElement | null>(null),
    stage = useRef<HTMLDivElement | null>(null),
    dataRef = useRef(data),
    canvasRef = useRef(canvas),
    metaRef = useRef({ title, visibility, space }),
    interaction = useRef<Interaction | null>(null),
    spaceHeld = useRef(false),
    alive = useRef(true),
    saved = useRef(""),
    savePromise = useRef<Promise<boolean> | null>(null),
    undo = useRef<CanvasData[]>([]),
    redo = useRef<CanvasData[]>([]);
  dataRef.current = data;
  canvasRef.current = canvas;
  metaRef.current = { title, visibility, space };
  const draftKey = `madi.canvas.${user.id}.${id}`;
  const syncHistory = () =>
    setHistoryCount({ undo: undo.current.length, redo: redo.current.length });
  const record = (before: CanvasData) => {
    undo.current = [...undo.current.slice(-49), structuredClone(before)];
    redo.current = [];
    syncHistory();
  };
  const updateData = (next: CanvasData, recordChange = true) => {
    if (!canvasRef.current?.can_write) return;
    if (recordChange) record(dataRef.current);
    dataRef.current = next;
    setData(next);
    setDirty(true);
  };
  const load = async () => {
    setLoading(true);
    setError("");
    try {
      const c = await api<CanvasRecord>("/canvases/" + id);
      if (!alive.current) return;
      setCanvas(c);
      canvasRef.current = c;
      setData(c.data);
      dataRef.current = c.data;
      setTitle(c.title);
      setVisibility(c.visibility);
      setSpace(c.space_id || "");
      metaRef.current = {
        title: c.title,
        visibility: c.visibility,
        space: c.space_id || "",
      };
      saved.current = snapshot(c.data, c.title, c.visibility, c.space_id || "");
      setDirty(false);
      setConflict(false);
      setSelection([]);
      undo.current = [];
      redo.current = [];
      syncHistory();
      try {
        const value = JSON.parse(sessionStorage.getItem(draftKey) || "null");
        if (
          value?.data &&
          snapshot(
            value.data,
            value.title,
            value.visibility,
            value.space_id || "",
          ) !== saved.current
        )
          setDraft(value);
      } catch {}
    } catch (e) {
      if (alive.current) setError((e as Error).message);
    } finally {
      if (alive.current) setLoading(false);
    }
  };
  useEffect(() => {
    alive.current = true;
    load();
    return () => {
      alive.current = false;
    };
  }, [id]);
  const save = async (): Promise<boolean> => {
    if (savePromise.current) {
      await savePromise.current;
      if (!alive.current) return false;
      if (
        snapshot(
          dataRef.current,
          metaRef.current.title,
          metaRef.current.visibility,
          metaRef.current.space,
        ) === saved.current
      )
        return true;
    }
    const c = canvasRef.current,
      m = metaRef.current,
      currentData = structuredClone(dataRef.current);
    if (!c?.can_write || conflict) return false;
    const encoded = snapshot(currentData, m.title, m.visibility, m.space);
    if (encoded === saved.current) {
      setDirty(false);
      sessionStorage.removeItem(draftKey);
      return true;
    }
    const request = (async () => {
      setSaving(true);
      try {
        const result = await api<CanvasRecord>("/canvases/" + id, "PUT", {
          version: c.version,
          title: m.title,
          visibility: m.visibility,
          space_id: m.space,
          data: currentData,
        });
        if (!alive.current) return false;
        canvasRef.current = result;
        setCanvas(result);
        saved.current = encoded;
        const stillDirty =
          snapshot(
            dataRef.current,
            metaRef.current.title,
            metaRef.current.visibility,
            metaRef.current.space,
          ) !== encoded;
        setDirty(stillDirty);
        setError("");
        if (!stillDirty) {
          sessionStorage.removeItem(draftKey);
          setDraft(null);
        }
        return true;
      } catch (e) {
        if (alive.current) {
          const message = (e as Error).message;
          setError(message);
          if (
            message.includes("다른 사용자") ||
            message.includes("변경했습니다")
          )
            setConflict(true);
        }
        return false;
      } finally {
        savePromise.current = null;
        if (alive.current) setSaving(false);
      }
    })();
    savePromise.current = request;
    return request;
  };
  useEffect(() => {
    if (!canvas || !dirty) return;
    try {
      sessionStorage.setItem(
        draftKey,
        JSON.stringify({
          data,
          title,
          visibility,
          space_id: space,
          base_version: canvas.version,
          saved_at: Date.now(),
        }),
      );
    } catch {
      setError(
        "브라우저 임시 저장 공간이 부족합니다. JSON으로 내보낸 뒤 저장하세요.",
      );
    }
    if (conflict || interacting) return;
    const timer = setTimeout(() => {
      void save();
    }, 1800);
    return () => clearTimeout(timer);
  }, [data, title, visibility, space, dirty, conflict, interacting]);
  useEffect(() => {
    const before = (event: BeforeUnloadEvent) => {
      if (dirty) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", before);
    return () => window.removeEventListener("beforeunload", before);
  }, [dirty]);
  const removeSelection = () => {
    if (!canvasRef.current?.can_write) return;
    if (!selection.length && !edgeSelection) return;
    const ids = new Set(selection);
    updateData({
      nodes: dataRef.current.nodes.filter((n) => !ids.has(n.id)),
      edges: dataRef.current.edges.filter(
        (e) =>
          !ids.has(e.source) && !ids.has(e.target) && e.id !== edgeSelection,
      ),
    });
    setSelection([]);
    setEdgeSelection("");
  };
  const changeHistory = (direction: "undo" | "redo") => {
    const source = direction === "undo" ? undo : redo,
      destination = direction === "undo" ? redo : undo;
    if (!source.current.length || !canvasRef.current?.can_write) return;
    destination.current.push(structuredClone(dataRef.current));
    const next = source.current.pop()!;
    updateData(next, false);
    syncHistory();
    setSelection([]);
  };
  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (
        (e.target as HTMLElement).closest(
          "input,textarea,select,[contenteditable=true],[role=dialog]",
        )
      )
        return;
      if (e.code === "Space") {
        spaceHeld.current = true;
        e.preventDefault();
      }
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        e.preventDefault();
        void save();
      }
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "z") {
        e.preventDefault();
        changeHistory(e.shiftKey ? "redo" : "undo");
      }
      if (["Delete", "Backspace"].includes(e.key)) {
        e.preventDefault();
        removeSelection();
      }
      if (e.key === "Escape") {
        setSelection([]);
        setEdgeSelection("");
        setTool("select");
      }
    };
    const up = (e: KeyboardEvent) => {
      if (e.code === "Space") spaceHeld.current = false;
    };
    window.addEventListener("keydown", down);
    window.addEventListener("keyup", up);
    return () => {
      window.removeEventListener("keydown", down);
      window.removeEventListener("keyup", up);
    };
  }, [selection, edgeSelection, conflict]);
  const point = (event: { clientX: number; clientY: number }) => {
    const rect = svg.current!.getBoundingClientRect();
    return {
      x: Math.max(
        -90000,
        Math.min(90000, (event.clientX - rect.left - pan.x) / scale),
      ),
      y: Math.max(
        -90000,
        Math.min(90000, (event.clientY - rect.top - pan.y) / scale),
      ),
    };
  };
  const zoom = (factor: number, anchor?: { x: number; y: number }) => {
    const rect = stage.current?.getBoundingClientRect();
    if (!rect) return;
    const center = anchor || { x: rect.width / 2, y: rect.height / 2 };
    const next = Math.min(2.5, Math.max(0.15, scale * factor));
    setPan({
      x: center.x - ((center.x - pan.x) * next) / scale,
      y: center.y - ((center.y - pan.y) * next) / scale,
    });
    setScale(next);
  };
  useEffect(() => {
    const element = stage.current;
    if (!element) return;
    const wheel = (event: WheelEvent) => {
      event.preventDefault();
      if (event.ctrlKey || event.metaKey) {
        const rect = element.getBoundingClientRect();
        zoom(event.deltaY < 0 ? 1.1 : 1 / 1.1, {
          x: event.clientX - rect.left,
          y: event.clientY - rect.top,
        });
      } else
        setPan((prev) => ({
          x: prev.x - event.deltaX,
          y: prev.y - event.deltaY,
        }));
    };
    element.addEventListener("wheel", wheel, { passive: false });
    return () => element.removeEventListener("wheel", wheel);
  }, [scale, pan, loading]);
  const fit = () => {
    const nodes = dataRef.current.nodes,
      rect = stage.current?.getBoundingClientRect();
    if (!nodes.length || !rect) {
      setPan({ x: 80, y: 70 });
      setScale(0.85);
      return;
    }
    const minX = Math.min(...nodes.map((n) => n.x)),
      minY = Math.min(...nodes.map((n) => n.y)),
      maxX = Math.max(...nodes.map((n) => n.x + n.width)),
      maxY = Math.max(...nodes.map((n) => n.y + n.height)),
      next = Math.min(
        1.2,
        Math.max(
          0.15,
          Math.min(
            (rect.width - 100) / (maxX - minX),
            (rect.height - 100) / (maxY - minY),
          ),
        ),
      );
    setScale(next);
    setPan({
      x: (rect.width - (maxX - minX) * next) / 2 - minX * next,
      y: (rect.height - (maxY - minY) * next) / 2 - minY * next,
    });
  };
  const nodeDown = (
    event: ReactPointerEvent<SVGGElement>,
    node: CanvasNode,
    resize = false,
  ) => {
    if ((event.target as Element).closest("button,a,input,textarea")) return;
    event.stopPropagation();
    if (tool === "pan" || spaceHeld.current) {
      pointerDown(event as unknown as ReactPointerEvent<SVGSVGElement>);
      return;
    }
    if (tool === "connect" && canvas?.can_write) {
      if (selection.length === 1 && selection[0] !== node.id) {
        updateData({
          ...dataRef.current,
          edges: [
            ...dataRef.current.edges,
            {
              id: crypto.randomUUID(),
              source: selection[0],
              target: node.id,
              color,
            },
          ],
        });
        setTool("select");
        setSelection([]);
      } else setSelection([node.id]);
      return;
    }
    let ids = selection.includes(node.id)
      ? selection
      : event.shiftKey
        ? [...selection, node.id]
        : [node.id];
    if (event.shiftKey && selection.includes(node.id))
      ids = selection.filter((id) => id !== node.id);
    setSelection(ids);
    setEdgeSelection("");
    if (!canvas?.can_write) return;
    setInteracting(true);
    interaction.current = {
      kind: resize ? "resize" : "move",
      start: point(event),
      before: structuredClone(dataRef.current),
      ids,
      id: node.id,
    };
    svg.current?.setPointerCapture(event.pointerId);
  };
  const pointerDown = (event: ReactPointerEvent<SVGSVGElement>) => {
    if (event.button !== 0 && event.button !== 1) return;
    setInteracting(true);
    const start = point(event),
      before = structuredClone(dataRef.current);
    svg.current?.setPointerCapture(event.pointerId);
    if (tool === "pan" || spaceHeld.current || event.button === 1) {
      interaction.current = {
        kind: "pan",
        start: { x: event.clientX, y: event.clientY },
        before,
        pan,
      };
      return;
    }
    if (!canvas?.can_write || tool === "select" || tool === "connect") {
      interaction.current = {
        kind: "marquee",
        start,
        before,
        shift: event.shiftKey,
      };
      setMarquee({ x: start.x, y: start.y, width: 0, height: 0 });
      if (!event.shiftKey) setSelection([]);
      setEdgeSelection("");
      return;
    }
    const kind = tool === "pen" ? "drawing" : tool;
    const node: CanvasNode = {
      id: crypto.randomUUID(),
      kind,
      x: start.x,
      y: start.y,
      width: 40,
      height: 40,
      color,
      ...(kind === "drawing" ? { points: [[0, 0] as [number, number]] } : {}),
    };
    updateData({ ...before, nodes: [...before.nodes, node] }, false);
    setSelection([node.id]);
    interaction.current = {
      kind: kind === "drawing" ? "draw" : "shape",
      start,
      before,
      id: node.id,
    };
  };
  const pointerMove = (event: ReactPointerEvent<SVGSVGElement>) => {
    const state = interaction.current;
    if (!state) return;
    if (state.kind === "pan") {
      setPan({
        x: state.pan!.x + event.clientX - state.start.x,
        y: state.pan!.y + event.clientY - state.start.y,
      });
      return;
    }
    const current = point(event),
      dx = current.x - state.start.x,
      dy = current.y - state.start.y;
    if (state.kind === "marquee") {
      setMarquee({
        x: Math.min(current.x, state.start.x),
        y: Math.min(current.y, state.start.y),
        width: Math.abs(dx),
        height: Math.abs(dy),
      });
      return;
    }
    let nodes = dataRef.current.nodes;
    if (state.kind === "move")
      nodes = nodes.map((n) => {
        const original = state.before.nodes.find((v) => v.id === n.id);
        return state.ids?.includes(n.id) && original
          ? {
              ...n,
              x: Math.max(-90000, Math.min(90000, original.x + dx)),
              y: Math.max(-90000, Math.min(90000, original.y + dy)),
            }
          : n;
      });
    if (state.kind === "resize")
      nodes = nodes.map((n) => {
        const original = state.before.nodes.find((v) => v.id === n.id);
        return n.id === state.id && original
          ? {
              ...n,
              width: Math.min(4000, Math.max(80, original.width + dx)),
              height: Math.min(4000, Math.max(60, original.height + dy)),
            }
          : n;
      });
    if (state.kind === "shape")
      nodes = nodes.map((n) =>
        n.id === state.id
          ? {
              ...n,
              x: Math.min(current.x, state.start.x),
              y: Math.min(current.y, state.start.y),
              width: Math.min(4000, Math.max(40, Math.abs(dx))),
              height: Math.min(4000, Math.max(40, Math.abs(dy))),
            }
          : n,
      );
    if (state.kind === "draw")
      nodes = nodes.map((n) =>
        n.id === state.id && (n.points?.length || 0) < 2000
          ? {
              ...n,
              points: [
                ...(n.points || []),
                [
                  Math.max(-2000, Math.min(2000, dx)),
                  Math.max(-2000, Math.min(2000, dy)),
                ],
              ],
            }
          : n,
      );
    updateData({ ...dataRef.current, nodes }, false);
  };
  const pointerUp = () => {
    setInteracting(false);
    const state = interaction.current;
    interaction.current = null;
    if (!state) return;
    if (state.kind === "marquee" && marquee) {
      const ids = state.before.nodes
        .filter(
          (n) =>
            n.x + n.width >= marquee.x &&
            n.x <= marquee.x + marquee.width &&
            n.y + n.height >= marquee.y &&
            n.y <= marquee.y + marquee.height,
        )
        .map((n) => n.id);
      setSelection(state.shift ? [...new Set([...selection, ...ids])] : ids);
      setMarquee(null);
      return;
    }
    if (state.kind === "draw") {
      const node = dataRef.current.nodes.find((n) => n.id === state.id),
        points = node?.points || [];
      if (node && points.length) {
        const minX = Math.min(...points.map((p) => p[0])),
          minY = Math.min(...points.map((p) => p[1])),
          maxX = Math.max(...points.map((p) => p[0])),
          maxY = Math.max(...points.map((p) => p[1]));
        updateData(
          {
            ...dataRef.current,
            nodes: dataRef.current.nodes.map((n) =>
              n.id === node.id
                ? {
                    ...n,
                    x: n.x + minX,
                    y: n.y + minY,
                    width: Math.max(40, maxX - minX),
                    height: Math.max(40, maxY - minY),
                    points: points.map((p) => [p[0] - minX, p[1] - minY]),
                  }
                : n,
            ),
          },
          false,
        );
      }
    }
    if (
      state.kind !== "pan" &&
      JSON.stringify(state.before) !== JSON.stringify(dataRef.current)
    )
      record(state.before);
    if (["draw", "shape"].includes(state.kind)) setTool("select");
  };
  const addNode = (node: Partial<CanvasNode>) => {
    const rect = stage.current?.getBoundingClientRect();
    const next: CanvasNode = {
      id: crypto.randomUUID(),
      kind: node.kind || "note",
      x: ((rect?.width || 800) / 2 - pan.x) / scale - 150,
      y: ((rect?.height || 600) / 2 - pan.y) / scale - 100,
      width: 310,
      height: 230,
      color,
      ...node,
    };
    updateData({ ...dataRef.current, nodes: [...dataRef.current.nodes, next] });
    setSelection([next.id]);
    setAddKind(null);
  };
  const saveNode = (node: CanvasNode) => {
    updateData({
      ...dataRef.current,
      nodes: dataRef.current.nodes.map((n) => (n.id === node.id ? node : n)),
    });
    setEditingNode(null);
  };
  const exportCanvas = () =>
    downloadJSON(`${title || "madi-canvas"}.canvas.json`, {
      format: "madi-canvas",
      version: 1,
      title,
      visibility,
      data,
    });
  if (loading) return <Loading />;
  if (!canvas) return <ErrorBox error={error} />;
  return (
    <div className="canvas-editor-page">
      <div className="canvas-heading">
        <Link
          className="icon-button"
          aria-label="캔버스 목록"
          to="/app/canvases"
        >
          <ArrowLeft size={21} />
        </Link>
        <div>
          <input
            aria-label="캔버스 제목"
            value={title}
            disabled={!canvas.can_write}
            onChange={(e) => {
              setTitle(e.target.value);
              setDirty(true);
            }}
          />
          <p>
            <span className="canvas-save-state">
              {saving
                ? "저장 중…"
                : conflict
                  ? "저장 충돌"
                  : dirty
                    ? "저장 대기"
                    : "저장됨"}
            </span>
            <span>
              ·{" "}
              {canvas.visibility === "private"
                ? "나만 보기"
                : canvas.visibility === "selected"
                  ? "선택한 사용자"
                  : "워크스페이스 공유"}
            </span>
          </p>
        </div>
        <div className="canvas-heading-actions">
          <Button onClick={exportCanvas}>
            <Download size={16} />
            내보내기
          </Button>
          {canvas.can_manage && (
            <Button onClick={() => setShareOpen(true)}>
              <Share2 size={16} />
              공유
            </Button>
          )}
          {canvas.can_write && (
            <Button
              variant="primary"
              disabled={saving || conflict}
              onClick={() => {
                void save();
              }}
            >
              <Save size={17} />
              저장
            </Button>
          )}
        </div>
      </div>
      <ErrorBox error={error} />
      {canvas.deleted_at && (
        <div className="notice error">
          <Trash2 size={19} />
          <span>휴지통에 있는 캔버스입니다.</span>
          <Button
            disabled={!canvas.can_restore}
            onClick={async () => {
              try {
                await api(`/canvases/${id}/restore`, "POST", {});
                await load();
              } catch (e) {
                notify((e as Error).message, "error");
              }
            }}
          >
            복원
          </Button>
        </div>
      )}
      {draft && (
        <div className="notice subtle">
          <RefreshCw size={19} />
          <span>
            이 브라우저에 미저장 작업이 남아 있습니다. 최신 내용과 비교한 뒤
            복구하세요.
          </span>
          <Button
            onClick={() => {
              if (
                draft.base_version !== canvas.version &&
                !window.confirm(
                  "서버의 내용이 이후 변경되었습니다. 임시 작업을 복원하면 충돌 상태로 열립니다. JSON 내보내기로 작업을 보존할 수 있습니다. 복원할까요?",
                )
              )
                return;
              setData(draft.data);
              dataRef.current = draft.data;
              setTitle(draft.title);
              setVisibility(draft.visibility);
              setSpace(draft.space_id || "");
              setDirty(true);
              if (draft.base_version !== canvas.version) setConflict(true);
              setDraft(null);
            }}
          >
            복구
          </Button>
          <Button
            onClick={() => {
              sessionStorage.removeItem(draftKey);
              setDraft(null);
            }}
          >
            버리기
          </Button>
        </div>
      )}
      {conflict && (
        <div className="notice error">
          <span>미저장 작업을 먼저 JSON으로 내보낼 수 있습니다.</span>
          <Button onClick={exportCanvas}>작업 내보내기</Button>
          <Button
            onClick={() => {
              if (
                window.confirm(
                  "현재 미저장 작업을 버리고 서버의 최신 캔버스를 불러올까요?",
                )
              ) {
                sessionStorage.removeItem(draftKey);
                setDraft(null);
                void load();
              }
            }}
          >
            최신 내용 불러오기
          </Button>
        </div>
      )}
      <div className="canvas-toolbar">
        <div className="canvas-tool-group">
          {(
            [
              ["select", "선택", MousePointer2],
              ["pan", "화면 이동", Hand],
              ["connect", "노드 연결", Network],
              ["pen", "펜", PenTool],
              ["rectangle", "사각형", Square],
              ["ellipse", "타원", Circle],
              ["diamond", "마름모", Diamond],
            ] as const
          ).map(([v, label, Icon]) => (
            <button
              key={v}
              className={tool === v ? "active" : ""}
              disabled={!canvas.can_write && !["select", "pan"].includes(v)}
              title={label}
              aria-label={label}
              onClick={() => {
                setTool(v);
                if (v === "connect") setSelection([]);
              }}
            >
              <Icon size={19} />
            </button>
          ))}
        </div>
        <div className="canvas-tool-group">
          {(
            [
              "note",
              "document",
              "image",
              "url",
              "database",
              "ai",
              "diagram",
            ] as NodeKind[]
          ).map((kind) => {
            const Icon = nodeIcon(kind);
            return (
              <button
                key={kind}
                disabled={!canvas.can_write || data.nodes.length >= 500}
                title={`${kindLabels[kind]} 추가`}
                aria-label={`${kindLabels[kind]} 추가`}
                onClick={() => setAddKind(kind)}
              >
                <Icon size={18} />
                <span>{kindLabels[kind]}</span>
              </button>
            );
          })}
        </div>
        <div className="canvas-tool-group">
          {Object.keys(colors).map((value) => (
            <button
              key={value}
              className={`canvas-color ${color === value ? "active" : ""}`}
              aria-label={`색상 ${value}`}
              title={value}
              style={{
                background: colors[value as NodeColor].fill,
                borderColor: colors[value as NodeColor].line,
              }}
              onClick={() => {
                setColor(value as NodeColor);
                if (selection.length && canvas.can_write)
                  updateData({
                    ...dataRef.current,
                    nodes: dataRef.current.nodes.map((n) =>
                      selection.includes(n.id)
                        ? { ...n, color: value as NodeColor }
                        : n,
                    ),
                  });
              }}
            />
          ))}
        </div>
        <div className="canvas-tool-group">
          <button
            disabled={!historyCount.undo || !canvas.can_write}
            aria-label="실행 취소"
            title="실행 취소 Ctrl+Z"
            onClick={() => changeHistory("undo")}
          >
            <Undo2 size={19} />
          </button>
          <button
            disabled={!historyCount.redo || !canvas.can_write}
            aria-label="다시 실행"
            onClick={() => changeHistory("redo")}
          >
            <Redo2 size={19} />
          </button>
          <button
            disabled={
              !canvas.can_write || (!selection.length && !edgeSelection)
            }
            aria-label="선택 삭제"
            onClick={removeSelection}
          >
            <Trash2 size={18} />
          </button>
          <label className="canvas-import-button" title="캔버스 JSON 가져오기">
            <Upload size={18} />
            <input
              type="file"
              accept=".json"
              disabled={!canvas.can_write}
              onChange={async (event) => {
                const file = event.target.files?.[0];
                if (!file) return;
                try {
                  if (file.size > 4 * 1024 * 1024)
                    throw new Error("캔버스 JSON은 4MB 이하여야 합니다.");
                  const value = JSON.parse(await file.text());
                  const next = value.data || value;
                  if (
                    !Array.isArray(next.nodes) ||
                    !Array.isArray(next.edges) ||
                    next.nodes.length > 500 ||
                    next.edges.length > 1000
                  )
                    throw new Error("madi 캔버스 JSON 형식을 확인하세요.");
                  if (
                    !window.confirm(
                      "현재 캔버스 배치를 가져온 JSON으로 바꿀까요? 실행 취소로 되돌릴 수 있습니다.",
                    )
                  )
                    return;
                  const validated = await api<CanvasData>(
                    `/canvases/${id}/validate`,
                    "POST",
                    next,
                  );
                  updateData(validated);
                  fit();
                } catch (e) {
                  notify((e as Error).message, "error");
                }
                event.target.value = "";
              }}
            />
          </label>
        </div>
      </div>
      <div className={`canvas-stage tool-${tool}`} ref={stage}>
        <svg
          ref={svg}
          aria-label="캔버스 편집 영역"
          tabIndex={0}
          onPointerDown={pointerDown}
          onPointerMove={pointerMove}
          onPointerUp={pointerUp}
          onPointerCancel={pointerUp}
        >
          <defs>
            <pattern
              id={`dots-${id}`}
              width={24 * scale}
              height={24 * scale}
              patternUnits="userSpaceOnUse"
              x={pan.x}
              y={pan.y}
            >
              <circle cx={1} cy={1} r={1} fill="#cbd8ce" />
            </pattern>
            <marker
              id={`arrow-${id}`}
              viewBox="0 0 10 10"
              refX="9"
              refY="5"
              markerWidth="7"
              markerHeight="7"
              orient="auto-start-reverse"
            >
              <path d="M 0 0 L 10 5 L 0 10 z" fill="#729584" />
            </marker>
          </defs>
          <rect width="100%" height="100%" fill={`url(#dots-${id})`} />
          <g transform={`translate(${pan.x} ${pan.y}) scale(${scale})`}>
            {data.edges.map((edge) => {
              const source = data.nodes.find((n) => n.id === edge.source),
                target = data.nodes.find((n) => n.id === edge.target);
              if (!source || !target) return null;
              const x1 = source.x + source.width,
                y1 = source.y + source.height / 2,
                x2 = target.x,
                y2 = target.y + target.height / 2,
                bend = Math.max(60, Math.abs(x2 - x1) * 0.45),
                d = `M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`;
              return (
                <g
                  key={edge.id}
                  onPointerDown={(event) => {
                    event.stopPropagation();
                    setEdgeSelection(edge.id);
                    setSelection([]);
                  }}
                  onDoubleClick={() => {
                    if (canvas.can_write) setEdgeEdit(edge);
                  }}
                >
                  <path
                    d={d}
                    stroke="transparent"
                    strokeWidth={18}
                    fill="none"
                  />
                  <path
                    d={d}
                    stroke={
                      edgeSelection === edge.id
                        ? "#174f3d"
                        : colors[edge.color]?.line || "#729584"
                    }
                    strokeWidth={edgeSelection === edge.id ? 3.5 : 2}
                    fill="none"
                    markerEnd={`url(#arrow-${id})`}
                  />
                  {edge.label && (
                    <text
                      x={(x1 + x2) / 2}
                      y={(y1 + y2) / 2 - 10}
                      fontSize={15}
                      textAnchor="middle"
                      fill="#547463"
                    >
                      {edge.label}
                    </text>
                  )}
                </g>
              );
            })}
            {data.nodes.map((node) => {
              const selected = selection.includes(node.id),
                palette = colors[node.color] || colors.mint,
                reference = canvas.resolved?.[node.id],
                Icon = nodeIcon(node.kind) || StickyNote;
              return (
                <g
                  key={node.id}
                  transform={`translate(${node.x} ${node.y})`}
                  onPointerDown={(event) => nodeDown(event, node)}
                  onDoubleClick={() => setEditingNode({ ...node })}
                  className="canvas-node"
                  data-node-id={node.id}
                  data-node-kind={node.kind}
                >
                  {node.kind === "drawing" ? (
                    <>
                      <rect
                        width={node.width}
                        height={node.height}
                        fill="transparent"
                        stroke={selected ? palette.line : "transparent"}
                        strokeDasharray="5 4"
                      />
                      <polyline
                        points={(node.points || [])
                          .map((p) => p.join(","))
                          .join(" ")}
                        stroke={palette.line}
                        strokeWidth={3}
                        fill="none"
                        strokeLinecap="round"
                        strokeLinejoin="round"
                      />
                    </>
                  ) : node.kind === "ellipse" ? (
                    <ellipse
                      cx={node.width / 2}
                      cy={node.height / 2}
                      rx={node.width / 2}
                      ry={node.height / 2}
                      fill={palette.fill}
                      stroke={palette.line}
                      strokeWidth={selected ? 3 : 1.5}
                    />
                  ) : node.kind === "diamond" ? (
                    <polygon
                      points={`${node.width / 2},0 ${node.width},${node.height / 2} ${node.width / 2},${node.height} 0,${node.height / 2}`}
                      fill={palette.fill}
                      stroke={palette.line}
                      strokeWidth={selected ? 3 : 1.5}
                    />
                  ) : (
                    <rect
                      width={node.width}
                      height={node.height}
                      rx={node.kind === "rectangle" ? 5 : 11}
                      fill={palette.fill}
                      stroke={selected ? palette.line : "#cfdcd3"}
                      strokeWidth={selected ? 3 : 1.2}
                    />
                  )}{" "}
                  {!["drawing", "ellipse", "diamond", "rectangle"].includes(
                    node.kind,
                  ) ? (
                    <>
                      <line
                        x1={0}
                        y1={37}
                        x2={node.width}
                        y2={37}
                        stroke="#7f99821f"
                      />
                      <foreignObject
                        x={12}
                        y={9}
                        width={node.width - 24}
                        height={24}
                      >
                        <div className="canvas-node-label">
                          <Icon size={15} />
                          <span>
                            {["document", "database", "image"].includes(
                              node.kind,
                            )
                              ? reference?.title || "원본 확인 필요"
                              : node.title || kindLabels[node.kind]}
                          </span>
                          {reference?.unavailable && <Lock size={13} />}
                        </div>
                      </foreignObject>
                      {node.kind === "diagram" ? (
                        <DiagramPreview
                          source={node.text || ""}
                          svgBounds={{
                            x: 12,
                            y: 48,
                            width: Math.max(10, node.width - 24),
                            height: Math.max(10, node.height - 60),
                          }}
                        />
                      ) : (
                        <foreignObject
                          x={12}
                          y={48}
                          width={Math.max(10, node.width - 24)}
                          height={Math.max(10, node.height - 60)}
                        >
                          <div className="canvas-node-content">
                            {node.kind === "note" ? (
                              <MarkdownView
                                markdown={
                                  node.text || "두 번 눌러 메모를 작성하세요."
                                }
                                documents={documents}
                              />
                            ) : node.kind === "image" ? (
                              reference?.url && !reference.unavailable ? (
                                <img
                                  draggable={false}
                                  src={reference.url}
                                  alt={reference.title}
                                />
                              ) : (
                                <p className="muted">
                                  {reference?.error ||
                                    "이미지를 확인할 수 없습니다."}
                                </p>
                              )
                            ) : node.kind === "url" ? (
                              <>
                                <p>{node.text || node.url}</p>
                                {safeLink(node.url) && (
                                  <a
                                    href={safeLink(node.url)}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    onPointerDown={(e) => e.stopPropagation()}
                                  >
                                    링크 열기 <ArrowUpRight size={14} />
                                  </a>
                                )}
                              </>
                            ) : node.kind === "ai" ? (
                              <>
                                <p>
                                  {node.text || "AI에게 질문을 작성하세요."}
                                </p>
                                <Button
                                  disabled={!canvas.can_write}
                                  onClick={async () => {
                                    if (await save()) setAINode(node);
                                  }}
                                  onPointerDown={(e) => e.stopPropagation()}
                                >
                                  <Sparkles size={14} />
                                  AI 실행
                                </Button>
                              </>
                            ) : (
                              <>
                                <p>
                                  {reference?.unavailable
                                    ? reference.error
                                    : reference?.snippet ||
                                      "현재 원본을 불러오려면 새로고침하세요."}
                                </p>
                                {reference?.url && !reference.unavailable && (
                                  <Link
                                    to={reference.url}
                                    onPointerDown={(e) => e.stopPropagation()}
                                  >
                                    원본 열기 <ArrowUpRight size={14} />
                                  </Link>
                                )}
                              </>
                            )}
                          </div>
                        </foreignObject>
                      )}
                    </>
                  ) : (
                    node.kind !== "drawing" && (
                      <foreignObject
                        x={node.width * 0.15}
                        y={node.height * 0.25}
                        width={node.width * 0.7}
                        height={node.height * 0.5}
                      >
                        <div className="canvas-shape-label">
                          {node.text || node.title}
                        </div>
                      </foreignObject>
                    )
                  )}
                  {selected && canvas.can_write && node.kind !== "drawing" && (
                    <g
                      onPointerDown={(event) => {
                        event.stopPropagation();
                        nodeDown(event, node, true);
                      }}
                      className="canvas-resize-handle"
                    >
                      <rect
                        x={node.width - 9}
                        y={node.height - 9}
                        width={18}
                        height={18}
                        rx={4}
                        fill={palette.line}
                      />
                      <path
                        d={`M ${node.width - 4} ${node.height + 3} l 7 -7`}
                        stroke="white"
                        strokeWidth={1.5}
                      />
                    </g>
                  )}
                </g>
              );
            })}
            {marquee && (
              <rect
                {...marquee}
                fill="#39886215"
                stroke="#3b8064"
                strokeDasharray="6 4"
              />
            )}
          </g>
        </svg>
        {!data.nodes.length && (
          <div className="canvas-empty-hint">
            <Network size={35} />
            <h2>아이디어를 놓을 준비가 됐어요</h2>
            <p>
              위 도구 모음에서 메모나 문서를 추가하세요.
              <br />빈 공간 드래그로 선택 · Space 드래그로 이동
            </p>
          </div>
        )}
        <div className="canvas-stage-bottom">
          <span>
            {tool === "connect"
              ? "연결할 두 노드를 순서대로 선택하세요"
              : tool === "pen"
                ? "캔버스에 드래그하여 그리세요"
                : `${data.nodes.length}개 노드 · ${selection.length}개 선택`}
          </span>
          <div>
            <button aria-label="축소" onClick={() => zoom(1 / 1.15)}>
              <ZoomOut size={18} />
            </button>
            <span>{Math.round(scale * 100)}%</span>
            <button aria-label="확대" onClick={() => zoom(1.15)}>
              <ZoomIn size={18} />
            </button>
            <button aria-label="전체 보기" title="전체 보기" onClick={fit}>
              <Expand size={18} />
            </button>
          </div>
        </div>
      </div>
      <div className="canvas-footer">
        <span>
          {canvas.can_write
            ? "두 번 클릭하여 내용 편집 · Shift로 다중 선택 · 연결선을 두 번 클릭하면 라벨 편집"
            : "읽기 전용 캔버스입니다."}
        </span>
        {canvas.can_write && (
          <button
            className="text-button danger-text"
            onClick={async () => {
              if (
                !window.confirm(
                  "캔버스를 휴지통으로 이동할까요? 나중에 복원할 수 있습니다.",
                )
              )
                return;
              try {
                await api("/canvases/" + id, "DELETE");
                sessionStorage.removeItem(draftKey);
                navigate("/app/canvases");
                notify("캔버스를 휴지통으로 이동했습니다.");
              } catch (e) {
                notify((e as Error).message, "error");
              }
            }}
          >
            <Trash2 size={15} />
            휴지통으로 이동
          </button>
        )}
      </div>
      <CanvasNodeDialog
        canvasId={id}
        kind={addKind}
        node={editingNode}
        workspaceID={canvas.workspace_id}
        readOnly={!canvas.can_write}
        onClose={() => {
          setAddKind(null);
          setEditingNode(null);
        }}
        onSave={(node) => {
          if (editingNode) saveNode(node as CanvasNode);
          else addNode(node);
        }}
      />
      <CanvasShareDialog
        open={shareOpen}
        canvas={canvas}
        onClose={() => setShareOpen(false)}
        onVisibility={async (value, nextSpace) => {
          const previous = { ...metaRef.current };
          setVisibility(value);
          setSpace(nextSpace);
          metaRef.current = { title, visibility: value, space: nextSpace };
          setDirty(true);
          const ok = await save();
          if (!ok) {
            setVisibility(previous.visibility);
            setSpace(previous.space);
            metaRef.current = previous;
          }
          return ok;
        }}
      />
      <Modal
        open={!!edgeEdit}
        onOpenChange={(open) => {
          if (!open) setEdgeEdit(null);
        }}
        title="연결선 편집"
      >
        <Field label="연결선 라벨">
          <input
            value={edgeEdit?.label || ""}
            maxLength={150}
            onChange={(e) =>
              setEdgeEdit((prev) =>
                prev ? { ...prev, label: e.target.value } : null,
              )
            }
          />
        </Field>
        <div className="modal-actions">
          <Button
            variant="primary"
            onClick={() => {
              if (edgeEdit)
                updateData({
                  ...dataRef.current,
                  edges: dataRef.current.edges.map((e) =>
                    e.id === edgeEdit.id ? edgeEdit : e,
                  ),
                });
              setEdgeEdit(null);
            }}
          >
            저장
          </Button>
        </div>
      </Modal>
      <CanvasAIResult
        canvasId={id}
        node={aiNode}
        onClose={() => setAINode(null)}
        onInsert={(text) => {
          addNode({
            kind: "note",
            title: "AI 답변",
            text,
            color: "lavender",
            x: (aiNode?.x || 0) + (aiNode?.width || 310) + 70,
            y: aiNode?.y || 0,
          });
          setAINode(null);
        }}
      />
    </div>
  );
}

function CanvasNodeDialog({
  canvasId,
  kind,
  node,
  workspaceID,
  readOnly,
  onClose,
  onSave,
}: {
  canvasId: string;
  kind: NodeKind | null;
  node: CanvasNode | null;
  workspaceID: string;
  readOnly: boolean;
  onClose: () => void;
  onSave: (node: Partial<CanvasNode>) => void;
}) {
  const { documents, notify } = useApp();
  const [title, setTitle] = useState(""),
    [text, setText] = useState(""),
    [refID, setRefID] = useState(""),
    [url, setURL] = useState(""),
    [databases, setDatabases] = useState<{ id: string; name: string }[]>([]),
    [imageDocument, setImageDocument] = useState(""),
    [attachments, setAttachments] = useState<
      { id: string; name: string; content_type: string }[]
    >([]),
    [error, setError] = useState("");
  const actual = node?.kind || kind;
  useEffect(() => {
    setTitle(node?.title || "");
    setText(
      node?.text ||
        (kind === "diagram"
          ? "flowchart LR\n  A[아이디어] --> B[연결]\n  B --> C[새로운 지식]"
          : ""),
    );
    setRefID(node?.ref_id || "");
    setURL(node?.url || "");
    setError("");
    setImageDocument("");
    setAttachments([]);
  }, [kind, node?.id]);
  useEffect(() => {
    if (actual !== "database") return;
    let active = true;
    api<{ id: string; name: string }[]>(
      "/databases?workspace_id=" + workspaceID,
    )
      .then((v) => {
        if (active) setDatabases(v);
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [actual, workspaceID]);
  const loadImages = async (id: string) => {
    setImageDocument(id);
    setAttachments([]);
    if (!id) return;
    try {
      setAttachments(
        await api<any[]>(`/canvases/${canvasId}/images?document_id=${id}`),
      );
    } catch (e) {
      setError((e as Error).message);
    }
  };
  return (
    <Modal
      open={!!actual}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
      title={`${actual ? kindLabels[actual] : ""} ${node ? (readOnly ? "보기" : "편집") : "추가"}`}
      wide={actual === "diagram"}
    >
      <ErrorBox error={error} />
      <form
        onSubmit={(e) => {
          e.preventDefault();
          if (readOnly) return;
          if (actual === "url" && !safeLink(url)) {
            setError("HTTP 또는 HTTPS 주소를 입력하세요.");
            return;
          }
          onSave({ ...node, kind: actual!, title, text, ref_id: refID, url });
        }}
      >
        <fieldset disabled={readOnly} className="canvas-form-fieldset">
          {!["document", "database", "image"].includes(actual || "") && (
            <Field label="노드 제목">
              <input
                maxLength={150}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder={actual ? kindLabels[actual] : ""}
              />
            </Field>
          )}
          {actual === "document" && (
            <Field label="연결할 문서">
              <select
                required
                value={refID}
                onChange={(e) => setRefID(e.target.value)}
              >
                <option value="" disabled>
                  문서를 선택하세요
                </option>
                {documents
                  .filter((d) => d.workspace_id === workspaceID)
                  .map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
              </select>
            </Field>
          )}
          {actual === "database" && (
            <Field label="연결할 데이터베이스">
              <select
                required
                value={refID}
                onChange={(e) => setRefID(e.target.value)}
              >
                <option value="" disabled>
                  데이터베이스를 선택하세요
                </option>
                {databases.map((d) => (
                  <option key={d.id} value={d.id}>
                    {d.name}
                  </option>
                ))}
              </select>
            </Field>
          )}
          {actual === "image" && (
            <>
              <Field label="이미지가 첨부된 문서">
                <select
                  value={imageDocument}
                  onChange={(e) => loadImages(e.target.value)}
                >
                  <option value="">문서를 선택하세요</option>
                  {documents
                    .filter((d) => d.workspace_id === workspaceID)
                    .map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.title}
                      </option>
                    ))}
                </select>
              </Field>
              <Field label="연결할 이미지">
                <select
                  required
                  value={refID}
                  onChange={(e) => setRefID(e.target.value)}
                >
                  <option value="" disabled>
                    이미지를 선택하세요
                  </option>
                  {node?.ref_id &&
                    !attachments.some((a) => a.id === node.ref_id) && (
                      <option value={node.ref_id}>현재 연결 이미지</option>
                    )}
                  {attachments.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </Field>
              {imageDocument && (
                <Field label="선택한 문서에 이미지 업로드">
                  <input
                    type="file"
                    accept="image/png,image/jpeg,image/gif,image/webp,image/avif"
                    onChange={async (e) => {
                      const file = e.target.files?.[0];
                      if (!file) return;
                      try {
                        const form = new FormData();
                        form.append("file", file);
                        const image = await api<{ id: string }>(
                          `/attachments?document_id=${imageDocument}`,
                          "POST",
                          form,
                        );
                        await loadImages(imageDocument);
                        setRefID(image.id);
                        notify("원본 문서에 이미지를 업로드했습니다.");
                      } catch (e) {
                        setError((e as Error).message);
                      }
                    }}
                  />
                </Field>
              )}
              <p className="muted">
                원본 문서의 접근 권한을 따릅니다. 파일을 복제하거나 외부
                주소에서 불러오지 않습니다.
              </p>
            </>
          )}
          {actual === "url" && (
            <Field label="링크 주소">
              <input
                type="url"
                required
                value={url}
                onChange={(e) => setURL(e.target.value)}
                placeholder="https://intranet.example/"
              />
            </Field>
          )}
          {!["document", "database", "image", "drawing"].includes(
            actual || "",
          ) && (
            <Field
              label={
                actual === "diagram"
                  ? "Mermaid 소스"
                  : actual === "ai"
                    ? "AI 질문"
                    : "내용"
              }
            >
              <textarea
                rows={actual === "diagram" ? 8 : 6}
                maxLength={actual === "diagram" ? 50000 : 60000}
                value={text}
                onChange={(e) => setText(e.target.value)}
                placeholder={
                  actual === "note"
                    ? "Markdown으로 아이디어를 기록하세요."
                    : actual === "ai"
                      ? "워크스페이스 지식을 바탕으로 질문해 보세요."
                      : ""
                }
              />
            </Field>
          )}
          {actual === "diagram" && <DiagramPreview source={text} />}
        </fieldset>
        <div className="modal-actions">
          <Button type="button" onClick={onClose}>
            닫기
          </Button>
          {!readOnly && (
            <Button
              variant="primary"
              disabled={
                ["document", "database", "image"].includes(actual || "") &&
                !refID
              }
            >
              {node ? "적용" : "캔버스에 추가"}
            </Button>
          )}
        </div>
      </form>
    </Modal>
  );
}

function CanvasShareDialog({
  open,
  canvas,
  onClose,
  onVisibility,
}: {
  open: boolean;
  canvas: CanvasRecord;
  onClose: () => void;
  onVisibility: (
    visibility: CanvasRecord["visibility"],
    space: string,
  ) => Promise<boolean>;
}) {
  const { notify } = useApp();
  const [visibility, setVisibility] = useState(canvas.visibility),
    [space, setSpace] = useState(canvas.space_id || ""),
    [spaces, setSpaces] = useState<{ id: string; name: string }[]>([]),
    [shares, setShares] = useState<
      { email: string; name: string; permission: string }[]
    >([]),
    [email, setEmail] = useState(""),
    [permission, setPermission] = useState("read"),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const load = () =>
    api<any[]>(`/canvases/${canvas.id}/shares`)
      .then(setShares)
      .catch((e) => setError(e.message));
  useEffect(() => {
    if (open) {
      setVisibility(canvas.visibility);
      setSpace(canvas.space_id || "");
      setError("");
      load();
      api<any[]>(`/spaces?workspace_id=${canvas.workspace_id}`)
        .then(setSpaces)
        .catch((e) => setError(e.message));
    }
  }, [open, canvas.id]);
  return (
    <Modal
      open={open}
      onOpenChange={(v) => {
        if (!v) onClose();
      }}
      title="캔버스 공유"
    >
      <ErrorBox error={error} />
      <Field label="공개 범위">
        <select
          value={visibility}
          onChange={(e) =>
            setVisibility(e.target.value as CanvasRecord["visibility"])
          }
        >
          <option value="private">나만 보기</option>
          <option value="workspace">워크스페이스 공유</option>
          <option value="selected">선택한 사용자와 공유</option>
        </select>
      </Field>
      <Field label="캔버스 공간">
        <select value={space} onChange={(e) => setSpace(e.target.value)}>
          <option value="">워크스페이스 기본 공간</option>
          {spaces.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
      </Field>
      {visibility === "selected" && (
        <>
          <p className="muted">
            아래 목록은 즉시 반영됩니다. 공개 범위는 하단 저장 버튼으로
            적용하세요.
          </p>
          <form
            className="form-grid"
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              try {
                await api(`/canvases/${canvas.id}/shares`, "PUT", {
                  email,
                  permission,
                });
                await load();
                setEmail("");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Field label="공유할 사용자 이메일">
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </Field>
            <Field label="공유 권한">
              <select
                value={permission}
                onChange={(e) => setPermission(e.target.value)}
              >
                <option value="read">읽기</option>
                <option value="write">편집</option>
              </select>
            </Field>
            <Button disabled={busy || !email}>사용자 추가</Button>
          </form>
          <div className="canvas-share-list">
            {shares.map((share) => (
              <div key={share.email}>
                <span>
                  {share.name}
                  <small>{share.email}</small>
                </span>
                <Badge>{share.permission === "write" ? "편집" : "읽기"}</Badge>
                <button
                  className="icon-button"
                  aria-label={`${share.name} 공유 제거`}
                  onClick={async () => {
                    try {
                      await api(`/canvases/${canvas.id}/shares`, "PUT", {
                        email: share.email,
                        permission: "remove",
                      });
                      await load();
                    } catch (e) {
                      setError((e as Error).message);
                    }
                  }}
                >
                  <X size={15} />
                </button>
              </div>
            ))}
          </div>
        </>
      )}
      <div className="modal-actions">
        <Button onClick={onClose}>닫기</Button>
        <Button
          variant="primary"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              if (await onVisibility(visibility, space)) {
                notify("공개 범위를 저장했습니다.");
                onClose();
              }
            } finally {
              setBusy(false);
            }
          }}
        >
          공개 범위 저장
        </Button>
      </div>
    </Modal>
  );
}

function CanvasAIResult({
  canvasId,
  node,
  onClose,
  onInsert,
}: {
  canvasId: string;
  node: CanvasNode | null;
  onClose: () => void;
  onInsert: (text: string) => void;
}) {
  const [output, setOutput] = useState(""),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [complete, setComplete] = useState(false);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => {
    setOutput("");
    setError("");
    setComplete(false);
    return () => controller.current?.abort();
  }, [node?.id]);
  const run = async () => {
    if (!node) return;
    const c = new AbortController();
    controller.current = c;
    setBusy(true);
    setOutput("");
    setError("");
    setComplete(false);
    try {
      const response = await fetch(
        `/api/v1/canvases/${canvasId}/ai/${node.id}`,
        {
          method: "POST",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            "X-Madi-Request": "1",
          },
          body: "{}",
          signal: c.signal,
        },
      );
      if (!response.ok) {
        const value = await response.json();
        throw new Error(value.error);
      }
      if (!response.body) throw new Error("스트리밍 응답이 없습니다.");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "",
        done = false;
      while (true) {
        const chunk = await reader.read();
        if (chunk.done) break;
        buffer += decoder.decode(chunk.value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const raw = line.slice(5).trim();
          if (raw === "[DONE]") {
            done = true;
            continue;
          }
          if (!raw) continue;
          const value = JSON.parse(raw);
          if (value.error) throw new Error(value.error);
          if (value.text) setOutput((v) => v + value.text);
        }
      }
      if (!done) throw new Error("응답이 완료되기 전에 연결이 끊어졌습니다.");
      setComplete(true);
    } catch (e) {
      setError(
        c.signal.aborted ? "생성을 중단했습니다." : (e as Error).message,
      );
    } finally {
      c.abort();
      setBusy(false);
    }
  };
  return (
    <Modal
      open={!!node}
      onOpenChange={(value) => {
        if (!value && !busy) onClose();
      }}
      title="캔버스 AI"
      description="권한이 있는 워크스페이스 문서를 바탕으로 답변합니다. 결과를 확인한 뒤 메모로 추가하세요."
      wide
    >
      <div className="notice subtle">
        <Sparkles size={18} />
        <span>{node?.text}</span>
      </div>
      <ErrorBox error={error} />
      {output && (
        <div className="canvas-ai-output">
          <MarkdownView markdown={output} documents={[]} />
        </div>
      )}
      <div className="modal-actions">
        {output && <CopyButton value={output} />}
        <Button disabled={busy} onClick={onClose}>
          닫기
        </Button>
        {busy ? (
          <Button variant="danger" onClick={() => controller.current?.abort()}>
            생성 중단
          </Button>
        ) : (
          <Button onClick={run}>
            <Sparkles size={16} />
            {output ? "다시 생성" : "AI 생성"}
          </Button>
        )}
        {complete && (
          <Button variant="primary" onClick={() => onInsert(output)}>
            <StickyNote size={16} />
            메모로 추가
          </Button>
        )}
      </div>
    </Modal>
  );
}

function CanvasStyles() {
  return (
    <>
      <style>{`
      .canvas-node-content .markdown-content{padding:0!important;margin:0;min-height:0;font-size:15px;line-height:1.7}
      .canvas-node-content .markdown-content>:first-child{margin-top:0}
      .canvas-node-content .markdown-content p{font-size:15px;line-height:1.7;margin:0 0 10px}
      .canvas-node-content .markdown-content h1,.canvas-node-content .markdown-content h2,.canvas-node-content .markdown-content h3{font-size:20px;margin:0 0 10px;line-height:1.5}
      .canvas-node-content .markdown-content ul,.canvas-node-content .markdown-content ol{padding-left:20px;margin:10px 0}
      .canvas-node-content .markdown-content li{margin:5px 0}
      .canvas-node-content .markdown-content pre{padding:10px;margin:10px 0;font-size:13px}
      .canvas-node-content .diagram-preview{height:100%;display:flex;align-items:center;justify-content:center;overflow:hidden!important}
      .canvas-node-content .diagram-preview img{display:block;width:100%;height:100%!important;object-fit:contain}
    `}</style>
      <style>{`.canvas-card-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:22px}.canvas-list-card{overflow:hidden}.canvas-list-card>a{display:block;color:inherit;text-decoration:none}.canvas-card-cover{height:150px;background:radial-gradient(#c3d7c8 1px,transparent 1px) 0 0/18px 18px,#edf4ee;position:relative;display:grid;place-items:center;color:#5a8e71}.canvas-card-cover span{position:absolute;width:54px;height:39px;left:18%;top:25%;border:1px solid #c8d5c9;border-radius:5px;background:#fff9e9;transform:rotate(-8deg)}.canvas-card-cover span:nth-child(2){left:65%;top:43%;background:#ece5f8;transform:rotate(6deg)}.canvas-card-cover span:nth-child(3){left:29%;top:65%;width:42px;height:28px;background:#e1edfa;transform:rotate(3deg)}.canvas-card-body{padding:22px}.canvas-card-body h2{font-size:1.05rem;margin:0 0 9px}.canvas-card-body p{font-size:14px;color:var(--muted);margin:0 0 15px}.canvas-list-card>.button{margin:0 22px 20px}.canvas-heading{display:flex;align-items:center;gap:15px;margin-bottom:22px}.canvas-heading>div:nth-child(2){min-width:0;flex:1}.canvas-heading input{font-size:1.45rem;font-weight:700;background:transparent;border:0;padding:3px 0;box-shadow:none;width:100%;color:var(--text)}.canvas-heading p{display:flex;gap:8px;color:var(--muted);font-size:13px;margin:7px 0 0}.canvas-heading-actions{display:flex;gap:9px;flex-shrink:0}.canvas-save-state{color:var(--primary)}.canvas-toolbar{display:flex;flex-wrap:wrap;gap:10px;padding:12px;border:1px solid var(--border);border-radius:11px 11px 0 0;background:var(--surface)}.canvas-tool-group{display:flex;gap:3px;align-items:center;border-right:1px solid var(--border);padding-right:10px;max-width:100%;overflow-x:auto}.canvas-tool-group:last-child{border:0}.canvas-tool-group>button,.canvas-import-button{display:flex;align-items:center;justify-content:center;gap:6px;width:35px;height:35px;background:transparent;border:0;border-radius:6px;color:var(--muted);flex-shrink:0}.canvas-tool-group>button:has(span){width:auto;padding:0 8px;font-size:14px}.canvas-tool-group>button:hover:not(:disabled),.canvas-tool-group>button.active{background:var(--soft);color:var(--primary)}.canvas-tool-group>button:disabled{opacity:.35}.canvas-tool-group>button.canvas-color{width:20px;height:20px;border:1px solid;margin:0 3px}.canvas-tool-group>button.canvas-color.active{outline:2px solid var(--primary);outline-offset:2px}.canvas-import-button input{display:none}.canvas-stage{position:relative;height:calc(100vh - 335px);min-height:540px;max-height:1000px;background:#f7faf5;border:1px solid var(--border);border-top:0;overflow:hidden;isolation:isolate;border-radius:0 0 11px 11px;touch-action:none}.canvas-stage>svg{width:100%;height:100%;display:block;outline:none;touch-action:none}.canvas-stage.tool-pan{cursor:grab}.canvas-stage.tool-pen,.canvas-stage.tool-rectangle,.canvas-stage.tool-ellipse,.canvas-stage.tool-diamond,.canvas-stage.tool-connect{cursor:crosshair}.canvas-node{cursor:move}.canvas-node-label{display:flex;align-items:center;gap:8px;font-family:'Noto Sans KR Variable',sans-serif;font-size:14px;color:#476451;font-weight:600;white-space:nowrap;overflow:hidden}.canvas-node-label>span{overflow:hidden;text-overflow:ellipsis;flex:1}.canvas-node-content{height:100%;overflow:auto;scrollbar-width:thin;font-size:15px;line-height:1.7;color:#344c3c}.canvas-node-content p{margin:0 0 10px;white-space:pre-wrap;overflow-wrap:anywhere}.canvas-node-content .markdown-view{font-size:15px}.canvas-node-content .markdown-view p{font-size:15px}.canvas-node-content .markdown-view h1,.canvas-node-content .markdown-view h2{font-size:20px;margin:0 0 10px}.canvas-node-content img{width:100%;height:100%;object-fit:contain;pointer-events:none}.canvas-node-content a{display:inline-flex;align-items:center;gap:5px;font-size:14px;color:#276d4b}.canvas-node-content .button{font-size:14px;min-height:32px;padding:5px 10px}.canvas-node-content .diagram-preview{max-height:100%}.canvas-node-content .diagram-preview svg{max-width:100%!important;height:auto!important}.canvas-resize-handle{cursor:nwse-resize}.canvas-shape-label{height:100%;display:grid;place-items:center;text-align:center;white-space:pre-wrap;overflow:hidden;font-size:17px;color:#36533e}.canvas-empty-hint{position:absolute;left:50%;top:45%;transform:translate(-50%,-50%);text-align:center;pointer-events:none;color:#7b9985;width:80%}.canvas-empty-hint h2{font-size:1.15rem;color:#476752}.canvas-empty-hint p{font-size:14px;line-height:1.9}.canvas-stage-bottom{position:absolute;left:18px;right:18px;bottom:16px;display:flex;justify-content:space-between;align-items:center;gap:15px;pointer-events:none}.canvas-stage-bottom>span{background:#ffffffd9;color:#708675;font-size:13px;padding:9px 12px;border-radius:7px;max-width:60%}.canvas-stage-bottom>div{display:flex;align-items:center;gap:10px;background:var(--surface);padding:6px 9px;border:1px solid var(--border);border-radius:8px;pointer-events:auto;box-shadow:0 4px 14px #1b493d0d}.canvas-stage-bottom button{background:none;border:0;color:var(--primary);display:flex;padding:4px}.canvas-stage-bottom>div>span{font-size:13px;min-width:38px;text-align:center}.canvas-footer{display:flex;justify-content:space-between;gap:15px;font-size:13px;color:var(--muted);padding:14px 2px}.canvas-footer .text-button{font-size:13px}.canvas-form-fieldset{border:0;padding:0;margin:0;min-width:0}.canvas-share-list{display:flex;flex-direction:column;gap:12px;margin-top:22px}.canvas-share-list>div{display:flex;align-items:center;gap:12px}.canvas-share-list>div>span:first-child{flex:1;font-size:15px}.canvas-share-list small{display:block;color:var(--muted);font-size:13px}.canvas-ai-output{max-height:450px;overflow:auto;padding:20px;border:1px solid var(--border);border-radius:9px}.canvas-editor-page>.notice{margin-bottom:14px;flex-wrap:wrap}.canvas-editor-page>.notice>span{flex:1;min-width:180px}.canvas-editor-page>.notice>.button{flex-shrink:0}@media(max-width:1150px){.canvas-card-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.canvas-tool-group>button:has(span)>span{display:none}.canvas-tool-group>button:has(span){width:35px;padding:0}.canvas-heading-actions .button{font-size:14px;padding:8px}}@media(max-width:640px){.canvas-card-grid{grid-template-columns:1fr}.canvas-heading{flex-wrap:wrap;gap:10px}.canvas-heading input{font-size:1.2rem}.canvas-heading-actions{width:100%;justify-content:flex-end}.canvas-heading-actions .button{font-size:14px}.canvas-stage{height:650px;min-height:440px}.canvas-toolbar{gap:7px;padding:9px}.canvas-tool-group{padding-right:7px}.canvas-tool-group>button,.canvas-import-button{width:31px;height:31px}.canvas-stage-bottom{left:9px;right:9px;gap:7px}.canvas-stage-bottom>span{font-size:12px;padding:7px;max-width:40%}.canvas-stage-bottom>div{gap:5px;padding:5px}.canvas-footer{flex-direction:column;font-size:12px}.canvas-empty-hint h2{font-size:1rem}}`}</style>
    </>
  );
}
