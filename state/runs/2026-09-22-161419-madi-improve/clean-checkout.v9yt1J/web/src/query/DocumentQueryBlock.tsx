import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Play, RefreshCw, Square } from "lucide-react";
import { ApiError } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field } from "../ui";
import {
  queryLabels,
  querySourceLabels,
  type QueryMetadata,
  type QueryResult,
} from "./types";
import "./query.css";
const DocumentPreview = lazy(() => import("../review/DocumentPreview"));
// Serialize only in-flight work, never cache definitions, result rows or tokens.
// A document may contain sixteen fences; mounting/checking them must not race
// all four server execution slots at once.
const requestQueue: (() => void)[] = [];
let activeRequests = 0;
function advanceQueue() {
  while (activeRequests < 2 && requestQueue.length) {
    activeRequests++;
    requestQueue.shift()!();
  }
}

async function request<T>(
  path: string,
  signal: AbortSignal,
  body?: unknown,
): Promise<T> {
  await new Promise<void>((resolve) => {
    requestQueue.push(resolve);
    advanceQueue();
  });
  try {
    signal.throwIfAborted();
    const response = await fetch(`/api/v1${path}`, {
      credentials: "same-origin",
      method: body === undefined ? "GET" : "POST",
      headers: { "X-Madi-Request": "1", "Content-Type": "application/json" },
      signal,
      ...(body === undefined ? {} : { body: JSON.stringify(body) }),
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok)
      throw new ApiError(
        data.error || "조회 결과를 확인할 수 없습니다.",
        response.status,
      );
    return data as T;
  } finally {
    activeRequests--;
    advanceQueue();
  }
}
export default function DocumentQueryBlock({
  source,
  documentId,
}: {
  source: string;
  documentId?: string;
}) {
  const app = useApp();
  if (!documentId || !app?.user)
    return (
      <section className="document-query inert">
        <strong>선언형 조회 정의</strong>
        <p>
          동적 결과는 정본 문서의 읽기 화면에서 직접 실행할 때만 표시합니다.
        </p>
        <pre>
          <code>{source}</code>
        </pre>
      </section>
    );
  return (
    <QueryBlock
      key={`${app.user.id}:${app.workspace?.id}:${documentId}:${source}`}
      source={source}
      documentId={documentId}
    />
  );
}
function QueryBlock({
  source,
  documentId,
}: {
  source: string;
  documentId: string;
}) {
  const { user, workspace } = useApp();
  const scope = `${user.id}:${workspace?.id}:${documentId}:${source}`,
    current = useRef(scope);
  current.current = scope;
  const [meta, setMeta] = useState<QueryMetadata | null>(null),
    [loadAttempt, setLoadAttempt] = useState(0),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [result, setResult] = useState<QueryResult | null>(null),
    [values, setValues] = useState<Record<string, string>>({}),
    [preview, setPreview] = useState<QueryResult["rows"][number] | null>(null);
  const controller = useRef<AbortController | null>(null),
    lifetime = useRef<AbortController | null>(null),
    sequence = useRef(0),
    currentResult = useRef<QueryResult | null>(null);
  currentResult.current = result;
  const base = `/documents/${documentId}/queries`,
    fence = meta?.queries.find((q) => q.source.trim() === source.trim()),
    definition = fence?.definition;
  useEffect(() => {
    const life = new AbortController();
    lifetime.current = life;
    const fresh = () => !life.signal.aborted && current.current === scope;
    setMeta(null);
    setResult(null);
    setPreview(null);
    setError(null);
    const load = async () => {
      try {
        const data = await request<QueryMetadata>(base, life.signal);
        if (!fresh()) return;
        setMeta(data);
        const found = data.queries.find(
          (q) => q.source.trim() === source.trim(),
        );
        const defaults: Record<string, string> = {};
        for (const [key, p] of Object.entries(
          found?.definition?.parameters || {},
        ))
          if (p.default !== undefined) defaults[key] = String(p.default);
        setValues(defaults);
      } catch (e) {
        if (fresh()) setError(e);
      }
    };
    void load();
    let checking = false;
    const check = async () => {
      if (!fresh() || checking || !currentResult.current) return;
      if (document.hidden) {
        setResult(null);
        setPreview(null);
        return;
      }
      const saved = currentResult.current;
      checking = true;
      try {
        await request(`${base}/check`, life.signal, {
          validation_token: saved.validation_token,
        });
      } catch (e) {
        if (fresh() && currentResult.current === saved) {
          setResult(null);
          setPreview(null);
          setError(e);
        }
      } finally {
        checking = false;
      }
    };
    const timer = window.setInterval(check, 3000);
    const visibility = () => {
      if (document.hidden) {
        sequence.current++;
        controller.current?.abort();
        setBusy(false);
        setResult(null);
        setPreview(null);
      } else void check();
    };
    document.addEventListener("visibilitychange", visibility);
    return () => {
      life.abort();
      controller.current?.abort();
      sequence.current++;
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [base, scope, source, loadAttempt]);
  const execute = async () => {
    if (!definition || !meta || !fence || busy) return;
    controller.current?.abort();
    const control = new AbortController();
    controller.current = control;
    const run = ++sequence.current;
    const fresh = () =>
      !control.signal.aborted &&
      !lifetime.current?.signal.aborted &&
      current.current === scope &&
      sequence.current === run;
    setBusy(true);
    setError(null);
    setResult(null);
    setPreview(null);
    try {
      const parameters: Record<string, unknown> = {};
      for (const [key, p] of Object.entries(definition.parameters || {})) {
        const raw = values[key] || "";
        if (raw === "") {
          if (p.required && p.default === undefined)
            throw new Error(`${p.label || key} 값을 입력하세요.`);
          continue;
        }
        if (p.type === "number") {
          const n = Number(raw);
          if (!Number.isFinite(n) || Math.abs(n) > Number.MAX_SAFE_INTEGER)
            throw new Error(`${p.label || key} 숫자를 확인하세요.`);
          parameters[key] = n;
        } else parameters[key] = p.type === "boolean" ? raw === "true" : raw;
      }
      const data = await request<QueryResult>(
        `${base}/execute`,
        control.signal,
        {
          document_version: meta.document_version,
          query_hash: fence.hash,
          parameters,
        },
      );
      if (fresh()) setResult(data);
    } catch (e) {
      if (fresh()) setError(e);
    } finally {
      if (fresh()) setBusy(false);
    }
  };
  const change = (key: string, value: string) => {
    sequence.current++;
    controller.current?.abort();
    setBusy(false);
    setResult(null);
    setPreview(null);
    setError(null);
    setValues((old) => ({ ...old, [key]: value }));
  };
  const labels: Record<string, Record<string, string>> = {
    status: {
      draft: "초안",
      published: "게시됨",
      review: "검토 중",
      backlog: "예정",
      todo: "할 일",
      doing: "진행 중",
      done: "완료",
    },
    priority: { low: "낮음", normal: "보통", high: "높음", urgent: "긴급" },
    classification: {
      public: "공개",
      internal: "내부",
      confidential: "기밀",
      restricted: "제한",
    },
    kind: {
      page: "일반 문서",
      runbook: "런북",
      policy: "정책",
      meeting: "회의록",
    },
    relation_type: {
      related: "관련",
      reference: "참조",
      policy: "정책",
      execution: "실행",
      data: "데이터",
    },
  };
  const text = (value: unknown, field: string) =>
    value == null
      ? "—"
      : typeof value === "boolean"
        ? value
          ? field === "done"
            ? "완료"
            : "참"
          : field === "done"
            ? "미완료"
            : "거짓"
        : Array.isArray(value)
          ? value.join(", ")
          : labels[field]?.[String(value)] || String(value);
  return (
    <section className="document-query" aria-label="문서 선언형 조회">
      <header>
        <div>
          <strong>
            {definition ? querySourceLabels[definition.source] : "선언형 조회"}
          </strong>
          <p>
            현재 내 권한으로만 계산하는 읽기 전용 표 · 원문에 결과를 저장하지
            않습니다.
          </p>
        </div>
        <Button onClick={execute} disabled={!definition || busy}>
          {result ? <RefreshCw size={17} /> : <Play size={17} />}
          {busy ? "조회 중…" : result ? "새로 조회" : "조회 실행"}
        </Button>
        {busy && (
          <Button
            onClick={() => {
              sequence.current++;
              controller.current?.abort();
              setBusy(false);
              setResult(null);
            }}
          >
            {" "}
            <Square size={17} />
            취소
          </Button>
        )}
      </header>
      <ErrorBox error={error} />
      {!!error && (
        <Button onClick={() => setLoadAttempt((v) => v + 1)}>
          다시 정의 확인
        </Button>
      )}
      {meta && !fence && (
        <p role="alert">
          {meta.truncated
            ? "한 문서의 처음 16개 조회 정의만 지원합니다. 조회 문서를 나누어 주세요."
            : "현재 저장된 원문에서 같은 조회 정의를 찾을 수 없습니다. 문서를 다시 불러오세요."}
        </p>
      )}
      {fence?.error && <p role="alert">{fence.error}</p>}
      {!!definition?.parameters && (
        <div className="query-parameters">
          {Object.entries(definition.parameters).map(([key, p]) => (
            <Field
              key={key}
              label={`${p.label || key}${p.required ? " (필수)" : ""}`}
            >
              {p.type === "boolean" ? (
                <select
                  value={values[key] || ""}
                  onChange={(e) => change(key, e.target.value)}
                >
                  <option value="">선택 안 함</option>
                  <option value="true">참</option>
                  <option value="false">거짓</option>
                </select>
              ) : (
                <input
                  type={
                    p.type === "number"
                      ? "number"
                      : p.type === "date" &&
                          (!values[key] || values[key].length <= 10)
                        ? "date"
                        : "text"
                  }
                  value={values[key] || ""}
                  maxLength={500}
                  onChange={(e) => change(key, e.target.value)}
                />
              )}
            </Field>
          ))}
        </div>
      )}
      {result ? (
        <div className="query-result" aria-live="polite">
          <p className="query-status">
            {new Date(result.executed_at).toLocaleString("ko-KR")} 실행 · 정의 v
            {result.document_version} · {result.rows.length}행 ·{" "}
            {result.duration_ms}ms
          </p>
          {(result.diagnostics.truncated ||
            result.diagnostics.invalid_properties > 0 ||
            result.diagnostics.truncated_fields > 0) && (
            <div className="query-warning" role="status">
              일부 범위만 표시합니다. 큰 본문 제외{" "}
              {result.diagnostics.oversized_documents}개 · 읽을 수 없는 속성{" "}
              {result.diagnostics.invalid_properties}개 · 잘린 필드{" "}
              {result.diagnostics.truncated_fields}개. 조건과 범위를 좁혀 다시
              조회하세요.
            </div>
          )}
          <div
            className="query-table"
            tabIndex={0}
            role="region"
            aria-label="조회 결과 표"
          >
            <table>
              <thead>
                <tr>
                  {result.columns.map((c) => (
                    <th key={c.field}>
                      {c.label ||
                        queryLabels[c.field] ||
                        c.field.replace("property.", "")}
                    </th>
                  ))}
                  <th>근거 원문</th>
                </tr>
              </thead>
              <tbody>
                {result.rows.map((row, i) => (
                  <tr key={`${row.document_id}:${row.line}:${i}`}>
                    {result.columns.map((c) => (
                      <td key={c.field}>
                        {text(row.values[c.field], c.field)}
                      </td>
                    ))}
                    <td>
                      <button
                        type="button"
                        className="text-button"
                        onClick={() => setPreview(row)}
                      >
                        원문 v{row.version} 보기
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {result.rows.length === 0 && (
            <p>
              현재 범위와 조건에서 표시할 행이 없습니다. 검색 조건을 바꾸거나
              범위를 확인하세요.
            </p>
          )}
          <details>
            <summary>조회 범위와 현재성 안내</summary>
            <p>{result.diagnostics.notice}</p>
            <p>
              문서 {result.diagnostics.documents_scanned}개 · 후보{" "}
              {result.diagnostics.candidates}개 ·{" "}
              {Math.ceil(result.diagnostics.scanned_bytes / 1024)} KiB 검사.
              화면이 보일 때 3초 주기로 표시된 근거만 다시 확인합니다. 새 문서와
              새 행은 자동 추가하지 않으며, 숨긴 화면의 결과는 폐기합니다.
            </p>
          </details>
        </div>
      ) : (
        !busy && (
          <p className="query-status">
            아직 실행하지 않았거나 이전 결과를 폐기했습니다. 조회 실행으로 현재
            결과를 확인하세요.
          </p>
        )
      )}
      <details className="query-definition">
        <summary>Markdown 조회 정의 보기</summary>
        <pre>
          <code>{source}</code>
        </pre>
      </details>
      {preview && (
        <Suspense fallback={null}>
          <DocumentPreview
            documentId={preview.document_id}
            expectedVersion={preview.version}
            sourceLine={preview.line}
            onClose={() => setPreview(null)}
          />
        </Suspense>
      )}
    </section>
  );
}
