import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  ArrowDown,
  ArrowUp,
  Plus,
  RefreshCw,
  Save,
  ShieldCheck,
  Terminal,
  Trash2,
} from "lucide-react";
import { api, ApiError } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  PageHeading,
  Toggle,
} from "../ui";
import { ApprovalReview } from "../approval/ApprovalPanel";
import type {
  Action,
  Definition,
  Execution,
  Parameter,
  Runbook,
  Runner,
  Step,
} from "./types";
import { phaseNames, statusNames } from "./types";
import "./style.css";
export { RunbookAdminPage } from "./RunbookAdminPage";

const phaseKey: Record<
  string,
  "steps" | "validation_steps" | "rollback_steps"
> = {
  execute: "steps",
  validate: "validation_steps",
  rollback: "rollback_steps",
};
function ParameterInputs({
  action,
  values,
  onChange,
  disabled = false,
}: {
  action?: Action;
  values: Record<string, unknown>;
  onChange: (v: Record<string, unknown>) => void;
  disabled?: boolean;
}) {
  return (
    <div className="runbook-grid">
      {action?.parameters?.map((p) => (
        <Field
          key={p.name}
          label={`${p.label || p.name}${p.required ? " *" : ""}`}
          hint={p.type === "string" ? `허용 형식: ${p.pattern}` : undefined}
        >
          {p.type === "enum" ? (
            <select
              disabled={disabled}
              value={String(values[p.name] ?? "")}
              onChange={(e) =>
                onChange({ ...values, [p.name]: e.target.value })
              }
            >
              <option value="">값 선택</option>
              {p.options.map((o) => (
                <option key={o} value={o}>
                  {o}
                </option>
              ))}
            </select>
          ) : p.type === "boolean" ? (
            <select
              disabled={disabled}
              value={values[p.name] === undefined ? "" : String(values[p.name])}
              onChange={(e) => {
                const next = { ...values };
                if (e.target.value === "") delete next[p.name];
                else next[p.name] = e.target.value === "true";
                onChange(next);
              }}
            >
              <option value="">값 선택</option>
              <option value="true">예</option>
              <option value="false">아니오</option>
            </select>
          ) : (
            <input
              disabled={disabled}
              type={p.type === "integer" ? "number" : "text"}
              min={p.min}
              max={p.max}
              step={p.type === "integer" ? 1 : undefined}
              value={String(values[p.name] ?? "")}
              onChange={(e) => {
                const next = { ...values };
                if (e.target.value === "") delete next[p.name];
                else
                  next[p.name] =
                    p.type === "integer"
                      ? Number(e.target.value)
                      : e.target.value;
                onChange(next);
              }}
            />
          )}
        </Field>
      ))}
    </div>
  );
}
function StepEditor({
  steps,
  runners,
  onChange,
  disabled,
}: {
  steps: Step[];
  runners: Runner[];
  onChange: (v: Step[]) => void;
  disabled: boolean;
}) {
  const change = (i: number, v: Step) =>
    onChange(steps.map((s, n) => (n === i ? v : s)));
  const move = (i: number, d: number) => {
    const copy = [...steps];
    [copy[i], copy[i + d]] = [copy[i + d], copy[i]];
    onChange(copy);
  };
  return (
    <div className="runbook-stack">
      {steps.map((step, i) => {
        const runner = runners.find((r) => r.id === step.runner_id),
          action = runner?.actions.find((a) => a.id === step.action_id);
        return (
          <div className="runbook-step" key={i}>
            <div className="runbook-step-heading">
              <strong>
                {i + 1}. {step.name || "새 단계"}
              </strong>
              {!disabled && (
                <div className="runbook-actions">
                  <Button
                    aria-label={`${i + 1}단계 위로`}
                    disabled={i === 0}
                    onClick={() => move(i, -1)}
                  >
                    <ArrowUp size={16} />
                  </Button>
                  <Button
                    aria-label={`${i + 1}단계 아래로`}
                    disabled={i === steps.length - 1}
                    onClick={() => move(i, 1)}
                  >
                    <ArrowDown size={16} />
                  </Button>
                  <Button
                    aria-label={`${i + 1}단계 삭제`}
                    onClick={() => onChange(steps.filter((_, n) => n !== i))}
                  >
                    <Trash2 size={16} />
                  </Button>
                </div>
              )}
            </div>
            <div className="runbook-grid">
              <Field label="단계 이름">
                <input
                  disabled={disabled}
                  value={step.name}
                  maxLength={200}
                  onChange={(e) => change(i, { ...step, name: e.target.value })}
                />
              </Field>
              <Field label="격리 실행기">
                <select
                  disabled={disabled}
                  value={step.runner_id}
                  onChange={(e) =>
                    change(i, {
                      ...step,
                      runner_id: e.target.value,
                      action_id: "",
                      parameters: {},
                    })
                  }
                >
                  <option value="">실행기 선택</option>
                  {!runner && step.runner_id && (
                    <option value={step.runner_id}>
                      현재 권한 없음 · 기존 실행기
                    </option>
                  )}
                  {runners.map((r) => (
                    <option key={r.id} value={r.id}>
                      {r.name} ·{" "}
                      {r.kind === "awx" ? "AWX / Ansible" : "Kubernetes"}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="관리자 허용 작업">
                <select
                  disabled={disabled}
                  value={step.action_id}
                  onChange={(e) =>
                    change(i, {
                      ...step,
                      action_id: e.target.value,
                      parameters: {},
                    })
                  }
                >
                  <option value="">작업 선택</option>
                  {!action && step.action_id && (
                    <option value={step.action_id}>
                      현재 허용 목록에 없음
                    </option>
                  )}
                  {runner?.actions.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </Field>
              <p className="runbook-meta">
                {action &&
                  `${action.timeout_seconds}초 제한 · ${runner?.kind === "awx" ? `고정 템플릿 #${action.template_id}` : "고정 이미지 digest / argv"}`}
              </p>
            </div>
            <ParameterInputs
              action={action}
              values={step.parameters}
              disabled={disabled}
              onChange={(parameters) => change(i, { ...step, parameters })}
            />
          </div>
        );
      })}
      {!disabled && (
        <Button
          disabled={steps.length >= 20}
          onClick={() =>
            onChange([
              ...steps,
              { name: "새 단계", runner_id: "", action_id: "", parameters: {} },
            ])
          }
        >
          <Plus size={17} />
          단계 추가
        </Button>
      )}
    </div>
  );
}

export function RunbookDocumentPage() {
  const { id = "" } = useParams(),
    navigate = useNavigate(),
    { publicInfo } = useApp();
  const [data, setData] = useState<Runbook | null>(null),
    [draft, setDraft] = useState<Definition | null>(null),
    [runners, setRunners] = useState<Runner[]>([]),
    [history, setHistory] = useState<Execution[]>([]),
    [phase, setPhase] = useState("execute"),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [dirty, setDirty] = useState(false),
    [unavailable, setUnavailable] = useState(false);
  const load = useCallback(async () => {
    try {
      const value = await api<Runbook>(`/documents/${id}/runbook`);
      setData(value);
      setDraft(value.definition);
      setDirty(false);
      const [rs, hs] = await Promise.all([
        api<Runner[]>(`/runbook/runners?workspace_id=${value.workspace_id}`),
        api<Execution[]>(`/documents/${id}/runbook/executions`),
      ]);
      setRunners(rs);
      setHistory(hs);
      setError("");
      setUnavailable(false);
    } catch (e) {
      setData(null);
      setDraft(null);
      setError((e as Error).message);
      setUnavailable(e instanceof ApiError && e.status === 404);
    }
  }, [id]);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    const protect = (e: BeforeUnloadEvent) => {
      if (dirty) e.preventDefault();
    };
    window.addEventListener("beforeunload", protect);
    return () => window.removeEventListener("beforeunload", protect);
  }, [dirty]);
  const edit = (v: Definition) => {
    setDraft(v);
    setDirty(true);
  };
  const save = async () => {
    if (!draft) return;
    setBusy(true);
    try {
      await api(`/documents/${id}/runbook`, "PUT", draft);
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const prepare = async () => {
    setBusy(true);
    try {
      const p = await api<Execution>(
        `/documents/${id}/runbook/prepare`,
        "POST",
        { phase },
      );
      navigate(`/app/runbook/executions/${p.id}`);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  if (unavailable)
    return (
      <div className="page-content">
        <Empty
          title="격리 실행을 사용할 수 없습니다"
          text="관리자가 기능을 켜고 허용 실행기를 설정해야 합니다. 문서 접근 권한도 확인하세요."
        />
        <Link to={`/app/documents/${id}`}>문서로 돌아가기</Link>
      </div>
    );
  if (!data || !draft)
    return (
      <div className="page-content">
        {error ? <ErrorBox error={error} /> : <Loading />}
      </div>
    );
  return (
    <div className="page-content">
      <div className="runbook-layout">
        <PageHeading
          eyebrow="운영 절차"
          title={data.title}
          description="목적·선행 조건과 관리자 허용 작업을 연결합니다. madi 서버에서는 명령을 실행하지 않습니다."
          actions={<Link to={`/app/documents/${id}`}>문서로 돌아가기</Link>}
        />
        {error && <ErrorBox error={error} />}
        <div className="runbook-warning">
          <ShieldCheck size={18} /> 실행은 사내 격리 실행기에서만 수행됩니다.{" "}
          {publicInfo.approval_enabled
            ? "관리자 승인 정책에 따라 정확한 계획을 검토한 뒤 직접 실행을 확인합니다."
            : "별도의 검토·승인 과정 없이 요청자가 정확한 계획을 직접 확인합니다."}
        </div>
        <section className="runbook-card">
          <div className="runbook-grid">
            {(
              ["purpose", "prerequisites", "validation", "rollback"] as const
            ).map((key) => (
              <Field
                key={key}
                label={
                  {
                    purpose: "목적",
                    prerequisites: "선행 조건",
                    validation: "검증 기준",
                    rollback: "롤백 기준",
                  }[key]
                }
              >
                <textarea
                  disabled={!data.can_write || busy}
                  maxLength={30000}
                  value={draft[key]}
                  onChange={(e) => edit({ ...draft, [key]: e.target.value })}
                />
              </Field>
            ))}
          </div>
          <p className="runbook-meta">
            문서 v{data.document_version} · 운영 절차 v{draft.version} · 마지막
            검증:{" "}
            {data.last_tested_at
              ? new Date(data.last_tested_at).toLocaleString("ko-KR")
              : "아직 실행되지 않음"}
          </p>
        </section>
        <section className="runbook-card runbook-stack">
          <div
            className="runbook-tabs"
            role="tablist"
            aria-label="운영 절차 유형"
          >
            {Object.entries(phaseNames).map(([key, label]) => (
              <Button
                key={key}
                role="tab"
                aria-selected={phase === key}
                onClick={() => setPhase(key)}
              >
                {label} 단계
              </Button>
            ))}
          </div>
          <StepEditor
            steps={draft[phaseKey[phase]] || []}
            runners={runners}
            disabled={!data.can_write || busy}
            onChange={(steps) => edit({ ...draft, [phaseKey[phase]]: steps })}
          />
          {data.can_write && (
            <div className="runbook-actions">
              <Button
                variant="primary"
                disabled={busy || !dirty}
                onClick={() => void save()}
              >
                <Save size={17} />
                운영 절차 저장
              </Button>
              <Button
                disabled={busy || dirty || !draft[phaseKey[phase]]?.length}
                onClick={() => void prepare()}
              >
                <Terminal size={17} />
                {busy ? "처리 중…" : `${phaseNames[phase]} 계획 준비`}
              </Button>
              {dirty && (
                <span className="runbook-meta">
                  변경 사항을 먼저 저장하세요.
                </span>
              )}
            </div>
          )}
        </section>
        <section className="runbook-card">
          <h2>실행 이력</h2>
          <div className="runbook-history">
            {history.map((x) => (
              <Link key={x.id} to={`/app/runbook/executions/${x.id}`}>
                <span>
                  {phaseNames[x.phase]} · {statusNames[x.status]}
                </span>
                <span className="runbook-meta">
                  {new Date(x.created_at).toLocaleString("ko-KR")}
                </span>
              </Link>
            ))}
            {!history.length && (
              <p className="muted">준비한 실행 계획이 없습니다.</p>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}

export function RunbookExecutionPage() {
  const { id = "" } = useParams(),
    { user } = useApp();
  const [data, setData] = useState<Execution | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [confirmation, setConfirmation] = useState(""),
    [review, setReview] = useState(false),
    [logs, setLogs] = useState(""),
    [streamError, setStreamError] = useState("");
  const cursor = useRef(0),
    [streamGeneration, setStreamGeneration] = useState(0);
  const load = useCallback(async () => {
    try {
      setData(await api<Execution>(`/runbook/executions/${id}`));
      setError("");
    } catch (e) {
      setData(null);
      setLogs("");
      setError((e as Error).message);
    }
  }, [id]);
  useEffect(() => {
    setConfirmation("");
    setLogs("");
    cursor.current = 0;
    void load();
    const timer = setInterval(() => void load(), 4000);
    return () => clearInterval(timer);
  }, [load]);
  useEffect(() => {
    if (!data?.can_read_logs || !data.steps?.some((s) => s.state !== "queued"))
      return;
    const source = new EventSource(
      `/api/v1/runbook/executions/${id}/events?after=${cursor.current}`,
      { withCredentials: true },
    );
    let ended = false;
    const receive = (e: MessageEvent) => {
      try {
        const item = JSON.parse(e.data);
        if (Number(item.id) > cursor.current) {
          cursor.current = Number(item.id);
          setLogs((old) =>
            (
              old +
              (item.kind === "output"
                ? item.message
                : `\n[상태] ${item.message}\n`)
            ).slice(-500000),
          );
        }
      } catch {
        setStreamError("실행 출력 형식을 확인하지 못했습니다.");
      }
    };
    source.addEventListener("event", receive as EventListener);
    source.addEventListener("done", () => {
      ended = true;
      source.close();
    });
    source.addEventListener("error", (e) => {
      if (ended) return;
      source.close();
      const message = (e as MessageEvent).data;
      if (message) {
        try {
          setStreamError(
            JSON.parse(message).error || "출력 권한이 변경되었습니다.",
          );
          setLogs("");
        } catch {
          setStreamError("출력 연결을 확인하세요.");
        }
      } else
        setStreamError(
          "출력 연결이 끊겼습니다. 다시 연결하여 저장된 출력부터 이어 볼 수 있습니다.",
        );
    });
    setStreamError("");
    return () => {
      ended = true;
      source.close();
    };
  }, [
    id,
    data?.can_read_logs,
    !!data?.steps?.some((s) => s.state !== "queued"),
    streamGeneration,
  ]);
  const action = async (kind: "approval" | "execute" | "cancel") => {
    if (!data) return;
    setBusy(true);
    try {
      await api(`/runbook/executions/${id}/${kind}`, "POST", {
        version: data.version,
        confirmation,
        approval_version: data.approval?.version || 0,
      });
      setConfirmation("");
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  if (!data)
    return (
      <div className="page-content">
        {error ? <ErrorBox error={error} /> : <Loading />}
      </div>
    );
  const owner = user?.id === data.owner_id,
    canCancel =
      ["queued", "running", "unknown"].includes(data.status) &&
      (owner || user?.role === "admin");
  return (
    <div className="page-content">
      <div className="runbook-layout">
        <PageHeading
          eyebrow={`격리 ${phaseNames[data.phase]} 계획`}
          title={data.snapshot.title}
          description="검토한 문서·운영 절차·허용 작업의 정확한 버전에만 유효합니다."
          actions={
            <Link to={`/app/documents/${data.document_id}/runbook`}>
              운영 절차로 돌아가기
            </Link>
          }
        />
        {error && <ErrorBox error={error} />}
        <div className="runbook-actions">
          <span className="badge">{statusNames[data.status]}</span>
          <span className="runbook-meta">
            문서 v{data.snapshot.document_version} · 운영 절차 v
            {data.snapshot.definition_version} · 계획 v{data.version}
          </span>
        </div>
        {data.status === "unknown" && (
          <div className="runbook-warning">
            외부 실행 여부를 확인할 수 없습니다. 자동 재실행하지 않습니다.
            관리자는 격리 실행기에서 이 계획의 작업과 출력을 확인하고, 작업 ID가
            표시된 경우 중지를 요청하세요.
          </div>
        )}
        {data.last_error && <ErrorBox error={data.last_error} />}
        <section className="runbook-card runbook-stack">
          <h2>변경할 수 없는 실행 원본</h2>
          {data.snapshot.steps.map((step, i) => (
            <article className="runbook-step" key={i}>
              <div className="runbook-step-heading">
                <strong>
                  {i + 1}. {step.name}
                </strong>
                <span className="badge">
                  {statusNames[data.steps?.[i]?.state || "queued"]}
                </span>
              </div>
              <p>
                {step.action.name} ·{" "}
                {step.kind === "awx"
                  ? `AWX / Ansible 템플릿 #${step.action.template_id}`
                  : "Kubernetes 격리 Job"}{" "}
                · 최대 {step.action.timeout_seconds}초
              </p>
              {step.kind === "kubernetes" && (
                <pre className="runbook-code">
                  {step.action.image}
                  {"\n"}
                  {JSON.stringify(step.argv)}
                </pre>
              )}
              <pre
                className="runbook-code"
                aria-label={`${i + 1}단계 매개변수`}
              >
                {JSON.stringify(step.parameters, null, 2)}
              </pre>
              {data.steps?.[i]?.external_id && (
                <p className="runbook-meta">
                  외부 작업 ID: {data.steps[i].external_id}
                </p>
              )}
              <details>
                <summary>
                  격리 환경 검증 원본 · 실행기 v{step.runner_revision}
                </summary>
                <pre className="runbook-code">
                  {JSON.stringify(step.remote, null, 2)}
                </pre>
              </details>
            </article>
          ))}
        </section>
        {data.approval_required && (
          <section className="runbook-card">
            <h2>실행 계획 검토</h2>
            <p>
              관리자가 지정한 검토자가 위 계획을 승인해야 합니다. 승인 후에도
              요청자가 직접 실행을 확인합니다.
            </p>
            <div className="runbook-actions">
              {owner &&
                ["prepared", "rejected", "cancelled"].includes(data.status) &&
                !data.cancel_requested && (
                  <Button
                    disabled={busy}
                    onClick={() => void action("approval")}
                  >
                    검토 요청
                  </Button>
                )}
              {data.approval_id && (
                <Button onClick={() => setReview(true)}>
                  검토 원본·결정 확인
                </Button>
              )}
            </div>
          </section>
        )}
        {data.can_confirm && (
          <section className="runbook-card runbook-stack">
            <h2>{phaseNames[data.phase]}을 직접 확인하세요</h2>
            <p>
              현재 문서와 허용 명령이 변경되면 실행이 거부됩니다. 같은 계획으로
              두 번 실행할 수 없습니다.
            </p>
            <Field
              label="실행 확인 문구"
              hint={`아래 문구를 그대로 입력하세요: EXECUTE ${id}`}
            >
              <input
                value={confirmation}
                autoComplete="off"
                spellCheck={false}
                onChange={(e) => setConfirmation(e.target.value)}
              />
            </Field>
            <Button
              variant="primary"
              disabled={busy || confirmation !== `EXECUTE ${id}`}
              onClick={() => void action("execute")}
            >
              <Terminal size={18} />
              확인한 계획 실행
            </Button>
          </section>
        )}
        {canCancel && (
          <section className="runbook-card runbook-stack">
            <h2>외부 작업 중지</h2>
            <Field label="중지 확인 문구" hint={`CANCEL ${id}`}>
              <input
                value={confirmation}
                autoComplete="off"
                onChange={(e) => setConfirmation(e.target.value)}
              />
            </Field>
            <Button
              disabled={
                busy || data.cancel_requested || confirmation !== `CANCEL ${id}`
              }
              onClick={() => void action("cancel")}
            >
              {data.cancel_requested
                ? "외부 중지 확인 중"
                : "격리 작업 중지 요청"}
            </Button>
          </section>
        )}
        {data.can_read_logs && (
          <section className="runbook-card">
            <div className="runbook-step-heading">
              <h2>실시간 실행 출력</h2>
              <Button onClick={() => setStreamGeneration((g) => g + 1)}>
                <RefreshCw size={16} />
                다시 연결
              </Button>
            </div>
            {streamError && <ErrorBox error={streamError} />}
            <pre className="runbook-code runbook-output" aria-live="polite">
              {logs || "작업이 시작되면 저장된 출력부터 표시합니다."}
            </pre>
            <p className="runbook-meta">
              단계당 최대 2MB, 실행당 최대 5MB 저장 · 화면에서는 최근 50만 자
              표시 · 현재 로그인·문서·실행 권한을 계속 확인합니다.
            </p>
          </section>
        )}
        {review && data.approval_id && (
          <ApprovalReview
            requestID={data.approval_id}
            onClose={() => setReview(false)}
            onChanged={() => void load()}
          />
        )}
      </div>
    </div>
  );
}
