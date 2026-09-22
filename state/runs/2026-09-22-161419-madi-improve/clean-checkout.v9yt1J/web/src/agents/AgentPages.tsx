import { useEffect, useRef, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  Bot,
  Play,
  Plus,
  Settings2,
  ShieldCheck,
  Square,
  RefreshCw,
} from "lucide-react";
import { api, ApiError, datetime, type Database } from "../api";
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

type Agent = {
  id: string;
  workspace_id: string;
  name: string;
  instructions: string;
  enabled: boolean;
  revision: number;
  space_ids: string[];
  document_ids: string[];
  database_ids: string[];
  tools: string[];
  max_steps: number;
  max_tokens: number;
  can_manage?: boolean;
};
type Action = {
  id: string;
  tool: string;
  arguments: {
    title: string;
    markdown: string;
    document_id?: string;
    space_id?: string;
    expected_version?: number;
    visibility?: string;
  };
  action_hash: string;
  status: string;
};
type Run = {
  id: string;
  agent_id: string;
  agent_name?: string;
  status: string;
  prompt: string;
  step: number;
  created_at: string;
  error: string;
  actions: Action[];
  can_confirm: boolean;
};
const tools: Record<string, string> = {
  search_documents: "문서 검색",
  get_document: "문서 읽기",
  query_database: "DB 조회·수식",
  get_graph: "문서 연결 그래프",
  create_document: "새 문서 작성 제안",
  update_document: "문서 수정 제안",
};
const states: Record<string, string> = {
  pending: "대기 중",
  running: "실행 중",
  awaiting_confirmation: "작업 확인 대기",
  succeeded: "완료",
  failed: "실패",
  cancelled: "취소됨",
};
const actionStates: Record<string, string> = {
  planned: "확인 필요",
  confirmed: "실행 대기",
  applied: "저장 완료",
  rejected: "거절됨",
  cancelled: "취소됨",
  failed: "실패",
};
const newAgent = (): Agent => ({
  id: "",
  workspace_id: "",
  name: "",
  instructions: "",
  enabled: false,
  revision: 1,
  space_ids: [],
  document_ids: [],
  database_ids: [],
  tools: ["search_documents", "get_document"],
  max_steps: 8,
  max_tokens: 4096,
});

