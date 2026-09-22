import { useCallback, useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ArrowDown, ArrowUp, Plus, RefreshCw, Trash2 } from "lucide-react";
import { api, type Settings } from "../api";
import { useApp } from "../context";
import {
  Button,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
  Toggle,
} from "../ui";
import { ApprovalReview } from "./ApprovalPanel";
import {
  type Policy,
  type Request,
  type Stage,
  kindNames,
  statusNames,
} from "./types";
import "./style.css";

export function ApprovalInboxPage() {
  const { publicInfo, workspace } = useApp();
  const [params, setParams] = useSearchParams(),
    selected = params.get("request") || "";
  const [items, setItems] = useState<Request[]>([]),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true);
  const load = useCallback(async () => {
    if (!publicInfo.approval_enabled) {
      setLoading(false);
      return;
    }
    try {
      setItems(
        await api<Request[]>(
          `/approvals/inbox${workspace ? `?workspace_id=${workspace.id}` : ""}`,
        ),
      );
      setError("");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setLoading(false);
    }
  }, [workspace?.id, publicInfo.approval_enabled]);
  useEffect(() => {
    void load();
    const timer = setInterval(() => void load(), 5000);
    return () => clearInterval(timer);
  }, [load]);
  if (!publicInfo.approval_enabled)
    return (
      <Empty
        title="검토 절차를 사용하지 않습니다"
        text="관리자가 검토·승인을 활성화하지 않아 별도의 검토 과정 없이 문서를 운영합니다."
      />
    );
  return (
    <div className="page-content">
      <PageHeading
        eyebrow="나의 업무"
        title="검토함"
        description="현재 단계에서 내 검토가 필요한 요청입니다."
        actions={
          <Button onClick={() => void load()}>
            <RefreshCw size={17} />
            새로고침
          </Button>
        }
      />
      {error && <ErrorBox error={error} />}{" "}
      {loading ? (
        <Loading />
      ) : items.length ? (
        <div className="approval-policy-list">
          {items.map((r) => (
            <article className="approval-inbox-card" key={r.id}>
              <div className="approval-summary">
                <strong>{r.title}</strong>
                <span className="badge">
                  {kindNames[r.resource_kind]} · {statusNames[r.status]}
                </span>
              </div>
              <p className="muted">
                {r.policy.name} · {r.current_stage + 1}/{r.policy.stages.length}
                단계 · 원본 v{r.resource_version}
              </p>
              {r.stale && <p className="notice">{r.reason}</p>}
              <div className="button-row">
                <Button onClick={() => setParams({ request: r.id })}>
                  검토 원본 열기
                </Button>
                {r.document_id && (
                  <Link to={`/app/documents/${r.document_id}`}>현재 문서</Link>
                )}
              </div>
            </article>
          ))}
        </div>
      ) : (
        <Empty
          title="지금 검토할 요청이 없습니다"
          text="현재 단계의 담당자로 지정되고 대상 조회 권한이 있는 요청이 여기에 표시됩니다."
        />
      )}
      {selected && (
        <ApprovalReview
          requestID={selected}
          onClose={() => setParams({})}
          onChanged={() => void load()}
        />
      )}
    </div>
  );
}

