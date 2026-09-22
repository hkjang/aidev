import { useEffect, useState } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import {
  ArrowDownToLine,
  ArrowRight,
  ArrowUpRight,
  BookOpen,
  CalendarDays,
  CheckCircle2,
  ChevronRight,
  Clock,
  Database,
  FileText,
  FolderOpen,
  Import,
  Link2,
  MoreHorizontal,
  Network,
  Plus,
  Search,
  ShieldCheck,
  Sparkles,
  Star,
  Tag,
  Trash2,
  Upload,
  Users,
  Workflow,
  X,
} from "lucide-react";
import { api, date, download, type DocSummary } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
  roleNames,
  statusNames,
  DocIcon,
} from "./ui";

export function HomePage() {
  const { user, documents, workspace, createDocument, notify } = useApp();
  const navigate = useNavigate();
  const [creating, setCreating] = useState(false);
  const tags = [...new Set(documents.flatMap((d) => d.tags || []))];
  const hour = new Date().getHours();
  return (
    <>
      <PageHeading
        eyebrow={new Date().toLocaleDateString("ko-KR", {
          year: "numeric",
          month: "long",
          day: "numeric",
          weekday: "long",
        })}
        title={`${hour < 12 ? "좋은 아침이에요" : hour < 18 ? "반가워요" : "오늘도 수고했어요"}, ${user.name || "사용자"}님`}
        description="오늘의 생각이 내일의 지식이 됩니다."
        actions={
          <Button
            variant="primary"
            disabled={creating}
            onClick={async () => {
              setCreating(true);
              const d = await createDocument();
              setCreating(false);
              if (d) navigate(`/app/documents/${d.id}`);
            }}
          >
            <Plus size={18} /> 새 문서
          </Button>
        }
      />
      <section className="welcome-banner">
        <div>
          <Badge tone="light">
            <span className="status-dot" /> CONNECT YOUR KNOWLEDGE
          </Badge>
          <h2>
            작은 기록에서,
            <br />
            함께 쓰는 지식으로.
          </h2>
          <p>
            문서를 연결하고, 아이디어를 나누고,
            <br className="mobile-break" /> 우리 팀의 다음 가능성을 발견하세요.
          </p>
          <Link className="banner-link" to="/app/graph">
            지식 그래프 둘러보기 <ArrowUpRight size={18} />
          </Link>
        </div>
        <div className="banner-art" aria-hidden="true">
          <div className="orbit orbit-one" />
          <div className="orbit orbit-two" />
          <div className="art-node node-main">
            <img src="/favicon.svg" alt="" />
          </div>
          <div className="art-node node-book">
            <BookOpen size={29} />
          </div>
          <div className="art-node node-idea">
            <Sparkles size={25} />
          </div>
          <div className="art-node node-team">
            <Users size={25} />
          </div>
          <span className="art-label label-one">생각을 연결하다</span>
          <span className="art-label label-two">함께 성장하다</span>
          <i className="art-dot dot-one" />
          <i className="art-dot dot-two" />
        </div>
      </section>
      <div className="stats-grid">
        <div className="stat-card">
          <span className="stat-icon teal">
            <FileText size={22} />
          </span>
          <div>
            <span>워크스페이스 문서</span>
            <strong>
              {documents.length}
              <small>개</small>
            </strong>
          </div>
          <Link to="/app/documents" aria-label="모든 문서 보기">
            <ArrowUpRight size={19} />
          </Link>
        </div>
        <div className="stat-card">
          <span className="stat-icon amber">
            <Star size={22} />
          </span>
          <div>
            <span>즐겨찾는 문서</span>
            <strong>
              {documents.filter((d) => d.is_favorite).length}
              <small>개</small>
            </strong>
          </div>
          <Link to="/app/favorites" aria-label="즐겨찾기 보기">
            <ArrowUpRight size={19} />
          </Link>
        </div>
        <div className="stat-card">
          <span className="stat-icon lavender">
            <Tag size={22} />
          </span>
          <div>
            <span>연결된 태그</span>
            <strong>
              {tags.length}
              <small>개</small>
            </strong>
          </div>
          <Link to="/app/search" aria-label="태그 검색">
            <ArrowUpRight size={19} />
          </Link>
        </div>
      </div>
      <div className="home-columns">
        <section className="panel recent-panel">
          <div className="section-heading">
            <div>
              <Clock size={19} />
              <h2>최근 업데이트</h2>
            </div>
            <Link to="/app/documents">
              모두 보기 <ChevronRight size={16} />
            </Link>
          </div>
          {documents.length ? (
            <div className="recent-list">
              {documents.slice(0, 6).map((d) => (
                <Link
                  to={`/app/documents/${d.id}`}
                  key={d.id}
                  className="recent-item"
                >
                  <span className="document-icon">
                    <DocIcon icon={d.icon} size={22} />
                  </span>
                  <div>
                    <strong>{d.title}</strong>
                    <span>
                      {(d.tags || [])
                        .slice(0, 2)
                        .map((t) => `#${t}`)
                        .join("  ") ||
                        workspace?.name ||
                        "워크스페이스"}
                    </span>
                  </div>
                  <Badge>{statusNames[d.status] || d.status}</Badge>
                  <time>{date(d.updated_at)}</time>
                  <ChevronRight size={17} />
                </Link>
              ))}
            </div>
          ) : (
            <Empty />
          )}
        </section>
        <section className="panel quick-panel">
          <div className="section-heading">
            <div>
              <Sparkles size={19} />
              <h2>빠르게 시작하기</h2>
            </div>
          </div>
          <button
            className="quick-action"
            onClick={async () => {
              const today = new Date().toLocaleDateString("sv-SE");
              const d =
                documents.find((d) => d.title === today) ||
                (await createDocument(
                  today,
                  `# ${today}\n\n## 오늘의 할 일\n\n- [ ] \n\n## 오늘의 기록\n`,
                ));
              if (d) navigate(`/app/documents/${d.id}`);
            }}
          >
            <span className="quick-icon peach">
              <CalendarDays size={22} />
            </span>
            <span>
              <strong>오늘의 노트</strong>
              <small>하루의 생각을 가볍게 기록하세요</small>
            </span>
            <ChevronRight size={17} />
          </button>
          <Link to="/app/templates" className="quick-action">
            <span className="quick-icon lavender">
              <BookOpen size={22} />
            </span>
            <span>
              <strong>템플릿으로 시작</strong>
              <small>회의록, 프로젝트, 의사결정 기록</small>
            </span>
            <ChevronRight size={17} />
          </Link>
          <Link to="/app/import" className="quick-action">
            <span className="quick-icon teal">
              <Import size={22} />
            </span>
            <span>
              <strong>기존 문서 가져오기</strong>
              <small>Markdown과 Obsidian Vault</small>
            </span>
            <ChevronRight size={17} />
          </Link>
          <div className="keyboard-tip">
            <CommandIcon />
            <p>
              <strong>생각의 속도로 탐색하세요</strong>
              <span>
                <kbd>Ctrl</kbd> + <kbd>K</kbd> 빠른 검색과 명령
              </span>
            </p>
          </div>
        </section>
      </div>
      <div className="workspace-footer">
        <span>
          <ShieldCheck size={15} /> 조직 안에서 안전하게 연결되는 지식
        </span>
        <span>BUILT FOR YOUR KNOWLEDGE · madi</span>
      </div>
    </>
  );
}
function CommandIcon() {
  return <span className="command-symbol">⌘</span>;
}

