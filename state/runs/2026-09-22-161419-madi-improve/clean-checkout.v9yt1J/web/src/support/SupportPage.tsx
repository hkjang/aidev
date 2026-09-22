import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  FileText,
  LockKeyhole,
  Search,
  ShieldCheck,
  Square,
  Settings2,
} from "lucide-react";
import { api, ApiError, datetime, type User } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import "./style.css";

type Session = {
  id: string;
  target_name: string;
  reason: string;
  expires_at: string;
  status: string;
};
type Meta = {
  enabled: boolean;
  can_start: boolean;
  max_minutes: number;
  notice: string;
};
type Workspace = { id: string; name: string; target_role: string };
type Item = {
  id: string;
  title?: string;
  name?: string;
  version?: number;
  visibility?: string;
};
type Snapshot = {
  id: string;
  properties: { id: string; name: string }[];
  rows: {
    id: string;
    values: Record<string, unknown>;
    errors: Record<string, string>;
  }[];
  dependency_database_ids: string[];
  formula_workspace_id: string;
};
type Graph = {
  nodes: Item[];
  edges: { source: string; target: string }[];
  truncated?: boolean;
};
type Pins = { documents: string[]; databases: string[]; formula: string };
const emptyPins = (): Pins => ({ documents: [], databases: [], formula: "" });
const roles: Record<string, string> = {
  owner: "소유자",
  admin: "관리자",
  editor: "편집자",
  viewer: "조회자",
  commenter: "댓글 작성자",
  guest: "게스트",
};
const visibility: Record<string, string> = {
  workspace: "워크스페이스",
  selected: "지정 사용자",
  private: "개인",
};
async function read<T>(path: string, signal: AbortSignal): Promise<T> {
  const response = await fetch("/api/v1" + path, {
    credentials: "same-origin",
    signal,
    headers: { "X-Madi-Request": "1" },
  });
  const value = await response.json();
  if (!response.ok)
    throw new ApiError(
      value.error || "진단 정보를 확인할 수 없습니다",
      response.status,
    );
  return value;
}

