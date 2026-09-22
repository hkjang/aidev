import { useCallback, useEffect, useState } from "react";
import {
  Link,
  useNavigate,
  useParams,
  useSearchParams,
} from "react-router-dom";
import {
  Activity,
  ArrowLeft,
  CheckCircle2,
  History,
  RefreshCw,
  Save,
  ShieldCheck,
} from "lucide-react";
import { api, datetime, type Doc } from "./api";
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
import "./knowledge.css";

const flags: Record<string, string> = {
  stale: "검토 기한 경과",
  orphan: "연결 없는 문서",
  duplicate: "본문 중복",
  missing_owner: "소유자 비활성·권한 없음",
  broken_links: "미해결 링크",
  short: "내용 보강 필요",
  no_tags: "태그 없음",
};
const kinds: Record<string, string> = {
  page: "문서",
  note: "개인 노트",
  daily: "일일 노트",
  meeting: "회의록",
  decision: "의사결정 기록",
  runbook: "운영 절차서",
  template: "템플릿",
  inbox: "수집함",
  entity: "개체 페이지",
};
const classes: Record<string, string> = {
  public: "공개",
  internal: "내부",
  confidential: "기밀",
  restricted: "최고 기밀",
};
const parts: Record<string, string> = {
  completeness: "완성도",
  structure: "문서 구조",
  references: "참조",
  tags: "태그",
  ownership: "소유권",
  freshness: "최신성",
};
type HealthDocument = {
  id: string;
  title: string;
  owner_name: string;
  classification: string;
  legal_hold: boolean;
  quality: { score: number; flags: string[]; parts: Record<string, number> };
  unresolved_links: string[] | null;
};
type Health = {
  score: number;
  documents: HealthDocument[];
  counts: Record<string, number>;
  sample_size: number;
  truncated: boolean;
  method: string;
};
export function KnowledgeHealthPage() {
  const { workspace, user } = useApp();
  const [query, setQuery] = useSearchParams();
  const [health, setHealth] = useState<Health | null>(null),
    [analytics, setAnalytics] = useState<any>(null),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<HealthDocument | null>(null);
  const manager =
    !!workspace &&
    ["owner", "admin"].includes(workspace.role) &&
    user.role !== "viewer";
  const load = useCallback(async () => {
    if (!workspace) return;
    setBusy(true);
    setError(null);
    try {
      const [h, a] = await Promise.all([
        api<Health>(`/knowledge/health?workspace_id=${workspace.id}`),
        manager
          ? api(`/knowledge/analytics?workspace_id=${workspace.id}`)
          : Promise.resolve(null),
      ]);
      setHealth(h);
      setAnalytics(a);
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }, [workspace?.id, manager]);
  useEffect(() => {
    void load();
  }, [load]);
  const filtered =
    health?.documents.filter(
      (d) =>
        (!query.get("flag") || d.quality.flags.includes(query.get("flag")!)) &&
        (!query.get("q") ||
          `${d.title} ${d.owner_name}`
            .toLowerCase()
            .includes(query.get("q")!.toLowerCase())),
    ) || [];
  const setFilter = (key: string, value: string) =>
    setQuery((old) => {
      const p = new URLSearchParams(old);
      if (value) p.set(key, value);
      else p.delete(key);
      return p;
    });
  return (
    <div className="page knowledge-page">
      <PageHeading
        eyebrow="KNOWLEDGE HEALTH"
        title="지식 품질 관리"
        description="문서의 최신성, 연결과 소유권을 확인하고 팀의 지식을 함께 돌봅니다."
        actions={
          <Button onClick={() => void load()} disabled={busy}>
            <RefreshCw size={17} />
            다시 분석
          </Button>
        }
      />
      <ErrorBox error={error} />
      {!health ? (
        busy ? (
          <Loading />
        ) : (
          <Empty title="워크스페이스를 선택하세요" />
        )
      ) : (
        <>
          <div className="knowledge-summary">
            <section className="panel knowledge-score">
              <Activity size={28} />
              <span>우리 지식의 현재 상태</span>
              <strong>
                {health.score}
                <small>/ 100</small>
              </strong>
              <p>열람 가능한 문서 {health.sample_size.toLocaleString()}개</p>
            </section>
            <section className="panel knowledge-flags">
              <h2>지금 살펴볼 항목</h2>
              <div>
                {Object.entries(flags).map(([key, label]) => (
                  <button
                    key={key}
                    className={query.get("flag") === key ? "active" : ""}
                    onClick={() =>
                      setFilter("flag", query.get("flag") === key ? "" : key)
                    }
                  >
                    <span>{label}</span>
                    <strong>{health.counts[key] || 0}</strong>
                  </button>
                ))}
              </div>
            </section>
          </div>
          <p className="muted knowledge-method">
            {health.method}
            {health.truncated &&
              " 전체 규모가 표본 한도를 초과하여 최근 문서만 분석했습니다."}
          </p>
          <div className="knowledge-filters">
            <Field label="문서·소유자 검색">
              <input
                value={query.get("q") || ""}
                onChange={(e) => setFilter("q", e.target.value)}
                placeholder="살펴볼 문서를 찾아보세요"
              />
            </Field>
            <Field label="진단 항목">
              <select
                value={query.get("flag") || ""}
                onChange={(e) => setFilter("flag", e.target.value)}
              >
                <option value="">전체 문서</option>
                {Object.entries(flags).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <div className="panel knowledge-documents">
            {filtered.length === 0 ? (
              <Empty title="조건에 맞는 문서가 없습니다" />
            ) : (
              filtered.map((d) => (
                <article key={d.id}>
                  <button
                    className="knowledge-mini-score"
                    onClick={() => setSelected(d)}
                    aria-label={`${d.title} 품질 점수 상세`}
                  >
                    {d.quality.score}
                  </button>
                  <div className="knowledge-document-title">
                    <Link to={`/app/documents/${d.id}`}>{d.title}</Link>
                    <small>
                      {d.owner_name} ·{" "}
                      {classes[d.classification] || d.classification}
                      {d.legal_hold && " · 법적 보존"}
                    </small>
                    <div>
                      {d.quality.flags.map((f) => (
                        <Badge key={f}>{flags[f] || f}</Badge>
                      ))}
                    </div>
                  </div>
                  <Link
                    className="button"
                    to={`/app/documents/${d.id}/knowledge`}
                  >
                    운영 속성
                  </Link>
                </article>
              ))
            )}
          </div>
          {analytics && (
            <section className="panel knowledge-analytics">
              <h2>최근 30일 지식 활용</h2>
              <div className="knowledge-metrics">
                {[
                  ["활성 작성자", analytics.active_writers],
                  ["활성 조회자", analytics.active_readers],
                  ["문서 조회", analytics.reads],
                  ["문서 작성·변경", analytics.writes],
                ].map(([k, v]) => (
                  <div key={k}>
                    <span>{k}</span>
                    <strong>{Number(v).toLocaleString()}</strong>
                  </div>
                ))}
              </div>
              <p className="muted">{analytics.note}</p>
              <h3>검색 결과가 없었던 질문</h3>
              {analytics.zero_result_searches.length === 0 ? (
                <p className="muted">아직 검색 공백이 발견되지 않았습니다.</p>
              ) : (
                analytics.zero_result_searches.map((q: any) => (
                  <div className="knowledge-gap" key={q.query}>
                    <span>{q.query}</span>
                    <Badge>{q.count}회</Badge>
                  </div>
                ))
              )}
            </section>
          )}
        </>
      )}
      <Modal
        open={!!selected}
        onOpenChange={(v) => !v && setSelected(null)}
        title="문서 품질 점수"
        description="AI 평가가 아닌 공개된 규칙으로 계산합니다. 점수는 문서의 사실성을 보증하지 않습니다."
      >
        {selected && (
          <>
            <h3>{selected.title}</h3>
            {Object.entries(selected.quality.parts).map(([k, v]) => (
              <div className="knowledge-gap" key={k}>
                <span>{parts[k] || k}</span>
                <strong>{v}점</strong>
              </div>
            ))}
            {(selected.unresolved_links || []).length > 0 && (
              <p className="muted">
                미해결 링크: {selected.unresolved_links!.join(", ")}. 권한 밖
                문서는 존재 여부를 확인하지 않습니다.
              </p>
            )}
          </>
        )}
      </Modal>
    </div>
  );
}

type Meta = Record<string, any>;
function retentionLocal(value?: string) {
  if (!value) return "";
  const d = new Date(value);
  return new Date(d.getTime() - d.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
}
export function DocumentKnowledgePage() {
  const { id } = useParams(),
    navigate = useNavigate(),
    { notify, reload } = useApp();
  const [meta, setMeta] = useState<Meta | null>(null),
    [form, setForm] = useState<Meta>({}),
    [doc, setDoc] = useState<Doc | null>(null),
    [members, setMembers] = useState<Meta[]>([]),
    [history, setHistory] = useState<Meta[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [transfer, setTransfer] = useState(false);
  const load = useCallback(async () => {
    if (!id) return;
    setBusy(true);
    setError(null);
    try {
      const [m, d, h] = await Promise.all([
        api(`/documents/${id}/knowledge`),
        api<Doc>(`/documents/${id}`),
        api<Meta[]>(`/documents/${id}/knowledge/history`),
      ]);
      const people = await api<Meta[]>(`/workspaces/${d.workspace_id}/members`);
      setMeta(m);
      setDoc(d);
      setHistory(h);
      setMembers(people);
      setForm({ ...m, retain_until: retentionLocal(m.retain_until) });
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  }, [id]);
  useEffect(() => {
    setMeta(null);
    void load();
  }, [load]);
  const set = (key: string, value: any) =>
    setForm((f) => ({ ...f, [key]: value }));
  const save = async () => {
    if (!meta || !id) return;
    setBusy(true);
    setError(null);
    setTransfer(false);
    try {
      const payload: Meta = { version: meta.version };
      for (const key of [
        "kind",
        "classification",
        "owner_id",
        "reviewer_id",
        "maintainer_id",
        "review_period_days",
      ])
        payload[key] = form[key] ?? null;
      if (form.status !== meta.status) payload.status = form.status;
      if (meta.can_set_retention) {
        payload.legal_hold = !!form.legal_hold;
        payload.retain_until = form.retain_until
          ? new Date(form.retain_until).toISOString()
          : null;
      }
      const next = await api(`/documents/${id}/knowledge`, "PUT", payload);
      notify("문서 운영 속성을 저장했습니다");
      await reload();
      if (next.access_changed) {
        navigate("/app/documents");
        return;
      }
      await load();
    } catch (e) {
      setError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="page knowledge-page">
      <Link className="knowledge-back" to={`/app/documents/${id}`}>
        <ArrowLeft size={17} />
        문서로 돌아가기
      </Link>
      <PageHeading
        eyebrow="DOCUMENT OPERATIONS"
        title={doc?.title || "문서 운영 속성"}
        description="담당자, 문서 등급과 검토 주기를 관리합니다."
        actions={
          <Button onClick={() => void load()} disabled={busy}>
            <RefreshCw size={17} />
            현재 값 다시 불러오기
          </Button>
        }
      />
      <ErrorBox error={error} />
      {!meta ? (
        busy ? (
          <Loading />
        ) : null
      ) : (
        <>
          <section className="panel knowledge-policy">
            <div className="knowledge-policy-heading">
              <ShieldCheck />
              <div>
                <h2>소유권과 문서 정책</h2>
                <p className="muted">
                  문서 v{meta.version} · 마지막 최신성 확인:{" "}
                  {meta.last_reviewed_at
                    ? datetime(meta.last_reviewed_at)
                    : "아직 확인하지 않음"}
                </p>
              </div>
            </div>
            {!meta.can_manage && (
              <p className="notice">
                열람 전용입니다. 문서 소유자 또는 접근 가능한 워크스페이스
                관리자가 수정할 수 있습니다.
              </p>
            )}
            <form
              onSubmit={(e) => {
                e.preventDefault();
                if (form.owner_id !== meta.owner_id) setTransfer(true);
                else void save();
              }}
            >
              <fieldset
                disabled={!meta.can_manage || busy || !!doc?.deleted_at}
                className="knowledge-fieldset"
              >
                <div className="knowledge-form-grid">
                  <Field
                    label="문서 상태"
                    hint="승인 기능이 켜져 있으면 검토 절차를 통과해야 게시할 수 있습니다."
                  >
                    <select
                      value={form.status || "draft"}
                      onChange={(e) => set("status", e.target.value)}
                    >
                      <option value="draft">초안</option>
                      <option value="published">게시</option>
                      <option value="stale">검토 기한 경과</option>
                      <option value="archived">보관</option>
                      {["review", "rejected"].includes(form.status) && (
                        <option value={form.status}>
                          {form.status === "review" ? "검토 중" : "반려됨"}
                        </option>
                      )}
                    </select>
                  </Field>
                  <Field label="문서 종류">
                    <select
                      value={form.kind || "page"}
                      onChange={(e) => set("kind", e.target.value)}
                    >
                      {Object.entries(kinds).map(([k, v]) => (
                        <option key={k} value={k}>
                          {v}
                        </option>
                      ))}
                    </select>
                  </Field>
                  <Field
                    label="문서 등급"
                    hint="분류 표시는 공개 링크를 만들거나 문서 열람 권한을 변경하지 않습니다."
                  >
                    <select
                      value={form.classification || "internal"}
                      onChange={(e) => set("classification", e.target.value)}
                    >
                      {Object.entries(classes).map(([k, v]) => (
                        <option key={k} value={k}>
                          {v}
                        </option>
                      ))}
                    </select>
                  </Field>
                  {["owner_id", "reviewer_id", "maintainer_id"].map(
                    (key, i) => (
                      <Field
                        key={key}
                        label={
                          ["문서 소유자", "검토 담당자", "유지관리 담당자"][i]
                        }
                        hint={
                          i === 0
                            ? "비공개 문서 소유권을 넘기면 현재 사용자의 접근이 해제될 수 있습니다."
                            : "담당자 지정은 열람 권한을 새로 부여하지 않습니다."
                        }
                      >
                        <select
                          required={i === 0}
                          value={form[key] || ""}
                          onChange={(e) => set(key, e.target.value)}
                        >
                          {i !== 0 && <option value="">지정하지 않음</option>}
                          {form[key] &&
                            !members.some((m) => m.id === form[key]) && (
                              <option value={form[key]}>
                                {meta[key.replace("_id", "_name")] ||
                                  "현재 담당자"}
                              </option>
                            )}
                          {members
                            .filter((m) =>
                              ["owner", "admin", "editor"].includes(m.role),
                            )
                            .map((m) => (
                              <option key={m.id} value={m.id}>
                                {m.name} · {m.email}
                              </option>
                            ))}
                        </select>
                      </Field>
                    ),
                  )}
                  <Field label="문서 검토 주기 (일)">
                    <input
                      type="number"
                      min={1}
                      max={3650}
                      step={1}
                      required
                      value={form.review_period_days ?? 90}
                      onChange={(e) =>
                        set(
                          "review_period_days",
                          e.target.value === "" ? "" : Number(e.target.value),
                        )
                      }
                    />
                  </Field>
                </div>
                {meta.can_set_retention ? (
                  <div className="knowledge-retention">
                    <h3>보존 정책</h3>
                    <label className="checkbox-line">
                      <input
                        type="checkbox"
                        checked={!!form.legal_hold}
                        onChange={(e) => set("legal_hold", e.target.checked)}
                      />
                      법적 보존 — 해제하기 전까지 삭제 금지
                    </label>
                    <Field
                      label="최소 보존 기한"
                      hint="현재 브라우저의 현지 시각입니다. 기한 전에는 사용자 삭제와 휴지통 자동 정리를 모두 차단합니다."
                    >
                      <input
                        type="datetime-local"
                        value={form.retain_until || ""}
                        onChange={(e) => set("retain_until", e.target.value)}
                      />
                    </Field>
                  </div>
                ) : (
                  <p className="notice">
                    법적 보존: {meta.legal_hold ? "적용 중" : "미적용"} · 보존
                    기한:{" "}
                    {meta.retain_until ? datetime(meta.retain_until) : "미지정"}
                  </p>
                )}
                <div className="knowledge-form-actions">
                  <Button variant="primary" disabled={!meta.can_manage || busy}>
                    <Save size={17} />
                    {busy ? "저장 중…" : "운영 속성 저장"}
                  </Button>
                </div>
              </fieldset>
            </form>
            {meta.can_review && !doc?.deleted_at && (
              <div className="knowledge-review">
                <div>
                  <h3>내용을 점검하셨나요?</h3>
                  <p className="muted">
                    최신성 확인은 검토·승인 절차와 별개입니다. 문서 상태를 자동
                    게시하지 않습니다.
                  </p>
                </div>
                <Button
                  disabled={busy}
                  onClick={async () => {
                    setBusy(true);
                    try {
                      await api(`/documents/${id}/reviewed`, "POST");
                      notify("문서 최신성을 확인했습니다");
                      await load();
                    } catch (e) {
                      setError(e);
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  <CheckCircle2 size={17} />
                  오늘 최신성 확인
                </Button>
              </div>
            )}
          </section>
          <section className="panel knowledge-policy">
            <h2>
              <History size={21} /> 운영 속성 변경 이력
            </h2>
            {history.length === 0 ? (
              <p className="muted">아직 변경 이력이 없습니다.</p>
            ) : (
              history.map((h) => (
                <details key={h.id}>
                  <summary>
                    {h.action === "reviewed" ? "최신성 확인" : "정책 변경"} ·{" "}
                    {h.user_name} · {datetime(h.created_at)}
                  </summary>
                  <div className="knowledge-history-values">
                    {[
                      "kind",
                      "classification",
                      "owner_name",
                      "review_period_days",
                      "legal_hold",
                      "retain_until",
                    ].map((k) => (
                      <div key={k}>
                        <span>
                          {
                            (
                              {
                                kind: "종류",
                                classification: "등급",
                                owner_name: "소유자",
                                review_period_days: "검토 주기",
                                legal_hold: "법적 보존",
                                retain_until: "보존 기한",
                              } as Record<string, string>
                            )[k]
                          }
                        </span>
                        <code>
                          {String(h.before_data[k] ?? "미지정")} →{" "}
                          {String(h.after_data[k] ?? "변경 없음")}
                        </code>
                      </div>
                    ))}
                  </div>
                  {h.action === "policy" && meta.can_manage && (
                    <Button
                      onClick={() => {
                        const v = h.before_data;
                        setForm((f) => ({
                          ...f,
                          ...Object.fromEntries(
                            [
                              "kind",
                              "classification",
                              "owner_id",
                              "reviewer_id",
                              "maintainer_id",
                              "review_period_days",
                            ].map((k) => [k, v[k]]),
                          ),
                          ...(meta.can_set_retention
                            ? {
                                legal_hold: v.legal_hold,
                                retain_until: retentionLocal(v.retain_until),
                              }
                            : {}),
                        }));
                        notify(
                          "이전 값을 양식에 불러왔습니다. 확인 후 저장하세요",
                        );
                      }}
                    >
                      변경 전 값을 양식에 불러오기
                    </Button>
                  )}
                </details>
              ))
            )}
          </section>
        </>
      )}
      <Modal
        open={transfer}
        onOpenChange={setTransfer}
        title="문서 소유권을 이전할까요?"
        description="비공개 문서인 경우 현재 사용자의 열람·수정 권한이 해제될 수 있습니다. 새 소유자는 활성 워크스페이스 작성자이며 상위 공간과 문서에 접근할 수 있어야 합니다."
      >
        <p>
          새 소유자:{" "}
          {members.find((m) => m.id === form.owner_id)?.name || "선택한 사용자"}
        </p>
        <div className="knowledge-form-actions">
          <Button onClick={() => setTransfer(false)}>취소</Button>
          <Button variant="primary" onClick={() => void save()} disabled={busy}>
            소유권 이전 및 저장
          </Button>
        </div>
      </Modal>
    </div>
  );
}
