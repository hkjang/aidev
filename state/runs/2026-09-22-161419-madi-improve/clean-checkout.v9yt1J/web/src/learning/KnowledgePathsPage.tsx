import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  ArrowDown,
  ArrowUp,
  BookOpen,
  Plus,
  RefreshCw,
  Trash2,
} from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
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
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import { ApprovalReview } from "../approval/ApprovalPanel";
import "./style.css";

type Kind = "read" | "practice" | "review";
type Step = {
  id?: string;
  document_id: string;
  kind: Kind;
  title: string;
  instruction: string;
  ordinal?: number;
  document_version?: number;
  document_title?: string;
  available?: boolean;
  current?: boolean;
  omitted?: boolean;
  can_confirm?: boolean;
  proof?: string;
  proof_hidden?: boolean;
  progress?: {
    id: string;
    revision: number;
    state: string;
    approval_id: string;
    document_version: number;
    updated_at: string;
  };
};
type Path = {
  id?: string;
  workspace_id: string;
  space_id: string;
  owner_id?: string;
  title: string;
  description: string;
  role_labels: string[];
  visibility: string;
  archived: boolean;
  revision: number;
  steps: Step[];
  confirm_shared?: boolean;
};
type Detail = {
  path: Path;
  steps: Step[];
  total: number;
  completed: number;
  omitted_reviews: number;
  review_policy_configured: boolean;
  can_manage: boolean;
  notice: string;
};
type Review = {
  path_id: string;
  approval_id: string;
  proof: string;
  preceding_records: {
    step_id: string;
    document_id: string;
    document_version: number;
    title: string;
    proof: string;
  }[];
  notice: string;
};
const kinds: Record<Kind, string> = {
  read: "읽기 확인",
  practice: "실습 결과",
  review: "독립 검토",
};
const states: Record<string, string> = {
  confirmed: "직접 확인",
  pending_review: "검토 대기",
  approved: "승인 완료",
  rejected: "반려",
  needs_recheck: "재확인 필요",
};
const blankStep = (): Step => ({
  document_id: "",
  kind: "read",
  title: "",
  instruction: "",
});
export default function KnowledgePathsPage() {
  const { workspace, user, documents, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const selected = params.get("path") || "",
    reviewID = params.get("review") || "";
  const scope = `${user.id}:${workspace?.id || ""}`;
  const identity = useRef(scope);
  identity.current = scope;
  const route = useRef(`${scope}:${selected}:${reviewID}`);
  route.current = `${scope}:${selected}:${reviewID}`;
  const mounted = useRef(true);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const [items, setItems] = useState<Path[]>([]),
    [detail, setDetail] = useState<Detail | null>(null),
    [review, setReview] = useState<Review | null>(null),
    [error, setError] = useState<unknown>(null),
    [loading, setLoading] = useState(true),
    [busy, setBusy] = useState(false);
  const [draft, setDraft] = useState<Path | null>(null),
    [roleText, setRoleText] = useState(""),
    [configReview, setConfigReview] = useState(false),
    [confirm, setConfirm] = useState<Step | null>(null),
    [proof, setProof] = useState(""),
    [approval, setApproval] = useState("");
  const [history, setHistory] = useState<Record<string, any>[] | null>(null),
    [spaces, setSpaces] = useState<{ id: string; name: string }[]>([]),
    [showArchived, setShowArchived] = useState(false),
    [filter, setFilter] = useState("");
  const [resultScope, setResultScope] = useState(""),
    [loadedSelection, setLoadedSelection] = useState("");
  const operation = useRef(0);
  const canManage =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const alive = (s: string) => mounted.current && identity.current === s;
  const refresh = useCallback(async () => {
    if (!workspace) return;
    const token = ++operation.current,
      expected = route.current;
    setLoading(true);
    setError(null);
    try {
      const [list, d, r] = await Promise.all([
        api<{ items: Path[] }>(
          `/knowledge-paths?workspace_id=${workspace.id}${showArchived ? "&archived=1" : ""}`,
        ),
        selected
          ? api<Detail>(`/knowledge-paths/${selected}`)
          : Promise.resolve(null),
        reviewID
          ? api<Review>(`/learning-progress/${reviewID}/review`)
          : Promise.resolve(null),
      ]);
      if (
        !mounted.current ||
        token !== operation.current ||
        expected !== route.current
      )
        return;
      setItems(list.items);
      setDetail(d);
      setReview(r);
      setResultScope(scope);
      setLoadedSelection(`${selected}:${reviewID}`);
    } catch (e) {
      if (
        mounted.current &&
        token === operation.current &&
        expected === route.current
        ) {
          setError(e);
          setItems([]);
          setDetail(null);
          setReview(null);
      }
    } finally {
      if (
        mounted.current &&
        token === operation.current &&
        expected === route.current
      )
        setLoading(false);
    }
  }, [workspace?.id, selected, reviewID, showArchived, scope]);
  useEffect(() => {
    setItems([]);
    setDetail(null);
    setReview(null);
    setDraft(null);
    setConfirm(null);
    setProof("");
    setHistory(null);
    setApproval("");
    setBusy(false);
    setConfigReview(false);
    setFilter("");
    setSpaces([]);
    setResultScope("");
  }, [scope]);
  useEffect(() => {
    void refresh();
    return () => {
      operation.current++;
    };
  }, [refresh]);
  useEffect(() => {
    setDraft(null);
    setConfirm(null);
    setProof("");
    setHistory(null);
    setApproval("");
    setBusy(false);
  }, [selected, reviewID]);
  useEffect(() => {
    if (!reviewID) return;
    let active = true;
    const expected = route.current;
    const timer = setInterval(() => {
      api<Review>(`/learning-progress/${reviewID}/review`)
        .then((v) => {
          if (active && route.current === expected) setReview(v);
        })
        .catch((e) => {
          if (active && route.current === expected) {
            setReview(null);
            setApproval("");
            setError(e);
          }
        });
    }, 5000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [scope, selected, reviewID]);
  useEffect(() => {
    if (!workspace) return;
    let active = true;
    api<{ id: string; name: string }[]>(`/spaces?workspace_id=${workspace.id}`)
      .then((v) => {
        if (active) setSpaces(v);
      })
      .catch(() => {
        if (active) setSpaces([]);
      });
    return () => {
      active = false;
    };
  }, [scope]);
  useEffect(() => {
    if (!draft && !confirm) return;
    const warn = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [draft, confirm]);
  const run = async (action: (valid: () => boolean) => Promise<void>) => {
    const s = scope,
      key = route.current;
    setBusy(true);
    setError(null);
    const valid = () => alive(s) && route.current === key;
    try {
      await action(valid);
    } catch (e) {
      if (valid()) setError(e);
    } finally {
      if (valid()) setBusy(false);
    }
  };
  const closeDraft = () => {
    if (busy) return;
    if (!window.confirm("저장하지 않은 경로 구성을 닫을까요?")) return;
    setDraft(null);
    setConfigReview(false);
  };
  const openNew = () => {
    if (!workspace || busy) return;
    setRoleText("");
    setDraft({
      workspace_id: workspace.id,
      space_id: "",
      title: "",
      description: "",
      role_labels: [],
      visibility: "private",
      archived: false,
      revision: 0,
      steps: [blankStep()],
    });
    setConfigReview(false);
  };
  const openEdit = () => {
    if (!detail || busy) return;
    setRoleText(detail.path.role_labels.join(", "));
    setDraft({
      ...detail.path,
      steps: detail.steps.map(
        ({ id, document_id, kind, title, instruction }) => ({
          id,
          document_id,
          kind,
          title,
          instruction,
        }),
      ),
      confirm_shared: false,
    });
    setConfigReview(false);
  };
  const matches =
    resultScope === scope && loadedSelection === `${selected}:${reviewID}`;
  const active = matches ? detail : null;
  const updateStep = (index: number, value: Partial<Step>) =>
    setDraft((old) =>
      old
        ? {
            ...old,
            steps: old.steps.map((x, i) =>
              i === index ? { ...x, ...value } : x,
            ),
          }
        : old,
    );
  const move = (index: number, delta: number) =>
    setDraft((old) => {
      if (!old) return old;
      const steps = [...old.steps];
      [steps[index], steps[index + delta]] = [
        steps[index + delta],
        steps[index],
      ];
      return { ...old, steps };
    });
  return (
    <section className="learning-page">
      <PageHeading
        eyebrow="ROLE-BASED KNOWLEDGE PATHS"
        title="지식 경로"
        description="역할별 원문을 순서대로 읽고 직접 실습합니다. 독립 검토와 개인 진행은 문서 읽음과 구분합니다."
        actions={
          <>
            <Button onClick={() => void refresh()} disabled={busy}>
              <RefreshCw size={17} />
              새로고침
            </Button>
            {canManage && (
              <Button variant="primary" onClick={openNew}>
                <Plus size={17} />
                경로 만들기
              </Button>
            )}
          </>
        }
      />
      <RecoveryNotice
        error={error}
        busy={busy}
        onReview={() => void refresh()}
        onRetry={() => void refresh()}
      />
      <div className="learning-layout">
        <aside className="panel learning-list">
          <Field label="경로·역할 검색">
            <input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="예: 운영자"
            />
          </Field>
          <label className="check-line">
            <input
              type="checkbox"
              checked={showArchived}
              onChange={(e) => setShowArchived(e.target.checked)}
            />
            보관 경로 포함
          </label>
          {resultScope === scope &&
            items
              .filter((p) =>
                (p.title + " " + p.role_labels.join(" ")).includes(filter),
              )
              .map((p) => (
                <Link
                  key={p.id}
                  className={`learning-path-link ${selected === p.id ? "active" : ""}`}
                  to={`?path=${p.id}`}
                >
                  <BookOpen size={19} />
                  <span>
                    <strong>{p.title}</strong>
                    <small>
                      {p.role_labels.join(" · ") || "역할 미지정"} ·{" "}
                      {p.visibility === "private" ? "비공개" : "공유"}
                      {p.archived ? " · 보관" : ""}
                    </small>
                  </span>
                </Link>
              ))}
          {!loading && items.length === 0 && (
            <p className="muted">접근 가능한 경로가 없습니다.</p>
          )}
        </aside>
        <main className="learning-content">
          {loading ? (
            <Loading />
          ) : active ? (
            <>
              <header className="panel learning-summary">
                <div>
                  <Badge>
                    {active.path.visibility === "private"
                      ? "비공개 경로"
                      : "공유 경로"}
                  </Badge>
                  <h2>{active.path.title}</h2>
                  <p>{active.path.description}</p>
                  <p>{active.path.role_labels.join(" · ")}</p>
                </div>
                <div className="learning-progress">
                  <strong>
                    {active.completed} / {active.total}
                  </strong>
                  <span>현재 유효한 완료</span>
                  <progress
                    value={active.completed}
                    max={Math.max(active.total, 1)}
                  />
                </div>
                <div className="learning-actions">
                  {active.can_manage && (
                    <Button onClick={openEdit}>경로 구성</Button>
                  )}
                  <Button
                    onClick={() =>
                      void run(async (valid) => {
                        const v = await api<{ items: Record<string, any>[] }>(
                          `/knowledge-paths/${selected}/history`,
                        );
                        if (valid()) setHistory(v.items);
                      })
                    }
                  >
                    개인 이력
                  </Button>
                </div>
              </header>
              <p className="notice">{active.notice}</p>
              {active.omitted_reviews > 0 && (
                <p className="notice">
                  명시적인 실습 검토 승인 정책이 없어 검토{" "}
                  {active.omitted_reviews}개를 분모에서 제외했습니다. 이는 검토
                  승인 완료가 아닙니다.
                </p>
              )}
              <ol className="learning-steps">
                {active.steps.map((step, i) => (
                  <li key={step.id} className="panel learning-step">
                    <div className="learning-step-heading">
                      <span className="learning-number">{i + 1}</span>
                      <div>
                        <Badge>{kinds[step.kind]}</Badge>
                        <h3>
                          {step.available
                            ? step.title
                            : "현재 접근할 수 없는 단계"}
                        </h3>
                      </div>
                      <Badge>
                        {step.omitted
                          ? "정책 미설정 · 제외"
                          : step.current
                            ? "현재 완료"
                            : step.progress?.id
                              ? "재확인 / " +
                                (states[step.progress.state] ||
                                  step.progress.state)
                              : "미완료"}
                      </Badge>
                    </div>
                    {step.available ? (
                      <>
                        <p className="learning-instruction">
                          {step.instruction}
                        </p>
                        <Link to={`/app/documents/${step.document_id}`}>
                          원문: {step.document_title} · v{step.document_version}
                        </Link>
                        {step.proof && (
                          <details>
                            <summary>내가 남긴 실습 기록</summary>
                            <pre>{step.proof}</pre>
                          </details>
                        )}
                        {step.proof_hidden && (
                          <p>
                            현재 정보 보호 정책으로 과거 실습 원문을 표시하지
                            않습니다.
                          </p>
                        )}
                        <div className="learning-actions">
                          <Button
                            disabled={!step.can_confirm || busy}
                            onClick={() => {
                              setConfirm(step);
                              setProof(step.proof || "");
                            }}
                          >
                            {step.kind === "review"
                              ? "독립 검토 요청"
                              : step.current
                                ? "다시 확인"
                                : kinds[step.kind]}
                          </Button>
                          {step.progress?.approval_id && (
                            <Link
                              className="button"
                              to={`?path=${selected}&review=${step.progress.id}`}
                            >
                              제출 기록·검토 상태
                            </Link>
                          )}
                        </div>
                      </>
                    ) : (
                      <p className="muted">
                        원문 권한은 경로 공유와 별개입니다. 접근을 허용받기
                        전에는 다음 단계를 완료할 수 없습니다.
                      </p>
                    )}
                  </li>
                ))}
              </ol>
            </>
          ) : (
            <Empty
              title="역할에 맞는 경로를 선택하세요"
              text="경로는 기존 원문과 실습을 안내할 뿐 권한이나 실행을 자동으로 부여하지 않습니다."
            />
          )}
        </main>
      </div>
      <Modal
        open={resultScope === scope && !!draft}
        onOpenChange={(v) => {
          if (!v) closeDraft();
        }}
        title={draft?.id ? "지식 경로 구성" : "지식 경로 만들기"}
        wide
      >
        {draft && (
          <>
            <ErrorBox error={error} />
            {configReview ? (
              <ChangeReview
                title="경로 변경 확인"
                description="경로 구성을 저장하면 기존 개인 완료는 새 경로 버전에서 재확인이 필요합니다."
                changes={[
                  {
                    label: "경로",
                    before: active?.path.title,
                    after: draft.title,
                  },
                  {
                    label: "공유 범위",
                    before: active?.path.visibility,
                    after:
                      draft.visibility === "private"
                        ? "소유자만"
                        : "현재 대상 공간 구성원",
                  },
                  {
                    label: "순서",
                    after: draft.steps.map((x, i) => (
                      <p key={i}>
                        {i + 1}. {kinds[x.kind]} · {x.title}
                      </p>
                    )),
                  },
                ]}
                warnings={[
                  "경로 안내를 공유해도 참조 문서의 접근 권한은 변경되지 않습니다.",
                  "실습 검토는 관리자가 명시적으로 설정한 승인 정책이 있을 때만 적용됩니다.",
                ]}
                confirmLabel="경로 저장"
                busy={busy}
                disabled={
                  draft.visibility === "workspace" && !draft.confirm_shared
                }
                onCancel={() => setConfigReview(false)}
                onConfirm={() =>
                  void run(async (valid) => {
                    const saved = await api<{ id: string }>(
                      draft.id
                        ? `/knowledge-paths/${draft.id}`
                        : "/knowledge-paths",
                      draft.id ? "PUT" : "POST",
                      draft,
                    );
                    if (!valid()) return;
                    setDraft(null);
                    setConfigReview(false);
                    notify("경로 구성을 저장했습니다");
                    if (saved.id === selected) await refresh();
                    else setParams({ path: saved.id });
                  })
                }
              >
                {draft.visibility === "workspace" && (
                  <label className="check-line">
                    <input
                      type="checkbox"
                      disabled={busy}
                      checked={!!draft.confirm_shared}
                      onChange={(e) =>
                        setDraft({ ...draft, confirm_shared: e.target.checked })
                      }
                    />
                    경로 제목·설명·단계 안내를 선택한 범위와 공유합니다.
                  </label>
                )}
              </ChangeReview>
            ) : (
              <form
                onSubmit={(e) => {
                  e.preventDefault();
                  setDraft({
                    ...draft,
                    role_labels: roleText
                      .split(",")
                      .map((x) => x.trim())
                      .filter(Boolean),
                  });
                  setConfigReview(true);
                }}
              >
                <fieldset disabled={busy}>
                  <Field label="경로 이름">
                    <input
                      required
                      maxLength={250}
                      value={draft.title}
                      onChange={(e) =>
                        setDraft({ ...draft, title: e.target.value })
                      }
                    />
                  </Field>
                  <Field label="경로 설명">
                    <textarea
                      maxLength={4000}
                      value={draft.description}
                      onChange={(e) =>
                        setDraft({ ...draft, description: e.target.value })
                      }
                    />
                  </Field>
                  <Field
                    label="역할 이름"
                    hint="쉼표로 구분합니다. 안내용이며 권한 그룹이 아닙니다."
                  >
                    <input
                      value={roleText}
                      onChange={(e) => setRoleText(e.target.value)}
                    />
                  </Field>
                  <Field label="경로 공유">
                    <select
                      value={draft.visibility}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          visibility: e.target.value,
                          confirm_shared: false,
                        })
                      }
                    >
                      <option value="private">비공개</option>
                      <option value="workspace">
                        워크스페이스 / 공간 공유
                      </option>
                    </select>
                  </Field>
                  <Field label="대상 공간">
                    <select
                      value={draft.space_id}
                      onChange={(e) =>
                        setDraft({
                          ...draft,
                          space_id: e.target.value,
                          confirm_shared: false,
                        })
                      }
                    >
                      <option value="">워크스페이스 전체</option>
                      {spaces.map((s) => (
                        <option key={s.id} value={s.id}>
                          {s.name}
                        </option>
                      ))}
                    </select>
                  </Field>
                  <label className="check-line">
                    <input
                      type="checkbox"
                      checked={draft.archived}
                      onChange={(e) =>
                        setDraft({ ...draft, archived: e.target.checked })
                      }
                    />
                    경로 보관 · 새 완료 확인 중단
                  </label>
                  <datalist id="learning-document-options">
                    {documents.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.title}
                      </option>
                    ))}
                  </datalist>
                  {draft.steps.map((step, i) => (
                    <section
                      className="learning-editor-step"
                      key={step.id || i}
                    >
                      <h3>단계 {i + 1}</h3>
                      <Field
                        label={`${i + 1}단계 문서 ID`}
                        hint="현재 문서 목록에서 고르거나 기존 문서 UUID를 붙여 넣으세요."
                      >
                        <input
                          required
                          readOnly={!!step.id}
                          list="learning-document-options"
                          value={step.document_id}
                          onChange={(e) =>
                            updateStep(i, {
                              document_id: e.target.value.trim(),
                            })
                          }
                        />
                      </Field>
                      <Field label={`${i + 1}단계 유형`}>
                        <select
                          disabled={!!step.id}
                          value={step.kind}
                          onChange={(e) =>
                            updateStep(i, { kind: e.target.value as Kind })
                          }
                        >
                          {Object.entries(kinds).map(([key, label]) => (
                            <option key={key} value={key}>
                              {label}
                            </option>
                          ))}
                        </select>
                      </Field>
                      <Field label={`${i + 1}단계 이름`}>
                        <input
                          required
                          maxLength={250}
                          value={step.title}
                          onChange={(e) =>
                            updateStep(i, { title: e.target.value })
                          }
                        />
                      </Field>
                      <Field label={`${i + 1}단계 안내`}>
                        <textarea
                          maxLength={4000}
                          value={step.instruction}
                          onChange={(e) =>
                            updateStep(i, { instruction: e.target.value })
                          }
                        />
                      </Field>
                      <div className="learning-actions">
                        <Button
                          type="button"
                          disabled={i === 0}
                          onClick={() => move(i, -1)}
                          aria-label={`${i + 1}단계 위로`}
                        >
                          <ArrowUp size={17} />
                        </Button>
                        <Button
                          type="button"
                          disabled={i === draft.steps.length - 1}
                          onClick={() => move(i, 1)}
                          aria-label={`${i + 1}단계 아래로`}
                        >
                          <ArrowDown size={17} />
                        </Button>
                        <Button
                          type="button"
                          disabled={draft.steps.length === 1}
                          onClick={() =>
                            setDraft({
                              ...draft,
                              steps: draft.steps.filter((_, n) => n !== i),
                            })
                          }
                        >
                          <Trash2 size={17} />
                          단계 제외
                        </Button>
                      </div>
                    </section>
                  ))}
                  <div className="learning-actions">
                    <Button
                      type="button"
                      disabled={draft.steps.length >= 100}
                      onClick={() =>
                        setDraft({
                          ...draft,
                          steps: [...draft.steps, blankStep()],
                        })
                      }
                    >
                      <Plus size={17} />
                      단계 추가
                    </Button>
                    <Button type="submit" variant="primary">
                      변경 검토
                    </Button>
                  </div>
                </fieldset>
              </form>
            )}
          </>
        )}
      </Modal>
      <Modal
        open={matches && !!confirm}
        onOpenChange={(v) => {
          if (!v && !busy) setConfirm(null);
        }}
        title={confirm ? kinds[confirm.kind] : "단계 확인"}
      >
        {confirm && active && (
          <>
            <ErrorBox error={error} />
            <ChangeReview
              title={confirm.title}
              changes={[
                {
                  label: "확인할 원문",
                  after: `${confirm.document_title} · v${confirm.document_version}`,
                },
                { label: "경로 버전", after: active.path.revision },
                {
                  label: "내 기록",
                  before: confirm.progress?.id
                    ? states[confirm.progress.state]
                    : "미완료",
                  after:
                    confirm.kind === "review"
                      ? "독립 검토 요청 · 완료 아님"
                      : "직접 확인 기록",
                },
              ]}
              warnings={[
                "문서가 열렸다는 사실만으로 완료 처리하지 않습니다.",
                "실습 기록은 내 개인 이력이며 독립 검토에 제출하면 지정 검토자가 현재 근거 권한 안에서 볼 수 있습니다.",
              ]}
              busy={busy}
              disabled={confirm.kind !== "read" && !proof.trim()}
              confirmLabel={
                confirm.kind === "review" ? "검토 요청 제출" : "직접 확인 완료"
              }
              onCancel={() => setConfirm(null)}
              onConfirm={() =>
                void run(async (valid) => {
                  const saved = await api<{
                    protection?: { changed: boolean; findings: unknown[] };
                  }>(
                    `/knowledge-paths/${selected}/steps/${confirm.id}/confirm`,
                    "POST",
                    {
                      path_revision: active.path.revision,
                      document_version: confirm.document_version,
                      revision: confirm.progress?.revision || 0,
                      proof,
                      confirmation: "CONFIRM",
                    },
                  );
                  if (!valid()) return;
                  setConfirm(null);
                  setProof("");
                  notify(
                    saved.protection?.changed
                      ? "정보보호 정책으로 정제한 개인 기록을 저장했습니다"
                      : saved.protection?.findings?.length
                        ? "개인 기록을 저장했습니다. 개인정보 탐지 경고를 확인하세요"
                        : "개인 진행을 기록했습니다",
                  );
                  await refresh();
                })
              }
            >
              {confirm.kind !== "read" && (
                <Field
                  label={
                    confirm.kind === "practice"
                      ? "직접 수행한 실습 결과"
                      : "독립 검토 요청 사유"
                  }
                  hint="16KiB 이하. 비밀값을 붙여 넣지 마세요. 정보보호 정책을 적용합니다."
                >
                    <textarea
                      rows={6}
                      disabled={busy}
                      value={proof}
                    onChange={(e) => setProof(e.target.value)}
                  />
                </Field>
              )}
            </ChangeReview>
          </>
        )}
      </Modal>
      <Modal
        open={!!reviewID}
        onOpenChange={(v) => {
          if (!v) setParams(selected ? { path: selected } : {});
        }}
        title="제출한 실습 원문 검토"
        wide
      >
        <ErrorBox error={error} />
        {review && matches ? (
          <div className="learning-review">
            <p className="notice">{review.notice}</p>
            {review.preceding_records.map((x) => (
              <section key={x.step_id}>
                <h3>{x.title}</h3>
                <Link to={`/app/documents/${x.document_id}`}>
                  근거 원문 v{x.document_version}
                </Link>
                <pre>{x.proof || "읽기 직접 확인 기록"}</pre>
              </section>
            ))}
            <h3>검토 요청 사유</h3>
            <pre>{review.proof}</pre>
            <Button
              variant="primary"
              onClick={() => setApproval(review.approval_id)}
            >
              승인 진행·결정 확인
            </Button>
          </div>
        ) : loading ? (
          <Loading />
        ) : (
          <p>
            현재 제출 기록을 다시 확인해야 합니다. 원문이 바뀌면 재제출 전의
            새로운 실습 기록을 표시하지 않습니다.
          </p>
        )}
      </Modal>
      {approval && (
        <ApprovalReview
          requestID={approval}
          onClose={() => setApproval("")}
          onChanged={() => {
            setApproval("");
            void refresh();
          }}
        />
      )}
      <Modal
        open={matches && history !== null}
        onOpenChange={(v) => {
          if (!v) setHistory(null);
        }}
        title="개인 진행 이력"
        wide
      >
        <p>
          과거 실습 원문은 현재 완료와 구분하며, 현재 근거 권한이 있을 때만
          표시합니다. 최근 200개입니다.
        </p>
        {history?.map((x) => (
          <section className="learning-history-entry" key={x.id}>
            <strong>
              {states[x.action] ||
                (
                  {
                    previous_progress: "이전 개인 기록 보관",
                    path_configured: "경로 구성 변경",
                  } as Record<string, string>
                )[x.action] ||
                x.action}
            </strong>
            <small>{datetime(x.created_at)}</small>
            {x.proof && <pre>{x.proof}</pre>}
            {!x.available && (
              <p className="muted">현재 접근 가능한 단계 원문 없음</p>
            )}
          </section>
        ))}
      </Modal>
    </section>
  );
}