export function DocumentList({ mode = "all" }: { mode?: string }) {
  const [searchParams] = useSearchParams();
  const { workspace, documents, reload, createDocument, notify } = useApp();
  const navigate = useNavigate();
  const [items, setItems] = useState<DocSummary[]>([]),
    [query, setQuery] = useState(searchParams.get("q") || ""),
    [tag, setTag] = useState(""),
    [loading, setLoading] = useState(false),
    [error, setError] = useState(""),
    [view, setView] = useState("list");
  useEffect(() => {
    if (!workspace) return;
    let active = true;
    const t = setTimeout(
      () => {
        setLoading(true);
        api<DocSummary[]>(
          `/documents?workspace_id=${workspace.id}${mode === "trash" ? "&trash=1" : mode === "favorites" ? "&favorite=1" : ""}&q=${encodeURIComponent(query)}&tag=${encodeURIComponent(tag)}`,
        )
          .then((v) => {
            if (active) {
              setItems(v);
              setError("");
            }
          })
          .catch((e) => active && setError(e.message))
          .finally(() => active && setLoading(false));
      },
      mode === "search" ? 200 : 0,
    );
    return () => {
      active = false;
      clearTimeout(t);
    };
  }, [workspace?.id, query, tag, mode, documents]);
  const names: Record<string, string> = {
    all: "모든 문서",
    search: "지식 검색",
    favorites: "즐겨찾기",
    trash: "휴지통",
  };
  return (
    <>
      <PageHeading
        eyebrow="KNOWLEDGE LIBRARY"
        title={names[mode]}
        description={
          mode === "trash"
            ? "삭제한 문서를 확인하고 다시 복원할 수 있습니다."
            : mode === "search"
              ? "문서의 제목, 내용, 태그에서 필요한 지식을 찾아보세요."
              : "우리 팀이 함께 쌓아가는 살아 있는 지식입니다."
        }
        actions={
          mode !== "trash" && (
            <Button
              variant="primary"
              onClick={async () => {
                const d = await createDocument();
                if (d) navigate(`/app/documents/${d.id}`);
              }}
            >
              <Plus size={18} /> 새 문서
            </Button>
          )
        }
      />
      <div className="filter-bar">
        <div className="search-field">
          <Search size={19} />
          <input
            aria-label="문서 검색"
            placeholder="제목, 내용, 태그 검색…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          {query && (
            <button
              aria-label="검색 지우기"
              className="icon-button"
              onClick={() => setQuery("")}
            >
              <X size={16} />
            </button>
          )}
        </div>
        <select
          aria-label="태그 필터"
          value={tag}
          onChange={(e) => setTag(e.target.value)}
        >
          <option value="">모든 태그</option>
          {[...new Set(documents.flatMap((d) => d.tags || []))].map((t) => (
            <option key={t} value={t}>
              #{t}
            </option>
          ))}
        </select>
        <Badge>{items.length}개 문서</Badge>
      </div>
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : items.length ? (
        <div className="panel document-table">
          <div className="document-table-header">
            <span>문서 이름</span>
            <span>태그</span>
            <span>상태</span>
            <span>업데이트</span>
            <span />
          </div>
          {items.map((d) => (
            <div className="document-table-row" key={d.id}>
              <Link to={`/app/documents/${d.id}`}>
                <span className="document-icon">
                  <DocIcon icon={d.icon} size={20} />
                </span>
                <div>
                  <strong>{d.title}</strong>
                  <small>
                    {(d.excerpt || "").replace(/[#*`\[\]]/g, "").slice(0, 85) ||
                      "내용을 추가해 보세요."}
                  </small>
                </div>
              </Link>
              <div className="tags">
                {(d.tags || []).slice(0, 2).map((t) => (
                  <Badge key={t}>{t}</Badge>
                ))}
              </div>
              <span>
                <Badge tone={d.status === "published" ? "green" : ""}>
                  {statusNames[d.status] || d.status}
                </Badge>
              </span>
              <time>{date(d.updated_at)}</time>
              <button
                className="icon-button"
                aria-label={
                  mode === "trash"
                    ? "문서 복원"
                    : d.is_favorite
                      ? "즐겨찾기 해제"
                      : "즐겨찾기 추가"
                }
                onClick={async () => {
                  try {
                    await api(
                      `/documents/${d.id}/${mode === "trash" ? "restore" : "favorite"}`,
                      "POST",
                      mode === "trash" ? {expected_version:d.version} : undefined,
                    );
                    await reload();
                    notify(
                      mode === "trash"
                        ? "문서를 복원했습니다."
                        : "즐겨찾기를 변경했습니다.",
                    );
                  } catch (e) {
                    notify((e as Error).message, "error");
                  }
                }}
              >
                {mode === "trash" ? (
                  <HistoryIcon />
                ) : (
                  <Star
                    size={18}
                    fill={d.is_favorite ? "currentColor" : "none"}
                  />
                )}
              </button>
            </div>
          ))}
        </div>
      ) : (
        <Empty
          title={
            query
              ? "검색 결과가 없어요"
              : mode === "trash"
                ? "휴지통이 비어 있어요"
                : mode === "favorites"
                  ? "자주 찾는 문서를 모아보세요"
                  : "아직 문서가 없어요"
          }
          text={
            query
              ? "다른 검색어나 태그로 다시 찾아보세요."
              : mode === "favorites"
                ? "문서 옆의 별 아이콘을 누르면 이곳에 모입니다."
                : mode === "trash"
                  ? "삭제한 문서는 보존 정책에 따라 이곳에 보관됩니다."
                  : "새 문서를 만들거나 기존 Markdown 문서를 가져오세요."
          }
        />
      )}
    </>
  );
}
function HistoryIcon() {
  return <ArrowUpRight size={18} />;
}

