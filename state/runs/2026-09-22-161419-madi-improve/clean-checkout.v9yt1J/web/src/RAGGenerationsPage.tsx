import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { Link } from "react-router-dom";
import { ArrowLeft, Play, Plus, RefreshCw, Trash2 } from "lucide-react";
import { api, ApiError } from "./api";
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
} from "./ui";
import "./rag-generations.css";
const DocumentRAGIndex = lazy(() => import("./DocumentRAGIndex"));
type Mode = "exact" | "ann" | "verify";
type Generation = {
  id: string;
  name: string;
  revision: number;
  status: string;
  dimensions: number;
  provider_fingerprint: string;
  ann_ready: boolean;
  provider: {
    base_url: string;
    model: string;
    api_key_configured: boolean;
    allow_http: boolean;
    rerank_enabled: boolean;
  };
  index: {
    consented_documents: number;
    ready_documents: number;
    pending_documents: number;
    failed_documents: number;
  };
  index_job_id: string | null;
  index_job: { status: string; cancel_requested: boolean } | null;
};
type State = {
  generations: Generation[];
  active_id: string;
  state_revision: number;
  mode: Mode;
  enabled: boolean;
  settings_fingerprint: string;
  scope: string;
};
type Verification = {
  validation_id: string;
  state_revision: number;
  generation_revision: number;
  expires_in_seconds: number;
  can_activate: boolean;
  report: {
    mode: Mode;
    recall_at_k: number;
    exact_candidates: number;
    ann_candidates: number;
    documents: number;
    truncated: boolean;
    dimensions: number;
    index_name?: string;
    planned_index_used: boolean;
    warnings: string[];
  };
};
const modes: Record<Mode, string> = {
  exact: "정확 검색",
  ann: "근사 검색 · HNSW",
  verify: "검증 검색 · ANN + 정확 비교",
};
const statuses: Record<string, string> = {
  building: "새 세대 준비",
  active: "현재 검색 중",
  retired: "이전 세대",
  disabled: "복원 후 비활성",
};
const jobs: Record<string, string> = {
  pending: "대기",
  running: "인덱스 생성 중",
  succeeded: "완료",
  failed: "실패",
  cancelled: "취소됨",
};