const newStage = (): Stage => ({
  name: "검토 및 승인",
  mode: "all",
  gates: [{ name: "검토 담당자", kind: "role", role: "admin" }],
});
export function ApprovalPolicyPage() {
  const { workspaces, workspace, notify, refreshPublic } = useApp();
  const [policies, setPolicies] = useState<Policy[]>([]),
    [settings, setSettings] = useState<Settings | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [draft, setDraft] = useState<Policy | null>(null);
  const [members, setMembers] = useState<Record<string, any>[]>([]),
    [teams, setTeams] = useState<Record<string, any>[]>([]),
    [spaces, setSpaces] = useState<Record<string, any>[]>([]),
    [targetsError, setTargetsError] = useState("");
  const load = useCallback(async () => {
    try {
      const [p, s] = await Promise.all([
        api<Policy[]>("/admin/approval/policies"),
        api<Settings>("/admin/settings"),
      ]);
      setPolicies(p);
      setSettings(s);
      setError("");
    } catch (e) {
      setError((e as Error).message);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  useEffect(() => {
    setMembers([]);
    setTeams([]);
    setSpaces([]);
    setTargetsError("");
    if (!draft?.workspace_id) return;
    let active = true;
    const wid = draft.workspace_id;
    Promise.all([
      api<Record<string, any>[]>(`/workspaces/${wid}/members`),
      api<Record<string, any>[]>(`/teams?workspace_id=${wid}`),
      api<Record<string, any>[]>(`/spaces?workspace_id=${wid}`),
      api<Record<string, any>[]>("/admin/users"),
    ])
      .then(([m, t, s, users]) => {
        if (active) {
          setMembers(
            m.filter((u) =>
              users.some(
                (account) =>
                  account.id === (u.user_id || u.id) &&
                  !account.disabled &&
                  account.kind === "user",
              ),
            ),
          );
          setTeams(t);
          setSpaces(s);
        }
      })
      .catch((e) => {
        if (active) setTargetsError(e.message);
      });
    return () => {
      active = false;
    };
  }, [draft?.workspace_id]);
  const run = async (fn: () => Promise<void>) => {
    setBusy(true);
    setError("");
    try {
      await fn();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const toggle = async (enabled: boolean) => {
    const previous = settings;
    setSettings((old) => (old ? { ...old, approval_enabled: enabled } : old));
    await run(async () => {
      try {
        await api("/admin/settings", "PUT", { approval_enabled: enabled });
        await load();
        await refreshPublic();
        notify(
          enabled
            ? "관리자가 설정한 승인 절차를 활성화했습니다."
            : "승인 절차를 사용하지 않도록 설정했습니다.",
        );
      } catch (e) {
        setSettings(previous);
        throw e;
      }
    });
  };
  const changeStage = (i: number, change: Partial<Stage>) =>
    setDraft((d) =>
      d
        ? {
            ...d,
            stages: d.stages.map((s, index) =>
              index === i ? { ...s, ...change } : s,
            ),
          }
        : d,
    );
  const move = (i: number, to: number) => {
    if (!draft || to < 0 || to >= draft.stages.length) return;
    const next = [...draft.stages];
    [next[i], next[to]] = [next[to], next[i]];
    setDraft({ ...draft, stages: next });
  };
  const create = () =>
    setDraft({
      id: "",
      workspace_id: workspace?.id || workspaces[0]?.id || "",
      space_id: "",
      resource_kind: "document",
      name: "문서 검토 정책",
      enabled: true,
      version: 1,
      stages: [newStage()],
    });
  return (
    <div className="page-content">
      <PageHeading
        eyebrow="서비스 관리"
        title="검토·승인 정책"
        description="관리자가 활성화한 경우에만 검토 절차가 적용됩니다. 사용자·팀·부서별 순차·병렬 승인을 설정하세요."
        actions={
          <Button onClick={create}>
            <Plus size={18} />
            정책 만들기
          </Button>
        }
      />
      {error && <ErrorBox error={error} />}
      {!settings ? (
        <Loading />
      ) : (
        <>
          <section className="approval-policy-card">
            <Toggle
              label="서비스 검토·승인 사용"
              checked={!!settings.approval_enabled}
              onChange={(enabled) => {
                if (!busy) void toggle(enabled);
              }}
            />
            <p className="muted">
              활성화 후 상세 정책이 없는 문서와 SQL 계획은 기본 검토 역할의 다른
              담당자 한 명이 승인합니다. 실행형 리소스는 명시적인 실행 승인
              정책이 필요합니다.
            </p>
            <Field label="기본 검토 역할">
              <select
                value={settings.reviewer_role || "admin"}
                disabled={busy}
                onChange={(e) => {
                  const reviewer_role = e.target.value;
                  const previous = settings;
                  setSettings((old) => (old ? { ...old, reviewer_role } : old));
                  void run(async () => {
                    try {
                      await api("/admin/settings", "PUT", { reviewer_role });
                      await load();
                      await refreshPublic();
                    } catch (e) {
                      setSettings(previous);
                      throw e;
                    }
                  });
                }}
              >
                <option value="admin">워크스페이스 관리자</option>
                <option value="editor">편집자 이상</option>
              </select>
            </Field>
            <p className="notice">
              승인 사용 여부·기본 역할·정책 변경 후 기존 대기 요청은 다시
              검토해야 합니다. 자신이 소유하거나 요청한 대상은 스스로 승인할 수
              없습니다.
            </p>
          </section>
          <div className="approval-policy-list">
            {policies.map((p) => (
              <article className="approval-policy-card" key={p.id}>
                <div className="approval-summary">
                  <strong>{p.name}</strong>
                  <span>
                    {p.enabled ? "적용" : "비활성"} · v{p.version}
                  </span>
                </div>
                <p>
                  {workspaces.find((w) => w.id === p.workspace_id)?.name ||
                    p.workspace_id}{" "}
                  · {kindNames[p.resource_kind]} ·{" "}
                  {p.space_id ? "선택 공간 및 하위 공간" : "워크스페이스 기본"}
                </p>
                <p className="muted">
                  {p.stages
                    .map(
                      (s) =>
                        `${s.name} (${s.mode === "all" ? "모든 대상" : "한 곳"})`,
                    )
                    .join(" → ")}
                </p>
                <div className="button-row">
                  <Button onClick={() => setDraft(structuredClone(p))}>
                    정책 편집
                  </Button>
                  {p.enabled && (
                    <Button
                      disabled={busy}
                      onClick={() =>
                        void run(async () => {
                          await api(
                            `/admin/approval/policies/${p.id}`,
                            "DELETE",
                          );
                          await load();
                          notify(
                            "정책을 비활성화했습니다. 상위 또는 기본 정책이 적용됩니다.",
                          );
                        })
                      }
                    >
                      비활성화
                    </Button>
                  )}
                </div>
              </article>
            ))}
          </div>
        </>
      )}
      <Modal
        open={!!draft}
        onOpenChange={(v) => {
          if (!v) setDraft(null);
        }}
        title={draft?.id ? "승인 정책 편집" : "승인 정책 만들기"}
        description="각 검토 대상은 후보 중 한 명이 승인합니다. 병렬 '모든 대상'은 모든 부서의 대표 승인을 요구합니다."
        wide
      >
        {draft && (
          <form
            className="approval-review"
            onSubmit={(e) => {
              e.preventDefault();
              void run(async () => {
                await api(
                  `/admin/approval/policies${draft.id ? `/${draft.id}` : ""}`,
                  draft.id ? "PUT" : "POST",
                  draft,
                );
                setDraft(null);
                await load();
                notify("승인 정책을 저장했습니다.");
              });
            }}
          >
            {error && <ErrorBox error={error} />}
            <Field label="정책 이름">
              <input
                required
                maxLength={250}
                value={draft.name}
                onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              />
            </Field>
            <div className="approval-grid">
              <Field label="워크스페이스">
                <select
                  required
                  disabled={!!draft.id}
                  value={draft.workspace_id}
                  onChange={(e) =>
                    setDraft({
                      ...draft,
                      workspace_id: e.target.value,
                      space_id: "",
                      stages: [newStage()],
                    })
                  }
                >
                  <option value="">선택하세요</option>
                  {workspaces.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="대상 유형">
                <select
                  disabled={!!draft.id}
                  value={draft.resource_kind}
                  onChange={(e) =>
                    setDraft({ ...draft, resource_kind: e.target.value })
                  }
                >
                  {Object.entries(kindNames).map(([value, name]) => (
                    <option key={value} value={value}>
                      {name}
                    </option>
                  ))}
                </select>
              </Field>
              <Field label="적용 공간">
                <select
                  disabled={!!draft.id}
                  value={draft.space_id || ""}
                  onChange={(e) =>
                    setDraft({ ...draft, space_id: e.target.value })
                  }
                >
                  <option value="">워크스페이스 전체 기본</option>
                  {spaces.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name}
                    </option>
                  ))}
                  {draft.space_id &&
                    !spaces.some((s) => s.id === draft.space_id) && (
                      <option value={draft.space_id}>기존 선택 공간</option>
                    )}
                </select>
              </Field>
              <Toggle
                label="이 정책 적용"
                checked={draft.enabled}
                onChange={(enabled) => setDraft({ ...draft, enabled })}
              />
            </div>
            {targetsError && <ErrorBox error={targetsError} />}
            {draft.stages.map((stage, si) => (
              <section className="approval-policy-stage" key={si}>
                <div className="approval-summary">
                  <strong>{si + 1}단계</strong>
                  <div className="button-row">
                    <Button
                      type="button"
                      aria-label={`${si + 1}단계 위로`}
                      disabled={si === 0}
                      onClick={() => move(si, si - 1)}
                    >
                      <ArrowUp size={16} />
                    </Button>
                    <Button
                      type="button"
                      aria-label={`${si + 1}단계 아래로`}
                      disabled={si === draft.stages.length - 1}
                      onClick={() => move(si, si + 1)}
                    >
                      <ArrowDown size={16} />
                    </Button>
                    <Button
                      type="button"
                      aria-label={`${si + 1}단계 삭제`}
                      disabled={draft.stages.length === 1}
                      onClick={() =>
                        setDraft({
                          ...draft,
                          stages: draft.stages.filter((_, i) => i !== si),
                        })
                      }
                    >
                      <Trash2 size={16} />
                    </Button>
                  </div>
                </div>
                <div className="approval-grid">
                  <Field label={`${si + 1}단계 이름`}>
                    <input
                      required
                      maxLength={250}
                      value={stage.name}
                      onChange={(e) =>
                        changeStage(si, { name: e.target.value })
                      }
                    />
                  </Field>
                  <Field label={`${si + 1}단계 승인 방식`}>
                    <select
                      value={stage.mode}
                      onChange={(e) =>
                        changeStage(si, {
                          mode: e.target.value as "any" | "all",
                        })
                      }
                    >
                      <option value="all">모든 검토 대상 승인 (병렬)</option>
                      <option value="any">검토 대상 중 한 곳 승인</option>
                    </select>
                  </Field>
                </div>
                {stage.gates.map((gate, gi) => {
                  const set = (changes: Partial<typeof gate>) =>
                    changeStage(si, {
                      gates: stage.gates.map((g, i) =>
                        i === gi ? { ...g, ...changes } : g,
                      ),
                    });
                  return (
                    <div className="approval-policy-gate" key={gi}>
                      <Field label={`${si + 1}단계 ${gi + 1}번 검토 대상 이름`}>
                        <input
                          required
                          maxLength={250}
                          value={gate.name}
                          onChange={(e) => set({ name: e.target.value })}
                        />
                      </Field>
                      <Field label={`${si + 1}단계 ${gi + 1}번 대상 유형`}>
                        <select
                          value={gate.kind}
                          onChange={(e) =>
                            set({ kind: e.target.value, id: "", role: "admin" })
                          }
                        >
                          <option value="role">워크스페이스 역할</option>
                          <option value="user">지정 사용자</option>
                          <option value="team">팀·부서 대표</option>
                          {draft.resource_kind !== "sql_query_plan" && (
                            <option value="document_reviewer">
                              문서 운영 속성의 검토자
                            </option>
                          )}
                        </select>
                      </Field>
                      {gate.kind === "role" && (
                        <Field label="검토 역할">
                          <select
                            value={gate.role || "admin"}
                            onChange={(e) => set({ role: e.target.value })}
                          >
                            <option value="admin">관리자 이상</option>
                            <option value="editor">편집자 이상</option>
                            <option value="commenter">댓글 작성자 이상</option>
                            <option value="viewer">조회자 이상</option>
                          </select>
                        </Field>
                      )}
                      {["user", "team"].includes(gate.kind) && (
                        <Field
                          label={
                            gate.kind === "user"
                              ? "검토 사용자"
                              : "검토 팀·부서"
                          }
                        >
                          <select
                            required
                            value={gate.id || ""}
                            onChange={(e) => set({ id: e.target.value })}
                          >
                            <option value="">선택하세요</option>
                            {(gate.kind === "user" ? members : teams).map(
                              (item) => (
                                <option
                                  key={item.user_id || item.id}
                                  value={item.user_id || item.id}
                                >
                                  {item.name}
                                </option>
                              ),
                            )}
                            {gate.id &&
                              !(gate.kind === "user" ? members : teams).some(
                                (item) => (item.user_id || item.id) === gate.id,
                              ) && (
                                <option value={gate.id}>
                                  현재 목록에 없는 기존 대상
                                </option>
                              )}
                          </select>
                        </Field>
                      )}
                      <Button
                        type="button"
                        disabled={stage.gates.length === 1}
                        onClick={() =>
                          changeStage(si, {
                            gates: stage.gates.filter((_, i) => i !== gi),
                          })
                        }
                      >
                        이 검토 대상 삭제
                      </Button>
                    </div>
                  );
                })}
                <Button
                  type="button"
                  disabled={stage.gates.length >= 20}
                  onClick={() =>
                    changeStage(si, {
                      gates: [
                        ...stage.gates,
                        { name: "추가 검토 대상", kind: "role", role: "admin" },
                      ],
                    })
                  }
                >
                  <Plus size={16} />
                  병렬 검토 대상 추가
                </Button>
              </section>
            ))}
            <div className="button-row">
              <Button
                type="button"
                disabled={draft.stages.length >= 20}
                onClick={() =>
                  setDraft({ ...draft, stages: [...draft.stages, newStage()] })
                }
              >
                다음 승인 단계 추가
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={busy || !!targetsError}
              >
                정책 저장
              </Button>
            </div>
          </form>
        )}
      </Modal>
    </div>
  );
}