export function GraphPage() {
  const { workspace, notify } = useApp();
  const navigate = useNavigate();
  const [data, setData] = useState<{
      nodes: { id: string; title: string; tags: string[] }[];
      edges: { source: string; target: string }[];
    }>({ nodes: [], edges: [] }),
    [query, setQuery] = useState(""),
    [tag, setTag] = useState(""),
    [focus, setFocus] = useState(""),
    [depth, setDepth] = useState("2"),
    [error, setError] = useState("");
  useEffect(() => {
    if (workspace)
      api("/graph?workspace_id=" + workspace.id)
        .then(setData)
        .catch((e) => setError(e.message));
  }, [workspace?.id]);
  let nodes = data.nodes.filter(
    (n) =>
      n.title.toLowerCase().includes(query.toLowerCase()) &&
      (!tag || n.tags?.includes(tag)),
  );
  if (focus) {
    const reachable = new Set([focus]);
    for (let i = 0; i < Number(depth); i++) {
      const previous = new Set(reachable);
      for (const e of data.edges) {
        if (previous.has(e.source) || previous.has(e.target)) {
          reachable.add(e.source);
          reachable.add(e.target);
        }
      }
    }
    nodes = nodes.filter((n) => reachable.has(n.id));
  }
  const count = nodes.length;
  const points = new Map(
    nodes.map((n, i) => [
      n.id,
      {
        x:
          480 +
          Math.cos((i / count) * Math.PI * 2 - Math.PI / 2) *
            (count === 1 ? 0 : count < 5 ? 170 : 240 + (i % 2) * 30),
        y:
          290 +
          Math.sin((i / count) * Math.PI * 2 - Math.PI / 2) *
            (count === 1 ? 0 : count < 5 ? 150 : 210),
        node: n,
      },
    ]),
  );
  const edges = data.edges.filter(
    (e) => points.has(e.source) && points.has(e.target),
  );
  return (
    <>
      <PageHeading
        eyebrow="CONNECTED IDEAS"
        title="지식 그래프"
        description="문서와 문서 사이, 새로운 연결을 발견하세요."
        actions={
          <Badge>
            <Network size={14} />
            {data.nodes.length}개 문서 · {data.edges.length}개 연결
          </Badge>
        }
      />
      <div className="filter-bar">
        <div className="search-field">
          <Search size={19} />
          <input
            aria-label="그래프 문서 검색"
            placeholder="문서 검색…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        <select
          aria-label="그래프 태그"
          value={tag}
          onChange={(e) => setTag(e.target.value)}
        >
          <option value="">모든 태그</option>
          {[...new Set(data.nodes.flatMap((n) => n.tags || []))].map((t) => (
            <option key={t}>{t}</option>
          ))}
        </select>
        <select
          aria-label="그래프 중심 문서"
          value={focus}
          onChange={(e) => setFocus(e.target.value)}
        >
          <option value="">전체 그래프</option>
          {data.nodes.map((n) => (
            <option value={n.id} key={n.id}>
              {n.title}
            </option>
          ))}
        </select>
        {focus && (
          <select
            aria-label="연결 깊이"
            value={depth}
            onChange={(e) => setDepth(e.target.value)}
          >
            {[1, 2, 3, 4, 5].map((n) => (
              <option key={n} value={n}>
                {n}단계 연결
              </option>
            ))}
          </select>
        )}
      </div>
      <ErrorBox error={error} />
      <div className="graph-panel">
        {nodes.length ? (
          <svg
            className="knowledge-graph"
            viewBox="0 0 960 580"
            role="img"
            aria-label="문서 연결 그래프"
          >
            <defs>
              <pattern
                id="dots"
                x="0"
                y="0"
                width="24"
                height="24"
                patternUnits="userSpaceOnUse"
              >
                <circle cx="1" cy="1" r="1" fill="var(--border)" />
              </pattern>
            </defs>
            <rect width="100%" height="100%" fill="url(#dots)" />
            {edges.map((e, i) => {
              const a = points.get(e.source)!,
                b = points.get(e.target)!;
              return (
                <line
                  key={i}
                  x1={a.x}
                  y1={a.y}
                  x2={b.x}
                  y2={b.y}
                  stroke="var(--graph-line)"
                  strokeWidth="2"
                />
              );
            })}
            {[...points.values()].map(({ x, y, node }, i) => (
              <g
                key={node.id}
                className="graph-node"
                tabIndex={0}
                role="link"
                aria-label={`${node.title} 문서 열기`}
                onClick={() => navigate(`/app/documents/${node.id}`)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") navigate(`/app/documents/${node.id}`);
                }}
              >
                <circle
                  cx={x}
                  cy={y}
                  r="30"
                  fill={["#deeee3", "#ede7f3", "#fae7d5", "#dbeaec"][i % 4]}
                  stroke={["#72a18a", "#ab97c5", "#d5ad78", "#7ca6b1"][i % 4]}
                  strokeWidth="2"
                />
                <text
                  x={x}
                  y={y + 5}
                  textAnchor="middle"
                  fontSize="19"
                  fill="#315447"
                >
                  {node.title.slice(0, 1)}
                </text>
                <text
                  x={x}
                  y={y + 54}
                  textAnchor="middle"
                  fontSize="15"
                  fill="var(--text)"
                >
                  {node.title.length > 22
                    ? node.title.slice(0, 22) + "…"
                    : node.title}
                </text>
              </g>
            ))}
          </svg>
        ) : (
          <Empty
            title="지식의 연결을 시작해 보세요"
            text="문서에 [[문서 제목]]을 입력하면 지식 그래프에서 연결됩니다."
          />
        )}
        <div className="graph-legend">
          <span>
            <i />
            문서
          </span>
          <span>
            <b />
            위키 링크
          </span>
          <small>문서를 클릭하면 바로 열 수 있어요</small>
        </div>
      </div>
      <div className="notice">
        <Link2 size={19} />
        <span>
          문서에 <code>[[문서 제목]]</code>을 입력해 보세요. 연결된 문서와
          백링크가 자동으로 만들어집니다.
        </span>
      </div>
    </>
  );
}

