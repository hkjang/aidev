import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { ArrowLeft, Plus, Save, Trash2, Play, Download } from "lucide-react";
import { api, downloadText } from "./api";
import { useApp } from "./context";
import { Button, Empty, ErrorBox, Field, Loading, PageHeading } from "./ui";
import "./search-operations.css";

type Entry = { id: string; canonical: string; aliases: string[] };
type Dictionary = {
  revision: number;
  entries: Entry[];
  protection_changed?: boolean;
};
type Evaluation = { query: string; expected_document_ids: string[] };
type Interpretation = {
  normalized: string;
  parts: string[][];
  matched_terms: string[];
  strategy: string;
  warnings: string[];
};
type Diagnostic = {
  interpretation: Interpretation;
  elapsed_ms: number;
  matched_in_page: number;
  has_more: boolean;
  scope: string;
  projection: Record<string, number>;
  plan: { depth: number; node: string; "Index Name"?: string }[];
  plan_executed: boolean;
  plan_warning?: string;
  planning_ms?: number;
  execution_ms?: number;
};
type EvaluationResult = {
  top_k: number;
  recall_at_k: number;
  mrr_at_k: number;
  scope: string;
  cases: {
    query: string;
    expected: { document_id: string; rank: number }[];
    recall_at_k: number;
    reciprocal_rank: number;
    elapsed_ms: number;
  }[];
};