export function AgentsPage() {
  const { workspace, user, documents, notify } = useApp();
  const navigate = useNavigate();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selected, setSelected] = useState("");
  const [history, setHistory] = useState<Run[]>([]);
  const [prompt, setPrompt] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [edit, setEdit] = useState<Agent | null>(null);
  const [spaces, setSpaces] = useState<{ id: string; name: string }[]>([]);
  const [databases, setDatabases] = useState<Database[]>([]);
  const generation = useRef(0);
  const canManage =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const load = async () => {
    if (!workspace) return;
    const token = ++generation.current;
    setError("");
    try {
      const values = await api<Agent[]>(`/workspaces/${workspace.id}/agents`);
      if (token !== generation.current) return;
      setAgents(values);
      setSelected((previous) =>
        values.some((a) => a.id === previous)
          ? previous
          : values.find((a) => a.enabled)?.id || values[0]?.id || "",
      );
    } catch (e) {
      if (token === generation.current) setError((e as Error).message);
    } finally {
      if (token === generation.current) setLoading(false);
    }
  };
  useEffect(() => {
    setSelected("");
    setHistory([]);
    setAgents([]);
    setEdit(null);
    setPrompt("");
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [workspace?.id, user.id]);
  useEffect(() => {
    let active = true;
    setHistory([]);
    if (selected)
      api<Run[]>(`/agents/${selected}/runs`)
        .then((v) => {
          if (active) setHistory(v);
        })
        .catch((e) => {
          if (active) setError(e.message);
        });
    return () => {
      active = false;
    };
  }, [selected]);
  useEffect(() => {
    let active = true;
    setSpaces([]);
    setDatabases([]);
    if (workspace && canManage)
      Promise.all([
        api<{ id: string; name: string }[]>(
          `/spaces?workspace_id=${workspace.id}`,
        ),
        api<Database[]>(`/databases?workspace_id=${workspace.id}`),
      ])
        .then(([s, d]) => {
          if (active) {
            setSpaces(s);
            setDatabases(d);
          }
        })
        .catch((e) => {
          if (active) setError(e.message);
        });
    return () => {
      active = false;
    };
  }, [workspace?.id, canManage]);
  const current = agents.find((a) => a.id === selected);
  const save = async () => {
    if (!workspace || !edit) return;
    setBusy(true);
    setError("");
    try {
      await api(
        edit.id ? `/agents/${edit.id}` : `/workspaces/${workspace.id}/agents`,
        edit.id ? "PUT" : "POST",
        edit,
      );
      setEdit(null);
      await load();
      notify("Agent 설정을 저장했습니다");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const start = async () => {
    if (!current || !prompt.trim()) return;
    setBusy(true);
    setError("");
    try {
      const run = await api<{ id: string }>(
        `/agents/${current.id}/runs`,
        "POST",
        { prompt, expected_agent_version: current.revision },
      );
      navigate(`/app/agents/runs/${run.id}`);
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const choices = (
    key: "space_ids" | "document_ids" | "database_ids",
    items: { id: string; label: string }[],
  ) => (
    <div className="agent-choice-list">
      {items.length ? (
        items.map((item) => (
          <label key={item.id}>
            <input
              type="checkbox"
              checked={!!edit?.[key].includes(item.id)}
              onChange={(e) =>
                setEdit((v) =>
                  v
                    ? {
                        ...v,
                        [key]: e.target.checked
                          ? [...v[key], item.id]
                          : v[key].filter((id) => id !== item.id),
                      }
                    : v,
                )
              }
            />
            <span>{item.label}</span>
          </label>
        ))
      ) : (
        <p className="muted">선택할 항목이 없습니다.</p>
      )}
    </div>
  );
  if (!workspace)
    return (
      <Empty
        title="워크스페이스를 선택하세요"
        text="Agent는 워크스페이스별 지식과 권한을 사용합니다."
      />
    );
  return (
    <div className="agent-page">
      <PageHeading
        title="워크스페이스 Agent"
        description="허용한 지식과 도구로 질문을 해결하고, 변경은 내가 확인합니다."
        actions={
          <>
            <Button onClick={() => void load()} aria-label="Agent 새로고침">
              <RefreshCw size={17} />
            </Button>
            {canManage && (
              <Button onClick={() => setEdit(newAgent())}>
                <Plus size={17} />
                Agent 만들기
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      {loading ? (
        <Loading />
      ) : !agents.length ? (
        <Empty
          title="아직 연결된 Agent가 없습니다"
          text="워크스페이스 관리자가 지식 범위와 사용할 도구를 설정하면 시작할 수 있습니다."
        />
      ) : (
        <div className="agent-layout">
          <aside className="agent-list">
            {agents.map((a) => (
              <button
                key={a.id}
                className={selected === a.id ? "selected" : ""}
                onClick={() => setSelected(a.id)}
              >
                <Bot size={22} />
                <span>
                  <strong>{a.name}</strong>
                  <small>
                    {a.enabled
                      ? `${a.tools.length}개 도구 · 최대 ${a.max_steps}단계`
                      : "비활성화"}
                  </small>
                </span>
              </button>
            ))}
          </aside>
          <section className="agent-content">
            {current && (
              <>
                <div className="agent-card">
                  <div className="agent-title">
                    <h2>{current.name}</h2>
                    {canManage && (
                      <Button onClick={() => setEdit({ ...current })}>
                        <Settings2 size={16} />
                        설정
                      </Button>
                    )}
                  </div>
                  <p className="muted">
                    {current.instructions ||
                      "설정된 지식 범위에서 도구를 사용해 답변합니다."}
                  </p>
                  <div className="agent-chips">
                    {current.tools.map((tool) => (
                      <span key={tool}>{tools[tool] || tool}</span>
                    ))}
                  </div>
                  <p className="agent-notice">
                    <ShieldCheck size={18} />내 현재 접근 권한만 사용합니다.
                    문서 내용은 관리자가 연결한 AI 공급자에 전송되며, 문서
                    저장은 작업별 확인이 필요합니다.
                  </p>
                  <Field label="Agent에게 요청하기">
                    <textarea
                      rows={5}
                      maxLength={12000}
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                      placeholder="관련 운영 문서를 찾아 비교하고, 개선할 내용을 제안해 주세요."
                    />
                  </Field>
                  <Button
                    variant="primary"
                    disabled={busy || !current.enabled || !prompt.trim()}
                    onClick={() => void start()}
                  >
                    <Play size={17} />
                    실행 시작
                  </Button>
                </div>
                <div className="agent-card">
                  <h3>내 실행 이력</h3>
                  {history.length ? (
                    history.map((run) => (
                      <Link
                        className="agent-history"
                        key={run.id}
                        to={`/app/agents/runs/${run.id}`}
                      >
                        <span>{run.prompt}</span>
                        <small>
                          {states[run.status]} · {datetime(run.created_at)}
                        </small>
                      </Link>
                    ))
                  ) : (
                    <p className="muted">
                      아직 실행 기록이 없습니다. 다른 사용자의 기록은 표시하지
                      않습니다.
                    </p>
                  )}
                </div>
              </>
            )}
          </section>
        </div>
      )}
      <Modal
        open={!!edit}
        onOpenChange={(open) => {
          if (!open && !busy) setEdit(null);
        }}
        title={edit?.id ? "Agent 설정" : "새 Agent"}
        description="설정 변경 즉시 진행 중 실행의 이전 권한과 공급자 사용이 중단됩니다."
        wide
      >
        {edit && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void save();
            }}
            className="agent-form"
          >
            <ErrorBox error={error} />
            <Field label="Agent 이름">
              <input
                required
                maxLength={160}
                value={edit.name}
                onChange={(e) => setEdit({ ...edit, name: e.target.value })}
              />
            </Field>
            <Field label="역할과 지시문">
              <textarea
                rows={4}
                maxLength={16000}
                value={edit.instructions}
                onChange={(e) =>
                  setEdit({ ...edit, instructions: e.target.value })
                }
              />
            </Field>
            <div className="agent-two">
              <Field label="최대 실행 단계">
                <input
                  type="number"
                  min={1}
                  max={24}
                  required
                  value={edit.max_steps}
                  onChange={(e) =>
                    setEdit({ ...edit, max_steps: Number(e.target.value) })
                  }
                />
              </Field>
              <Field
                label="단계별 최대 출력 토큰"
                hint="서비스 AI 한도와 교집합 · 최대 262144"
              >
                <input
                  type="number"
                  min={1}
                  max={262144}
                  required
                  value={edit.max_tokens}
                  onChange={(e) =>
                    setEdit({ ...edit, max_tokens: Number(e.target.value) })
                  }
                />
              </Field>
            </div>
            <fieldset>
              <legend>허용 도구</legend>
              <div className="agent-choice-list">
                {Object.entries(tools).map(([key, label]) => (
                  <label key={key}>
                    <input
                      type="checkbox"
                      checked={edit.tools.includes(key)}
                      onChange={(e) =>
                        setEdit({
                          ...edit,
                          tools: e.target.checked
                            ? [...edit.tools, key]
                            : edit.tools.filter((t) => t !== key),
                        })
                      }
                    />
                    <span>
                      {label}
                      {key.endsWith("document") && key !== "get_document"
                        ? " · 사용자 확인 필수"
                        : ""}
                    </span>
                  </label>
                ))}
              </div>
            </fieldset>
            <fieldset>
              <legend>지식 공간 · 하위 공간 포함</legend>
              {choices(
                "space_ids",
                spaces.map((s) => ({ id: s.id, label: s.name })),
              )}
            </fieldset>
            <fieldset>
              <legend>개별 문서 · 현재 읽을 수 있는 문서만</legend>
              {choices(
                "document_ids",
                documents
                  .filter(
                    (d) =>
                      d.workspace_id === workspace.id && !d.deleted_at,
                  )
                  .map((d) => ({ id: d.id, label: d.title })),
              )}
            </fieldset>
            <fieldset>
              <legend>데이터베이스 · 관계/롤업 대상도 선택 필요</legend>
              {choices(
                "database_ids",
                databases.map((d) => ({ id: d.id, label: d.name })),
              )}
            </fieldset>
            <label className="agent-check">
              <input
                type="checkbox"
                checked={edit.enabled}
                onChange={(e) =>
                  setEdit({ ...edit, enabled: e.target.checked })
                }
              />
              설정된 지식과 도구로 Agent 활성화
            </label>
            <p className="muted">
              새 문서 작성은 명시적으로 선택한 공간에서만 가능합니다. 셸·임의
              URL·Runbook 실행 도구는 제공하지 않습니다.
            </p>
            <Button variant="primary" type="submit" disabled={busy}>
              설정 저장
            </Button>
          </form>
        )}
      </Modal>
    </div>
  );
}

export function AgentRunPage() {
  const { id = "" } = useParams();
  const { user } = useApp();
  const [run, setRun] = useState<Run | null>(null);
  const [error, setError] = useState("");
  const [steps, setSteps] = useState<Record<string, string>>({});
  const [stepStates, setStepStates] = useState<Record<string, string>>({});
  const [results, setResults] = useState<{ tool: string; result: unknown }[]>(
    [],
  );
  const [busy, setBusy] = useState(false);
  const [chosen, setChosen] = useState<Action | null>(null);
  const [confirm, setConfirm] = useState(false);
  const cursor = useRef(0);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current;
    let active = true;
    let streaming = false;
    let abort: AbortController | null = null;
    cursor.current = 0;
    setRun(null);
    setError("");
    setSteps({});
    setStepStates({});
    setResults([]);
    setChosen(null);
    setConfirm(false);
    const clear = (message: string) => {
      if (!active) return;
      setSteps({});
      setStepStates({});
      setResults([]);
      setRun(null);
      setChosen(null);
      setError(message);
    };
    const apply = (event: string, value: any) => {
      if (!active || token !== generation.current) return;
      if (event === "retract") {
        clear(value.error || "권한이 변경되어 출력을 숨겼습니다");
        return;
      }
      if (event === "step_start") {
        setSteps((v) => ({ ...v, [value.step]: "" }));
        setStepStates((v) => ({ ...v, [value.step]: "running" }));
      }
      if (event === "step_end") {
        setStepStates((v) => ({ ...v, [value.step]: value.status }));
        setRun((v) =>
          v
            ? { ...v, step: Math.max(v.step, value.step), status: value.status }
            : v,
        );
      }
      if (event === "status")
        setRun((v) => (v ? { ...v, status: value.status } : v));
      if (event === "delta")
        setSteps((v) => ({
          ...v,
          [value.step]: (v[value.step] || "") + value.text,
        }));
      if (event === "tool_result")
        setResults((v) => [...v, { tool: value.tool, result: value.result }]);
    };
    const stream = async () => {
      if (streaming || !active || document.hidden) return;
      streaming = true;
      abort = new AbortController();
      try {
        const response = await fetch(
          `/api/v1/agent-runs/${id}/events?after=${cursor.current}`,
          { credentials: "same-origin", signal: abort.signal },
        );
        if (!response.ok) {
          const v = await response.json();
          throw new ApiError(
            v.error || "실행 출력을 가져오지 못했습니다",
            response.status,
          );
        }
        if (!response.body) throw Error("스트리밍 응답이 없습니다");
        const reader = response.body.getReader(),
          decoder = new TextDecoder();
        let buffer = "";
        for (;;) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder
            .decode(value, { stream: true })
            .replace(/\r\n/g, "\n");
          if (buffer.length > 1048576)
            throw Error("스트림 프레임 한도를 초과했습니다");
          let boundary;
          while ((boundary = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, boundary);
            buffer = buffer.slice(boundary + 2);
            let event = "message",
              data = "",
              eventId = 0;
            for (const line of frame.split("\n")) {
              if (line.startsWith("event:")) event = line.slice(6).trim();
              if (line.startsWith("data:")) data += line.slice(5).trimStart();
              if (line.startsWith("id:"))
                eventId = Number(line.slice(3).trim());
            }
            if (data) {
              apply(event, JSON.parse(data));
              if (eventId) cursor.current = eventId;
            }
          }
        }
      } catch (e) {
        if (active && (e as Error).name !== "AbortError")
          clear((e as Error).message);
      } finally {
        streaming = false;
      }
    };
    const poll = async () => {
      if (!active || document.hidden) return;
      try {
        const value = await api<Run>(`/agent-runs/${id}`);
        if (!active || token !== generation.current) return;
        setRun(value);
        setChosen((old) =>
          old &&
          value.actions.some(
            (a) =>
              a.id === old.id &&
              a.status === "planned" &&
              a.action_hash === old.action_hash,
          )
            ? old
            : null,
        );
        await stream();
      } catch (e) {
        if (active) clear((e as Error).message);
      }
    };
    void poll();
    const timer = setInterval(() => void poll(), 1800);
    const visibility = () => {
      if (document.hidden) abort?.abort();
      else void poll();
    };
    document.addEventListener("visibilitychange", visibility);
    return () => {
      active = false;
      generation.current++;
      clearInterval(timer);
      abort?.abort();
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [id, user.id]);
  useEffect(() => setConfirm(false), [chosen?.id, chosen?.action_hash]);
  const decide = async (reject = false) => {
    if (!chosen) return;
    setBusy(true);
    setError("");
    try {
      await api(`/agent-runs/${id}/actions/${chosen.id}/confirm`, "POST", {
        action_hash: chosen.action_hash,
        confirm: !reject,
        reject,
      });
      setChosen(null);
      setConfirm(false);
      setRun(await api<Run>(`/agent-runs/${id}`));
    } catch (e) {
      setError((e as Error).message);
      setChosen(null);
      setConfirm(false);
    } finally {
      setBusy(false);
    }
  };
  const cancel = async () => {
    setBusy(true);
    try {
      await api(`/agent-runs/${id}/cancel`, "POST", {});
      setRun(await api<Run>(`/agent-runs/${id}`));
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="agent-page">
      <Link to="/app/agents" className="muted">
        ← 워크스페이스 Agent
      </Link>
      <PageHeading
        title={run?.agent_name || "Agent 실행"}
        description={
          run
            ? `${states[run.status]} · ${run.step}단계 · ${datetime(run.created_at)}`
            : "권한과 실행 상태를 확인합니다."
        }
        actions={
          run &&
          ["pending", "running", "awaiting_confirmation"].includes(
            run.status,
          ) ? (
            <Button disabled={busy} onClick={() => void cancel()}>
              <Square size={16} />
              실행 취소
            </Button>
          ) : undefined
        }
      />
      <ErrorBox error={error || run?.error || ""} />
      {!run && !error ? (
        <Loading />
      ) : (
        run && (
          <>
            <div className="agent-card">
              <h3>내 요청</h3>
              <p className="agent-answer">{run.prompt}</p>
            </div>
            <section aria-live="polite" className="agent-steps">
              {Object.entries(steps).map(([step, text]) => (
                <article key={step} className="agent-card">
                  <small className="muted">단계 {step}</small>
                  <p className="agent-answer">
                    {text ||
                      (stepStates[step] === "awaiting_confirmation"
                        ? "확인할 문서 변경 계획을 준비했습니다."
                        : stepStates[step] && stepStates[step] !== "running"
                          ? "도구 처리를 완료했습니다."
                          : run.status === "succeeded"
                            ? "도구 처리를 완료했습니다."
                            : "도구를 확인하고 있습니다…")}
                  </p>
                </article>
              ))}
            </section>
            {results.map((result, index) => (
              <details className="agent-card" key={index}>
                <summary>{tools[result.tool] || result.tool} 결과</summary>
                <pre className="agent-json">
                  {JSON.stringify(result.result, null, 2)}
                </pre>
              </details>
            ))}
            {run.actions.length > 0 && (
              <section className="agent-card">
                <h2>문서 변경 계획</h2>
                <p className="muted">
                  각 계획은 내용과 대상 버전을 고정합니다. 확인 전에는 문서가
                  저장되지 않습니다.
                </p>
                {run.actions.map((action) => (
                  <div className="agent-action" key={action.id}>
                    <div>
                      <strong>{action.arguments.title}</strong>
                      <small>
                        {tools[action.tool]} · {actionStates[action.status]}
                      </small>
                    </div>
                    {action.status === "planned" && run.can_confirm && (
                      <Button
                        onClick={() => {
                          setChosen(action);
                          setConfirm(false);
                        }}
                      >
                        <ShieldCheck size={17} />
                        내용 확인
                      </Button>
                    )}
                    {action.arguments.document_id && (
                      <Link
                        to={`/app/documents/${action.arguments.document_id}`}
                      >
                        원본 문서
                      </Link>
                    )}
                  </div>
                ))}
              </section>
            )}
          </>
        )
      )}
      <Modal
        open={!!chosen}
        onOpenChange={(open) => {
          if (!open && !busy) setChosen(null);
        }}
        title="이 문서 변경을 확인하시겠습니까?"
        description="이 계획 한 건만 승인합니다. 실제 문서 승인 프로세스가 설정되어 있으면 기존 검토 정책이 그대로 적용됩니다."
        wide
      >
        {chosen && (
          <>
            <p>
              <strong>{chosen.arguments.title}</strong>
            </p>
            <p className="muted">
              {chosen.arguments.document_id
                ? `대상 버전 ${chosen.arguments.expected_version} · ${chosen.arguments.document_id}`
                : `새 문서 · ${chosen.arguments.visibility === "private" ? "개인 문서" : "워크스페이스 문서"}`}
            </p>
            <pre className="agent-plan-body">{chosen.arguments.markdown}</pre>
            <label className="agent-check">
              <input
                type="checkbox"
                checked={confirm}
                onChange={(e) => setConfirm(e.target.checked)}
              />
              대상과 본문을 읽었으며 이 변경 한 건을 저장하는 데 동의합니다.
            </label>
            <div className="agent-title">
              <Button disabled={busy} onClick={() => void decide(true)}>
                이 변경 거절
              </Button>
              <Button
                variant="primary"
                disabled={busy || !confirm}
                onClick={() => void decide(false)}
              >
                확인한 변경 저장
              </Button>
            </div>
          </>
        )}
      </Modal>
    </div>
  );
}
