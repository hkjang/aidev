import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  Search,
  SlidersHorizontal,
  ShieldCheck,
  RefreshCw,
  ArrowRight,
  FileText,
  Database,
  MessageSquare,
  Paperclip,
  Hash,
  CheckSquare,
  Code2,
  Layers,
  Users,
  Eye,
  X,
  Settings2,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  PageHeading,
} from "./ui";
import "./search.css";
import NaturalSearch from "./NaturalSearch";
const SearchAI = lazy(() => import("./AI"));
const DocumentPreview = lazy(() => import("./review/DocumentPreview"));

// Navigation-only, in-memory state: no snippets, Markdown or AI answers are
// retained. Actor/workspace scoping prevents reuse after an account switch.
const searchNavigation = new Map<
  string,
  { scroll: number; cursors: string[] }
>();
const filterKeys = [
  "q",
  "type",
  "space_id",
  "author_id",
  "tag",
  "status",
  "from",
  "to",
  "has_attachment",
  "sort",
  "cursor",
];
function navigationState(key: string) {
  let value = searchNavigation.get(key);
  if (!value) {
    if (searchNavigation.size >= 30)
      searchNavigation.delete(searchNavigation.keys().next().value!);
    value = { scroll: 0, cursors: [""] };
    searchNavigation.set(key, value);
  }
  return value;
}