export default function SearchOperationsPage() {
  const { user, workspace, documents } = useApp();
  const scope = `${user.id}:${workspace?.id}`,
    current = useRef(scope);
  current.current = scope;
  const [loaded, setLoaded] = useState(""),
    [revision, setRevision] = useState(0),
    [entries, setEntries] = useState<Entry[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(""),
    [notice, setNotice] = useState(""),
    [refresh, setRefresh] = useState(0);
  const [tab, setTab] = useState<"dictionary" | "diagnostic" | "evaluation">(
      "dictionary",
    ),
    [confirmed, setConfirmed] = useState(false),
    [dirty, setDirty] = useState(false),
    [query, setQuery] = useState(""),
    [plan, setPlan] = useState(true),
    [diagnostic, setDiagnostic] = useState<Diagnostic | null>(null),
    [evaluation, setEvaluation] = useState<EvaluationResult | null>(null),
    [cases, setCases] = useState<Evaluation[]>([
      { query: "", expected_document_ids: [] },
    ]),
    [topK, setTopK] = useState(5);
  const allowed = !!workspace && ["owner", "admin"].includes(workspace.role);
  useEffect(() => {
    setLoaded("");
    setEntries([]);
    setError(null);
    setNotice("");
    setBusy("");
    setConfirmed(false);
    setDirty(false);
    setDiagnostic(null);
    setEvaluation(null);
    setCases([{ query: "", expected_document_ids: [] }]);
    setQuery("");
    if (!workspace || !allowed) return;
    let stopped = false;
    void api<Dictionary>(`/workspaces/${workspace.id}/search-dictionary`)
      .then((data) => {
        if (!stopped && current.current === scope) {
          setEntries(data.entries);
          setRevision(data.revision);
          setLoaded(scope);
        }
      })
      .catch((e) => {
        if (!stopped && current.current === scope) {
          setError(e);
          setLoaded(scope);
        }
      });
    return () => {
      stopped = true;
    };
  }, [scope, refresh, allowed]);
  useEffect(() => {
    if (!dirty) return;
    const before = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", before);
    return () => window.removeEventListener("beforeunload", before);
  }, [dirty]);
  async function action(name: string, run: () => Promise<void>) {
    if (busy) return;
    const started = scope;
    setBusy(name);
    setError(null);
    setNotice("");
    try {
      await run();
    } catch (e) {
      if (current.current === started) setError(e);
    } finally {
      if (current.current === started) setBusy("");
    }
  }
  function update(id: string, value: Partial<Entry>) {
    setEntries((old) =>
      old.map((entry) => (entry.id === id ? { ...entry, ...value } : entry)),
    );
    setDirty(true);
    setConfirmed(false);
  }
  async function save() {
    if (!workspace || !confirmed) return;
    const started = scope;
    await action("save", async () => {
      const result = await api<Dictionary>(
        `/workspaces/${workspace.id}/search-dictionary`,
        "PUT",
        {
          revision,
          entries: entries.map((entry) => ({
            ...entry,
            aliases: entry.aliases.filter((alias) => alias.trim()),
          })),
          confirm_shared: true,
        },
      );
      if (current.current !== started) return;
      setRevision(result.revision);
      setEntries(result.entries);
      setConfirmed(false);
      setDirty(false);
      setNotice(
        result.protection_changed
          ? "정보 보호 정책으로 정제된 사전을 저장했습니다. 결과를 확인하세요."
          : "사전을 저장했습니다. 다음 검색부터 새 용어 해석을 적용합니다.",
      );
    });
  }
  async function diagnose() {
    if (!workspace) return;
    const started = scope;
    setDiagnostic(null);
    await action("diagnostic", async () => {
      const result = await api<Diagnostic>(
        `/workspaces/${workspace.id}/search-diagnostics`,
        "POST",
        {
          filters: { q: query, group: "document" },
          run_plan: plan,
          confirm: true,
        },
      );
      if (current.current === started) setDiagnostic(result);
    });
  }
  async function evaluate() {
    if (!workspace) return;
    const started = scope;
    setEvaluation(null);
    await action("evaluation", async () => {
      const result = await api<EvaluationResult>(
        `/workspaces/${workspace.id}/search-evaluation`,
        "POST",
        { cases, top_k: topK, confirm: true },
      );
      if (current.current === started) setEvaluation(result);
    });
  }
  async function sample() {
    if (!workspace) return;
    const started = scope;
    await action("sample", async () => {
      const result = await api<Record<string, unknown>>(
        `/workspaces/${workspace.id}/search-evaluation/sample`,
      );
      if (current.current === started)
        downloadText(
          "madi-한국어-검색-평가셋.json",
          JSON.stringify(result, null, 2),
          "application/json",
        );
    });
  }
  if (!allowed)
    return (
      <Empty
        title="검색 운영 권한이 필요합니다"
        text="현재 워크스페이스의 소유자 또는 관리자에게 문의하세요."
      />
    );
  return (
    <section className="search-operations">
      <PageHeading
        eyebrow="워크스페이스 관리"
        title="검색 운영"
        description="조직 용어를 맞추고 현재 권한의 검색을 검증합니다. 검색어와 문서를 외부 AI로 전송하지 않습니다."
        actions={
          <>
            <Link className="button" to="/app/search/generations">
              벡터 색인 세대
            </Link>
            <Link className="button" to="/app/search">
              <ArrowLeft size={17} />
              검색으로 돌아가기
            </Link>
          </>
        }
      />
      <div
        className="search-operations-tabs"
        role="tablist"
        aria-label="검색 운영 메뉴"
      >
        {(
          [
            ["dictionary", "조직 용어 사전"],
            ["diagnostic", "검색 진단"],
            ["evaluation", "한국어 평가"],
          ] as const
        ).map(([key, label]) => (
          <button
            key={key}
            role="tab"
            aria-selected={tab === key}
            onClick={() => {
              setTab(key);
              setError(null);
            }}
          >
            {label}
          </button>
        ))}
      </div>
      <ErrorBox error={error} />
      {notice && (
        <p className="search-operation-notice" role="status">
          {notice}
        </p>
      )}
      {loaded !== scope ? (
        <Loading />
      ) : (
        <>
          {tab === "dictionary" && (
            <div role="tabpanel" className="panel padded">
              <div className="search-operation-heading">
                <div>
                  <h2>조직 용어 사전</h2>
                  <p className="muted">
                    {entries.length}/300개 · 사전 버전 {revision} ·{" "}
                    {dirty
                      ? "아직 저장하지 않은 변경이 있습니다."
                      : "서버 설정과 같습니다."}
                  </p>
                </div>
                <Button
                  disabled={!!busy || entries.length >= 300}
                  onClick={() => {
                    setEntries((old) => [
                      ...old,
                      { id: crypto.randomUUID(), canonical: "", aliases: [] },
                    ]);
                    setDirty(true);
                    setConfirmed(false);
                  }}
                >
                  <Plus size={17} />
                  용어 추가
                </Button>
              </div>
              <p>
                예: 표준 용어 <strong>Kubernetes</strong>에{" "}
                <strong>k8s, 쿠버네티스</strong>를 연결합니다. 공백·대소문자를
                정규화하며 원문은 바꾸지 않습니다. 사전은 워크스페이스 구성원이
                검색할 때 사용하는 공유 설정입니다. 비밀이나 비공개 문서의
                내용을 입력하지 마세요.
              </p>
              <div className="search-dictionary-list">
                {entries.map((entry, index) => (
                  <div className="search-dictionary-row" key={entry.id}>
                    <Field label={`표준 용어 ${index + 1}`}>
                      <input
                        value={entry.canonical}
                        disabled={!!busy}
                        maxLength={200}
                        onChange={(e) =>
                          update(entry.id, { canonical: e.target.value })
                        }
                      />
                    </Field>
                    <Field label={`별칭 ${index + 1} · 쉼표로 구분`}>
                      <input
                        value={entry.aliases.join(", ")}
                        disabled={!!busy}
                        onChange={(e) =>
                          update(entry.id, {
                            aliases: e.target.value
                              .split(",")
                              .map((v) => v.trim()),
                          })
                        }
                        placeholder="k8s, 쿠버네티스"
                      />
                    </Field>
                    <Button
                      aria-label={`용어 ${index + 1} 삭제`}
                      disabled={!!busy}
                      onClick={() => {
                        setEntries((old) =>
                          old.filter((v) => v.id !== entry.id),
                        );
                        setDirty(true);
                        setConfirmed(false);
                      }}
                    >
                      <Trash2 size={17} />
                    </Button>
                  </div>
                ))}
              </div>
              {!entries.length && (
                <p className="muted">
                  등록한 용어가 없습니다. 기본 공백·대소문자 검색은 계속 사용할
                  수 있습니다.
                </p>
              )}
              <label className="search-operation-check">
                <input
                  type="checkbox"
                  checked={confirmed}
                  disabled={!!busy}
                  onChange={(e) => setConfirmed(e.target.checked)}
                />
                이 용어를 워크스페이스 구성원의 검색에 공유하는 것을
                확인했습니다.
              </label>
              <div className="modal-actions">
                <Button
                  disabled={!!busy}
                  onClick={() => {
                    if (
                      !dirty ||
                      window.confirm(
                        "저장하지 않은 사전 변경을 버리고 다시 불러올까요?",
                      )
                    )
                      setRefresh((v) => v + 1);
                  }}
                >
                  다시 불러오기
                </Button>
                <Button
                  variant="primary"
                  disabled={
                    !!busy ||
                    !confirmed ||
                    entries.some((e) => !e.canonical.trim())
                  }
                  onClick={() => void save()}
                >
                  <Save size={17} />
                  {busy === "save" ? "저장 중…" : "사전 저장"}
                </Button>
              </div>
            </div>
          )}
          {tab === "diagnostic" && (
            <div role="tabpanel" className="panel padded">
              <h2>현재 권한으로 검색 진단</h2>
              <p>
                검색 실행과 선택적인 읽기 전용 실행 계획을 확인합니다. SQL
                상수·비공개 문서 수는 표시하지 않으며 검색어를 자동 보관하지
                않습니다.
              </p>
              <Field label="진단할 검색어">
                <input
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  maxLength={500}
                  placeholder="예: k8s 장애"
                  disabled={!!busy}
                />
              </Field>
              <label className="search-operation-check">
                <input
                  type="checkbox"
                  checked={plan}
                  onChange={(e) => setPlan(e.target.checked)}
                  disabled={!!busy}
                />
                실제 실행 계획도 확인합니다. 별도 읽기 실행은 최대 3초로
                제한됩니다.
              </label>
              <Button
                variant="primary"
                disabled={!!busy}
                onClick={() => void diagnose()}
              >
                <Play size={17} />
                {busy === "diagnostic" ? "진단 중…" : "현재 권한으로 진단 실행"}
              </Button>
              {diagnostic && (
                <div className="search-operation-result" aria-live="polite">
                  <h3>진단 결과</h3>
                  <dl>
                    <dt>검색 실행</dt>
                    <dd>{diagnostic.elapsed_ms.toFixed(1)} ms</dd>
                    <dt>현재 페이지</dt>
                    <dd>
                      {diagnostic.matched_in_page}개
                      {diagnostic.has_more ? " · 다음 결과 있음" : ""}
                    </dd>
                    <dt>정규화 색인</dt>
                    <dd>
                      {diagnostic.projection.normalized_current}/
                      {diagnostic.projection.readable_documents}개 문서
                    </dd>
                    <dt>색인 대기</dt>
                    <dd>{diagnostic.projection.normalized_pending}개</dd>
                    <dt>정확 확인용 사전 필터 생략</dt>
                    <dd>{diagnostic.projection.incomplete_gram_prefilter}개</dd>
                  </dl>
                  <p>{diagnostic.scope}</p>
                  <details open>
                    <summary>검색어 해석</summary>
                    <p>
                      정규화:{" "}
                      {diagnostic.interpretation.normalized || "빈 검색어"}
                    </p>
                    <p>
                      연결된 조직 용어:{" "}
                      {diagnostic.interpretation.matched_terms.join(", ") ||
                        "없음"}
                    </p>
                    <p>
                      검색 부분:{" "}
                      {diagnostic.interpretation.parts
                        .map((part) => `(${part.join(" 또는 ")})`)
                        .join(" 그리고 ") || "기존 검색 문법"}
                    </p>
                    {diagnostic.interpretation.warnings.map((value) => (
                      <p key={value}>{value}</p>
                    ))}
                  </details>
                  {diagnostic.plan_executed && (
                    <details open>
                      <summary>
                        실제 실행 계획 · {diagnostic.execution_ms?.toFixed(1)}{" "}
                        ms
                      </summary>
                      <ol className="search-plan">
                        {diagnostic.plan.map((node, i) => (
                          <li
                            key={i}
                            style={{
                              paddingInlineStart: Math.min(node.depth, 8) * 12,
                            }}
                          >
                            <span>{node.node}</span>
                            {node["Index Name"] && (
                              <code>{node["Index Name"]}</code>
                            )}
                          </li>
                        ))}
                      </ol>
                      <p className="muted">
                        이름과 실행 경로만 표시합니다. DB 전체 통계나 미열람
                        결과 수는 제공하지 않습니다.
                      </p>
                    </details>
                  )}
                  {diagnostic.plan_warning && (
                    <p role="status">{diagnostic.plan_warning}</p>
                  )}
                </div>
              )}
            </div>
          )}
          {tab === "evaluation" && (
            <div role="tabpanel" className="panel padded">
              <div className="search-operation-heading">
                <h2>실제 문서로 평가</h2>
                <Button disabled={!!busy} onClick={() => void sample()}>
                  <Download size={17} />
                  합성 평가셋 다운로드
                </Button>
              </div>
              <p>
                검색어와 기대하는 문서를 직접 지정해 Recall@k와 MRR@k를
                계산합니다. 평가 입력·결과는 이 화면에서만 다루며 자동 보관하지
                않습니다. 합성 평가셋은 실제 고객 자료가 아닌 회귀 검증용
                예제입니다.
              </p>
              <Field label="상위 결과 수 k">
                <select
                  value={topK}
                  disabled={!!busy}
                  onChange={(e) => setTopK(Number(e.target.value))}
                >
                  {[1, 3, 5, 10, 20].map((n) => (
                    <option key={n} value={n}>
                      {n}개
                    </option>
                  ))}
                </select>
              </Field>
              <div className="search-evaluation-list">
                {cases.map((item, index) => (
                  <div className="search-evaluation-row" key={index}>
                    <Field label={`평가 검색어 ${index + 1}`}>
                      <input
                        disabled={!!busy}
                        value={item.query}
                        maxLength={500}
                        onChange={(e) =>
                          setCases((old) =>
                            old.map((v, i) =>
                              i === index ? { ...v, query: e.target.value } : v,
                            ),
                          )
                        }
                      />
                    </Field>
                    <Field
                      label={`정답 문서 ${index + 1} · 여러 개는 Ctrl/⌘로 선택`}
                    >
                      <select
                        multiple
                        size={3}
                        disabled={!!busy}
                        value={item.expected_document_ids}
                        onChange={(e) => {
                          const ids = [...e.target.selectedOptions].map(
                            (o) => o.value,
                          );
                          setCases((old) =>
                            old.map((v, i) =>
                              i === index
                                ? { ...v, expected_document_ids: ids }
                                : v,
                            ),
                          );
                        }}
                      >
                        {documents.map((doc) => (
                          <option key={doc.id} value={doc.id}>
                            {doc.title}
                          </option>
                        ))}
                      </select>
                    </Field>
                    <Button
                      aria-label={`평가 ${index + 1} 삭제`}
                      disabled={!!busy || cases.length === 1}
                      onClick={() =>
                        setCases((old) => old.filter((_, i) => i !== index))
                      }
                    >
                      <Trash2 size={17} />
                    </Button>
                  </div>
                ))}
              </div>
              <div className="modal-actions">
                <Button
                  disabled={!!busy || cases.length >= 30}
                  onClick={() =>
                    setCases((old) => [
                      ...old,
                      { query: "", expected_document_ids: [] },
                    ])
                  }
                >
                  <Plus size={17} />
                  평가 추가
                </Button>
                <Button
                  variant="primary"
                  disabled={
                    !!busy ||
                    cases.some(
                      (v) =>
                        !v.query.trim() ||
                        !v.expected_document_ids.length ||
                        v.expected_document_ids.length > 20,
                    )
                  }
                  onClick={() => void evaluate()}
                >
                  <Play size={17} />
                  {busy === "evaluation"
                    ? "평가 중…"
                    : "제공한 정답으로 평가 실행"}
                </Button>
              </div>
              {evaluation && (
                <div className="search-operation-result">
                  <h3>
                    Recall@{evaluation.top_k}:{" "}
                    {(evaluation.recall_at_k * 100).toFixed(1)}% · MRR@
                    {evaluation.top_k}: {evaluation.mrr_at_k.toFixed(3)}
                  </h3>
                  <p>{evaluation.scope}</p>
                  <div className="table-scroll">
                    <table>
                      <thead>
                        <tr>
                          <th>검색어</th>
                          <th>정답 순위</th>
                          <th>Recall</th>
                          <th>실행 시간</th>
                        </tr>
                      </thead>
                      <tbody>
                        {evaluation.cases.map((item, index) => (
                          <tr key={index}>
                            <td>{item.query}</td>
                            <td>
                              {item.expected
                                .map(
                                  (v) =>
                                    `${documents.find((d) => d.id === v.document_id)?.title || "제공한 문서"}: ${v.rank ? `${v.rank}위` : "상위 결과에 없음"}`,
                                )
                                .join(" · ")}
                            </td>
                            <td>{(item.recall_at_k * 100).toFixed(1)}%</td>
                            <td>{item.elapsed_ms.toFixed(1)} ms</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </section>
  );
}