export function TasksPage() {
  const { workspace, documents, reload, notify } = useApp();
  const [items, setItems] = useState<any[]>([]),
    [filter, setFilter] = useState("all"),
    [error, setError] = useState("");
  const load = () =>
    workspace &&
    api<any[]>("/tasks?workspace_id=" + workspace.id)
      .then(setItems)
      .catch((e) => setError(e.message));
  useEffect(() => {
    load();
  }, [workspace?.id, documents]);
  const filtered = items.filter(
    (t) => filter === "all" || (filter === "done" ? t.done : !t.done),
  );
  return (
    <>
      <PageHeading
        eyebrow="ONE STEP AT A TIME"
        title="내 할 일"
        description="문서 속 체크리스트를 한곳에서 관리하세요."
      />
      <div className="stats-grid">
        <div className="mini-stat">
          <strong>{items.length}</strong>
          <span>전체 할 일</span>
        </div>
        <div className="mini-stat">
          <strong>{items.filter((t) => !t.done).length}</strong>
          <span>진행 중</span>
        </div>
        <div className="mini-stat">
          <strong>{items.filter((t) => t.done).length}</strong>
          <span>완료</span>
        </div>
      </div>
      <div className="tabs">
        {[
          ["all", "전체"],
          ["open", "진행 중"],
          ["done", "완료"],
        ].map(([id, label]) => (
          <button
            key={id}
            className={filter === id ? "active" : ""}
            onClick={() => setFilter(id)}
          >
            {label}
          </button>
        ))}
      </div>
      <ErrorBox error={error} />
      <div className="panel">
        {filtered.length ? (
          filtered.map((t, i) => (
            <div
              className={`task-row ${t.done ? "completed" : ""}`}
              key={`${t.document_id}-${t.line}`}
            >
              <input
                type="checkbox"
                checked={t.done}
                aria-label={`${t.text} 완료 여부`}
                onChange={async (e) => {
                  try {
                    await api("/tasks", "PUT", {
                      document_id: t.document_id,
                      line: t.line,
                      done: e.target.checked,
                      version: t.version,
                    });
                    load();
                    await reload();
                  } catch (e) {
                    notify((e as Error).message, "error");
                  }
                }}
              />
              <div>
                <strong>{t.text}</strong>
                <Link to={`/app/documents/${t.document_id}`}>
                  <FileText size={14} />
                  {t.title}
                </Link>
              </div>
              <Badge tone={t.done ? "green" : ""}>
                {t.done ? "완료" : "진행 중"}
              </Badge>
            </div>
          ))
        ) : (
          <Empty
            title={
              filter === "done"
                ? "완료한 할 일이 아직 없어요"
                : "모든 할 일을 한눈에"
            }
            text="문서에 '- [ ] 할 일'을 작성하면 이곳에 자동으로 모입니다."
          />
        )}
      </div>
    </>
  );
}

