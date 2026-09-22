import { useCallback, useEffect, useState } from "react";
import { Plus, Save, ShieldCheck, Trash2 } from "lucide-react";
import { api, type Workspace } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading, PageHeading, Toggle } from "../ui";
import type { Action, Parameter, Runner } from "./types";
import "./style.css";
const blankAction = (): Action => ({
  id: "check",
  name: "안전 점검",
  roles: ["admin"],
  teams: [],
  parameters: [],
  timeout_seconds: 300,
  template_id: 0,
  image: "",
  argv: ["/usr/local/bin/check"],
  cpu_milli: 100,
  memory_mi: 128,
});
const blankRunner = (workspace_id: string): Runner => ({
  id: "",
  workspace_id,
  name: "",
  kind: "kubernetes",
  enabled: false,
  revision: 0,
  config: {
    base_url: "",
    token: "",
    ca_pem: "",
    namespace: "madi-execution",
    runtime_class: "",
  },
  actions: [blankAction()],
});
function TeamListInput({
  values,
  disabled,
  onChange,
}: {
  values: string[];
  disabled: boolean;
  onChange: (v: string[]) => void;
}) {
  const [text, setText] = useState(values.join(", "));
  const stable = values.join(",");
  useEffect(() => setText(values.join(", ")), [stable]);
  return (
    <input
      disabled={disabled}
      value={text}
      onChange={(e) => setText(e.target.value)}
      onBlur={() =>
        onChange(
          text
            .split(",")
            .map((x) => x.trim())
            .filter(Boolean),
        )
      }
    />
  );
}
function ActionEditor({
  action,
  kind,
  onChange,
  onDelete,
  disabled,
}: {
  action: Action;
  kind: Runner["kind"];
  onChange: (a: Action) => void;
  onDelete: () => void;
  disabled: boolean;
}) {
  const set = <K extends keyof Action>(key: K, value: Action[K]) =>
    onChange({ ...action, [key]: value });
  const parameter = (i: number, p: Parameter) =>
    set(
      "parameters",
      action.parameters.map((v, n) => (n === i ? p : v)),
    );
  return (
    <section className="runbook-step">
      <div className="runbook-step-heading">
        <h3>{action.name || "허용 작업"}</h3>
        <Button
          disabled={disabled}
          aria-label="허용 작업 삭제"
          onClick={onDelete}
        >
          <Trash2 size={16} />
        </Button>
      </div>
      <div className="runbook-grid">
        <Field label="작업 ID" hint="영문 소문자·숫자·밑줄·하이픈">
          <input
            disabled={disabled}
            value={action.id}
            onChange={(e) => set("id", e.target.value)}
          />
        </Field>
        <Field label="작업 이름">
          <input
            disabled={disabled}
            value={action.name}
            onChange={(e) => set("name", e.target.value)}
          />
        </Field>
        <Field label="실행 허용 역할">
          <select
            disabled={disabled}
            value={
              action.roles.includes("editor")
                ? "editor"
                : action.roles.includes("admin")
                  ? "admin"
                  : "team"
            }
            onChange={(e) =>
              set("roles", e.target.value === "team" ? [] : [e.target.value])
            }
          >
            <option value="admin">워크스페이스 관리자 이상</option>
            <option value="editor">워크스페이스 편집자 이상</option>
            <option value="team">아래 지정 팀만</option>
          </select>
        </Field>
        <Field
          label="추가 허용 팀 ID"
          hint="같은 워크스페이스의 팀 UUID · 쉼표 구분"
        >
          <TeamListInput
            disabled={disabled}
            values={action.teams}
            onChange={(v) => set("teams", v)}
          />
        </Field>
        <Field
          label="단계 제한 시간 (초)"
          hint="5~1,800초 · 전체 계획은 여유 시간을 포함해 1시간 이하"
        >
          <input
            disabled={disabled}
            type="number"
            min={5}
            max={1800}
            value={action.timeout_seconds}
            onChange={(e) => set("timeout_seconds", Number(e.target.value))}
          />
        </Field>
        {kind === "awx" ? (
          <Field label="AWX 고정 Job Template ID">
            <input
              disabled={disabled}
              type="number"
              min={1}
              value={action.template_id || ""}
              onChange={(e) => set("template_id", Number(e.target.value))}
            />
          </Field>
        ) : (
          <>
            <Field
              label="이미지 digest"
              hint="사내 registry/image@sha256:64자리 해시"
            >
              <input
                disabled={disabled}
                value={action.image}
                onChange={(e) => set("image", e.target.value)}
              />
            </Field>
            <Field label="CPU (millicore)">
              <input
                disabled={disabled}
                type="number"
                min={10}
                max={8000}
                value={action.cpu_milli}
                onChange={(e) => set("cpu_milli", Number(e.target.value))}
              />
            </Field>
            <Field label="메모리 (MiB)">
              <input
                disabled={disabled}
                type="number"
                min={16}
                max={16384}
                value={action.memory_mi}
                onChange={(e) => set("memory_mi", Number(e.target.value))}
              />
            </Field>
            <div className="runbook-span">
              <Field
                label="고정 argv · 한 줄에 한 인자"
                hint="첫 줄은 이미지 안의 절대 실행 경로입니다. 매개변수는 ${name} 단독 인자로만 사용합니다. sh -c 등 인라인 스크립트 실행은 허용하지 않습니다."
              >
                <textarea
                  disabled={disabled}
                  value={action.argv.join("\n")}
                  spellCheck={false}
                  onChange={(e) => set("argv", e.target.value.split("\n"))}
                />
              </Field>
            </div>
          </>
        )}
      </div>
      <h3>허용 매개변수</h3>
      {action.parameters.map((p, i) => (
        <div key={i} className="runbook-step">
          <div className="runbook-grid">
            <Field label="매개변수 이름">
              <input
                disabled={disabled}
                value={p.name}
                onChange={(e) => parameter(i, { ...p, name: e.target.value })}
              />
            </Field>
            <Field label="표시 이름">
              <input
                disabled={disabled}
                value={p.label}
                onChange={(e) => parameter(i, { ...p, label: e.target.value })}
              />
            </Field>
            <Field label="자료형">
              <select
                disabled={disabled}
                value={p.type}
                onChange={(e) =>
                  parameter(i, {
                    ...p,
                    type: e.target.value as Parameter["type"],
                  })
                }
              >
                <option value="enum">선택 목록</option>
                <option value="string">문자열 · 허용 정규식</option>
                <option value="integer">정수</option>
                <option value="boolean">예 / 아니오</option>
              </select>
            </Field>
            <Field label="필수 여부">
              <select
                disabled={disabled}
                value={String(p.required)}
                onChange={(e) =>
                  parameter(i, { ...p, required: e.target.value === "true" })
                }
              >
                <option value="true">필수</option>
                <option value="false">선택</option>
              </select>
            </Field>
            {p.type === "enum" && (
              <Field label="선택 옵션 · 한 줄에 하나">
                <textarea
                  disabled={disabled}
                  value={p.options.join("\n")}
                  onChange={(e) =>
                    parameter(i, { ...p, options: e.target.value.split("\n") })
                  }
                />
              </Field>
            )}
            {p.type === "string" && (
              <Field label="허용 정규식" hint="RE2 · 전체 문자열 일치">
                <input
                  disabled={disabled}
                  value={p.pattern}
                  spellCheck={false}
                  onChange={(e) =>
                    parameter(i, { ...p, pattern: e.target.value })
                  }
                />
              </Field>
            )}
            {p.type === "integer" && (
              <>
                <Field label="최솟값">
                  <input
                    disabled={disabled}
                    type="number"
                    value={p.min ?? ""}
                    onChange={(e) =>
                      parameter(i, {
                        ...p,
                        min:
                          e.target.value === ""
                            ? undefined
                            : Number(e.target.value),
                      })
                    }
                  />
                </Field>
                <Field label="최댓값">
                  <input
                    disabled={disabled}
                    type="number"
                    value={p.max ?? ""}
                    onChange={(e) =>
                      parameter(i, {
                        ...p,
                        max:
                          e.target.value === ""
                            ? undefined
                            : Number(e.target.value),
                      })
                    }
                  />
                </Field>
              </>
            )}
          </div>
          <Button
            disabled={disabled}
            onClick={() =>
              set(
                "parameters",
                action.parameters.filter((_, n) => n !== i),
              )
            }
          >
            매개변수 삭제
          </Button>
        </div>
      ))}
      <Button
        disabled={disabled || action.parameters.length >= 30}
        onClick={() =>
          set("parameters", [
            ...action.parameters,
            {
              name: "",
              label: "",
              type: "enum",
              required: true,
              options: [],
              pattern: "",
            },
          ])
        }
      >
        <Plus size={16} />
        매개변수 추가
      </Button>
    </section>
  );
}
export function RunbookAdminPage() {
  const { refreshPublic } = useApp();
  const [settings, setSettings] = useState<{
      enabled: boolean;
      revision: number;
    } | null>(null),
    [runners, setRunners] = useState<Runner[]>([]),
    [workspaces, setWorkspaces] = useState<Workspace[]>([]),
    [draft, setDraft] = useState<Runner | null>(null),
    [error, setError] = useState(""),
    [notice, setNotice] = useState(""),
    [busy, setBusy] = useState(false),
    [report, setReport] = useState<unknown>(null),
    [uncertain, setUncertain] = useState<
      { id: string; phase: string; created_at: string; steps: unknown[] }[]
    >([]),
    [resolveID, setResolveID] = useState(""),
    [resolveReason, setResolveReason] = useState(""),
    [resolveConfirmation, setResolveConfirmation] = useState(""),
    [externalChecked, setExternalChecked] = useState(false);
  const load = useCallback(async () => {
    try {
      const [cfg, rs, ws, unknown] = await Promise.all([
        api<{ enabled: boolean; revision: number }>("/admin/runbook/settings"),
        api<Runner[]>("/admin/runbook/runners"),
        api<Workspace[]>("/workspaces"),
        api<typeof uncertain>("/admin/runbook/uncertain"),
      ]);
      setSettings(cfg);
      setRunners(rs);
      setWorkspaces(ws);
      setUncertain(unknown);
      setError("");
    } catch (e) {
      setError((e as Error).message);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  const toggle = async (value: boolean) => {
    if (!settings) return;
    const previous = settings;
    setSettings({ ...settings, enabled: value });
    setBusy(true);
    try {
      setSettings(
        await api("/admin/runbook/settings", "PUT", {
          ...settings,
          enabled: value,
        }),
      );
      await refreshPublic();
    } catch (e) {
      setSettings(previous);
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const save = async () => {
    if (!draft) return;
    setBusy(true);
    setNotice("");
    setReport(null);
    try {
      const saved = await api<Runner>(
        `/admin/runbook/runners${draft.id ? "/" + draft.id : ""}`,
        draft.id ? "PUT" : "POST",
        draft,
      );
      await load();
      setDraft(saved);
      setNotice(
        "실행기 설정을 저장했습니다. 새로운 설정 버전으로만 실행 계획을 준비할 수 있습니다.",
      );
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const test = async () => {
    if (!draft?.id) return;
    setBusy(true);
    setReport(null);
    try {
      setReport(
        await api(`/admin/runbook/runners/${draft.id}/test`, "POST", {}),
      );
      setError("");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const config = (key: string, value: string) =>
    draft && setDraft({ ...draft, config: { ...draft.config, [key]: value } });
  const resolve = async () => {
    setBusy(true);
    try {
      await api(`/admin/runbook/executions/${resolveID}/resolve`, "POST", {
        confirmation: resolveConfirmation,
        reason: resolveReason,
        external_checked: externalChecked,
      });
      setResolveID("");
      setResolveReason("");
      setResolveConfirmation("");
      setExternalChecked(false);
      await load();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page-content">
      <div className="runbook-layout">
        <PageHeading
          eyebrow="서비스 관리"
          title="격리 실행 · Runbook"
          description="사내 AWX 또는 전용 Kubernetes 실행기를 연결하고, 허용할 명령과 실행 역할을 명시합니다."
        />
        {error && <ErrorBox error={error} />}
        <div className="runbook-warning">
          <ShieldCheck size={18} /> madi 호스트의 shell·Docker socket은 사용하지
          않습니다. 실행기 자격 증명은 암호화해 보관하며 브라우저에 다시
          공개하지 않습니다. 먼저 외부 격리 정책과 최소 권한을 검증하세요.
        </div>
        {settings ? (
          <section className="runbook-card">
            <fieldset
              disabled={busy}
              style={{ border: 0, padding: 0, margin: 0 }}
            >
              <Toggle
                label="격리 실행 사용"
                description="기본은 꺼짐입니다. 끄면 새로운 계획·실행이 차단되고 진행 중 작업은 중지를 시도합니다."
                checked={settings.enabled}
                onChange={(v) => void toggle(v)}
              />
            </fieldset>
          </section>
        ) : (
          <Loading />
        )}
        {!!uncertain.length && (
          <section className="runbook-card runbook-stack">
            <h2>외부 실행 확인 필요</h2>
            <p className="runbook-warning">
              아래 작업은 자동으로 재실행되지 않으며 같은 문서의 새로운 실행도
              차단됩니다. 외부 실행기에서 실행 추적
              ID(madi_execution_id)·시간·출력을 확인하고 미실행 또는 중지를 직접
              확인한 경우에만 수동 종결하세요. 실행 성공으로 기록하지 않습니다.
            </p>
            {uncertain.map((x) => (
              <div key={x.id} className="runbook-step">
                <strong>{x.id}</strong>
                <span className="runbook-meta">
                  {new Date(x.created_at).toLocaleString("ko-KR")}
                </span>
                <pre className="runbook-code">
                  {JSON.stringify(x.steps, null, 2)}
                </pre>
                <Button
                  onClick={() => {
                    setResolveID(x.id);
                    setResolveReason("");
                    setResolveConfirmation("");
                    setExternalChecked(false);
                  }}
                >
                  외부 확인 후 수동 종결
                </Button>
              </div>
            ))}
            {resolveID && (
              <div className="runbook-step">
                <h3>수동 종결 · {resolveID}</h3>
                <Field label="외부에서 확인한 근거 (20자 이상)">
                  <textarea
                    value={resolveReason}
                    minLength={20}
                    maxLength={2000}
                    onChange={(e) => setResolveReason(e.target.value)}
                  />
                </Field>
                <label>
                  <input
                    type="checkbox"
                    checked={externalChecked}
                    onChange={(e) => setExternalChecked(e.target.checked)}
                  />{" "}
                  외부 실행기에서 미실행 또는 중지를 직접 확인했습니다. 자동
                  검증이 아님을 이해합니다.
                </label>
                <Field
                  label="수동 종결 확인 문구"
                  hint={`RESOLVE ${resolveID}`}
                >
                  <input
                    value={resolveConfirmation}
                    autoComplete="off"
                    onChange={(e) => setResolveConfirmation(e.target.value)}
                  />
                </Field>
                <Button
                  disabled={
                    busy ||
                    !externalChecked ||
                    resolveReason.trim().length < 20 ||
                    resolveConfirmation !== `RESOLVE ${resolveID}`
                  }
                  onClick={() => void resolve()}
                >
                  수동 확인으로 취소 종결
                </Button>
              </div>
            )}
          </section>
        )}
        <section className="runbook-card runbook-stack">
          <div className="runbook-step-heading">
            <h2>등록된 실행기</h2>
            <Button
              disabled={busy || !workspaces.length}
              onClick={() => {
                setDraft(blankRunner(workspaces[0]?.id || ""));
                setReport(null);
                setNotice("");
              }}
            >
              <Plus size={17} />
              실행기 추가
            </Button>
          </div>
          <div className="runbook-runner-list">
            {runners.map((r) => (
              <Button
                key={r.id}
                disabled={busy}
                onClick={() => {
                  setDraft(structuredClone(r));
                  setReport(null);
                  setNotice("");
                }}
              >
                {r.name} · {r.enabled ? "사용" : "중지"} · v{r.revision}
              </Button>
            ))}
          </div>
          {!runners.length && (
            <p className="muted">아직 연결된 격리 실행기가 없습니다.</p>
          )}
        </section>
        {draft && (
          <section className="runbook-card runbook-stack">
            <h2>{draft.id ? "실행기 설정" : "새 실행기"}</h2>
            <div className="runbook-grid">
              <Field label="실행기 이름">
                <input
                  disabled={busy}
                  value={draft.name}
                  maxLength={200}
                  onChange={(e) => setDraft({ ...draft, name: e.target.value })}
                />
              </Field>
              <Field label="워크스페이스">
                <select
                  disabled={busy || !!draft.id}
                  value={draft.workspace_id}
                  onChange={(e) =>
                    setDraft({ ...draft, workspace_id: e.target.value })
                  }
                >
                  <option value="">워크스페이스 선택</option>
                  {workspaces.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="실행기 유형">
                <select
                  disabled={busy || !!draft.id}
                  value={draft.kind}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      kind: e.target.value as Runner["kind"],
                    })
                  }
                >
                  <option value="kubernetes">
                    Kubernetes · 제한된 컨테이너 Job
                  </option>
                  <option value="awx">AWX · 고정 Ansible Job Template</option>
                </select>
              </Field>
              <Field label="실행기 상태">
                <select
                  disabled={busy}
                  value={String(draft.enabled)}
                  onChange={(e) =>
                    setDraft({ ...draft, enabled: e.target.value === "true" })
                  }
                >
                  <option value="false">중지</option>
                  <option value="true">사용</option>
                </select>
              </Field>
              <Field
                label="HTTPS API 기본 주소"
                hint="인증서 검증 필수 · 내부 주소 사용 가능"
              >
                <input
                  disabled={busy}
                  type="url"
                  value={draft.config.base_url || ""}
                  placeholder="https://runner.internal"
                  onChange={(e) => config("base_url", e.target.value)}
                />
              </Field>
              <Field
                label="Bearer 자격 증명"
                hint={
                  draft.config.token_configured
                    ? "저장됨 · 비워 두면 기존 값 유지"
                    : "최소 권한의 실행기 API 토큰"
                }
              >
                <input
                  disabled={busy}
                  type="password"
                  autoComplete="new-password"
                  value={draft.config.token || ""}
                  onChange={(e) => config("token", e.target.value)}
                />
              </Field>
              <Field
                label="사내 CA 인증서 (PEM)"
                hint="공인 인증서를 사용하면 비워 둘 수 있습니다."
              >
                <textarea
                  disabled={busy}
                  spellCheck={false}
                  value={draft.config.ca_pem || ""}
                  onChange={(e) => config("ca_pem", e.target.value)}
                />
              </Field>
              {draft.kind === "kubernetes" && (
                <>
                  <Field
                    label="전용 Namespace"
                    hint="restricted/latest + 양방향 deny-all + ResourceQuota 필요"
                  >
                    <input
                      disabled={busy}
                      value={draft.config.namespace || ""}
                      onChange={(e) => config("namespace", e.target.value)}
                    />
                  </Field>
                  <Field
                    label="RuntimeClass (선택)"
                    hint="gVisor/Kata 등 사내 클러스터에 실제 등록된 이름"
                  >
                    <input
                      disabled={busy}
                      value={draft.config.runtime_class || ""}
                      onChange={(e) => config("runtime_class", e.target.value)}
                    />
                  </Field>
                </>
              )}
            </div>
            <div className="runbook-warning">
              {draft.kind === "kubernetes"
                ? "전용 namespace에는 양방향 차단 정책만 허용됩니다. 실행 Pod는 non-root·읽기 전용 루트·권한 제거·서비스 계정 토큰 미사용으로 생성합니다. CNI가 NetworkPolicy를 실제로 적용하는지 운영자가 별도로 확인해야 합니다."
                : "AWX Job Template의 timeout과 고정 SCM revision, 실행 시 프로젝트 업데이트 끄기를 확인하세요. 입력 변수가 있으면 Prompt on launch를 허용해야 합니다. 인벤토리·인증 정보·실행 노드 격리는 AWX 관리자의 책임입니다."}
            </div>
            <h2>명시적 허용 작업</h2>
            {draft.actions.map((a, i) => (
              <ActionEditor
                key={i}
                action={a}
                kind={draft.kind}
                disabled={busy}
                onChange={(v) =>
                  setDraft({
                    ...draft,
                    actions: draft.actions.map((x, n) => (n === i ? v : x)),
                  })
                }
                onDelete={() =>
                  setDraft({
                    ...draft,
                    actions: draft.actions.filter((_, n) => n !== i),
                  })
                }
              />
            ))}
            <Button
              disabled={busy || draft.actions.length >= 50}
              onClick={() =>
                setDraft({
                  ...draft,
                  actions: [
                    ...draft.actions,
                    {
                      ...blankAction(),
                      id: `action-${draft.actions.length + 1}`,
                    },
                  ],
                })
              }
            >
              <Plus size={17} />
              허용 작업 추가
            </Button>
            <div className="runbook-actions">
              <Button
                variant="primary"
                disabled={busy}
                onClick={() => void save()}
              >
                <Save size={17} />
                {busy ? "처리 중…" : "실행기 저장"}
              </Button>
              <Button disabled={busy || !draft.id} onClick={() => void test()}>
                저장된 설정으로 연결·격리 검사
              </Button>
            </div>
            {notice && <p className="notice">{notice}</p>}
            {report !== null && (
              <details open>
                <summary>
                  사전 검사 결과 · 원격 작업은 실행하지 않았습니다
                </summary>
                <pre className="runbook-code">
                  {JSON.stringify(report, null, 2)}
                </pre>
              </details>
            )}
          </section>
        )}
      </div>
    </div>
  );
}