const kinds: Record<string, { label: string; icon: typeof Search }> = {
  document: { label: "문서", icon: FileText },
  block: { label: "블록", icon: Layers },
  code: { label: "코드", icon: Code2 },
  task: { label: "할 일", icon: CheckSquare },
  file: { label: "파일", icon: Paperclip },
  comment: { label: "댓글", icon: MessageSquare },
  tag: { label: "태그", icon: Hash },
  database: { label: "데이터베이스", icon: Database },
  row: { label: "데이터 행", icon: Database },
  user: { label: "사용자", icon: Users },
  ai_conversation: { label: "내 AI 대화", icon: MessageSquare },
};
type Result = {
  kind: string;
  id: string;
  title: string;
  snippet: string;
  url: string;
  updated_at: string;
  document_id: string;
  score: number;
  metadata: Record<string, any>;
};
type SearchResponse = {
  results: Result[];
  has_more: boolean;
  next_offset: number;
  available_types: string[];
  ranking: string;
  history_status?: string;
  next_cursor: string;
  outcome: string;
  interpretation?: {
    normalized: string;
    matched_terms: string[];
    parts: string[][];
    warnings: string[];
    dictionary_revision: number;
  };
};
function Excerpt({ text, query }: { text: string; query: string }) {
  if (!query) return <>{text}</>;
  const index = text.toLocaleLowerCase().indexOf(query.toLocaleLowerCase());
  return index < 0 ? (
    <>{text}</>
  ) : (
    <>
      {text.slice(0, index)}
      <mark>{text.slice(index, index + query.length)}</mark>
      {text.slice(index + query.length)}
    </>
  );
}
export default function SearchPage() {
  const { user, workspace, documents } = useApp();
  const [params, setParams] = useSearchParams();
  const [selected, setSelected] = useState<{ id: string; version: number }[]>(
      [],
    ),
    [summary, setSummary] = useState(false);
  const [query, setQuery] = useState(params.get("q") || "");
  const [data, setData] = useState<SearchResponse | null>(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true),
    [refresh, setRefresh] = useState(0),
    [filters, setFilters] = useState(false);
  const [failure, setFailure] = useState(""),
    [loadedScope, setLoadedScope] = useState(""),
    [wide, setWide] = useState(
      () => window.matchMedia("(min-width: 1100px)").matches,
    );
  const [spaces, setSpaces] = useState<Record<string, any>[]>([]),
    [members, setMembers] = useState<Record<string, any>[]>([]),
    [index, setIndex] = useState<Record<string, any> | null>(null);
  const generation = useRef(0),
    input = useRef<HTMLInputElement>(null),
    openingPreview = useRef(false),
    leavingSearch = useRef(false);
  const term = params.get("q") || "",
    kind = params.get("type") || "",
    page = Math.max(0, Number(params.get("page")) || 0);
  const requestParams = new URLSearchParams();
  for (const key of filterKeys)
    if (params.get(key)) requestParams.set(key, params.get(key)!);
  requestParams.set("group", "document");
  const queryString = requestParams.toString();
  const scope = `${user.id}:${workspace?.id}:${queryString}`;
  const scopeRef = useRef(scope);
  scopeRef.current = scope;
  const baseParams = new URLSearchParams(requestParams);
  baseParams.delete("cursor");
  const navigationKey = `${user.id}:${workspace?.id}:${baseParams}`;
  const previewID = /^[0-9a-f-]{36}$/i.test(params.get("preview") || "")
    ? params.get("preview")
    : null;
  const visibleData = loadedScope === scope ? data : null;
  const rememberPosition = () => {
    navigationState(scope).scroll = window.scrollY;
    leavingSearch.current = true;
  };
  const selectPreview = (
    id: string | null,
    version?: number,
    line?: number,
  ) => {
    if (id) navigationState(scope).scroll = window.scrollY;
    setParams(
      (old) => {
        const next = new URLSearchParams(old);
        for (const key of ["preview", "preview_version", "preview_line"])
          next.delete(key);
        if (id) {
          next.set("preview", id);
          if (version) next.set("preview_version", String(version));
          if (line) next.set("preview_line", String(line));
        }
        return next;
      },
      { replace: true },
    );
  };
  useEffect(() => {
    const media = window.matchMedia("(min-width: 1100px)");
    const change = () => setWide(media.matches);
    media.addEventListener("change", change);
    return () => media.removeEventListener("change", change);
  }, []);
  useEffect(() => {
    if (!visibleData) return;
    leavingSearch.current = false;
    let restoring = true;
    const target = navigationState(scope).scroll;
    const save = () => {
      if (restoring || leavingSearch.current) return;
      navigationState(scope).scroll = window.scrollY;
    };
    window.addEventListener("scroll", save, { passive: true });
    const frame = requestAnimationFrame(() => {
      window.scrollTo({ top: target, behavior: "instant" });
    });
    // Lazy preview/content layout may settle after the first paint. Do not
    // overwrite the saved position with a temporary loading-page clamp.
    const timer = window.setTimeout(() => {
      if (!leavingSearch.current && scopeRef.current === scope)
        window.scrollTo({ top: target, behavior: "instant" });
      restoring = false;
    }, 250);
    return () => {
      cancelAnimationFrame(frame);
      clearTimeout(timer);
      window.removeEventListener("scroll", save);
    };
  }, [visibleData, scope]);
  useEffect(() => setQuery(term), [term]);
  useEffect(() => {
    setSelected([]);
    setSummary(false);
  }, [user.id, workspace?.id, queryString, refresh]);
  useEffect(() => {
    input.current?.focus({ preventScroll: true });
  }, []);
  useEffect(() => {
    setSpaces([]);
    setMembers([]);
    setIndex(null);
    if (!workspace) return;
    let active = true;
    void api<Record<string, any>[]>(`/spaces?workspace_id=${workspace.id}`)
      .then((v) => active && setSpaces(v))
      .catch(() => {});
    void api<Record<string, any>[]>(`/workspaces/${workspace.id}/members`)
      .then((v) => active && setMembers(v))
      .catch(() => {});
    void api<Record<string, any>>(
      `/search/index-status?workspace_id=${workspace.id}`,
    )
      .then((v) => active && setIndex(v))
      .catch(() => {});
    return () => {
      active = false;
    };
  }, [user.id, workspace?.id, refresh]);
  useEffect(() => {
    const run = ++generation.current;
    setData(null);
    setError("");
    setFailure("");
    setLoadedScope("");
    setLoading(true);
    if (!workspace) {
      setLoading(false);
      return;
    }
    const request = new URLSearchParams(queryString);
    request.set("workspace_id", workspace.id);
    request.set("limit", "40");
    const controller = new AbortController();
    void fetch(`/api/v1/search?${request}`, {
      credentials: "same-origin",
      headers: { "X-Madi-Request": "1" },
      signal: controller.signal,
    })
      .then(async (response) => {
        const result = await response.json();
        if (!response.ok) {
          if (run === generation.current)
            setFailure(
              result.outcome ||
                (response.status === 403
                  ? "scope_unavailable"
                  : response.status === 400
                    ? "invalid_query"
                    : "unavailable"),
            );
          throw new Error(result.error || "검색을 완료하지 못했습니다.");
        }
        return result as SearchResponse;
      })
      .then((v) => {
        if (run === generation.current && scopeRef.current === scope) {
          setData(v);
          setLoadedScope(scope);
          navigationState(navigationKey).cursors[page] =
            requestParams.get("cursor") || "";
        }
      })
      .catch((e) => {
        if (run === generation.current) setError(e.message);
      })
      .finally(() => {
        if (run === generation.current) setLoading(false);
      });
    return () => {
      generation.current++;
      controller.abort();
    };
  }, [user.id, workspace?.id, queryString, refresh]);
  const change = (key: string, value: string) =>
    setParams((old) => {
      const next = new URLSearchParams(old);
      if (value) next.set(key, value);
      else next.delete(key);
      for (const name of [
        "offset",
        "cursor",
        "page",
        "preview",
        "preview_version",
        "preview_line",
      ])
        next.delete(name);
      return next;
    });
  const firstPage = () => change("cursor", "");
  const turnPage = (forward: boolean) => {
    const nextPage = forward ? page + 1 : Math.max(0, page - 1);
    const cursor = forward
      ? visibleData?.next_cursor
      : navigationState(navigationKey).cursors[nextPage];
    if (!cursor && nextPage > 0) {
      firstPage();
      return;
    }
    setParams((old) => {
      const next = new URLSearchParams(old);
      for (const key of [
        "cursor",
        "offset",
        "preview",
        "preview_version",
        "preview_line",
        "page",
      ])
        next.delete(key);
      if (cursor) next.set("cursor", cursor);
      if (nextPage) next.set("page", String(nextPage));
      return next;
    });
  };
  const activeFilters = [
    "space_id",
    "author_id",
    "tag",
    "status",
    "from",
    "to",
    "has_attachment",
  ].filter((k) => params.get(k));
  const tags = [...new Set(documents.flatMap((d) => d.tags || []))].sort();
  const statusNames: Record<string, string> = {
    draft: "초안",
    review: "검토 중",
    published: "게시됨",
    rejected: "반려됨",
    stale: "검토 필요",
    archived: "보관됨",
  };
  const choices = (
    key: string,
    items: { id: string; name: string }[],
    all: string,
  ) => (
    <select
      aria-label={all}
      value={params.get(key) || ""}
      onChange={(e) => change(key, e.target.value)}
    >
      <option value="">{all}</option>
      {params.get(key) && !items.some((i) => i.id === params.get(key)) && (
        <option value={params.get(key)!}>
          현재 필터 · 접근 범위 확인 필요
        </option>
      )}
      {items.map((i) => (
        <option key={i.id} value={i.id}>
          {i.name}
        </option>
      ))}
    </select>
  );
  return (
    <section className="universal-search">
      <PageHeading
        eyebrow="지식 탐색"
        title="통합 검색"
        description="문서와 블록, 할 일부터 댓글·파일·데이터베이스까지. 지금 접근할 수 있는 지식만 찾습니다."
        actions={
          <>
            <Link className="button" to="/app/knowledge-time">
              날짜 기준 검색
            </Link>
            {workspace && ["owner", "admin"].includes(workspace.role) && (
              <Link className="button" to="/app/search/operations">
                <Settings2 size={17} />
                검색 운영
              </Link>
            )}
            <NaturalSearch
              initialQuery={query}
              onApply={(proposed) => {
                const next = new URLSearchParams(proposed);
                for (const name of ["space_id", "author_id"]) {
                  const v = new URLSearchParams(window.location.search).get(
                    name,
                  );
                  if (v) next.set(name, v);
                }
                setParams(next);
              }}
            />
            <Button onClick={() => setRefresh((v) => v + 1)} disabled={loading}>
              <RefreshCw size={17} />
              결과 새로고침
            </Button>
          </>
        }
      />
      <form
        className="universal-search-box"
        onSubmit={(e) => {
          e.preventDefault();
          change("q", query.trim());
        }}
      >
        <Search size={24} aria-hidden="true" />
        <input
          ref={input}
          aria-label="통합 검색어"
          value={query}
          maxLength={500}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="어떤 지식을 찾고 계신가요?"
        />
        <Button type="submit" variant="primary">
          검색
        </Button>
      </form>
      <div className="search-kind-tabs" role="group" aria-label="검색 종류">
        <button
          className={!kind ? "active" : ""}
          aria-pressed={!kind}
          onClick={() => change("type", "")}
        >
          전체
        </button>
        {Object.entries(kinds).map(([key, { label, icon: Icon }]) => (
          <button
            key={key}
            className={kind === key ? "active" : ""}
            aria-pressed={kind === key}
            onClick={() => change("type", key)}
          >
            <Icon size={16} />
            {label}
          </button>
        ))}
      </div>
      <div className="search-refinements">
        <Button
          onClick={() => setFilters((v) => !v)}
          aria-expanded={filters}
          aria-controls="search-filters"
        >
          <SlidersHorizontal size={17} />
          상세 필터
          {activeFilters.length > 0 && <Badge>{activeFilters.length}</Badge>}
        </Button>
        <select
          aria-label="검색 정렬"
          value={params.get("sort") || "relevance"}
          onChange={(e) => change("sort", e.target.value)}
        >
          <option value="relevance">관련성 높은 순</option>
          <option value="newest">최근 수정 순</option>
          <option value="oldest">오래된 순</option>
          <option value="title">제목 순</option>
        </select>
        {activeFilters.length > 0 && (
          <Button
            onClick={() =>
              setParams((old) => {
                const next = new URLSearchParams(old);
                for (const key of activeFilters) next.delete(key);
                for (const key of [
                  "offset",
                  "cursor",
                  "page",
                  "preview",
                  "preview_version",
                  "preview_line",
                ])
                  next.delete(key);
                return next;
              })
            }
          >
            필터 초기화
          </Button>
        )}
      </div>
      <div className="search-scope" aria-label="현재 검색 범위">
        <ShieldCheck size={17} />
        <span>
          {workspace?.name || "워크스페이스"} · 현재 내 접근 권한 · 문서별 묶음
        </span>
      </div>
      {activeFilters.length > 0 && (
        <div
          className="search-active-chips"
          role="group"
          aria-label="적용 중인 검색 조건"
        >
          {activeFilters.map((key) => {
            const value = params.get(key)!;
            const label =
              key === "space_id"
                ? `공간: ${spaces.find((v) => v.id === value)?.name || "선택한 공간"}`
                : key === "author_id"
                  ? `소유자: ${members.find((v) => (v.user_id || v.id) === value)?.name || "선택한 사용자"}`
                  : key === "tag"
                    ? `태그: #${value}`
                    : key === "status"
                      ? `상태: ${statusNames[value] || value}`
                      : key === "from"
                        ? `시작일: ${value}`
                        : key === "to"
                          ? `종료일: ${value}`
                          : "첨부파일 있음";
            return (
              <button
                key={key}
                onClick={() => change(key, "")}
                aria-label={`${label} 조건 해제`}
              >
                {label}
                <X size={14} />
              </button>
            );
          })}
        </div>
      )}
      {filters && (
        <div id="search-filters" className="panel search-filter-grid">
          <Field label="공간">
            {choices(
              "space_id",
              spaces.map((v) => ({ id: v.id, name: v.name })),
              "모든 공간",
            )}
          </Field>
          <Field label="문서 소유자">
            {choices(
              "author_id",
              members.map((v) => ({
                id: v.user_id || v.id,
                name: v.name || v.email,
              })),
              "모든 소유자",
            )}
          </Field>
          <Field label="태그">
            {choices(
              "tag",
              tags.map((v) => ({ id: v, name: `#${v}` })),
              "모든 태그",
            )}
          </Field>
          <Field label="문서 상태">
            {choices(
              "status",
              Object.entries(statusNames).map(([id, name]) => ({ id, name })),
              "모든 상태",
            )}
          </Field>
          <Field label="수정일 시작 (UTC)">
            <input
              type="date"
              aria-label="수정일 시작"
              value={params.get("from") || ""}
              onChange={(e) => change("from", e.target.value)}
            />
          </Field>
          <Field label="수정일 종료 (UTC)">
            <input
              type="date"
              aria-label="수정일 종료"
              value={params.get("to") || ""}
              onChange={(e) => change("to", e.target.value)}
            />
          </Field>
          <label className="search-attachment-filter">
            <input
              type="checkbox"
              checked={params.get("has_attachment") === "1"}
              onChange={(e) =>
                change("has_attachment", e.target.checked ? "1" : "")
              }
            />
            첨부파일이 있는 문서만
          </label>
          <p className="search-filter-note">
            소유자·태그·상태·첨부 필터는 문서 기반 결과에 적용됩니다.
            데이터베이스 날짜는 생성일 기준이며, 조건이 적용되지 않는 종류는
            결과에서 제외됩니다.
          </p>
        </div>
      )}
      <ErrorBox error={error} />
      {error && (
        <div className="search-recovery" role="group" aria-label="검색 복구">
          <p>
            {failure === "search_changed"
              ? "사전·조건이 변경되었거나 15분이 지나 페이지가 만료되었습니다."
              : failure === "timeout"
                ? "검색 시간이 길어졌습니다. 조건을 좁히거나 다시 시도하세요."
                : "현재 범위에서 검색을 완료하지 못했습니다. 조건과 연결 상태를 확인하세요."}
          </p>
          <Button onClick={firstPage}>첫 페이지에서 다시 검색</Button>
          <Button
            onClick={() => {
              setRefresh((v) => v + 1);
            }}
          >
            다시 시도
          </Button>
        </div>
      )}
      {visibleData?.interpretation && term && (
        <details className="search-interpretation">
          <summary>
            검색어 해석{" "}
            {visibleData.interpretation.matched_terms.length
              ? `· 조직 용어 ${visibleData.interpretation.matched_terms.join(", ")}`
              : "· 문자·공백 정규화"}
          </summary>
          <p>기준: {visibleData.interpretation.normalized}</p>
          <p>
            조건별 교집합:{" "}
            {visibleData.interpretation.parts
              .map((part) => `(${part.join(" / ")})`)
              .join(" + ") || "기존 검색 문법 유지"}
          </p>
          <p>
            형태소 분석이 아닌 NFKC·대소문자·공백 정규화와 관리자가 등록한
            동의어를 적용합니다. 색인 갱신 전에는 새 버전의 정규화 결과가
            준비되지 않을 수 있습니다.
          </p>
          {visibleData.interpretation.warnings.map((warning) => (
            <p key={warning}>{warning}</p>
          ))}
        </details>
      )}
      {visibleData && (
        <p className="muted">
          {visibleData.history_status === "saved"
            ? "내 개인 검색 기록에 저장했습니다."
            : visibleData.history_status === "protection_blocked"
              ? "정보보호 정책으로 검색어 기록을 저장하지 않았습니다."
              : visibleData.history_status === "unavailable"
                ? "검색 결과는 조회했지만 개인 기록 저장은 일시적으로 실패했습니다."
                : ""}{" "}
          <Link to="/app/search-history">내 검색 기록 · 저장 동의 설정</Link>
        </p>
      )}
      {!!selected.length && (
        <div className="panel padded">
          <span>문서 {selected.length}/10개 선택 · 관련 원문 조각만 참조</span>
          <div className="modal-actions">
            <Button onClick={() => setSelected([])}>선택 해제</Button>
            <Button variant="primary" onClick={() => setSummary(true)}>
              선택 문서 AI 요약
            </Button>
          </div>
        </div>
      )}
      {summary && (
        <Suspense fallback={<Loading />}>
          <SearchAI
            open
            onClose={() => setSummary(false)}
            selectedDocuments={selected}
          />
        </Suspense>
      )}
      <div className="search-results-header">
        <span aria-live="polite">
          {loading
            ? "지식을 찾고 있습니다…"
            : visibleData?.results.length
              ? `${page + 1}페이지 · ${visibleData.results.length}개 묶음`
              : "검색 결과 0개"}
        </span>
        <span>
          <ShieldCheck size={15} />
          현재 권한 적용
        </span>
      </div>
      <div
        className={`search-reading-layout ${previewID && wide ? "has-preview" : ""}`}
      >
        <div className="search-reading-results">
          {loading ? (
            <Loading />
          ) : visibleData?.results.length ? (
            <div className="universal-results">
              {visibleData.results.map((result) => {
                const entry = kinds[result.kind] || kinds.document,
                  Icon = entry.icon;
                return (
                  <article
                    key={`${result.kind}:${result.id}`}
                    className={`search-result ${previewID === result.document_id ? "is-selected" : ""}`}
                  >
                    <div className="search-result-icon">
                      <Icon size={21} />
                      {result.kind === "document" &&
                        result.metadata.version && (
                          <input
                            type="checkbox"
                            aria-label={`${result.title} 요약에 선택`}
                            checked={selected.some((v) => v.id === result.id)}
                            disabled={
                              selected.length >= 10 &&
                              !selected.some((v) => v.id === result.id)
                            }
                            onChange={(e) =>
                              setSelected((old) =>
                                e.target.checked
                                  ? [
                                      ...old,
                                      {
                                        id: result.id,
                                        version: result.metadata.version,
                                      },
                                    ]
                                  : old.filter((v) => v.id !== result.id),
                              )
                            }
                          />
                        )}
                    </div>
                    <div className="search-result-body">
                      <div className="search-result-meta">
                        <Badge>{entry.label}</Badge>
                        {result.metadata.matched_count > 1 && (
                          <Badge>{result.metadata.matched_count}개 일치</Badge>
                        )}
                        <time dateTime={result.updated_at}>
                          {datetime(result.updated_at)}
                        </time>
                        {result.metadata.start_line && (
                          <span>{result.metadata.start_line}행</span>
                        )}
                      </div>
                      <Link
                        to={
                          result.kind === "tag"
                            ? `/app/search?type=document&tag=${encodeURIComponent(result.title)}`
                            : result.url
                        }
                        className="search-result-title"
                        onClick={rememberPosition}
                      >
                        <Excerpt text={result.title} query={term} />
                        <ArrowRight size={17} />
                      </Link>
                      <p
                        className={
                          result.kind === "code" ? "search-code-excerpt" : ""
                        }
                      >
                        <Excerpt text={result.snippet} query={term} />
                      </p>
                      {result.document_id && (
                        <Button
                          className="search-preview-button"
                          onClick={() =>
                            selectPreview(
                              result.document_id,
                              result.metadata.version,
                              result.metadata.start_line,
                            )
                          }
                        >
                          <Eye size={16} />
                          문서 미리보기
                        </Button>
                      )}
                      {result.metadata.matches?.filter(
                        (match: { kind: string; snippet: string }) =>
                          match.kind !== "document" && match.snippet,
                      ).length > 0 && (
                        <ul
                          className="search-group-matches"
                          aria-label="이 문서의 일치 위치"
                        >
                          {result.metadata.matches
                            .filter(
                              (match: { kind: string; snippet: string }) =>
                                match.kind !== "document" && match.snippet,
                            )
                            .map(
                              (
                                match: {
                                  kind: string;
                                  snippet: string;
                                  metadata: Record<string, any>;
                                },
                                n: number,
                              ) => (
                                <li key={n}>
                                  <button
                                    onClick={() =>
                                      selectPreview(
                                        result.document_id,
                                        result.metadata.version ||
                                          match.metadata.version,
                                        match.metadata.start_line,
                                      )
                                    }
                                  >
                                    <span>
                                      {kinds[match.kind]?.label || "문서 조각"}
                                      {match.metadata.start_line
                                        ? ` · ${match.metadata.start_line}행`
                                        : ""}
                                    </span>
                                    <span>
                                      <Excerpt
                                        text={match.snippet}
                                        query={term}
                                      />
                                    </span>
                                  </button>
                                </li>
                              ),
                            )}
                        </ul>
                      )}
                      {result.metadata.tags?.length > 0 && (
                        <div className="search-result-tags">
                          {result.metadata.tags.map((tag: string) => (
                            <button
                              key={tag}
                              onClick={() => change("tag", tag)}
                            >
                              #{tag}
                            </button>
                          ))}
                        </div>
                      )}
                    </div>
                  </article>
                );
              })}
            </div>
          ) : (
            !error && (
              <div className="search-empty-recovery">
                <Empty
                  title="찾는 지식이 아직 보이지 않아요"
                  text="현재 조건과 접근 범위에서 일치하는 결과가 없습니다. 다른 범위에 문서가 있는지는 표시하지 않습니다."
                />
                {visibleData?.outcome === "index_pending" && (
                  <p className="muted">
                    현재 열람 가능한 문서 중 정규화 색인 갱신을 기다리는 문서가
                    있습니다. 이 검색어에 일치하는 문서가 있다는 의미는
                    아닙니다.
                  </p>
                )}
                <div className="search-recovery">
                  <Button
                    onClick={() => {
                      input.current?.focus();
                      input.current?.select();
                    }}
                  >
                    검색어 다시 작성
                  </Button>
                  <Button onClick={() => setParams(term ? { q: term } : {})}>
                    필터 모두 해제
                  </Button>
                  <Button onClick={() => setRefresh((v) => v + 1)}>
                    색인 상태 새로 확인
                  </Button>
                </div>
              </div>
            )
          )}
          {visibleData && (page > 0 || visibleData.has_more) && (
            <div className="search-pagination">
              <Button disabled={page === 0} onClick={() => turnPage(false)}>
                이전 결과
              </Button>
              <Button
                disabled={!visibleData.has_more}
                onClick={() => turnPage(true)}
              >
                다음 결과
              </Button>
            </div>
          )}
        </div>
        {previewID && (
          <Suspense fallback={<Loading />}>
            <DocumentPreview
              documentId={previewID}
              expectedVersion={
                Number(params.get("preview_version")) || undefined
              }
              sourceLine={Number(params.get("preview_line")) || undefined}
              onClose={() => {
                // Shared preview closes its local panel when opening a document.
                // Keep this history entry's URL selection for browser Back.
                if (openingPreview.current) {
                  openingPreview.current = false;
                  return;
                }
                selectPreview(null);
              }}
              onOpen={() => {
                leavingSearch.current = true;
                openingPreview.current = true;
              }}
              variant={wide ? "panel" : "modal"}
            />
          </Suspense>
        )}
      </div>
      <footer className="search-index-note">
        {index && (
          <span>
            내 접근 범위의 로컬 색인 {index.indexed}/{index.documents}개 문서
            {index.pending > 0 ? ` · ${index.pending}개 갱신 대기` : ""}
            {typeof index.normalized === "number"
              ? ` · 공백 정규화 ${index.normalized}개 준비`
              : ""}
          </span>
        )}
        <p>
          문서 내용은 외부로 전송하지 않습니다. 파일은 이름·형식으로 검색하며,
          변경 전 버전의 블록·코드는 표시하지 않습니다.
        </p>
        {visibleData && (
          <details>
            <summary>검색 순위 안내</summary>
            <p>{visibleData.ranking}</p>
            <p>
              페이지마다 현재 권한과 문서 버전을 다시 확인합니다. 실시간
              편집으로 결과 순서가 바뀔 수 있으며, 페이지는 고정된 데이터
              스냅샷이 아닙니다. 검색 페이지는 15분 후 또는 사전 변경 시 첫
              페이지부터 다시 조회합니다.
            </p>
          </details>
        )}
      </footer>
    </section>
  );
}