export const templates = [
  {
    name: "회의록",
    icon: "📝",
    category: "팀 협업",
    description: "논의와 결정을 명확하게 남기세요.",
    markdown:
      "# 회의록\n\n## 회의 정보\n\n- 일시: \n- 참석자: \n\n## 안건\n\n1. \n\n## 논의 내용\n\n\n## 결정 사항\n\n\n## 다음 할 일\n\n- [ ] 담당자 · 기한 · 할 일\n",
  },
  {
    name: "프로젝트 계획",
    icon: "🚀",
    category: "프로젝트",
    description: "목표부터 실행까지 하나의 문서로.",
    markdown:
      "# 프로젝트 계획\n\n## 프로젝트 개요\n\n## 목표와 성공 기준\n\n- \n\n## 범위\n\n### 포함\n\n### 제외\n\n## 일정\n\n| 단계 | 담당자 | 완료일 |\n| --- | --- | --- |\n| 기획 | | |\n\n## 실행 체크리스트\n\n- [ ] \n\n## 관련 문서\n",
  },
  {
    name: "의사결정 기록",
    icon: "🧭",
    category: "지식 관리",
    description: "무엇을, 왜 결정했는지 기록하세요.",
    markdown:
      "# 의사결정 기록\n\n## 배경\n\n## 검토한 선택지\n\n1. \n2. \n\n## 결정\n\n## 결정 사유\n\n## 예상 결과와 후속 작업\n\n- [ ] \n",
  },
  {
    name: "운영 런북",
    icon: "🛠️",
    category: "IT 운영",
    description: "반복되는 운영 작업을 안전하게.",
    markdown:
      "# 운영 런북\n\n## 목적\n\n## 사전 조건\n\n- [ ] \n\n## 실행 절차\n\n### 1. 상태 확인\n\n```shell\n# 확인 명령\n```\n\n## 검증\n\n- [ ] \n\n## 롤백\n\n## 담당자와 마지막 검증일\n",
  },
  {
    name: "주간 회고",
    icon: "🌱",
    category: "팀 협업",
    description: "한 주를 돌아보고 함께 성장하세요.",
    markdown:
      "# 주간 회고\n\n## 이번 주 성과\n\n## 잘한 점\n\n## 개선할 점\n\n## 배운 점\n\n## 다음 주 시도할 일\n\n- [ ] \n",
  },
  {
    name: "지식 정리",
    icon: "📚",
    category: "지식 관리",
    description: "배운 내용을 오래 남는 지식으로.",
    markdown:
      "# 지식 정리\n\n## 한 줄 요약\n\n> \n\n## 핵심 개념\n\n## 적용 예시\n\n## 참고 자료\n\n## 연결된 지식\n\n[[관련 문서]]\n",
  },
];
export function TemplatesPage() {
  const { createDocument } = useApp();
  const navigate = useNavigate();
  const [busy, setBusy] = useState("");
  return (
    <>
      <PageHeading
        eyebrow="A LITTLE HELP TO GET STARTED"
        title="템플릿"
        description="좋은 시작을 위한 작은 틀. 우리 팀에 맞게 자유롭게 바꿔보세요."
      />
      <div className="template-grid">
        {templates.map((t) => (
          <article className="template-card" key={t.name}>
            <div className="template-preview">
              <span>{t.icon}</span>
              <div className="preview-line long" />
              <div className="preview-line" />
              <div className="preview-line short" />
            </div>
            <div className="template-body">
              <Badge>{t.category}</Badge>
              <h2>{t.name}</h2>
              <p>{t.description}</p>
              <Button
                disabled={!!busy}
                onClick={async () => {
                  setBusy(t.name);
                  const d = await createDocument(t.name, t.markdown, {
                    icon: t.icon,
                  });
                  setBusy("");
                  if (d) navigate(`/app/documents/${d.id}`);
                }}
              >
                {busy === t.name ? "만드는 중…" : "이 템플릿 사용하기"}
                <ArrowUpRight size={16} />
              </Button>
            </div>
          </article>
        ))}
      </div>
    </>
  );
}

