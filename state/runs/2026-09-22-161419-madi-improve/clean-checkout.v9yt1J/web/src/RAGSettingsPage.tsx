import { useEffect, useRef, useState, type ReactNode } from "react";
import { Link } from "react-router-dom";
import {
  Database,
  History,
  KeyRound,
  Network,
  RefreshCw,
  Search,
  ShieldCheck,
  TestTube2,
  TriangleAlert,
} from "lucide-react";
import { api } from "./api";
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
import "./rag-settings.css";

const defaults = {
  rag_enabled: false,
  rag_embedding_base_url: "",
  rag_embedding_model: "",
  rag_embedding_api_key: "",
  rag_embedding_dimensions: 0,
  rag_allow_http: false,
  rag_ca_pem: "",
  rag_backend: "array",
  rag_top_k: 8,
  rag_candidates: 60,
  rag_scan_limit: 5000,
  rag_search_mode: "hybrid",
  rag_rerank_enabled: false,
  rag_rerank_base_url: "",
  rag_rerank_model: "",
  rag_rerank_api_key: "",
};
type Key = keyof typeof defaults;
type Value = string | number | boolean;
type Config = Record<Key, Value> & Record<string, unknown>;
type Snapshot = { data: Config; version: number; overrides: string[] };
type Diagnostic = {
  ok: boolean;
  dimensions: number;
  pgvector_installed: boolean;
  content_sent: string;
};
const keys = Object.keys(defaults) as Key[];
const secretKeys: Key[] = ["rag_embedding_api_key", "rag_rerank_api_key"];
function onlyRAG(data: Record<string, unknown>): Config {
  const result: Config = { ...defaults };
  for (const key of keys) {
    if (data[key] !== undefined) result[key] = data[key] as Value;
    if (secretKeys.includes(key)) {
      // A secret is write-only, including if a future server mistakenly returns it.
      result[key] = "";
      result[key + "_configured"] = data[key + "_configured"] === true;
    }
  }
  return result;
}

export function RAGSettingsPage({
  workspaceMode = false,
}: {
  workspaceMode?: boolean;
}) {
  const { user, workspace } = useApp();
  const allowed = workspaceMode
    ? !!workspace &&
      ["owner", "admin"].includes(workspace.role) &&
      user.role !== "viewer"
    : user.role === "admin";
  if (!allowed)
    return (
      <div className="page">
        <Empty
          title="검색 AI 설정 권한이 필요합니다"
          text={
            workspaceMode
              ? "현재 워크스페이스 소유자 또는 관리자로 접근하세요."
              : "서비스 관리자로 접근하세요."
          }
        />
      </div>
    );
  // A workspace/account change discards secret drafts and all old request state.
  return (
    <RAGSettingsForm
      key={`${user.id}:${workspaceMode ? workspace!.id : "service"}`}
      workspaceID={workspaceMode ? workspace!.id : undefined}
    />
  );
}

