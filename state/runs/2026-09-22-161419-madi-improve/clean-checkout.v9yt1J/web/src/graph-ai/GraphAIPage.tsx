import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  Check,
  GitBranch,
  History,
  LockKeyhole,
  RefreshCw,
  Search,
  Sparkles,
  Square,
  Trash2,
} from "lucide-react";
import { api, ApiError, bytes, datetime } from "../api";
import { useApp } from "../context";
import CitationViewer, { type CitationSource } from "../CitationViewer";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
} from "../ui";
import "./style.css";

type Kind = "relation" | "duplicate" | "topic" | "entity" | "gap";
const kinds: { id: Kind; name: string; help: string }[] = [
  { id: "relation", name: "문서 관계", help: "서로 참조할 문서를 연결합니다." },
  {
    id: "duplicate",
    name: "유사·중복 후보",
    help: "선택 자료의 유사성을 제안합니다. 병합하지 않습니다.",
  },
  {
    id: "topic",
    name: "문서 주제",
    help: "문서 하나만 분석하여 해당 문서의 운영 속성에 추가합니다.",
  },
  {
    id: "entity",
    name: "엔터티 초안",
    help: "시스템·기술·프로젝트 등의 개인 초안을 만듭니다.",
  },
  {
    id: "gap",
    name: "지식 공백 후보",
    help: "제공 자료에서 보완할 질문을 개인 문서로 만듭니다.",
  },
];
const kindName = (key: string) => kinds.find((k) => k.id === key)?.name || key;
const statusName: Record<string, string> = {
  running: "분석 중",
  ready: "검토 가능",
  failed: "분석 실패",
  cancelled: "취소됨",
  proposed: "확인 대기",
  applied: "적용됨",
  rejected: "제외됨",
};
const visibilityName: Record<string, string> = {
  private: "개인 문서",
  workspace: "워크스페이스",
  selected: "지정 사용자",
};
const classificationName: Record<string, string> = {
  public: "공개",
  internal: "내부",
  confidential: "기밀",
  restricted: "극비",
};
type Snapshot = {
  document_id: string;
  version: number;
  document_hash: string;
  start_byte: number;
  end_byte: number;
  content_hash: string;
};
type Preview = {
  snapshots: Snapshot[];
  sources: CitationSource[];
  documents: {
    id: string;
    title: string;
    version: number;
    visibility: string;
    classification: string;
    truncated: boolean;
    sent_bytes: number;
    total_bytes: number;
  }[];
  provider: {
    base_url: string;
    model: string;
    fingerprint: string;
    configured: boolean;
  };
  enabled: boolean;
  notice: string;
};
type Candidate = {
  kind: Kind;
  source_id?: string;
  target_id?: string;
  relation_type?: string;
  title?: string;
  description?: string;
  entity_type?: string;
  topic?: string;
  reason: string;
  evidence: { document_id: string; citation?: CitationSource }[];
};
type Action = {
  id: string;
  action_hash: string;
  kind: Kind;
  payload: Candidate;
  status: string;
  result: { document_id?: string; access_revoked?: boolean };
};
type Run = {
  id: string;
  workspace_id: string;
  status: string;
  error: string;
  created_at: string;
  kinds: Kind[];
  actions: Action[];
  sources: CitationSource[];
  can_apply: boolean;
};
type HistoryItem = Pick<
  Run,
  "id" | "status" | "error" | "created_at" | "kinds"
>;
const previewSignature = (v: Preview) =>
  JSON.stringify({
    snapshots: v.snapshots,
    provider: v.provider,
    enabled: v.enabled,
  });
async function read<T>(path: string, signal: AbortSignal): Promise<T> {
  const response = await fetch("/api/v1" + path, {
    credentials: "same-origin",
    signal,
    headers: { "X-Madi-Request": "1" },
  });
  const value = await response.json();
  if (!response.ok)
    throw new ApiError(
      value.error || "현재 권한으로 분석 정보를 확인할 수 없습니다",
      response.status,
    );
  return value;
}