export default function RAGGenerationsPage() {
  const { user, workspace, documents, reload } = useApp();
  const scope = `${user.id}:${workspace?.id}`,
    current = useRef(scope);
  current.current = scope;
  const [loaded, setLoaded] = useState(""),
    [state, setState] = useState<State | null>(null),
    [selected, setSelected] = useState(""),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(""),
    [notice, setNotice] = useState("");
  const [query, setQuery] = useState(""),
    [mode, setMode] = useState<Mode>("exact"),
    [cohort, setCohort] = useState<string[]>([]),
    [consent, setConsent] = useState(false),
    [coverage, setCoverage] = useState(false),
    [verification, setVerification] = useState<Verification | null>(null),
    [expires, setExpires] = useState(0),
    [now, setNow] = useState(Date.now());
  const [create, setCreate] = useState(false),
    [name, setName] = useState(""),
    [model, setModel] = useState(""),
    [dimensions, setDimensions] = useState(3),
    [custom, setCustom] = useState(false),
    [baseURL, setBaseURL] = useState(""),
    [key, setKey] = useState(""),
    [http, setHTTP] = useState(false),
    [createConfirmed, setCreateConfirmed] = useState(false);
  const [indexDocument, setIndexDocument] = useState<string | null>(null),
    [documentQuery, setDocumentQuery] = useState(""),
    [confirm, setConfirm] = useState<
      "index" | "activate" | "delete" | "cancel" | null
    >(null);
  const sequence = useRef(0),
    mutation = useRef(false),
    createDraftEdited = useRef(false),
    actionAbort = useRef<AbortController | null>(null),
    confirmation = useRef<{
      id: string;
      revision: number;
      stateRevision: number;
      settings: string;
      jobID: string | null;
      validationID: string | null;
    } | null>(null);
  const allowed =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const visible = loaded === scope ? state : null;
  const generation = visible?.generations.find((g) => g.id === selected);
  const stateRef = useRef(visible);
  stateRef.current = visible;
  const generationRef = useRef(generation);
  generationRef.current = generation;
  const listPath = workspace
    ? `/workspaces/${workspace.id}/rag-generations`
    : "";
  const resetValidation = () => {
    setVerification(null);
    setExpires(0);
    setConsent(false);
    setCoverage(false);
    setConfirm(null);
    confirmation.current = null;
  };
  const load = useCallback(async () => {
    if (!allowed || !listPath) return;
    const request = ++sequence.current;
    try {
      const value = await api<State>(listPath);
      if (current.current !== scope || request !== sequence.current) return;
      setState(value);
      setLoaded(scope);
      setSelected((old) =>
        value.generations.some((g) => g.id === old)
          ? old
          : value.active_id || value.generations[0]?.id || "",
      );
    } catch (e) {
      if (current.current !== scope || request !== sequence.current) return;
      setState(null);
      setLoaded(scope);
      setError(e);
      resetValidation();
    }
  }, [allowed, listPath, scope]);
  useEffect(() => {
    setLoaded("");
    setState(null);
    setSelected("");
    setBusy("");
    mutation.current = false;
    setError(null);
    setNotice("");
    setCohort([]);
    setQuery("");
    setCreate(false);
    setIndexDocument(null);
    setKey("");
    resetValidation();
    void load();
    const tick = setInterval(() => {
      if (document.visibilityState === "visible") void load();
    }, 5000);
    return () => {
      sequence.current++;
      clearInterval(tick);
      actionAbort.current?.abort();
    };
  }, [scope, load]);
  const documentVersions = cohort
    .map((id) => `${id}:${documents.find((d) => d.id === id)?.version || 0}`)
    .join("|");
  useEffect(
    () => resetValidation(),
    [
      selected,
      generation?.revision,
      generation?.provider_fingerprint,
      visible?.state_revision,
      visible?.settings_fingerprint,
      query,
      mode,
      documentVersions,
    ],
  );
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, []);
  useEffect(
    () => setCreateConfirmed(false),
    [name, model, dimensions, custom, baseURL, key, http],
  );
  const fresh = (id: string, revision: number, stateRevision: number) =>
    current.current === scope &&
    generationRef.current?.id === id &&
    generationRef.current.revision === revision &&
    stateRef.current?.state_revision === stateRevision;
  const run = async (kind: string, fn: () => Promise<void>) => {
    if (mutation.current || !visible || !allowed) return;
    mutation.current = true;
    setBusy(kind);
    setError(null);
    setNotice("");
    try {
      await fn();
    } catch (e) {
      if (current.current === scope) {
        if (e instanceof DOMException && e.name === "AbortError")
          setNotice(
            "검증 요청을 취소했습니다. 전환 결과를 적용하지 않았습니다.",
          );
        else setError(e);
        resetValidation();
      }
    } finally {
      if (current.current === scope) {
        mutation.current = false;
        setBusy("");
        void load();
      }
    }
  };
  const openCreate = async () => {
    createDraftEdited.current = false;
    setError(null);
    setName("");
    setModel("");
    setKey("");
    setBaseURL("");
    setCustom(false);
    setCreateConfirmed(false);
    setCreate(true);
    try {
      const settings = await api<{ data: Record<string, any> }>(
        `/workspaces/${workspace!.id}/search-ai`,
      );
      if (current.current !== scope || createDraftEdited.current) return;
      setModel(settings.data.rag_embedding_model || "");
      setDimensions(settings.data.rag_embedding_dimensions || 768);
      setHTTP(settings.data.rag_allow_http === true);
    } catch (e) {
      if (current.current === scope) setError(e);
    }
  };
  const createGeneration = () =>
    run("create", async () => {
      const config: Record<string, unknown> = {
        rag_embedding_model: model.trim(),
      };
      if (custom) {
        config.rag_embedding_base_url = baseURL.trim();
        config.rag_embedding_api_key = key;
        config.rag_allow_http = http;
      }
      const result = await api<{ id: string }>(listPath, "POST", {
        name: name.trim(),
        dimensions,
        config,
        confirm: true,
      });
      if (current.current !== scope) return;
      setKey("");
      setCreate(false);
      setSelected(result.id);
      setNotice(
        "새 세대 설정을 만들었습니다. 문서별 전송 동의와 색인 작업을 먼저 준비하세요. 현재 검색 세대는 바뀌지 않았습니다.",
      );
    });
  const verify = () =>
    run("verify", async () => {
      if (!generation || !visible || !consent) return;
      const g = generation,
        s = visible,
        controller = new AbortController();
      actionAbort.current = controller;
      const response = await fetch(`/api/v1/rag-generations/${g.id}/verify`, {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/json", "X-Madi-Request": "1" },
        signal: controller.signal,
        body: JSON.stringify({
          revision: g.revision,
          provider_fingerprint: g.provider_fingerprint,
          query,
          mode,
          consent: true,
          documents: cohort.map((id) => ({
            document_id: id,
            version: documents.find((d) => d.id === id)?.version || 0,
          })),
        }),
      });
      const body = await response.json();
      if (!response.ok)
        throw new ApiError(
          body.error || "세대 검증을 완료하지 못했습니다",
          response.status,
        );
      if (
        !fresh(g.id, g.revision, s.state_revision) ||
        stateRef.current?.settings_fingerprint !== s.settings_fingerprint
      )
        return;
      setVerification(body);
      setExpires(Date.now() + body.expires_in_seconds * 1000);
      setCoverage(false);
      setNotice(
        "실제 검증이 완료되었습니다. 결과와 검색 범위 변경을 확인하기 전에는 세대를 전환하지 않습니다.",
      );
    });
  const ask = (action: "index" | "activate" | "delete" | "cancel") => {
    if (!generation || !visible || busy) return;
    confirmation.current = {
      id: generation.id,
      revision: generation.revision,
      stateRevision: visible.state_revision,
      settings: visible.settings_fingerprint,
      jobID: generation.index_job_id,
      validationID: verification?.validation_id || null,
    };
    setConfirm(action);
  };
  const perform = () =>
    run(confirm || "", async () => {
      if (!generation || !visible || !confirm) return;
      const expected = confirmation.current;
      if (
        !expected ||
        !fresh(expected.id, expected.revision, expected.stateRevision) ||
        expected.settings !== visible.settings_fingerprint ||
        expected.jobID !== generation.index_job_id ||
        (confirm === "activate" &&
          expected.validationID !== verification?.validation_id)
      ) {
        resetValidation();
        throw new Error(
          "확인 이후 선택한 세대나 설정이 변경되었습니다. 현재 상태를 다시 확인하세요.",
        );
      }
      const g = generation,
        s = visible,
        action = confirm;
      setConfirm(null);
      if (action === "index")
        await api(`/rag-generations/${g.id}/index`, "POST", {
          revision: g.revision,
          confirm: true,
        });
      if (action === "cancel" && g.index_job_id)
        await api(`/jobs/${g.index_job_id}/cancel`, "POST", {});
      if (action === "delete")
        await api(`/rag-generations/${g.id}`, "DELETE", {
          revision: g.revision,
          confirm: true,
        });
      if (action === "activate") {
        if (
          !verification ||
          !coverage ||
          Date.now() >= expires ||
          verification.state_revision !== s.state_revision
        )
          return;
        await api(listPath + "/activate", "POST", {
          validation_id: verification.validation_id,
          state_revision: verification.state_revision,
          mode: verification.report.mode,
          confirm: true,
          confirm_coverage: true,
        });
      }
      if (current.current !== scope) return;
      resetValidation();
      setNotice(
        action === "activate"
          ? "검증한 세대와 공급자 설정을 함께 전환했습니다. 현재 권한과 동의가 일치하는 자료만 검색합니다."
          : action === "delete"
            ? "선택한 세대의 설정·동의·검증 기록·파생 색인을 삭제했습니다. 원본 문서는 유지했습니다."
            : action === "cancel"
              ? "인덱스 작업 취소를 요청했습니다. 완료 상태를 확인하세요."
              : "HNSW 인덱스 작업을 요청했습니다. 완료 후 실제 검증을 실행하세요.",
      );
    });
  const pending =
    !!generation?.index_job &&
    ["pending", "running"].includes(generation.index_job.status);
  const readyReceipt =
    !!verification &&
    verification.can_activate &&
    now < expires &&
    verification.state_revision === visible?.state_revision;
  const docs = documents.filter(
    (d) =>
      d.workspace_id === workspace?.id &&
      (!documentQuery ||
        d.title
          .toLocaleLowerCase()
          .includes(documentQuery.toLocaleLowerCase())),
  );
  if (!allowed)
    return (
      <Empty
        title="워크스페이스 관리 권한이 필요합니다"
        text="서비스 관리자도 현재 워크스페이스 관리 멤버십과 문서 권한을 우회할 수 없습니다."
      />
    );
  return (
    <section className="rag-generations-page">
      <PageHeading
        eyebrow="검색 운영"
        title="벡터 색인 세대"
        description="새 모델을 따로 준비하고, 실제 검증과 명시적인 전환으로 현재 검색을 바꿉니다."
        actions={
          <>
            <Link className="button" to="/app/search/operations">
              <ArrowLeft size={16} />
              검색 운영
            </Link>
            <Link className="button" to="/app/search-ai-settings">
              공급자 설정
            </Link>
            <Button
              onClick={() => void openCreate()}
              disabled={!!busy || !visible?.enabled}
            >
              <Plus size={17} />새 세대
            </Button>
          </>
        }
      />
      <ErrorBox error={error} />
      {notice && (
        <p className="notice" role="status">
          {notice}
        </p>
      )}
      {!visible ? (
        loaded === scope ? (
          <Button onClick={() => void load()}>
            현재 권한으로 다시 불러오기
          </Button>
        ) : (
          <Loading />
        )
      ) : (
        <>
          {!visible.enabled && (
            <p className="notice">
              검색 AI가 꺼져 있습니다. 공급자 설정에서 활성화한 뒤 세대를
              준비하세요.
            </p>
          )}
          <p className="notice subtle">
            {visible.scope} 검증 문서 외 자료나 다른 사용자 권한에서의 검색
            품질을 보증하지 않습니다.
          </p>
          <div className="rag-generation-toolbar">
            <strong>현재 모드 · {modes[visible.mode]}</strong>
            <Button onClick={() => void load()} disabled={!!busy}>
              <RefreshCw size={16} />
              상태 새로 확인
            </Button>
          </div>
          {!visible.generations.length ? (
            <Empty
              title="아직 관리 중인 세대가 없습니다"
              text="새 세대를 만들면 현재 공급자의 기존 색인을 기준 세대로 보존합니다. 새 공급자로 문서를 자동 전송하지 않습니다."
            />
          ) : (
            <div className="rag-generation-layout">
              <nav className="rag-generation-list" aria-label="색인 세대 목록">
                {visible.generations.map((g) => (
                  <button
                    key={g.id}
                    disabled={!!busy}
                    className={g.id === selected ? "selected" : ""}
                    onClick={() => {
                      setSelected(g.id);
                      setCohort([]);
                      setIndexDocument(null);
                      resetValidation();
                    }}
                    aria-pressed={g.id === selected}
                  >
                    <strong>{g.name}</strong>
                    <span>
                      {statuses[g.status] || g.status} ·{" "}
                      {g.dimensions || "미고정"}차원
                    </span>
                    <small>{g.provider.model}</small>
                  </button>
                ))}
              </nav>
              {generation && (
                <div className="rag-generation-detail" key={generation.id}>
                  <div className="rag-generation-header">
                    <div>
                      <h2>{generation.name}</h2>
                      <p>
                        {generation.provider.model} ·{" "}
                        {generation.dimensions || "차원 확인 필요"}차원
                      </p>
                    </div>
                    <Badge>{statuses[generation.status]}</Badge>
                  </div>
                  <dl className="rag-generation-provider">
                    <div>
                      <dt>전송 공급자</dt>
                      <dd>{generation.provider.base_url}</dd>
                    </div>
                    <div>
                      <dt>설정 지문</dt>
                      <dd>{generation.provider_fingerprint}</dd>
                    </div>
                  </dl>
                  {generation.provider.allow_http && (
                    <p className="notice">
                      내부 HTTP 허용 설정입니다. 질문과 문서가 암호화되지 않은
                      연결로 전송될 수 있습니다.
                    </p>
                  )}
                  {generation.status === "disabled" ? (
                    <p className="notice">
                      복원으로 비활성화된 세대입니다. 새 세대를 만들고 원본
                      문서에 다시 동의해야 합니다.
                    </p>
                  ) : (
                    <>
                      <h3>1. 문서 동의와 새 세대 색인</h3>
                      <p>
                        현재 열람 가능한 동의 문서{" "}
                        {generation.index.consented_documents}개 · 현재 버전
                        준비 {generation.index.ready_documents}개 · 대기{" "}
                        {generation.index.pending_documents}개 · 실패/취소{" "}
                        {generation.index.failed_documents}개
                      </p>
                      <p className="muted">
                        동의·색인은 이 세대에만 적용됩니다. 다른 사용자의 비공개
                        자료는 상태 집계나 검증 목록에 포함하지 않습니다.
                        아래에서 검증에 사용할 문서를 최대 30개 선택하세요.
                      </p>
                      <Field label="검증 문서 찾기">
                        <input
                          value={documentQuery}
                          onChange={(e) => setDocumentQuery(e.target.value)}
                          placeholder="문서 제목"
                        />
                      </Field>
                      <div
                        className="rag-generation-documents"
                        aria-label="검증 문서 선택"
                      >
                        {docs.map((d) => (
                          <div key={d.id}>
                            <label>
                              <input
                                type="checkbox"
                                checked={cohort.includes(d.id)}
                                disabled={
                                  !!busy ||
                                  (!cohort.includes(d.id) &&
                                    cohort.length >= 30)
                                }
                                onChange={(e) =>
                                  setCohort((old) =>
                                    e.target.checked
                                      ? [...old, d.id]
                                      : old.filter((id) => id !== d.id),
                                  )
                                }
                              />
                              <span>
                                {d.title}
                                <small>
                                  저장본 v{d.version} ·{" "}
                                  {d.visibility === "private"
                                    ? "나만 보기"
                                    : "현재 접근 가능한 문서"}
                                </small>
                              </span>
                            </label>
                            <Button
                              onClick={() => setIndexDocument(d.id)}
                              disabled={!!busy}
                            >
                              세대별 동의·색인
                            </Button>
                          </div>
                        ))}
                      </div>
                      {!docs.length && (
                        <p className="muted">
                          현재 목록과 조건에서 열람 가능한 문서가 없습니다.
                        </p>
                      )}
                      <h3>2. 실제 벡터 실행 방식 준비</h3>
                      <Field label="검증하고 전환할 모드">
                        <select
                          value={mode}
                          onChange={(e) => setMode(e.target.value as Mode)}
                          disabled={!!busy}
                        >
                          {Object.entries(modes).map(([value, label]) => (
                            <option value={value} key={value}>
                              {label}
                            </option>
                          ))}
                        </select>
                      </Field>
                      <p className="muted">
                        정확 검색도 설정된 scan_limit 이내의 현재 권한 후보만
                        확인합니다. HNSW는 pgvector 0.8 이상과 1~2000차원 세대가
                        필요하며, 확장은 운영자가 먼저 설치해야 합니다. 검증
                        모드는 실제 ANN 결과와 정확 결과를 함께 계산해 비용이 더
                        듭니다.
                      </p>
                      <div className="rag-generation-toolbar">
                        <Badge>
                          {generation.ann_ready ? "HNSW 준비됨" : "HNSW 미준비"}
                        </Badge>
                        {generation.index_job && (
                          <span>
                            작업 ·{" "}
                            {jobs[generation.index_job.status] ||
                              generation.index_job.status}
                            {generation.index_job.cancel_requested
                              ? " · 취소 요청됨"
                              : ""}
                          </span>
                        )}
                        <Button
                          onClick={() => ask("index")}
                          disabled={
                            !!busy ||
                            pending ||
                            generation.dimensions < 1 ||
                            generation.dimensions > 2000
                          }
                        >
                          HNSW 인덱스 준비
                        </Button>
                        {pending && (
                          <Button
                            onClick={() => ask("cancel")}
                            disabled={
                              !!busy || generation.index_job?.cancel_requested
                            }
                          >
                            인덱스 작업 취소
                          </Button>
                        )}
                      </div>
                      <h3>3. 현재 권한으로 검증</h3>
                      <Field label="검증 질문">
                        <textarea
                          value={query}
                          onChange={(e) => setQuery(e.target.value)}
                          rows={3}
                          disabled={!!busy}
                          placeholder="선택한 문서에서 찾을 실제 질문"
                        />
                      </Field>
                      <label className="rag-generation-check">
                        <input
                          type="checkbox"
                          checked={consent}
                          onChange={(e) => setConsent(e.target.checked)}
                          disabled={!!busy}
                        />
                        <span>
                          위 공급자·모델로 이 질문을 전송하는 데 동의합니다.
                          문서 본문 전송 동의는 각 문서에서 별도로 관리하고,
                          여기서는 이미 준비된 벡터를 현재 권한으로 비교합니다.
                        </span>
                      </label>
                      <div className="rag-generation-toolbar">
                        <Button
                          variant="primary"
                          onClick={() => void verify()}
                          disabled={
                            !!busy ||
                            !visible.enabled ||
                            !query.trim() ||
                            !consent ||
                            !cohort.length ||
                            (mode !== "exact" && !generation.ann_ready)
                          }
                        >
                          <Play size={16} />
                          {busy === "verify"
                            ? "실제 검증 중…"
                            : "실제 검증 실행"}
                        </Button>
                        {busy === "verify" && (
                          <Button onClick={() => actionAbort.current?.abort()}>
                            검증 요청 취소
                          </Button>
                        )}
                        <span>선택 {cohort.length}/30개</span>
                      </div>
                      {verification && (
                        <div
                          className="rag-generation-report"
                          aria-label="벡터 검증 결과"
                        >
                          <h3>검증 결과 · {modes[verification.report.mode]}</h3>
                          <p>
                            {verification.report.documents}개 문서 · 정확 후보{" "}
                            {verification.report.exact_candidates}개
                            {verification.report.mode !== "exact"
                              ? ` · ANN 후보 ${verification.report.ann_candidates}개 · Recall@k ${(verification.report.recall_at_k * 100).toFixed(1)}%`
                              : " · ANN 비교를 실행하지 않은 정확 모드 검증"}
                          </p>
                          <p>
                            {verification.report.planned_index_used
                              ? "실제 HNSW 실행 계획 확인"
                              : "정확 배열 경로"}{" "}
                            ·{" "}
                            {verification.report.truncated
                              ? "정확 기준 범위 잘림 — 전환 불가"
                              : "정확 기준 잘림 없음"}
                          </p>
                          {verification.report.warnings.map((warning, i) => (
                            <p className="muted" key={i}>
                              {warning}
                            </p>
                          ))}
                          <p>
                            {now >= expires
                              ? "검증 영수증이 만료되었습니다. 다시 검증하세요."
                              : `현재 상태에 묶인 영수증 · ${Math.ceil((expires - now) / 60000)}분 이내 전환 가능`}
                          </p>
                        </div>
                      )}
                      <h3>4. 명시적인 전환 또는 이전 세대로 복귀</h3>
                      <label className="rag-generation-check">
                        <input
                          type="checkbox"
                          checked={coverage}
                          onChange={(e) => setCoverage(e.target.checked)}
                          disabled={!readyReceipt || !!busy}
                        />
                        <span>
                          검증은 선택한 문서와 질문 한 건에 한정됩니다. 새
                          세대에 동의하지 않았거나 현재 버전 색인이 준비되지
                          않은 자료는 전환 후 검색에서 빠질 수 있음을
                          확인했습니다. 이전 세대로 돌아갈 때도 새 검증이
                          필요합니다.
                        </span>
                      </label>
                      <Button
                        variant="primary"
                        onClick={() => ask("activate")}
                        disabled={!!busy || !readyReceipt || !coverage}
                      >
                        {generation.id === visible.active_id
                          ? "검증한 모드 적용"
                          : generation.status === "retired"
                            ? "검증한 이전 세대로 복귀"
                            : "검증한 세대로 전환"}
                      </Button>
                    </>
                  )}
                  {generation.id !== visible.active_id && (
                    <div className="rag-generation-delete">
                      <Button
                        variant="danger"
                        onClick={() => ask("delete")}
                        disabled={!!busy || pending}
                      >
                        <Trash2 size={16} />이 세대 삭제
                      </Button>
                      <p className="muted">
                        원본 문서를 지우지 않고 이 세대의 설정·동의·검증
                        기록·파생 벡터만 삭제합니다. 세대를 재사용하려면 새
                        동의가 필요합니다.
                      </p>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </>
      )}
      <Modal
        open={create}
        onOpenChange={(open) => {
          if (!busy) {
            setCreate(open);
            if (!open) setKey("");
          }
        }}
        title="새 벡터 색인 세대"
        description="불변 모델·차원 설정을 준비합니다. 문서 전송과 현재 세대 전환은 자동 실행하지 않습니다."
        wide
      >
        <ErrorBox error={error} />
        <form
          onChange={() => {
            createDraftEdited.current = true;
          }}
          onSubmit={(e) => {
            e.preventDefault();
            void createGeneration();
          }}
          className="rag-generation-form"
        >
          <Field label="세대 이름">
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              maxLength={200}
              required
            />
          </Field>
          <Field label="임베딩 모델">
            <input
              value={model}
              onChange={(e) => setModel(e.target.value)}
              maxLength={256}
              required
            />
          </Field>
          <Field label="고정 벡터 차원">
            <input
              type="number"
              min={1}
              max={8192}
              value={dimensions}
              onChange={(e) => setDimensions(Number(e.target.value))}
              required
            />
          </Field>
          <label className="rag-generation-check">
            <input
              type="checkbox"
              checked={custom}
              onChange={(e) => {
                setCustom(e.target.checked);
                setKey("");
                setCreateConfirmed(false);
              }}
            />
            <span>현재 공급자 대신 다른 임베딩 API 주소를 사용합니다</span>
          </label>
          {custom && (
            <>
              <Field label="임베딩 API 주소">
                <input
                  value={baseURL}
                  onChange={(e) => {
                    setBaseURL(e.target.value);
                    setKey("");
                    setCreateConfirmed(false);
                  }}
                  required
                  placeholder="https://llm.internal/v1"
                />
              </Field>
              <Field label="새 공급자 API 키">
                <input
                  type="password"
                  value={key}
                  onChange={(e) => setKey(e.target.value)}
                  autoComplete="new-password"
                  placeholder="필요한 경우 입력"
                />
              </Field>
              <label className="rag-generation-check">
                <input
                  type="checkbox"
                  checked={http}
                  onChange={(e) => setHTTP(e.target.checked)}
                />
                <span>내부 HTTP 연결을 명시적으로 허용합니다</span>
              </label>
            </>
          )}
          <p className="muted">
            다른 주소를 선택하면 기존 API 키를 자동 전달하지 않습니다. 같은
            주소를 유지하면 현재 공급자 설정을 복사합니다. 차원·모델을 바꾸면
            문서별 새 전송 동의와 재색인이 필요합니다. 최대 20개 세대를 보관할
            수 있습니다.
          </p>
          <label className="rag-generation-check">
            <input
              type="checkbox"
              checked={createConfirmed}
              onChange={(e) => setCreateConfirmed(e.target.checked)}
            />
            <span>
              모델·차원·전송 주소를 확인했습니다. 이 작업은 현재 검색 세대를
              전환하지 않습니다.
            </span>
          </label>
          <Button
            type="submit"
            variant="primary"
            disabled={
              !!busy || !createConfirmed || !name.trim() || !model.trim()
            }
          >
            세대 만들기
          </Button>
        </form>
      </Modal>
      <Modal
        open={!!confirm}
        onOpenChange={(open) => {
          if (!open && !busy) setConfirm(null);
        }}
        title={
          confirm === "activate"
            ? "검증한 검색 세대로 전환할까요?"
            : confirm === "delete"
              ? "이 세대의 파생 색인을 삭제할까요?"
              : confirm === "cancel"
                ? "인덱스 작업을 취소할까요?"
                : "HNSW 인덱스를 준비할까요?"
        }
        description={
          confirm === "activate"
            ? "공급자 설정과 활성 세대를 함께 바꿉니다. 현재 권한·문서 버전·동의·검증 영수증이 달라졌으면 적용하지 않습니다."
            : confirm === "delete"
              ? "원본 문서는 유지하지만 세대의 설정·동의·검증 기록·벡터 인덱스는 삭제합니다. 재생성하려면 새 동의와 색인이 필요합니다."
              : confirm === "cancel"
                ? "이미 완료된 파생 인덱스가 있으면 보존됩니다. 취소 완료 여부를 상태에서 확인하세요."
                : "로컬 PostgreSQL의 실제 HNSW 인덱스를 작업 큐에서 만듭니다. 외부 공급자로 문서를 전송하거나 검색 세대를 바꾸지 않습니다."
        }
      >
        <Button
          variant={confirm === "delete" ? "danger" : "primary"}
          onClick={() => void perform()}
          disabled={!!busy}
        >
          확인하고 계속
        </Button>
      </Modal>
      {indexDocument && generation && (
        <Suspense fallback={<Loading />}>
          <DocumentRAGIndex
            documentID={indexDocument}
            generationID={generation.id}
            onClose={() => {
              setIndexDocument(null);
              void load();
              void reload();
            }}
          />
        </Suspense>
      )}
    </section>
  );
}