function RAGSettingsForm({ workspaceID }: { workspaceID?: string }) {
  const { workspace, notify } = useApp();
  const [snapshot, setSnapshot] = useState<Snapshot | null>(null);
  const [patch, setPatch] = useState<Partial<Record<Key, Value | null>>>({});
  const [clears, setClears] = useState<Key[]>([]);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState<"load" | "save" | "test" | null>("load");
  const [diagnostic, setDiagnostic] = useState<Diagnostic | null>(null);
  const [reset, setReset] = useState(false);
  const generation = useRef(0),
    mounted = useRef(true);
  const workspaceMode = !!workspaceID;
  const settingsPath = workspaceMode
    ? `/workspaces/${workspaceID}/settings`
    : "/admin/settings";
  const readPath = workspaceMode
    ? `/workspaces/${workspaceID}/search-ai`
    : settingsPath;
  const testPath = workspaceMode
    ? `/workspaces/${workspaceID}/search-ai/test`
    : "/admin/search-ai/test";
  const dirty = Object.keys(patch).length > 0 || clears.length > 0;
  const current = (key: Key): Value =>
    patch[key] ?? snapshot?.data[key] ?? defaults[key];
  const overridden = (key: Key) =>
    Object.hasOwn(patch, key)
      ? patch[key] !== null
      : !!snapshot?.overrides.includes(key);
  const writable = (key: Key) => !busy && (!workspaceMode || overridden(key));
  const update = (key: Key, value: Value) => {
    setPatch((old) => ({ ...old, [key]: value }));
    setDiagnostic(null);
  };
  const fetchSnapshot = async (): Promise<Snapshot> => {
    const result = await api(readPath);
    return workspaceMode
      ? {
          data: onlyRAG(result.data),
          version: result.version,
          overrides: result.overrides,
        }
      : { data: onlyRAG(result), version: 0, overrides: [] };
  };
  const load = async () => {
    const request = ++generation.current;
    setBusy("load");
    setError(null);
    try {
      const next = await fetchSnapshot();
      if (!mounted.current || generation.current !== request) return;
      setSnapshot(next);
      setPatch({});
      setClears([]);
      setDiagnostic(null);
    } catch (e) {
      if (mounted.current && generation.current === request) setError(e);
    } finally {
      if (mounted.current && generation.current === request) setBusy(null);
    }
  };
  useEffect(() => {
    mounted.current = true;
    void load();
    return () => {
      mounted.current = false;
      generation.current++;
    };
  }, []);
  useEffect(() => {
    const prevent = (event: BeforeUnloadEvent) => {
      if (dirty) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    addEventListener("beforeunload", prevent);
    return () => removeEventListener("beforeunload", prevent);
  }, [dirty]);

  const field = (
    key: Key,
    label: string,
    control: ReactNode,
    hint?: string,
  ) => (
    <div
      className={`rag-setting ${workspaceMode && !overridden(key) ? "inherited" : ""}`}
      key={key}
    >
      {workspaceMode && (
        <div className="rag-override">
          <label>
            {label} 설정 출처
            <select
              aria-label={`${label} 설정 출처`}
              disabled={!!busy}
              value={overridden(key) ? "workspace" : "service"}
              onChange={(e) => {
                setPatch((old) => ({
                  ...old,
                  [key]: e.target.value === "service" ? null : current(key),
                }));
                setDiagnostic(null);
              }}
            >
              <option value="service">서비스 설정 상속</option>
              <option value="workspace">워크스페이스 개별 설정</option>
            </select>
          </label>
        </div>
      )}
      <Field label={label} hint={hint}>
        {control}
      </Field>
      {patch[key] === null && snapshot?.overrides.includes(key) && (
        <p className="rag-pending">
          저장하면 개별 설정이 해제됩니다. 저장 후 서비스의 실제 값을 다시
          불러옵니다.
        </p>
      )}
    </div>
  );
  const text = (key: Key, label: string, hint?: string, maxLength = 4096) =>
    field(
      key,
      label,
      <input
        value={String(current(key))}
        onChange={(e) => update(key, e.target.value)}
        disabled={!writable(key)}
        maxLength={maxLength}
        autoComplete="off"
        spellCheck={false}
      />,
      hint,
    );
  const select = (
    key: Key,
    label: string,
    options: [string, string][],
    hint?: string,
  ) =>
    field(
      key,
      label,
      <select
        disabled={!writable(key)}
        value={String(current(key))}
        onChange={(e) =>
          update(
            key,
            typeof defaults[key] === "boolean"
              ? e.target.value === "true"
              : e.target.value,
          )
        }
      >
        {options.map(([value, name]) => (
          <option key={value} value={value}>
            {name}
          </option>
        ))}
      </select>,
      hint,
    );
  const number = (
    key: Key,
    label: string,
    min: number,
    max: number,
    hint?: string,
  ) =>
    field(
      key,
      label,
      <input
        type="number"
        min={min}
        max={max}
        step={1}
        required
        value={String(current(key))}
        disabled={!writable(key)}
        onChange={(e) =>
          update(key, e.target.value === "" ? "" : Number(e.target.value))
        }
      />,
      hint,
    );
  const secret = (key: Key, label: string) =>
    field(
      key,
      label,
      <div className="rag-secret">
        <input
          type="password"
          autoComplete="new-password"
          spellCheck={false}
          maxLength={16384}
          disabled={!writable(key) || clears.includes(key)}
          value={String(patch[key] ?? "")}
          onChange={(e) => update(key, e.target.value)}
          placeholder={
            snapshot?.data[key + "_configured"]
              ? "저장된 키 유지 · 변경할 때만 입력"
              : "인증이 필요한 경우 입력"
          }
        />
        <span className="rag-secret-status">
          <KeyRound size={15} />
          저장된 설정:{" "}
          {snapshot?.data[key + "_configured"]
            ? "키 등록됨 · 내용은 표시하지 않음"
            : "등록된 키 없음"}
        </span>
        {!workspaceMode && snapshot?.data[key + "_configured"] === true && (
          <label className="rag-checkbox">
            <input
              type="checkbox"
              disabled={!!busy}
              checked={clears.includes(key)}
              onChange={(e) => {
                setClears((old) =>
                  e.target.checked
                    ? [...old, key]
                    : old.filter((v) => v !== key),
                );
                setPatch((old) => {
                  const next = { ...old };
                  delete next[key];
                  return next;
                });
                setDiagnostic(null);
              }}
            />
            저장할 때 {label} 삭제
          </label>
        )}
      </div>,
      workspaceMode
        ? "빈 입력은 기존 키를 유지합니다. 개별 키를 해제하려면 설정 출처를 서비스 상속으로 변경하세요. 다른 API 주소에는 서비스 키가 상속되지 않습니다."
        : "빈 입력은 기존 키를 유지합니다. 키는 서버에서 암호화하며 다시 표시하지 않습니다.",
    );

  const save = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!snapshot || busy || !dirty) return;
    setBusy("save");
    setError(null);
    setDiagnostic(null);
    const request = ++generation.current;
    try {
      const data: Record<string, unknown> = { ...patch };
      for (const key of secretKeys) if (data[key] === "") delete data[key];
      for (const key of clears) data[key + "_clear"] = true;
      if (!Object.keys(data).length)
        throw new Error(
          "변경할 값을 입력하세요. 빈 비밀키는 기존 값을 유지합니다.",
        );
      await api(
        settingsPath,
        "PUT",
        workspaceMode ? { version: snapshot.version, data } : data,
      );
      const next = await fetchSnapshot();
      if (!mounted.current || generation.current !== request) return;
      setSnapshot(next);
      setPatch({});
      setClears([]);
      notify("검색 AI 설정을 저장했습니다. 연결 진단으로 확인하세요.");
    } catch (e) {
      if (mounted.current && generation.current === request) setError(e);
    } finally {
      if (mounted.current && generation.current === request) setBusy(null);
    }
  };
  const test = async () => {
    if (busy || dirty) return;
    const request = ++generation.current;
    setBusy("test");
    setError(null);
    setDiagnostic(null);
    try {
      const result = await api<Diagnostic>(testPath, "POST", {});
      if (mounted.current && generation.current === request)
        setDiagnostic(result);
    } catch (e) {
      if (mounted.current && generation.current === request) setError(e);
    } finally {
      if (mounted.current && generation.current === request) setBusy(null);
    }
  };
  const hasHTTP = ["rag_embedding_base_url", "rag_rerank_base_url"].some(
    (key) => /^http:/i.test(String(current(key as Key))),
  );
  const history = workspaceMode
    ? "/app/workspace-settings"
    : "/admin/settings?tab=history";
  return (
    <div className="page rag-settings-page">
      <PageHeading
        eyebrow={workspaceMode ? "WORKSPACE SEARCH AI" : "SERVICE SEARCH AI"}
        title={workspaceMode ? "워크스페이스 검색 AI" : "검색 AI 설정"}
        description={
          workspaceMode
            ? `${workspace?.name}의 검색·임베딩 설정입니다. 선택한 항목만 서비스 기본값 대신 사용합니다.`
            : "문서 검색에 사용할 임베딩과 재정렬 모델, 검색 범위 및 안전한 연결 정책을 관리합니다."
        }
        actions={
          <Link className="button secondary" to={history}>
            <History size={17} />
            설정 변경 이력
          </Link>
        }
      />
      <ErrorBox error={error} />
      {!snapshot ? (
        busy ? (
          <Loading />
        ) : (
          <Empty
            title="설정을 불러오지 못했습니다"
            text="권한과 서버 연결을 확인한 뒤 다시 시도하세요."
            action={
              <Button onClick={load}>
                <RefreshCw size={17} />
                다시 불러오기
              </Button>
            }
          />
        )
      ) : (
        <form onSubmit={save}>
          <div className="rag-layout">
            <aside className="panel padded rag-overview">
              <Search size={32} />
              <h2>검색 연결 정책</h2>
              <Badge>
                {snapshot.data.rag_enabled
                  ? "저장된 상태: 활성"
                  : "저장된 상태: 비활성"}
              </Badge>
              {workspaceMode && (
                <p>
                  개별 설정 {snapshot.overrides.length}개 · 버전{" "}
                  {snapshot.version}
                </p>
              )}
              <p>
                이 화면은 설정과 연결 진단을 관리합니다. 설정 저장만으로 색인
                완료나 검색 품질을 보장하지 않습니다.
              </p>
              <div className="notice">
                <ShieldCheck size={19} />
                <span>
                  연결 진단에는 고정된 테스트 문구만 전송합니다. 실제 문서나
                  비밀키는 응답에 포함하지 않습니다.
                </span>
              </div>
              <p>
                임베딩 공급자·모델·차원을 바꾸면 기존 색인과 호환되지 않을 수
                있습니다. 변경 후 색인 상태를 확인하세요.
              </p>
            </aside>
            <div className="rag-form-sections">
              <section className="panel padded">
                <div className="settings-section-title">
                  <Search size={24} />
                  <div>
                    <h2>검색 방식</h2>
                    <p>키워드와 의미 검색의 사용 정책을 설정합니다.</p>
                  </div>
                </div>
                {select("rag_enabled", "검색 AI 사용", [
                  ["false", "사용 안 함"],
                  ["true", "사용"],
                ])}
                <div className="rag-grid">
                  {select("rag_search_mode", "기본 검색 방식", [
                    ["hybrid", "하이브리드 · 키워드 + 의미"],
                    ["keyword", "키워드 검색"],
                    ["semantic", "의미 검색"],
                  ])}
                  {select(
                    "rag_backend",
                    "벡터 저장 방식",
                    [
                      ["array", "PostgreSQL 배열 · 확장 불필요"],
                      ["pgvector", "pgvector · 선택 확장"],
                    ],
                    "폐쇄망 기본 설치는 배열 저장을 사용할 수 있습니다. pgvector를 선택하려면 PostgreSQL에 vector 확장을 별도로 설치해야 합니다.",
                  )}
                </div>
                <div className="rag-grid">
                  {number("rag_top_k", "최종 출처 수", 1, 20, "1~20개")}
                  {number(
                    "rag_candidates",
                    "검색 후보 수",
                    10,
                    200,
                    "10~200개 · 최종 출처 수 이상",
                  )}
                  {number(
                    "rag_scan_limit",
                    "검색 검사 한도",
                    100,
                    50000,
                    "100~50,000개 · 큰 값은 검색 비용을 높일 수 있습니다.",
                  )}
                </div>
              </section>
              <section className="panel padded">
                <div className="settings-section-title">
                  <Network size={24} />
                  <div>
                    <h2>임베딩 연결</h2>
                    <p>OpenAI 호환 임베딩 API를 사용합니다.</p>
                  </div>
                </div>
                {text(
                  "rag_embedding_base_url",
                  "임베딩 API 주소",
                  "예: https://llm.example.internal/v1 · 문서가 전송될 승인된 내부 API 주소를 입력하세요.",
                )}
                <div className="rag-grid">
                  {text(
                    "rag_embedding_model",
                    "임베딩 모델",
                    "공급자가 지원하는 정확한 모델 이름",
                    256,
                  )}
                  {number(
                    "rag_embedding_dimensions",
                    "임베딩 차원",
                    0,
                    8192,
                    "0은 모델 기본 차원 · 1~8,192는 고정 차원",
                  )}
                </div>
                {secret("rag_embedding_api_key", "임베딩 API 키")}
              </section>
              <section className="panel padded">
                <div className="settings-section-title">
                  <Database size={24} />
                  <div>
                    <h2>검색 결과 재정렬</h2>
                    <p>검색 후보의 관련도를 재평가할 별도 모델을 설정합니다.</p>
                  </div>
                </div>
                {select(
                  "rag_rerank_enabled",
                  "재정렬 사용",
                  [
                    ["false", "사용 안 함"],
                    ["true", "사용"],
                  ],
                  "검색 AI가 활성화된 경우에 사용합니다. 아래 연결 진단은 임베딩 연결만 확인합니다.",
                )}
                {text(
                  "rag_rerank_base_url",
                  "재정렬 API 주소",
                  "예: https://rerank.example.internal/v1",
                )}
                {text("rag_rerank_model", "재정렬 모델", undefined, 256)}
                {secret("rag_rerank_api_key", "재정렬 API 키")}
              </section>
              <section className="panel padded">
                <div className="settings-section-title">
                  <ShieldCheck size={24} />
                  <div>
                    <h2>연결 보안</h2>
                    <p>오프라인망에서도 인증서 검증을 유지합니다.</p>
                  </div>
                </div>
                {select(
                  "rag_allow_http",
                  "내부 HTTP 연결 허용",
                  [
                    ["false", "허용 안 함 · HTTPS 사용"],
                    ["true", "위험을 확인하고 HTTP 허용"],
                  ],
                  "HTTP는 API 키와 전송 내용을 암호화하지 않습니다. 신뢰할 수 있는 격리된 내부망에서만 명시적으로 허용하세요.",
                )}
                {(hasHTTP || current("rag_allow_http")) && (
                  <div className="notice warning">
                    <TriangleAlert size={20} />
                    <span>
                      HTTP를 사용하면 문서 내용과 API 키가 평문으로 전송될 수
                      있습니다. 폐쇄망도 HTTPS 사용을 권장합니다.
                    </span>
                  </div>
                )}
                {field(
                  "rag_ca_pem",
                  "내부 CA 인증서",
                  <textarea
                    rows={5}
                    maxLength={65536}
                    spellCheck={false}
                    disabled={!writable("rag_ca_pem")}
                    value={String(current("rag_ca_pem"))}
                    onChange={(e) => update("rag_ca_pem", e.target.value)}
                    placeholder="-----BEGIN CERTIFICATE-----"
                  />,
                  "선택 · PEM 형식, 최대 64KiB. 비워 두면 시스템의 신뢰할 수 있는 인증서를 사용합니다. 개인키는 입력하지 마세요.",
                )}
              </section>
              <section className="panel padded rag-diagnostic">
                <h2>
                  <TestTube2 size={22} />
                  저장된 설정 연결 진단
                </h2>
                <p>
                  고정 문구의 임베딩 응답과 차원을 확인하고, PostgreSQL의
                  pgvector 설치 여부를 조회합니다. 재정렬 모델이나 실제 문서
                  색인은 이 진단에 포함되지 않습니다.
                </p>
                <Button
                  type="button"
                  variant="secondary"
                  disabled={
                    !!busy ||
                    dirty ||
                    !snapshot.data.rag_embedding_base_url ||
                    !snapshot.data.rag_embedding_model
                  }
                  onClick={test}
                >
                  <TestTube2 size={18} />
                  {busy === "test" ? "진단 중…" : "임베딩 연결 진단"}
                </Button>
                {dirty && (
                  <p className="rag-pending">
                    변경사항을 먼저 저장해야 진단할 수 있습니다.
                  </p>
                )}
                {!snapshot.data.rag_embedding_base_url && !dirty && (
                  <p className="muted">
                    임베딩 API 주소와 모델을 저장하면 진단할 수 있습니다.
                  </p>
                )}
                {diagnostic && (
                  <div role="status" className="rag-test-result">
                    <Badge tone={diagnostic.ok ? "green" : "red"}>
                      {diagnostic.ok ? "임베딩 연결 확인" : "진단 실패"}
                    </Badge>
                    <dl>
                      <div>
                        <dt>응답 차원</dt>
                        <dd>{diagnostic.dimensions.toLocaleString("ko-KR")}</dd>
                      </div>
                      <div>
                        <dt>pgvector</dt>
                        <dd>
                          {diagnostic.pgvector_installed
                            ? "설치됨"
                            : "미설치 · 기본 배열 저장 사용 가능"}
                        </dd>
                      </div>
                      <div>
                        <dt>전송 내용</dt>
                        <dd>{diagnostic.content_sent}</dd>
                      </div>
                    </dl>
                    {snapshot.data.rag_backend === "pgvector" &&
                      !diagnostic.pgvector_installed && (
                        <div className="notice warning">
                          <TriangleAlert size={18} />
                          <span>
                            현재 pgvector를 선택했지만 확장이 설치되지
                            않았습니다. 배열 저장으로 변경하거나 DB 운영자에게
                            vector 확장 설치를 요청하세요.
                          </span>
                        </div>
                      )}
                  </div>
                )}
              </section>
              <div className="settings-save rag-save">
                <span aria-live="polite">
                  {dirty
                    ? "저장하지 않은 변경사항이 있습니다."
                    : "모든 설정이 저장되어 있습니다."}
                </span>
                <div>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={!!busy}
                    onClick={() => (dirty ? setReset(true) : void load())}
                  >
                    <RefreshCw size={17} />
                    다시 불러오기
                  </Button>
                  <Button disabled={!!busy || !dirty}>
                    {busy === "save" ? "저장 중…" : "검색 AI 설정 저장"}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </form>
      )}
      <Modal
        open={reset}
        onOpenChange={setReset}
        title="변경사항을 버리고 다시 불러올까요?"
        description="입력한 API 키를 포함해 저장하지 않은 변경사항을 버리고 서버의 최신 설정을 불러옵니다."
      >
        <div className="modal-actions">
          <Button variant="secondary" onClick={() => setReset(false)}>
            취소
          </Button>
          <Button
            onClick={() => {
              setReset(false);
              void load();
            }}
          >
            변경사항 버리기
          </Button>
        </div>
      </Modal>
    </div>
  );
}
