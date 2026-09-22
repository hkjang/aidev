import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  ArrowUpRight,
  CircleDot,
  FilterX,
  GitFork,
  Link2,
  Network,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  Unlink,
} from "lucide-react";
import { api } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  DocIcon,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "./ui";
import GraphCanvas from "./graph/GraphCanvas";
import {
  emptyGraph,
  filterGraph,
  type GraphData,
  type GraphEdge,
} from "./graph/model";
import "./graph/style.css";
const DocumentPreview = lazy(() => import("./review/DocumentPreview"));

const types = {
  reference: "참조",
  related: "관련",
  parent: "상위 문서",
  policy: "정책 의존",
  execution: "실행 의존",
  data: "데이터 의존",
};
const origins = { wiki: "위키 링크", manual: "직접 연결", tree: "문서 계층" };

export default function GraphPage() {
  const { user, workspace } = useApp();
  return <WorkspaceGraph key={`${user.id}:${workspace?.id || ""}`} />;
}
function WorkspaceGraph() {
  const [preview, setPreview] = useState<string | null>(null);
  const { user, workspace, notify } = useApp();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const [data, setData] = useState<GraphData>(emptyGraph),
    [loading, setLoading] = useState(true),
    [error, setError] = useState<unknown>(null);
  const [spaces, setSpaces] = useState<{ id: string; name: string }[]>([]),
    [members, setMembers] = useState<{ id: string; name: string }[]>([]);
  const [relation, setRelation] = useState(false),
    [target, setTarget] = useState(""),
    [relationType, setRelationType] = useState("related"),
    [remove, setRemove] = useState<GraphEdge | null>(null),
    [busy, setBusy] = useState(false),
    [relationError, setRelationError] = useState<unknown>(null);
  const [page, setPage] = useState(0),
    [filtersExpanded, setFiltersExpanded] = useState(false);
  const generation = useRef(0),
    active = useRef(true);
  const query = params.get("q") || "",
    tag = params.get("tag") || "",
    space = params.get("space") || "",
    owner = params.get("owner") || "",
    focus = params.get("focus") || "",
    selected = params.get("node") || "";
  const depth = /^[1-5]$/.test(params.get("depth") || "")
    ? Number(params.get("depth"))
    : 2;
  const type = Object.keys(types).includes(params.get("type") || "")
    ? params.get("type")!
    : "";
  const layout = ["cose", "concentric", "breadthfirst"].includes(
    params.get("layout") || "",
  )
    ? params.get("layout")!
    : "cose";
  const set = (key: string, value: string) => {
    // BrowserRouter commits history before its transition renders. Read that
    // canonical URL so a rapid reset + select cannot resurrect stale filters;
    // useSearchParams callbacks do not queue updates like React setState does.
    const next = new URLSearchParams(window.location.search);
    if (value) next.set(key, value);
    else next.delete(key);
    setParams(next, { replace: key === "q" });
  };
  const load = useCallback(
    async (quiet = false) => {
      if (!workspace) {
        setLoading(false);
        return;
      }
      const request = ++generation.current;
      if (!quiet) setLoading(true);
      try {
        const next = await api<GraphData>(
          `/graph?workspace_id=${workspace.id}`,
        );
        if (!active.current || generation.current !== request) return;
        const normalized = {
          ...emptyGraph,
          ...next,
          nodes: [...next.nodes].sort((a, b) => a.id.localeCompare(b.id)),
          edges: [...next.edges].sort((a, b) =>
            `${a.source}:${a.target}:${a.type}:${a.origin}`.localeCompare(
              `${b.source}:${b.target}:${b.type}:${b.origin}`,
            ),
          ),
          unresolved: [...(next.unresolved || [])].sort((a, b) =>
            `${a.source}:${a.target}:${a.reason}`.localeCompare(
              `${b.source}:${b.target}:${b.reason}`,
            ),
          ),
          diagnostics: { ...emptyGraph.diagnostics, ...next.diagnostics },
        };
        // Quiet permission checks must not rebuild a stable graph or discard
        // its manually arranged positions and active filter inputs.
        setData((previous) =>
          JSON.stringify(previous) === JSON.stringify(normalized)
            ? previous
            : normalized,
        );
        setError(null);
      } catch (e) {
        if (active.current && generation.current === request) {
          setError(e);
          setData(emptyGraph);
        }
      } finally {
        if (active.current && generation.current === request) setLoading(false);
      }
    },
    [workspace?.id],
  );
  useEffect(() => {
    active.current = true;
    void load();
    if (workspace) {
      api<{ id: string; name: string }[]>(
        `/spaces?workspace_id=${workspace.id}`,
      )
        .then((v) => active.current && setSpaces(v))
        .catch(() => {});
      api<{ id: string; name: string }[]>(`/workspaces/${workspace.id}/members`)
        .then((v) => active.current && setMembers(v))
        .catch(() => {});
    }
    return () => {
      active.current = false;
      generation.current++;
    };
  }, [load, workspace?.id]);
  useEffect(() => {
    const timer = setInterval(
      () => {
        if (document.visibilityState === "visible" && !busy) void load(true);
      },
      data.diagnostics.pending ? 3000 : 15000,
    );
    return () => clearInterval(timer);
  }, [load, busy, data.diagnostics.pending]);
  useEffect(() => {
    setRelation(false);
    setRemove(null);
    setRelationError(null);
  }, [selected]);
  const filtered = useMemo(
    () =>
      filterGraph(data, { q: query, tag, space, owner, focus, depth, type }),
    [data, query, tag, space, owner, focus, depth, type],
  );
  const sorted = useMemo(
    () =>
      [...filtered.nodes].sort((a, b) => a.title.localeCompare(b.title, "ko")),
    [filtered.nodes],
  );
  useEffect(() => setPage(0), [query, tag, space, owner, focus, depth, type]);
  const visiblePage = Math.min(
    page,
    Math.max(0, Math.ceil(sorted.length / 50) - 1),
  );
  const nodesByID = useMemo(
    () => new Map(data.nodes.map((node) => [node.id, node])),
    [data.nodes],
  );
  const tags = [...new Set(data.nodes.flatMap((node) => node.tags || []))].sort(
    (a, b) => a.localeCompare(b, "ko"),
  );
  const ownerIDs = [
    ...new Set(data.nodes.map((node) => node.owner_id).filter(Boolean)),
  ];
  const spaceIDs = [
    ...new Set(
      data.nodes
        .map((node) => node.space_id)
        .filter((id): id is string => !!id),
    ),
  ];
  const selectedNode = nodesByID.get(selected);
  const source = selectedNode;
  useEffect(() => {
    if (!selectedNode) {
      setRelation(false);
      setRemove(null);
    }
  }, [!!selectedNode]);
  const connected = data.edges.filter(
    (edge) => edge.source === selected || edge.target === selected,
  );
  const writeRelation = async () => {
    if (!source?.can_write || !target || busy) return;
    setBusy(true);
    setRelationError(null);
    try {
      await api(`/documents/${source.id}/relations`, "POST", {
        target_id: target,
        type: relationType,
        expected_version: source.version,
      });
      if (!active.current) return;
      setRelation(false);
      setTarget("");
      await load(true);
      notify("문서 관계를 연결했습니다.");
    } catch (e) {
      if (active.current) setRelationError(e);
    } finally {
      if (active.current) setBusy(false);
    }
  };
  const deleteRelation = async () => {
    if (!remove || !source?.can_write || remove.source !== source.id || busy)
      return;
    setBusy(true);
    setRelationError(null);
    try {
      await api(
        `/documents/${source.id}/relations/${remove.target}?type=${encodeURIComponent(remove.type)}&expected_version=${source.version}`,
        "DELETE",
      );
      if (!active.current) return;
      setRemove(null);
      await load(true);
      notify("직접 연결한 관계를 해제했습니다. 문서는 그대로 유지됩니다.");
    } catch (e) {
      if (active.current) setRelationError(e);
    } finally {
      if (active.current) setBusy(false);
    }
  };
  return (
    <div className="page graph-page">
      <PageHeading
        eyebrow="CONNECTED KNOWLEDGE"
        title="지식 그래프"
        description="문서의 연결을 따라 생각을 확장하고, 아직 연결되지 않은 지식을 발견하세요."
        actions={
          <Button
            variant="secondary"
            disabled={loading || busy}
            onClick={() => void load()}
          >
            <RefreshCw size={17} />
            새로고침
          </Button>
        }
      />
      <ErrorBox error={error} />
      {data.diagnostics.notice && (
        <div className="notice">{data.diagnostics.notice}</div>
      )}
      {(data.diagnostics.truncated || data.diagnostics.pending > 0) && (
        <div className="notice">
          <Network size={20} />
          <span>
            {data.diagnostics.truncated
              ? `현재 접근 가능한 문서 중 최대 ${data.diagnostics.limit.toLocaleString("ko-KR")}개를 표시합니다. 필터 결과도 이 조회 범위에 한정됩니다. `
              : ""}
            {data.diagnostics.pending > 0
              ? `${data.diagnostics.pending.toLocaleString("ko-KR")}개 문서의 링크 색인을 처리 중입니다. 연결 정보는 자동 갱신됩니다.`
              : ""}
          </span>
        </div>
      )}
      <section
        className={`panel padded graph-filters${filtersExpanded ? " expanded" : ""}`}
        aria-label="그래프 필터"
      >
        <button
          className="graph-filter-toggle"
          aria-expanded={filtersExpanded}
          onClick={() => setFiltersExpanded((v) => !v)}
        >
          <Search size={17} />
          그래프 필터 {filtersExpanded ? "접기" : "펼치기"}
          {[query, tag, space, owner, focus, type].filter(Boolean).length >
            0 && (
            <Badge>
              {[query, tag, space, owner, focus, type].filter(Boolean).length}개
              적용
            </Badge>
          )}
        </button>
        <div className="graph-filter-grid">
          <Field label="그래프 문서 검색">
            <div className="search-field">
              <Search size={18} />
              <input
                placeholder="제목 또는 태그로 찾기"
                value={query}
                onChange={(e) => set("q", e.target.value)}
              />
            </div>
          </Field>
          <Field label="그래프 태그">
            <select value={tag} onChange={(e) => set("tag", e.target.value)}>
              <option value="">모든 태그</option>
              {tag && !tags.includes(tag) && (
                <option value={tag}>현재 조건: {tag}</option>
              )}
              {tags.map((t) => (
                <option key={t}>{t}</option>
              ))}
            </select>
          </Field>
          <Field label="그래프 공간">
            <select
              value={space}
              onChange={(e) => set("space", e.target.value)}
            >
              <option value="">모든 공간</option>
              <option value="none">공간 미지정</option>
              {space && space !== "none" && !spaceIDs.includes(space) && (
                <option value={space}>선택한 공간을 찾을 수 없음</option>
              )}
              {spaceIDs.map((id) => (
                <option value={id} key={id}>
                  {spaces.find((item) => item.id === id)?.name ||
                    `공간 ${id.slice(0, 8)}`}
                </option>
              ))}
            </select>
          </Field>
          <Field label="그래프 작성자">
            <select
              value={owner}
              onChange={(e) => set("owner", e.target.value)}
            >
              <option value="">모든 작성자</option>
              {owner && !ownerIDs.includes(owner) && (
                <option value={owner}>선택한 작성자를 찾을 수 없음</option>
              )}
              {ownerIDs.map((id) => (
                <option key={id} value={id}>
                  {members.find((member) => member.id === id)?.name ||
                    `사용자 ${id.slice(0, 8)}`}
                </option>
              ))}
            </select>
          </Field>
          <Field label="관계 유형">
            <select value={type} onChange={(e) => set("type", e.target.value)}>
              <option value="">모든 관계</option>
              {Object.entries(types).map(([id, name]) => (
                <option key={id} value={id}>
                  {name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="그래프 중심 문서">
            <select
              value={focus}
              onChange={(e) => set("focus", e.target.value)}
            >
              <option value="">전체 그래프</option>
              {focus && !nodesByID.has(focus) && (
                <option value={focus}>중심 문서에 접근할 수 없음</option>
              )}
              {data.nodes.map((node) => (
                <option value={node.id} key={node.id}>
                  {node.title} · {node.id.slice(0, 6)}
                </option>
              ))}
            </select>
          </Field>
          <Field label="연결 깊이">
            <select
              value={String(depth)}
              disabled={!focus}
              onChange={(e) => set("depth", e.target.value)}
            >
              {[1, 2, 3, 4, 5].map((n) => (
                <option key={n} value={n}>
                  {n}단계 연결
                </option>
              ))}
            </select>
          </Field>
          <Field label="그래프 배치">
            <select
              value={layout}
              onChange={(e) => set("layout", e.target.value)}
            >
              <option value="cose">연결 중심 배치</option>
              <option value="concentric">동심원 배치</option>
              <option value="breadthfirst">계층 배치</option>
            </select>
          </Field>
        </div>
        <div className="graph-filter-footer">
          <span>
            {focus
              ? "중심 문서는 다른 필터와 관계없이 표시합니다. 연결 깊이는 화면에 포함된 문서 사이에서 계산합니다."
              : "필터와 중심 문서, 연결 깊이는 주소에 저장되어 새로고침 후에도 유지됩니다."}
          </span>
          <Button variant="secondary" onClick={() => setParams({})}>
            <FilterX size={16} />
            필터 초기화
          </Button>
        </div>
      </section>
      {loading && !data.nodes.length ? (
        <Loading />
      ) : (
        <>
          <div className="graph-workbench">
            <section className="panel graph-visual">
              <div className="graph-section-heading">
                <h2>
                  <Network size={20} />
                  {focus ? "문서 중심 그래프" : "워크스페이스 그래프"}
                </h2>
                <Badge>
                  {filtered.nodes.length.toLocaleString("ko-KR")}개 문서 ·{" "}
                  {filtered.edges.length.toLocaleString("ko-KR")}개 연결
                </Badge>
              </div>
              {filtered.nodes.length ? (
                <GraphCanvas
                  nodes={filtered.nodes}
                  edges={filtered.edges}
                  selected={selected}
                  focus={focus}
                  layout={layout}
                  theme={user.preferences?.theme || "light"}
                  select={(id) => set("node", id)}
                  open={(id) => navigate(`/app/documents/${id}`)}
                />
              ) : (
                <Empty
                  title="표시할 문서가 없습니다"
                  text={
                    focus && !nodesByID.has(focus)
                      ? "중심 문서가 삭제되었거나 접근 권한이 변경되었습니다."
                      : "검색 조건을 조정하거나 문서에 위키 링크를 추가해 연결해 보세요."
                  }
                />
              )}
              <div className="graph-legend">
                <span>
                  <i className="reference" />
                  참조
                </span>
                <span>
                  <i className="related" />
                  관련
                </span>
                <span>
                  <i className="parent" />
                  상위 문서
                </span>
                {filtered.nodes.length > 300 && layout === "cose" && (
                  <small>
                    큰 그래프는 동심원으로 배치해 화면 반응성을 유지합니다.
                  </small>
                )}
              </div>
            </section>
            <aside className="panel padded graph-detail">
              <h2>선택한 문서</h2>
              {selectedNode ? (
                <>
                  <div className="graph-selected-title">
                    <DocIcon icon={selectedNode.icon} />
                    <h3>{selectedNode.title}</h3>
                  </div>
                  <div className="tag-list">
                    {selectedNode.tags?.map((t) => (
                      <Badge key={t}>#{t}</Badge>
                    ))}
                  </div>
                  <div className="graph-detail-actions">
                    <Link
                      className="button"
                      to={`/app/documents/${selectedNode.id}`}
                    >
                      <ArrowUpRight size={16} />
                      문서 열기
                    </Link>
                    <Button onClick={() => setPreview(selectedNode.id)}>
                      문서 미리보기
                    </Button>
                    <Link
                      className="button"
                      to={`/app/knowledge-impact?document_id=${selectedNode.id}`}
                    >
                      변경 영향 분석
                    </Link>
                    <Button
                      variant="secondary"
                      onClick={() => set("focus", selectedNode.id)}
                    >
                      <CircleDot size={16} />
                      중심으로 보기
                    </Button>
                  </div>
                  <ErrorBox error={relationError} />
                  <h3>연결 {connected.length}개</h3>
                  <div className="graph-relations">
                    {connected.map((edge, index) => {
                      const otherID =
                        edge.source === selected ? edge.target : edge.source;
                      return (
                        <div
                          key={`${edge.source}:${edge.target}:${edge.type}:${index}`}
                        >
                          <button onClick={() => set("node", otherID)}>
                            {nodesByID.get(otherID)?.title ||
                              "현재 조회 범위 밖의 문서"}
                            <small>
                              {edge.source === selected ? "나가는" : "들어오는"}{" "}
                              {types[edge.type] || "참조"} ·{" "}
                              {origins[edge.origin] || "위키 링크"}
                            </small>
                          </button>
                          <Button
                            aria-label={`${nodesByID.get(otherID)?.title || "연결 문서"} 미리보기`}
                            onClick={() => setPreview(otherID)}
                          >
                            미리보기
                          </Button>
                          {edge.origin === "manual" &&
                            edge.source === source?.id &&
                            source.can_write && (
                              <button
                                className="icon-button"
                                aria-label={`${nodesByID.get(otherID)?.title || "문서"} 관계 해제`}
                                onClick={() => setRemove(edge)}
                              >
                                <Unlink size={15} />
                              </button>
                            )}
                        </div>
                      );
                    })}
                    {!connected.length && (
                      <p className="muted">
                        현재 조회 범위에서 연결을 찾지 못했습니다.
                      </p>
                    )}
                  </div>
                  <Button
                    variant="secondary"
                    disabled={!source?.can_write}
                    onClick={() => {
                      setTarget("");
                      setRelationType("related");
                      setRelationError(null);
                      setRelation(true);
                    }}
                  >
                    <Plus size={17} />
                    관계 연결
                  </Button>
                  <p className="graph-detail-hint">
                    위키 링크는 원문에서 수정하고, 상위 관계는 문서 이동으로
                    변경합니다.
                  </p>
                </>
              ) : (
                <Empty
                  title="연결을 살펴보세요"
                  text="그래프의 노드나 아래 문서 목록을 선택하면 연결 정보가 나타납니다."
                />
              )}
            </aside>
          </div>
          <section className="panel padded graph-document-list">
            <div className="graph-section-heading">
              <h2>그래프 문서 목록</h2>
              <span>
                {sorted.length.toLocaleString("ko-KR")}개 · 키보드로 선택하고
                문서를 열 수 있습니다.
              </span>
            </div>
            <ul aria-label="그래프 문서 목록">
              {sorted
                .slice(visiblePage * 50, (visiblePage + 1) * 50)
                .map((node) => (
                  <li key={node.id}>
                    <button
                      className={selected === node.id ? "active" : ""}
                      aria-pressed={selected === node.id}
                      aria-label={`${node.title} 그래프에서 선택`}
                      onClick={() => set("node", node.id)}
                    >
                      <DocIcon icon={node.icon} size={18} />
                      <span>{node.title}</span>
                      {node.indexed === false && <small>색인 대기</small>}
                    </button>
                    <Link
                      to={`/app/documents/${node.id}`}
                      aria-label={`${node.title} 문서 열기`}
                    >
                      <ArrowUpRight size={17} />
                    </Link>
                  </li>
                ))}
            </ul>
            {sorted.length > 50 && (
              <div className="graph-pagination">
                <Button
                  variant="secondary"
                  disabled={visiblePage === 0}
                  onClick={() => setPage(visiblePage - 1)}
                >
                  이전
                </Button>
                <span>
                  {visiblePage + 1} / {Math.ceil(sorted.length / 50)}
                </span>
                <Button
                  variant="secondary"
                  disabled={(visiblePage + 1) * 50 >= sorted.length}
                  onClick={() => setPage(visiblePage + 1)}
                >
                  다음
                </Button>
              </div>
            )}
          </section>
          <div className="graph-diagnostics">
            <section className="panel padded">
              <h2>
                <Unlink size={20} />
                확인할 위키 링크 <Badge>{filtered.unresolved.length}</Badge>
              </h2>
              <p>
                연결할 문서가 없거나 같은 이름의 문서가 여러 개입니다. 문서 ID
                링크로 대상을 명확히 지정할 수 있습니다.
              </p>
              <ul>
                {filtered.unresolved.slice(0, 100).map((link, index) => (
                  <li key={`${link.source}:${index}`}>
                    <Link to={`/app/documents/${link.source}`}>
                      {nodesByID.get(link.source)?.title || "원본 문서"}
                    </Link>
                    <span>→ {link.target}</span>
                    <Badge>
                      {link.reason === "ambiguous"
                        ? "이름 중복"
                        : "대상 미확인"}
                    </Badge>
                  </li>
                ))}
              </ul>
              {filtered.unresolved.length > 100 && (
                <p>현재 필터의 첫 100개 항목을 표시합니다.</p>
              )}
              {!filtered.unresolved.length && (
                <p className="muted">
                  현재 표시 범위에는 확인할 위키 링크가 없습니다.
                </p>
              )}
            </section>
            <section className="panel padded">
              <h2>
                <GitFork size={20} />
                연결 없는 문서 <Badge>{filtered.isolated.length}</Badge>
              </h2>
              <p>
                현재 필터·조회 범위에서 연결이 없는 문서입니다. 아직 색인 중인
                문서는 제외합니다.
              </p>
              <ul>
                {filtered.isolated.slice(0, 100).map((node) => (
                  <li key={node.id}>
                    <Link to={`/app/documents/${node.id}`}>{node.title}</Link>
                    <button
                      className="text-button"
                      onClick={() => set("node", node.id)}
                    >
                      관계 살펴보기
                    </button>
                  </li>
                ))}
              </ul>
              {filtered.isolated.length > 100 && (
                <p>현재 필터의 첫 100개 항목을 표시합니다.</p>
              )}
              {!filtered.isolated.length && (
                <p className="muted">
                  현재 표시 범위에는 고립된 문서가 없습니다.
                </p>
              )}
            </section>
          </div>
        </>
      )}
      {preview && (
        <Suspense fallback={<Loading />}>
          <DocumentPreview
            documentId={preview}
            onClose={() => setPreview(null)}
          />
        </Suspense>
      )}
      <Modal
        open={relation}
        onOpenChange={setRelation}
        title="문서 관계 연결"
        description="참조 또는 관련 관계를 추가합니다. 문서의 공개 범위나 권한은 변경하지 않습니다."
      >
        <ErrorBox error={relationError} />
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void writeRelation();
          }}
        >
          <Field label="연결할 문서">
            <select
              value={target}
              required
              disabled={busy}
              onChange={(e) => setTarget(e.target.value)}
            >
              <option value="">문서를 선택하세요</option>
              {data.nodes
                .filter((node) => node.id !== source?.id)
                .map((node) => (
                  <option value={node.id} key={node.id}>
                    {node.title} · {node.id.slice(0, 6)}
                  </option>
                ))}
            </select>
          </Field>
          <Field label="새 관계 유형">
            <select
              value={relationType}
              disabled={busy}
              onChange={(e) => setRelationType(e.target.value)}
            >
              <option value="related">관련</option>
              <option value="reference">참조</option>
              <option value="policy">
                정책 의존 · 이 문서가 대상 정책을 따름
              </option>
              <option value="execution">
                실행 의존 · 이 문서가 대상 절차에 의존
              </option>
              <option value="data">
                데이터 의존 · 이 문서가 대상 데이터를 사용
              </option>
            </select>
          </Field>
          <div className="modal-actions">
            <Button
              type="button"
              variant="secondary"
              disabled={busy}
              onClick={() => setRelation(false)}
            >
              취소
            </Button>
            <Button disabled={busy || !target || !source?.can_write}>
              <Link2 size={17} />
              {busy ? "연결 중…" : "관계 연결"}
            </Button>
          </div>
        </form>
      </Modal>
      <Modal
        open={!!remove}
        onOpenChange={(v) => {
          if (!v) setRemove(null);
        }}
        title="직접 연결한 관계를 해제할까요?"
        description="이 관계만 해제하며 원본 문서와 대상 문서는 삭제하지 않습니다."
      >
        <ErrorBox error={relationError} />
        <div className="modal-actions">
          <Button
            variant="secondary"
            disabled={busy}
            onClick={() => setRemove(null)}
          >
            취소
          </Button>
          <Button variant="danger" disabled={busy} onClick={deleteRelation}>
            <Trash2 size={16} />
            관계 해제
          </Button>
        </div>
      </Modal>
    </div>
  );
}