export function ImportPage() {
  const { workspace, reload, notify } = useApp();
  const [file, setFile] = useState<File | null>(null),
    [busy, setBusy] = useState(false),
    [result, setResult] = useState<any>(null),
    [error, setError] = useState("");
  return (
    <>
      <PageHeading
        eyebrow="YOUR KNOWLEDGE, YOUR WAY"
        title="가져오기 / 내보내기"
        description="쌓아온 지식을 연결하고, 언제든 내 문서로 간직하세요."
      />
      <div className="two-columns">
        <section className="panel padded">
          <span className="feature-icon teal">
            <Upload size={28} />
          </span>
          <h2>기존 문서 가져오기</h2>
          <p className="muted">
            Markdown 파일 또는 Obsidian Vault ZIP을 현재 워크스페이스로
            가져옵니다.
          </p>
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              if (!file || !workspace) return;
              setBusy(true);
              setError("");
              setResult(null);
              const data = new FormData();
              data.append("file", file);
              try {
                setResult(
                  await api(
                    "/import?workspace_id=" + workspace.id,
                    "POST",
                    data,
                  ),
                );
                await reload();
                notify("문서 가져오기를 완료했습니다.");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <label className="upload-zone">
              <Import size={32} />
              <strong>
                {file ? file.name : "파일을 선택하거나 끌어 놓으세요"}
              </strong>
              <span>.md, .markdown, .zip</span>
              <input
                type="file"
                accept=".md,.markdown,.zip"
                required
                onChange={(e) => setFile(e.target.files?.[0] || null)}
              />
            </label>
            <Button variant="primary" disabled={busy || !file}>
              {busy ? "가져오는 중…" : "문서 가져오기"}
              <ArrowRight size={17} />
            </Button>
          </form>
          <ErrorBox error={error} />
          {result && (
            <div className="notice">
              <CheckCircle2 size={20} />
              <span>
                {result.imported}개 문서를 가져왔습니다.
                {result.errors?.length > 0 && (
                  <ul>
                    {result.errors.map((s: string, i: number) => (
                      <li key={i}>{s}</li>
                    ))}
                  </ul>
                )}
              </span>
            </div>
          )}
        </section>
        <section className="panel padded">
          <span className="feature-icon lavender">
            <ArrowDownToLine size={28} />
          </span>
          <h2>워크스페이스 내보내기</h2>
          <p className="muted">
            문서를 Markdown ZIP으로 다운로드합니다. Front Matter와 문서 링크를
            함께 보관할 수 있습니다.
          </p>
          <div className="export-summary">
            <FolderOpen size={28} />
            <div>
              <strong>{workspace?.name || "워크스페이스"}</strong>
              <small>Markdown · ZIP 아카이브</small>
            </div>
            <Badge>열린 형식</Badge>
          </div>
          <Button
            onClick={async () => {
              if (!workspace) return;
              try {
                await download(
                  "/export?workspace_id=" + workspace.id,
                  `madi-${workspace.slug || workspace.id}.zip`,
                );
                notify("내보내기 파일을 다운로드했습니다.");
              } catch (e) {
                notify((e as Error).message, "error");
              }
            }}
          >
            <DownloadIcon /> Markdown ZIP 다운로드
          </Button>
          <div className="notice subtle">
            <ShieldCheck size={18} />
            <span>
              내보내기는 현재 사용자가 접근할 수 있는 문서만 포함합니다.
            </span>
          </div>
        </section>
      </div>
    </>
  );
}
function DownloadIcon() {
  return <ArrowDownToLine size={18} />;
}

export function MembersPage() {
  const { workspace, user, notify } = useApp();
  const [items, setItems] = useState<any[]>([]),
    [email, setEmail] = useState(""),
    [role, setRole] = useState("editor"),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  const load = () =>
    workspace &&
    api<any[]>(`/workspaces/${workspace.id}/members`)
      .then(setItems)
      .catch((e) => setError(e.message));
  useEffect(() => {
    load();
  }, [workspace?.id]);
  const canManage =
    user.role === "admin" || ["admin", "owner"].includes(workspace?.role || "");
  return (
    <>
      <PageHeading
        eyebrow="BETTER TOGETHER"
        title="워크스페이스 멤버"
        description="함께할 멤버를 추가하고 역할을 관리하세요."
      />
      {canManage && (
        <form
          className="panel padded member-form"
          onSubmit={async (e) => {
            e.preventDefault();
            if (!workspace) return;
            setBusy(true);
            try {
              await api(`/workspaces/${workspace.id}/members`, "PUT", {
                email,
                role,
              });
              setEmail("");
              load();
              notify("멤버 권한을 저장했습니다.");
            } catch (e) {
              notify((e as Error).message, "error");
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field label="멤버 이메일">
            <input
              required
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="member@company.com"
            />
          </Field>
          <Field label="워크스페이스 역할">
            <select value={role} onChange={(e) => setRole(e.target.value)}>
              {["admin", "editor", "commenter", "viewer"].map((r) => (
                <option key={r} value={r}>
                  {roleNames[r]}
                </option>
              ))}
            </select>
          </Field>
          <Button variant="primary" disabled={busy}>
            <Plus size={18} /> 멤버 추가 / 변경
          </Button>
        </form>
      )}
      <ErrorBox error={error} />
      <div className="panel">
        {items.map((m) => (
          <div className="member-row" key={m.user_id || m.id}>
            <span className="avatar">
              {(m.name || m.user_name || m.email || "M").slice(0, 1)}
            </span>
            <div>
              <strong>{m.name || m.user_name || m.email}</strong>
              <small>{m.email}</small>
            </div>
            <Badge>{roleNames[m.role] || m.role}</Badge>
            {canManage && m.role !== "owner" && (
              <Button
                variant="ghost danger"
                onClick={async () => {
                  if (
                    !window.confirm(
                      `${m.name || m.email} 님을 워크스페이스에서 제외할까요?`,
                    )
                  )
                    return;
                  try {
                    await api(
                      `/workspaces/${workspace?.id}/members/${m.user_id || m.id}`,
                      "DELETE",
                    );
                    load();
                    notify("멤버를 제외했습니다.");
                  } catch (e) {
                    notify((e as Error).message, "error");
                  }
                }}
              >
                제외
              </Button>
            )}
          </div>
        ))}
        {!items.length && !error && (
          <Empty
            title="등록된 멤버가 없어요"
            text="기존 서비스 계정의 이메일로 멤버를 추가하세요."
          />
        )}
      </div>
    </>
  );
}