export function GraphAIPage() {
  const { user, workspace, documents, notify, reload } = useApp();
  const [params, setParams] = useSearchParams();
  const runID = params.get("run") || "",
    wid = workspace?.id || "";
  const [selected, setSelected] = useState<string[]>([]);
  const [wanted, setWanted] = useState<Kind[]>([
    "relation",
    "duplicate",
    "entity",
    "gap",
  ]);
  const [query, setQuery] = useState("");
  const [preview, setPreview] = useState<Preview>();
  const [consent, setConsent] = useState(false);
  const [run, setRun] = useState<Run>();
  const [history, setHistory] = useState<HistoryItem[]>([]);
  const [historyOpen, setHistoryOpen] = useState(false);
  const [error, setError] = useState<unknown>();
  const [busy, setBusy] = useState(false);
  const [streaming, setStreaming] = useState(false);
  const [received, setReceived] = useState(0);
  const [confirm, setConfirm] = useState<Action>();
  const [confirmed, setConfirmed] = useState(false);
  const [citation, setCitation] = useState<CitationSource | null>(null);
  const [deleteID, setDeleteID] = useState("");
  const identity = user.id + ":" + wid;
  const current = useRef(identity),
    alive = useRef(true),
    epoch = useRef(0);
  const stream = useRef<AbortController | null>(null);
  const selectedKey = selected.slice().sort().join(",");
  current.current = identity;
  const currentRunID = useRef(runID);
  currentRunID.current = runID;
  const clearSensitive = () => {
    setRun(undefined);
    setConfirm(undefined);
    setConfirmed(false);
    setCitation(null);
  };
  const loadHistory = async () => {
    if (!wid) return;
    const owner = identity;
    const result = await api<HistoryItem[]>(`/workspaces/${wid}/graph-ai/runs`);
    if (alive.current && current.current === owner) setHistory(result);
  };
  const loadRun = async (id: string) => {
    const owner = identity;
    const value = await api<Run>(`/graph-ai/runs/${id}`);
    if (
      alive.current &&
      current.current === owner &&
      currentRunID.current === id
    )
      setRun(value);
    return value;
  };
  useEffect(() => {
    alive.current = true;
    epoch.current++;
    setSelected([]);
    setPreview(undefined);
    setConsent(false);
    setHistory([]);
    clearSensitive();
    setBusy(false);
    setStreaming(false);
    setError(undefined);
    void loadHistory().catch(setError);
    return () => {
      alive.current = false;
      epoch.current++;
      stream.current?.abort();
    };
  }, [identity]);
  useEffect(() => {
    setPreview(undefined);
    setConsent(false);
    setCitation(null);
    if (selected.length !== 1)
      setWanted((old) => old.filter((x) => x !== "topic"));
  }, [selectedKey]);
  // Preview polling never calls a provider. A changed version, permission or
  // endpoint invalidates consent; a fresh preview requires a new human click.
  useEffect(() => {
    if (!preview || streaming || runID) return;
    const expected = previewSignature(preview),
      owner = identity;
    let stopped = false,
      checking = false;
    const controller = new AbortController();
    const check = async () => {
      if (stopped || checking) return;
      if (document.visibilityState === "hidden") {
        setPreview(undefined);
        setConsent(false);
        setCitation(null);
        return;
      }
      checking = true;
      const deadline = setTimeout(() => controller.abort(), 1500);
      try {
        const next = await read<Preview>(
          `/workspaces/${wid}/graph-ai/context?document_ids=${selectedKey}`,
          controller.signal,
        );
        if (
          !stopped &&
          current.current === owner &&
          previewSignature(next) !== expected
        ) {
          setPreview(undefined);
          setConsent(false);
          setCitation(null);
          setError(
            "원문·공급자·기능 설정이 변경되었습니다. 전송 범위를 다시 확인하고 동의하세요.",
          );
        }
      } catch (e) {
        if (!stopped && current.current === owner) {
          setPreview(undefined);
          setConsent(false);
          setCitation(null);
          setError(e);
        }
      } finally {
        clearTimeout(deadline);
        checking = false;
      }
    };
    const interval = setInterval(check, 1000);
    document.addEventListener("visibilitychange", check);
    return () => {
      stopped = true;
      controller.abort();
      clearInterval(interval);
      document.removeEventListener("visibilitychange", check);
    };
  }, [preview, streaming, runID, selectedKey, identity]);
  useEffect(() => {
    const n = ++epoch.current;
    clearSensitive();
    setError(undefined);
    if (!runID) return;
    let stopped = false,
      checking = false;
    const requests = new Set<AbortController>();
    const check = async () => {
      if (stopped || checking || epoch.current !== n) return;
      if (document.visibilityState === "hidden") {
        clearSensitive();
        return;
      }
      checking = true;
      const controller = new AbortController();
      requests.add(controller);
      const deadline = setTimeout(() => controller.abort(), 1500);
      try {
        const value = await read<Run>(
          `/graph-ai/runs/${runID}`,
          controller.signal,
        );
        if (stopped || epoch.current !== n) return;
        if (value.workspace_id !== wid)
          throw new Error("현재 워크스페이스의 분석 기록이 아닙니다.");
        setRun(value);
        if (!value.can_apply) {
          setConfirm(undefined);
          setConfirmed(false);
        }
        if (!value.actions.length || value.status !== "ready")
          setCitation(null);
      } catch (e) {
        if (!stopped && epoch.current === n) {
          clearSensitive();
          // The request ID is placed in the URL before the streaming POST has
          // committed its row. A short initial 404 is not a revoked result.
          if (!(stream.current && e instanceof ApiError && e.status === 404))
            setError(e);
        }
      } finally {
        clearTimeout(deadline);
        requests.delete(controller);
        checking = false;
      }
    };
    void check();
    const interval = setInterval(check, 1000);
    document.addEventListener("visibilitychange", check);
    return () => {
      stopped = true;
      requests.forEach((r) => r.abort());
      clearInterval(interval);
      document.removeEventListener("visibilitychange", check);
    };
  }, [runID, identity]);
  async function makePreview() {
    const owner = identity,
      ids = selectedKey;
    setBusy(true);
    setError(undefined);
    setConsent(false);
    setPreview(undefined);
    try {
      const v = await api<Preview>(
        `/workspaces/${wid}/graph-ai/context?document_ids=${ids}`,
      );
      if (alive.current && current.current === owner) setPreview(v);
    } catch (e) {
      if (current.current === owner) setError(e);
    } finally {
      if (current.current === owner) setBusy(false);
    }
  }
  async function start() {
    if (!preview || !consent || !wanted.length || streaming || busy) return;
    const owner = identity,
      id = crypto.randomUUID(),
      controller = new AbortController();
    stream.current = controller;
    setStreaming(true);
    setReceived(0);
    setError(undefined);
    setConsent(false);
    setParams({ run: id });
    currentRunID.current = id;
    const valid = () =>
      alive.current && current.current === owner && currentRunID.current === id;
    try {
      const response = await fetch(
        `/api/v1/workspaces/${wid}/graph-ai/analyze`,
        {
          method: "POST",
          credentials: "same-origin",
          signal: controller.signal,
          headers: {
            "Content-Type": "application/json",
            "X-Madi-Request": "1",
          },
          body: JSON.stringify({
            request_id: id,
            snapshots: preview.snapshots,
            provider_fingerprint: preview.provider.fingerprint,
            kinds: wanted,
            consent: true,
          }),
        },
      );
      if (!response.ok) {
        const v = await response.json();
        throw new ApiError(
          v.error || "분석을 시작하지 못했습니다",
          response.status,
        );
      }
      if (!response.body) throw new Error("스트리밍 응답을 받을 수 없습니다");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "",
        completed = false;
      while (true) {
        const { value, done } = await reader.read();
        buffer += decoder.decode(value, { stream: !done });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const raw = line.slice(5).trim();
          if (!raw || raw === "[DONE]") continue;
          const v = JSON.parse(raw);
          if (!valid()) {
            controller.abort();
            return;
          }
          if (v.error || v.retract) {
            clearSensitive();
            throw new Error(
              v.error || "권한 또는 원문 변경으로 분석 결과를 지웠습니다.",
            );
          }
          if (typeof v.text === "string")
            setReceived((old) => old + v.text.length);
          if (v.proposal?.id === id) {
            completed = true;
            await loadRun(id);
          }
        }
        if (done) break;
      }
      if (!completed)
        throw new Error(
          "분석 연결이 종료되었습니다. 자동 재전송하지 않습니다. 현재 기록을 확인하세요.",
        );
      if (valid()) {
        setPreview(undefined);
        await loadHistory();
      }
    } catch (e) {
      if (valid() && !controller.signal.aborted) setError(e);
    } finally {
      if (stream.current === controller) stream.current = null;
      if (valid()) setStreaming(false);
    }
  }
  async function stop() {
    stream.current?.abort();
    setStreaming(false);
    setConfirm(undefined);
    setConfirmed(false);
    try {
      await api(`/graph-ai/runs/${runID}/cancel`, "POST", {});
      await loadRun(runID);
      await loadHistory();
    } catch (e) {
      setError(e);
    }
  }
  async function decide(action: Action, accepted: boolean) {
    if (busy || (accepted && !confirmed)) return;
    const owner = identity,
      id = runID;
    setBusy(true);
    setError(undefined);
    try {
      await api(`/graph-ai/actions/${action.id}/confirm`, "POST", {
        action_hash: action.action_hash,
        ...(accepted ? { confirm: true } : { reject: true }),
      });
      if (
        !alive.current ||
        current.current !== owner ||
        currentRunID.current !== id
      )
        return;
      setConfirm(undefined);
      setConfirmed(false);
      await loadRun(id);
      await reload();
      notify(
        accepted
          ? "확인한 후보 한 건만 적용했습니다"
          : "이 후보를 제외했습니다",
      );
    } catch (e) {
      if (current.current === owner) {
        setConfirm(undefined);
        setConfirmed(false);
        setError(e);
      }
    } finally {
      if (current.current === owner) setBusy(false);
    }
  }
  function openRun(id: string) {
    stream.current?.abort();
    setStreaming(false);
    setPreview(undefined);
    setConsent(false);
    setHistoryOpen(false);
    setParams(id ? { run: id } : {});
  }
  const title = (id?: string) =>
    run?.sources.find((s) => s.id === id)?.title ||
    documents.find((s) => s.id === id)?.title ||
    "참조 문서";
  const filtered = documents.filter(
    (d) =>
      !d.deleted_at &&
      d.workspace_id === wid &&
      d.title.toLocaleLowerCase().includes(query.toLocaleLowerCase()),
  );
  const evidence = (action: Action) => (
    <div className="graph-ai-evidence">
      {action.payload.evidence.map(
        (e, i) =>
          e.citation && (
            <Button
              key={i}
              variant="ghost"
              onClick={() =>
                setCitation({ ...e.citation!, title: title(e.document_id) })
              }
            >
              출처 {i + 1} · {title(e.document_id)} · {e.citation.start_line}–
              {e.citation.end_line}행
            </Button>
          ),
      )}
    </div>
  );
  const details = (a: Action) => (
    <>
      {(a.kind === "relation" || a.kind === "duplicate") && (
        <div className="graph-ai-pair">
          <strong>{title(a.payload.source_id)}</strong>
          <GitBranch size={20} />
          <strong>{title(a.payload.target_id)}</strong>
          <Badge>
            {a.payload.relation_type === "reference" ? "참조" : "관련"}
          </Badge>
        </div>
      )}
      {a.kind === "topic" && (
        <p>
          <strong>{a.payload.topic}</strong> → {title(a.payload.source_id)}의
          운영 속성
        </p>
      )}
      {a.payload.description && (
        <p className="graph-ai-description">{a.payload.description}</p>
      )}
      <p>{a.payload.reason}</p>
      {evidence(a)}
    </>
  );
  if (!workspace)
    return (
      <Empty
        title="워크스페이스를 선택하세요"
        text="선택한 문서에서만 AI 지식 제안을 만듭니다."
      />
    );
  return (
    <div className="page graph-ai-page">
      <PageHeading
        eyebrow="KNOWLEDGE INTELLIGENCE"
        title="AI 지식 제안"
        description="연결할 지식, 보완할 질문. 출처를 확인하고 한 건씩 반영하세요."
        actions={
          <>
            <Button
              variant="ghost"
              onClick={() => {
                void loadHistory().catch(setError);
                setHistoryOpen(true);
              }}
            >
              <History size={18} />내 분석 기록
            </Button>
            {runID && <Button onClick={() => openRun("")}>새 분석</Button>}
          </>
        }
      />
      <div className="graph-ai-boundary">
        <LockKeyhole size={21} />
        <div>
          <strong>선택 자료에 한정 · 자동 적용 없음</strong>
          <p>
            개인 문서도 동의한 구간만 AI 공급자에게 전송합니다. 검색 이력은
            수집하지 않으며 전체 지식의 완전성을 판정하지 않습니다.
          </p>
        </div>
      </div>
      <ErrorBox error={error} />
      {!runID ? (
        <div className="graph-ai-setup">
          <section className="graph-ai-card">
            <h2>1. 분석할 문서</h2>
            <p className="muted">현재 열람 가능한 문서 1~12개를 선택하세요.</p>
            <Field label="문서 찾기">
              <div className="graph-ai-search">
                <Search size={18} />
                <input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="문서 제목 검색"
                />
              </div>
            </Field>
            <div className="graph-ai-documents">
              {filtered.map((d) => (
                <label key={d.id}>
                  <input
                    type="checkbox"
                    checked={selected.includes(d.id)}
                    disabled={
                      busy ||
                      (!selected.includes(d.id) && selected.length >= 12)
                    }
                    onChange={(e) =>
                      setSelected((old) =>
                        e.target.checked
                          ? [...old, d.id]
                          : old.filter((id) => id !== d.id),
                      )
                    }
                  />
                  <span>
                    <strong>{d.title}</strong>
                    <small>
                      {visibilityName[d.visibility] || d.visibility} · v
                      {d.version}
                    </small>
                  </span>
                </label>
              ))}
              {!filtered.length && (
                <p className="muted">일치하는 문서가 없습니다.</p>
              )}
            </div>
            <p className="muted">{selected.length} / 12개 선택</p>
          </section>
          <section className="graph-ai-card">
            <h2>2. 제안할 내용</h2>
            <fieldset className="graph-ai-kinds">
              <legend className="sr-only">제안 종류</legend>
              {kinds.map((k) => (
                <label key={k.id}>
                  <input
                    type="checkbox"
                    checked={wanted.includes(k.id)}
                    disabled={
                      busy || (k.id === "topic" && selected.length !== 1)
                    }
                    onChange={(e) => {
                      setWanted((old) =>
                        e.target.checked
                          ? [...old, k.id]
                          : old.filter((x) => x !== k.id),
                      );
                      setConsent(false);
                    }}
                  />
                  <span>
                    <strong>{k.name}</strong>
                    <small>{k.help}</small>
                  </span>
                </label>
              ))}
            </fieldset>
            <Button
              variant="primary"
              disabled={!selected.length || !wanted.length || busy}
              onClick={makePreview}
            >
              <Search size={18} />
              전송 범위 확인
            </Button>
          </section>
          {preview && (
            <section className="graph-ai-card graph-ai-consent">
              <h2>3. 출처와 AI 전송 동의</h2>
              <div className="graph-ai-provider">
                <strong>{preview.provider.model || "모델 미설정"}</strong>
                <span>{preview.provider.base_url || "공급자 미설정"}</span>
                <small>
                  공급자 지문 {preview.provider.fingerprint.slice(0, 16)}…
                </small>
              </div>
              {preview.documents.map((d) => (
                <div className="graph-ai-source" key={d.id}>
                  <span>
                    <strong>{d.title}</strong>
                    <small>
                      v{d.version} · {visibilityName[d.visibility]} ·{" "}
                      {classificationName[d.classification]} ·{" "}
                      {bytes(d.sent_bytes)} / {bytes(d.total_bytes)}
                      {d.truncated ? " (앞부분만 전송)" : " (전체 구간)"}
                    </small>
                  </span>
                  <Button
                    variant="ghost"
                    onClick={() =>
                      setCitation(
                        preview.sources.find((x) => x.id === d.id) || null,
                      )
                    }
                  >
                    전송 원문 확인
                  </Button>
                </div>
              ))}
              <p className="muted">{preview.notice}</p>
              {preview.provider.base_url.startsWith("http:") && (
                <p className="notice warning">
                  HTTP 공급자는 전송 구간이 암호화되지 않습니다. 폐쇄망의 신뢰
                  가능한 주소인지 확인하세요.
                </p>
              )}
              {(!preview.enabled || !preview.provider.configured) && (
                <p className="notice warning">
                  AI 지식 제안 기능과 AI 공급자가 모두 활성화되어야 분석할 수
                  있습니다.
                </p>
              )}
              <label className="graph-ai-checkbox">
                <input
                  type="checkbox"
                  checked={consent}
                  onChange={(e) => setConsent(e.target.checked)}
                />
                <span>
                  개인·기밀 문서를 포함한 위 구간을 표시된 AI 공급자에게
                  전송하는 데 동의합니다. 후보는 별도 확인 후에만 적용됩니다.
                </span>
              </label>
              <Button
                variant="primary"
                disabled={
                  !consent ||
                  !preview.enabled ||
                  !preview.provider.configured ||
                  !wanted.length ||
                  busy
                }
                onClick={start}
              >
                <Sparkles size={18} />
                동의하고 분석
              </Button>
            </section>
          )}
        </div>
      ) : (
        <>
          {(streaming || run?.status === "running") && (
            <section
              className="graph-ai-card graph-ai-progress"
              aria-live="polite"
            >
              <Sparkles size={25} />
              <div>
                <h2>선택한 지식을 살펴보고 있어요</h2>
                <p>
                  응답 {received.toLocaleString()}자 수신 · 출처·형식·정보보호
                  검증 후 후보를 표시합니다.
                </p>
                <small>
                  새로고침하거나 연결이 끊기면 자동 재전송하지 않습니다.
                </small>
              </div>
              <Button onClick={stop}>
                <Square size={17} />
                분석 취소
              </Button>
            </section>
          )}
          {run && (
            <section className="graph-ai-results">
              <div className="graph-ai-results-heading">
                <div>
                  <h2>
                    제안 검토{" "}
                    <Badge>{statusName[run.status] || run.status}</Badge>
                  </h2>
                  <p className="muted">
                    {datetime(run.created_at)} · 출처 {run.sources.length}개 ·
                    후보 {run.actions.length}개
                  </p>
                </div>
                <div className="heading-actions">
                  {run.status === "ready" && (
                    <Button variant="ghost" onClick={stop}>
                      남은 후보 취소
                    </Button>
                  )}
                  <Button variant="ghost" onClick={() => setDeleteID(run.id)}>
                    <Trash2 size={17} />
                    기록 삭제
                  </Button>
                </div>
              </div>
              {run.error && <ErrorBox error={run.error} />}
              {run.status === "ready" && !run.can_apply && (
                <div className="notice warning">
                  현재 권한·원래 로그인·공급자 동의 또는 기능 정책으로는 적용할
                  수 없습니다. 필요한 조건을 확인하고 새 분석을 시작하세요.
                </div>
              )}
              <div className="graph-ai-actions">
                {run.actions.map((a) => (
                  <article
                    className="graph-ai-card"
                    key={a.id}
                    data-action-kind={a.kind}
                  >
                    <div className="graph-ai-action-heading">
                      <Badge>{kindName(a.kind)}</Badge>
                      <Badge tone={a.status === "applied" ? "green" : ""}>
                        {statusName[a.status] || a.status}
                      </Badge>
                    </div>
                    <h3>
                      {a.payload.title || a.payload.topic || kindName(a.kind)}
                    </h3>
                    {details(a)}
                    {(a.kind === "gap" || a.kind === "entity") && (
                      <p className="graph-ai-private">
                        <LockKeyhole size={16} />
                        적용하면 입력 중 가장 높은 보안 등급의 개인 초안으로
                        생성합니다.
                      </p>
                    )}
                    {a.kind === "duplicate" && (
                      <p className="muted">
                        유사성 후보입니다. 문서를 삭제하거나 병합하지 않습니다.
                      </p>
                    )}
                    <div className="graph-ai-action-controls">
                      {a.status === "proposed" && (
                        <>
                          <Button
                            variant="primary"
                            disabled={!run.can_apply || busy}
                            onClick={() => {
                              setConfirm(a);
                              setConfirmed(false);
                            }}
                          >
                            내용 확인 후 적용
                          </Button>
                          <Button
                            variant="ghost"
                            disabled={busy}
                            onClick={() => void decide(a, false)}
                          >
                            후보 제외
                          </Button>
                        </>
                      )}
                      {a.status === "applied" &&
                        a.result.document_id &&
                        !a.result.access_revoked && (
                          <Link
                            className="button"
                            to={`/app/documents/${a.result.document_id}?mode=read`}
                          >
                            <Check size={17} />
                            적용 문서 열기
                          </Link>
                        )}
                      {a.result.access_revoked && (
                        <span className="muted">
                          결과 문서의 현재 접근 권한이 없습니다.
                        </span>
                      )}
                    </div>
                  </article>
                ))}
              </div>
              {run.status === "ready" && !run.actions.length && (
                <Empty
                  title="현재 표시할 후보가 없습니다"
                  text="모델이 근거 있는 후보를 찾지 못했거나 현재 기능 정책으로 표시가 제한되었습니다."
                />
              )}
            </section>
          )}
          {!run && !streaming && !error && <Loading />}
        </>
      )}
      <Modal
        open={!!confirm}
        onOpenChange={(v) => {
          if (!v) {
            setConfirm(undefined);
            setConfirmed(false);
          }
        }}
        title="이 후보 한 건 적용"
        description="확인한 내용의 해시와 현재 원문·권한을 다시 검증합니다. 다른 후보는 적용하지 않습니다."
        wide
      >
        {confirm && (
          <div className="graph-ai-confirm">
            <Badge>{kindName(confirm.kind)}</Badge>
            <h3>
              {confirm.payload.title ||
                confirm.payload.topic ||
                kindName(confirm.kind)}
            </h3>
            {details(confirm)}
            <p className="muted">
              후보 지문 {confirm.action_hash.slice(0, 20)}…
            </p>
            <label className="graph-ai-checkbox">
              <input
                type="checkbox"
                checked={confirmed}
                onChange={(e) => setConfirmed(e.target.checked)}
              />
              <span>
                출처와 변경 내용을 확인했으며, 이 후보 한 건의 적용에
                동의합니다.
              </span>
            </label>
            <Button
              variant="primary"
              disabled={!confirmed || busy || !run?.can_apply}
              onClick={() => void decide(confirm, true)}
            >
              <Check size={18} />
              확인한 한 건 적용
            </Button>
          </div>
        )}
      </Modal>
      <Modal
        open={historyOpen}
        onOpenChange={setHistoryOpen}
        title="내 AI 지식 분석 기록"
        description="현재 모든 입력 문서를 열람할 수 있는 본인 기록만 표시합니다. 새로 열어도 AI를 다시 호출하지 않습니다."
        wide
      >
        <Button
          variant="ghost"
          onClick={() => void loadHistory().catch(setError)}
        >
          <RefreshCw size={17} />
          목록 새로고침
        </Button>
        <div className="graph-ai-history">
          {history.map((h) => (
            <button key={h.id} onClick={() => openRun(h.id)}>
              <strong>{h.kinds.map(kindName).join(" · ")}</strong>
              <span>
                {datetime(h.created_at)} · {statusName[h.status] || h.status}
              </span>
            </button>
          ))}
          {!history.length && (
            <Empty
              title="표시할 분석 기록이 없습니다"
              text="선택한 문서로 첫 분석을 시작하세요."
            />
          )}
        </div>
      </Modal>
      <Modal
        open={!!deleteID}
        onOpenChange={(v) => {
          if (!v) setDeleteID("");
        }}
        title="분석 기록 삭제"
        description="개인 분석 기록과 미적용 후보를 삭제합니다. 이미 적용한 문서·관계·주제는 그대로 유지됩니다."
      >
        <Button
          variant="danger"
          disabled={busy}
          onClick={async () => {
            const id = deleteID;
            setBusy(true);
            try {
              await api(`/graph-ai/runs/${id}`, "DELETE");
              setDeleteID("");
              if (id === runID) openRun("");
              await loadHistory();
            } catch (e) {
              setError(e);
            } finally {
              setBusy(false);
            }
          }}
        >
          기록만 삭제
        </Button>
      </Modal>
      <CitationViewer
        source={citation}
        onClose={() => setCitation(null)}
        onNavigate={() => {
          stream.current?.abort();
          setStreaming(false);
        }}
      />
    </div>
  );
}