export function SupportPage() {
  const { user, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const sessionID = params.get("session") || "";
  const [meta, setMeta] = useState<Meta>();
  const [targets, setTargets] = useState<Item[]>([]);
  const [target, setTarget] = useState("");
  const [reason, setReason] = useState("");
  const [minutes, setMinutes] = useState(10);
  const [consent, setConsent] = useState(false);
  const [session, setSession] = useState<Session>();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [workspace, setWorkspace] = useState("");
  const [view, setView] = useState("documents");
  const [query, setQuery] = useState("");
  const [items, setItems] = useState<Item[]>([]);
  const [document, setDocument] = useState<Item & { markdown: string }>();
  const [database, setDatabase] = useState<Snapshot>();
  const [graph, setGraph] = useState<Graph>();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<unknown>();
  const [stopped, setStopped] = useState(false);
  const [now, setNow] = useState(Date.now());
  const [configOpen, setConfigOpen] = useState(false);
  const [config, setConfig] = useState({
    support_enabled: false,
    support_operator_ids: [] as string[],
    support_max_minutes: 30,
  });
  const [admins, setAdmins] = useState<User[]>([]);
  const generation = useRef(0),
    pins = useRef<Pins>(emptyPins()),
    alive = useRef(true);
  const resourceGeneration = useRef(0);
  const pending = useRef(new Set<AbortController>());
  const currentUser = useRef(user.id);
  currentUser.current = user.id;
  const clearContent = () => {
    resourceGeneration.current++;
    setBusy(false);
    pins.current = emptyPins();
    setItems([]);
    setDocument(undefined);
    setDatabase(undefined);
    setGraph(undefined);
  };
  const invalidate = (message: unknown) => {
    generation.current++;
    pending.current.forEach((request) => request.abort());
    pending.current.clear();
    clearContent();
    setSession(undefined);
    setWorkspaces([]);
    setStopped(true);
    setBusy(false);
    setError(message);
  };
  async function loadMeta() {
    const actor = user.id;
    const value = await api<Meta>("/support/meta");
    if (!alive.current || actor !== currentUser.current) return;
    setMeta(value);
    if (value.max_minutes > 0)
      setMinutes((old) => Math.min(old, value.max_minutes));
    const list = value.can_start ? await api<Item[]>("/support/targets") : [];
    if (alive.current && actor === currentUser.current) {
      setTargets(list);
      setTarget((old) => (list.some((x) => x.id === old) ? old : ""));
    }
  }
  useEffect(() => {
    alive.current = true;
    setMeta(undefined);
    setConsent(false);
    setReason("");
    setTargets([]);
    setConfigOpen(false);
    loadMeta().catch(setError);
    return () => {
      alive.current = false;
      generation.current++;
      pending.current.forEach((request) => request.abort());
      pending.current.clear();
    };
  }, [user.id]);

  useEffect(() => {
    const epoch = ++generation.current;
    let disposed = false,
      checking = false;
    setStopped(false);
    setError(undefined);
    setSession(undefined);
    setWorkspaces([]);
    setWorkspace("");
    clearContent();
    if (!sessionID) return;
    async function check() {
      if (disposed || checking || epoch !== generation.current) return;
      if (window.document.visibilityState === "hidden") {
        // Hidden tabs retain no displayed source. Returning requires revalidation.
        clearContent();
        return;
      }
      checking = true;
      const controller = new AbortController();
      pending.current.add(controller);
      const deadline = setTimeout(() => controller.abort(), 1000);
      const q = new URLSearchParams();
      if (pins.current.documents.length)
        q.set("document_ids", [...new Set(pins.current.documents)].join(","));
      if (pins.current.databases.length)
        q.set("database_ids", [...new Set(pins.current.databases)].join(","));
      if (pins.current.formula)
        q.set("formula_workspace_id", pins.current.formula);
      try {
        const result = await read<{
          session: Session;
          workspaces: Workspace[];
        }>(`/support/sessions/${sessionID}?${q}`, controller.signal);
        if (disposed || epoch !== generation.current) return;
        if (Date.parse(result.session.expires_at) <= Date.now())
          throw new Error("지원 시간이 만료되었습니다");
        setSession(result.session);
        setWorkspaces(result.workspaces);
        setWorkspace((old) =>
          result.workspaces.some((x) => x.id === old)
            ? old
            : result.workspaces[0]?.id || "",
        );
        setNow(Date.now());
      } catch (e) {
        if (!disposed && epoch === generation.current)
          invalidate(
            e instanceof DOMException
              ? "연결을 확인할 수 없어 진단 내용을 지웠습니다. 다시 확인해 주세요."
              : e,
          );
      } finally {
        clearTimeout(deadline);
        pending.current.delete(controller);
        checking = false;
      }
    }
    void check();
    const timer = setInterval(check, 500);
    window.document.addEventListener("visibilitychange", check);
    return () => {
      disposed = true;
      clearInterval(timer);
      window.document.removeEventListener("visibilitychange", check);
      pending.current.forEach((request) => request.abort());
      pending.current.clear();
    };
  }, [sessionID, user.id]);

  async function start() {
    setBusy(true);
    setError(undefined);
    const actor = user.id;
    try {
      const result = await api<{ session: Session; reason_masked: boolean }>(
        "/support/sessions",
        "POST",
        { target_id: target, reason, duration_minutes: minutes },
      );
      if (!alive.current || actor !== currentUser.current) return;
      if (result.reason_masked)
        notify("지원 사유의 민감정보를 마스킹하여 기록했습니다");
      setParams({ session: result.session.id });
      setConsent(false);
      setReason("");
    } catch (e) {
      if (alive.current && actor === currentUser.current) setError(e);
    } finally {
      if (alive.current && actor === currentUser.current) setBusy(false);
    }
  }
  async function end() {
    const id = sessionID;
    invalidate(undefined);
    try {
      await api(`/support/sessions/${id}/end`, "POST");
      setParams({});
      notify("지원 진단을 종료했습니다");
      await loadMeta();
    } catch (e) {
      setError(e);
    }
  }
  async function load(
    path: string,
    kind: "documents" | "databases" | "graph" | "document" | "database",
  ) {
    const expected = generation.current;
    const controller = new AbortController();
    pending.current.add(controller);
    setError(undefined);
    clearContent();
    setBusy(true);
    const resource = resourceGeneration.current;
    try {
      const result = await read<any>(
        `/support/sessions/${sessionID}${path}`,
        controller.signal,
      );
      if (
        !alive.current ||
        generation.current !== expected ||
        resourceGeneration.current !== resource ||
        controller.signal.aborted
      )
        return;
      if (kind === "documents") {
        pins.current.documents = result.map((x: Item) => x.id);
        setItems(result);
      }
      if (kind === "databases") {
        pins.current.databases = result.map((x: Item) => x.id);
        setItems(result);
      }
      if (kind === "document") {
        pins.current.documents = [result.id];
        setDocument(result);
      }
      if (kind === "database") {
        pins.current.databases = result.dependency_database_ids;
        pins.current.formula = result.formula_workspace_id;
        setDatabase(result);
      }
      if (kind === "graph") {
        pins.current.documents = result.nodes.map((x: Item) => x.id);
        setGraph(result);
      }
    } catch (e) {
      if (
        alive.current &&
        generation.current === expected &&
        resourceGeneration.current === resource &&
        !controller.signal.aborted
      )
        invalidate(e);
    } finally {
      pending.current.delete(controller);
      if (
        alive.current &&
        generation.current === expected &&
        resourceGeneration.current === resource
      )
        setBusy(false);
    }
  }
  async function openConfig() {
    const actor = user.id;
    try {
      const [settings, users] = await Promise.all([
        api("/admin/settings"),
        api<User[]>("/admin/users"),
      ]);
      if (!alive.current || currentUser.current !== actor) return;
      setConfig({
        support_enabled: settings.support_enabled,
        support_operator_ids: settings.support_operator_ids || [],
        support_max_minutes: settings.support_max_minutes,
      });
      setAdmins(
        users.filter(
          (u) => u.role === "admin" && u.kind === "user" && !u.disabled,
        ),
      );
      setConfigOpen(true);
    } catch (e) {
      setError(e);
    }
  }
  async function saveConfig() {
    setBusy(true);
    try {
      await api("/admin/settings", "PUT", config);
      setConfigOpen(false);
      await loadMeta();
      notify("지원 진단 정책을 저장했습니다");
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }
  const remaining = session
    ? Math.max(0, Math.ceil((Date.parse(session.expires_at) - now) / 1000))
    : 0;
  const selectedWorkspace = workspaces.find((w) => w.id === workspace);
  return (
    <div className="page support-page">
      <PageHeading
        eyebrow="보안 · 사용자 지원"
        title="읽기 전용 지원 진단"
        description="계정을 대신 사용하지 않고, 두 사람에게 공통으로 허용된 화면과 접근 상태를 확인합니다."
        actions={
          !sessionID && (
            <Button onClick={openConfig}>
              <Settings2 size={18} />
              지원 정책
            </Button>
          )
        }
      />
      <ErrorBox error={error} />
      {!sessionID && (
        <>
          <div className="support-principle">
            <ShieldCheck size={28} />
            <div>
              <h2>본인의 권한을 넘어가지 않는 지원</h2>
              <p>
                관리자와 대상 사용자의 현재 권한이 모두 허용한 문서·DB만
                표시합니다. 타인의 개인 문서, 개인 AI 대화, 비밀 키는 열람하지
                않습니다.
              </p>
              <p>
                쓰기·AI 호출·외부 전송·첨부 다운로드·블록 실행은 제공하지
                않습니다. 다른 탭은 관리자 본인의 권한을 그대로 유지합니다.
              </p>
            </div>
          </div>
          {!meta ? (
            <Loading />
          ) : !meta.can_start ? (
            <Empty
              title="지원 진단이 허용되지 않았습니다"
              text="기본값은 꺼짐입니다. 지원 정책에서 기능을 켜고, 현재 서비스 관리자를 지원 담당자로 명시해야 시작할 수 있습니다."
              action={<Button onClick={openConfig}>지원 정책 설정</Button>}
            />
          ) : (
            <section className="support-start card">
              <h2>진단 시작</h2>
              <div className="support-fields">
                <Field
                  label="지원 대상"
                  hint="공통 워크스페이스에 속한 활성 사용자만 표시합니다."
                >
                  <select
                    value={target}
                    onChange={(e) => {
                      setTarget(e.target.value);
                      setConsent(false);
                    }}
                  >
                    <option value="">사용자를 선택하세요</option>
                    {targets.map((x) => (
                      <option key={x.id} value={x.id}>
                        {x.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field
                  label="지원 시간 (분)"
                  hint={`최대 ${meta.max_minutes}분 후 자동 만료됩니다.`}
                >
                  <input
                    type="number"
                    min={1}
                    max={meta.max_minutes}
                    value={minutes}
                    onChange={(e) => {
                      setMinutes(Number(e.target.value));
                      setConsent(false);
                    }}
                  />
                </Field>
              </div>
              <Field
                label="지원 사유"
                hint="10~500자. 문서 내용·비밀번호를 적지 마세요. 사유는 감사 기록에 보관되며 민감정보 정책이 적용됩니다."
              >
                <textarea
                  maxLength={500}
                  rows={3}
                  value={reason}
                  onChange={(e) => {
                    setReason(e.target.value);
                    setConsent(false);
                  }}
                  placeholder="예: 문서 메뉴 접근 오류를 함께 확인합니다"
                />
              </Field>
              <label className="support-check">
                <input
                  type="checkbox"
                  checked={consent}
                  onChange={(e) => setConsent(e.target.checked)}
                />
                계정 대체가 아닌 읽기 전용 권한 교집합 진단이며, 시작·자료
                접근·종료가 감사 기록에 남는다는 점을 확인했습니다.
              </label>
              <Button
                variant="primary"
                onClick={start}
                disabled={
                  busy ||
                  !consent ||
                  !target ||
                  [...reason.trim()].length < 10 ||
                  !Number.isInteger(minutes) ||
                  minutes < 1 ||
                  minutes > meta.max_minutes
                }
              >
                <ShieldCheck size={18} />
                진단 시작
              </Button>
            </section>
          )}
        </>
      )}
      {sessionID && (
        <>
          <aside className="support-banner" role="status">
            <LockKeyhole size={24} />
            <div>
              <strong>읽기 전용 · {session?.target_name || "지원 진단"}</strong>
              <span>관리자 ∩ 대상 사용자 권한 · 계정 전환 없음</span>
              {session && (
                <>
                  <small>사유: {session.reason}</small>
                  <small>
                    남은 시간 {Math.floor(remaining / 60)}분 {remaining % 60}초
                    · {datetime(session.expires_at)}까지
                  </small>
                </>
              )}
            </div>
            <Button onClick={end}>
              <Square size={17} />
              진단 종료
            </Button>
          </aside>
          {stopped ? (
            <Empty
              title="안전을 위해 진단 내용을 지웠습니다"
              text="권한·계정·정책·연결 또는 만료 상태가 바뀌었습니다. 진단을 종료한 뒤 현재 허용 범위로 다시 시작하세요."
              action={<Button onClick={end}>진단 종료</Button>}
            />
          ) : !session ? (
            <Loading />
          ) : (
            <>
              <section className="support-toolbar">
                <Field label="공통 워크스페이스">
                  <select
                    value={workspace}
                    onChange={(e) => {
                      clearContent();
                      setWorkspace(e.target.value);
                    }}
                  >
                    <option value="" disabled>
                      워크스페이스 선택
                    </option>
                    {workspaces.map((x) => (
                      <option key={x.id} value={x.id}>
                        {x.name}
                      </option>
                    ))}
                  </select>
                </Field>
                <p className="muted">
                  대상 역할:{" "}
                  {roles[selectedWorkspace?.target_role || ""] || "확인 중"}
                  <br />
                  표시는 양측 권한의 교집합이며 대상의 전체 화면과 다를 수
                  있습니다.
                </p>
              </section>
              <nav className="support-tabs" aria-label="진단 자료">
                {[
                  ["documents", "문서·검색"],
                  ["databases", "데이터베이스"],
                  ["graph", "문서 연결"],
                  ["restrictions", "진단 제한"],
                ].map(([key, label]) => (
                  <Button
                    key={key}
                    variant={view === key ? "primary" : ""}
                    onClick={() => {
                      clearContent();
                      setView(key);
                    }}
                  >
                    {label}
                  </Button>
                ))}
              </nav>
              {view === "restrictions" ? (
                <section className="card support-start">
                  <h2>진단 범위와 제한</h2>
                  <ul>
                    <li>
                      대상의 역할은 참고 정보이며 관리자에게 부여되지 않습니다.
                    </li>
                    <li>
                      공통 워크스페이스와 문서·공간 ACL을 매 요청마다 양쪽 모두
                      확인합니다.
                    </li>
                    <li>
                      표시된 자료의 현재 권한을 0.5초마다 다시 확인합니다. 연결
                      확인 실패·숨긴 탭에서는 내용을 지웁니다.
                    </li>
                    <li>
                      문서는 안전한 원문 텍스트로 표시합니다.
                      플러그인·이미지·첨부·링크는 실행하거나 다운로드하지
                      않습니다.
                    </li>
                    <li>
                      DB 관계·수식·롤업은 참조 대상까지 양측 권한을 검사합니다.
                      버튼 속성은 실행하지 않습니다.
                    </li>
                    <li>
                      문서·DB 목록은 각각 100개, DB 결과는 100행·4MiB까지
                      표시합니다. 그래프는 100문서·500연결로 제한됩니다.
                    </li>
                    <li>
                      다른 탭과 일반 서비스 API는 원래 관리자 본인의 권한이며
                      대상 사용자로 바뀌지 않습니다.
                    </li>
                  </ul>
                </section>
              ) : (
                <>
                  <form
                    className="support-query"
                    onSubmit={(e) => {
                      e.preventDefault();
                      if (workspace)
                        void load(
                          `/${view}?workspace_id=${workspace}&q=${encodeURIComponent(query)}`,
                          view as "documents" | "databases" | "graph",
                        );
                    }}
                  >
                    {view === "documents" && (
                      <Field label="문서 검색">
                        <input
                          value={query}
                          onChange={(e) => setQuery(e.target.value)}
                          placeholder="제목·본문 검색 (선택)"
                          maxLength={200}
                        />
                      </Field>
                    )}
                    <Button type="submit" disabled={busy || !workspace}>
                      <Search size={18} />
                      {view === "documents"
                        ? "문서 확인"
                        : view === "databases"
                          ? "DB 목록 확인"
                          : "연결 확인"}
                    </Button>
                  </form>
                  {busy && <Loading />}
                  {!busy && items.length > 0 && (
                    <section className="support-list">
                      {items.map((item) => (
                        <button
                          key={item.id}
                          onClick={() =>
                            void load(
                              `/${view}/${item.id}`,
                              view === "documents" ? "document" : "database",
                            )
                          }
                        >
                          <FileText size={20} />
                          <span>
                            <strong>{item.title || item.name}</strong>
                            {item.visibility && (
                              <small>
                                {visibility[item.visibility] || item.visibility}{" "}
                                · 버전 {item.version}
                              </small>
                            )}
                          </span>
                          <span>읽기</span>
                        </button>
                      ))}
                    </section>
                  )}
                  {document && (
                    <article className="support-source card">
                      <h2>{document.title}</h2>
                      <p className="muted">
                        안전한 Markdown 원문 · 버전 {document.version} ·
                        미리보기·블록 실행 없음
                      </p>
                      <pre tabIndex={0} aria-label="진단 문서 원문">
                        {document.markdown}
                      </pre>
                    </article>
                  )}
                  {database && (
                    <section className="support-db card">
                      <h2>읽기 전용 DB 조회</h2>
                      <p className="muted">
                        최대 100행 · 참조 DB{" "}
                        {database.dependency_database_ids.length}개 현재 권한
                        검사
                      </p>
                      <div className="table-scroll">
                        <table>
                          <thead>
                            <tr>
                              {database.properties.map((p) => (
                                <th key={p.id}>{p.name}</th>
                              ))}
                            </tr>
                          </thead>
                          <tbody>
                            {database.rows.map((row) => (
                              <tr key={row.id}>
                                {database.properties.map((p) => (
                                  <td key={p.id}>
                                    {row.errors[p.id] ||
                                      (row.values[p.id] == null
                                        ? "—"
                                        : typeof row.values[p.id] === "string"
                                          ? String(row.values[p.id])
                                          : JSON.stringify(row.values[p.id]))}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </section>
                  )}
                  {graph && (
                    <section className="card support-start">
                      <h2>권한 교집합의 문서 연결</h2>
                      <p>
                        {graph.nodes.length}개 문서 · {graph.edges.length}개
                        연결
                      </p>
                      <p className="muted">
                        최대 100문서·500연결, 문서당 앞 32,000자와 제한된 alias
                        32개를 확인합니다.
                        {graph.truncated &&
                          " 범위를 초과하여 일부 원문·별칭·연결이 제외되었습니다."}
                      </p>
                      {graph.edges.length ? (
                        <ul>
                          {graph.edges.map((edge, i) => (
                            <li key={i}>
                              {
                                graph.nodes.find((x) => x.id === edge.source)
                                  ?.title
                              }{" "}
                              →{" "}
                              {
                                graph.nodes.find((x) => x.id === edge.target)
                                  ?.title
                              }
                            </li>
                          ))}
                        </ul>
                      ) : (
                        <p className="muted">
                          현재 허용된 문서 사이의 연결이 없습니다.
                        </p>
                      )}
                    </section>
                  )}
                </>
              )}
            </>
          )}
        </>
      )}
      <Modal
        open={configOpen}
        onOpenChange={setConfigOpen}
        title="지원 진단 정책"
        description="서비스 관리자 중 명시적으로 지정된 담당자만 진단을 시작합니다. 이 설정은 현재 관리자 본인의 권한으로 변경합니다."
      >
        <ErrorBox error={error} />
        <label className="support-check">
          <input
            type="checkbox"
            checked={config.support_enabled}
            onChange={(e) =>
              setConfig({ ...config, support_enabled: e.target.checked })
            }
          />
          읽기 전용 지원 진단 허용 (기본 꺼짐)
        </label>
        <Field label="최대 지원 시간 (분)">
          <input
            type="number"
            min={1}
            max={30}
            value={config.support_max_minutes}
            onChange={(e) =>
              setConfig({
                ...config,
                support_max_minutes: Number(e.target.value),
              })
            }
          />
        </Field>
        <fieldset className="support-operators">
          <legend>명시적 지원 담당자</legend>
          {admins.map((admin) => (
            <label key={admin.id} className="support-check">
              <input
                type="checkbox"
                checked={config.support_operator_ids.includes(admin.id)}
                onChange={(e) =>
                  setConfig({
                    ...config,
                    support_operator_ids: e.target.checked
                      ? [...config.support_operator_ids, admin.id]
                      : config.support_operator_ids.filter(
                          (id) => id !== admin.id,
                        ),
                  })
                }
              />
              {admin.name}
              {admin.id === user.id ? " (본인)" : ""}
            </label>
          ))}
        </fieldset>
        <Button
          variant="primary"
          onClick={saveConfig}
          disabled={
            busy ||
            !Number.isInteger(config.support_max_minutes) ||
            config.support_max_minutes < 1 ||
            config.support_max_minutes > 30
          }
        >
          정책 저장
        </Button>
      </Modal>
    </div>
  );
}
