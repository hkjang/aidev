import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { api, datetime } from "../api";
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
import "./evidence.css";
const DocumentPreview = lazy(() => import("../review/DocumentPreview"));
const ImpactExceptions = lazy(() => import("./ImpactExceptions"));
type Target = {
  id: string;
  title: string;
  version: number;
  owner_id: string;
  kind: string;
  via: string;
  relation_type: string;
  depth: number;
  can_write: boolean;
};
type Impact = {
  source: { id: string; title: string; version: number; can_write: boolean };
  from: number;
  depth: number;
  categories: string[];
  documents: Target[];
  packages: {
    id: string;
    created_at: string;
    stale: boolean;
    source_version: number;
  }[];
  agents: { id: string; name: string; revision: number; reason: string }[];
  limited: boolean;
  notice: string;
  diff: {
    rows: {
      kind: string;
      text: string;
      old_line: number;
      new_line: number;
      count: number;
    }[];
    notice: string;
    truncated: boolean;
  };
};
type Review = {
  id: string;
  source_id: string;
  source_title: string;
  source_version: number;
  target_id: string;
  target_title: string;
  target_version: number;
  current_target_version: number;
  owner_id: string;
  owner_name: string;
  status: string;
  revision: number;
  stale: boolean;
  can_write: boolean;
  updated_at: string;
};
const kinds: Record<string, string> = {
  related: "관련",
  reference: "단순 참조",
  policy: "정책 의존",
  execution: "실행 의존",
  data: "데이터 의존",
};
const categories: Record<string, string> = {
  access: "문서 접근 조건 변경",
  metadata: "제목·태그 변경",
  number: "수치 변경 후보",
  procedure: "절차·권한 표현 변경 후보",
  wording_candidate: "표현 변경 후보 · 의미 검토 필요",
  incomplete: "분석 범위 제한",
};
const states: Record<string, string> = {
  pending: "검토 대기",
  no_impact: "영향 없음",
  needs_change: "수정 필요",
  done: "수정 완료",
  exception_requested: "예외 요청 · 승인 아님",
};
export default function KnowledgeImpactPage() {
  const { user, workspace, documents, notify, publicInfo } = useApp();
  const [params, setParams] = useSearchParams();
  const selected = params.get("document_id") || "";
  const [from, setFrom] = useState(1),
    [depth, setDepth] = useState(1),
    [impact, setImpact] = useState<Impact | null>(null),
    [reviews, setReviews] = useState<Review[]>([]),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [preview, setPreview] = useState<string | null>(null);
  const [exceptionReview, setExceptionReview] = useState<string | null>(null);
  const [editing, setEditing] = useState<Review | null>(null),
    [state, setState] = useState("needs_change"),
    [note, setNote] = useState(""),
    [history, setHistory] = useState<
      { revision: number; status: string; note: string; created_at: string }[]
    >([]);
  const generation = useRef(0);
  const currentScope = useRef("");
  currentScope.current = `${user.id}:${workspace?.id}:${selected}`;
  useEffect(() => {
    generation.current++;
    setImpact(null);
    setEditing(null);
    setPreview(null);
    setExceptionReview(null);
    setError("");
    setFrom(
      Math.max(1, (documents.find((d) => d.id === selected)?.version || 2) - 1),
    );
    setBusy(false);
  }, [user.id, workspace?.id, selected]);
  useEffect(() => {
    let active = true,
      pending = false;
    setReviews([]);
    if (!workspace) return;
    const load = async () => {
      if (pending) return;
      pending = true;
      try {
        const value = await api<Review[]>(
          `/knowledge/impact-reviews?workspace_id=${workspace.id}`,
        );
        if (active) {
          setReviews(value);
          setEditing((old) =>
            old && value.some((row) => row.id === old.id) ? old : null,
          );
          setExceptionReview((old) =>
            old && value.some((row) => row.id === old) ? old : null,
          );
        }
      } catch (e) {
        if (active) {
          setReviews([]);
          setEditing(null);
          setExceptionReview(null);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
      }
    };
    void load();
    const timer = setInterval(() => void load(), 3000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [user.id, workspace?.id]);
  useEffect(() => {
    if (!impact || !workspace) return;
    let active = true,
      pending = false;
    const check = async () => {
      if (pending) return;
      pending = true;
      try {
        const value = await api<{ valid: boolean }>(
          "/knowledge/impact-check",
          "POST",
          {
            workspace_id: workspace.id,
            documents: [impact.source, ...impact.documents].map((d) => ({
              id: d.id,
              version: d.version,
            })),
            packages: impact.packages.map((p) => p.id),
            agents: impact.agents.map((a) => ({
              id: a.id,
              revision: a.revision,
            })),
          },
        );
        if (active && !value.valid) {
          setImpact(null);
          setEditing(null);
          setError(
            "원문·연결 자료 또는 현재 접근 범위가 변경되었습니다. 다시 분석하세요.",
          );
        }
      } catch (e) {
        if (active) {
          setImpact(null);
          setEditing(null);
          setError((e as Error).message);
        }
      } finally {
        pending = false;
      }
    };
    const timer = setInterval(() => void check(), 2000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [impact, user.id, workspace?.id]);
  const perform = async (action: () => Promise<void>) => {
    if (busy) return;
    const request = generation.current;
    setBusy(true);
    setError("");
    try {
      await action();
    } catch (e) {
      if (request === generation.current) setError((e as Error).message);
    } finally {
      if (request === generation.current) setBusy(false);
    }
  };
  const analyze = () =>
    perform(async () => {
      const scope = currentScope.current;
      setImpact(null);
      const result = await api<Impact>(
        `/documents/${selected}/impact?from=${from}&depth=${depth}`,
      );
      if (scope === currentScope.current) setImpact(result);
    });
  const requestReview = (target: Target) =>
    perform(async () => {
      if (!impact) return;
      const scope = currentScope.current;
      await api(`/documents/${impact.source.id}/impact-reviews`, "POST", {
        source_version: impact.source.version,
        target_id: target.id,
        target_version: target.version,
        relation_type: target.relation_type,
      });
      if (scope === currentScope.current) {
        notify("대상 문서 소유자에게 영향 검토를 요청했습니다");
        setReviews(
          await api<Review[]>(
            `/knowledge/impact-reviews?workspace_id=${workspace?.id}`,
          ),
        );
      }
    });
  const openReview = (review: Review) =>
    perform(async () => {
      const scope = currentScope.current;
      const value = await api<typeof history>(
        `/knowledge/impact-reviews/${review.id}/history`,
      );
      if (scope === currentScope.current) {
        setEditing(review);
        setHistory(value);
        setState(review.status);
        setNote("");
      }
    });
  const saveReview = () =>
    perform(async () => {
      if (!editing) return;
      const scope = currentScope.current;
      await api(`/knowledge/impact-reviews/${editing.id}`, "PUT", {
        revision: editing.revision,
        target_version: editing.current_target_version,
        status: state,
        note,
      });
      if (scope === currentScope.current) {
        setEditing(null);
        setReviews(
          await api<Review[]>(
            `/knowledge/impact-reviews?workspace_id=${workspace?.id}`,
          ),
        );
        notify("영향 검토 기록을 저장했습니다");
      }
    });
  return (
    <div className="page evidence-page">
      <PageHeading
        title="문서 변경 영향"
        description="변경 원문에서 의존 자료·Runbook·Agent·내 지식 패키지까지 확인합니다."
      />
      <ErrorBox error={error} />
      <section className="card">
        <h2>변경 분석</h2>
        <Field label="변경된 원문">
          <select
            value={selected}
            onChange={(e) =>
              setParams(e.target.value ? { document_id: e.target.value } : {})
            }
          >
            <option value="">문서를 선택하세요</option>
            {documents.map((d) => (
              <option key={d.id} value={d.id}>
                {d.title} · v{d.version}
              </option>
            ))}
          </select>
        </Field>
        <div className="evidence-compare">
          <Field label="이전 원문 버전">
            <input
              type="number"
              min={1}
              max={2147483647}
              step={1}
              value={from}
              onChange={(e) => setFrom(Number(e.target.value))}
            />
          </Field>
          <Field label="역방향 의존 탐색 깊이">
            <select
              value={depth}
              onChange={(e) => setDepth(Number(e.target.value))}
            >
              {[1, 2, 3, 4, 5].map((n) => (
                <option key={n} value={n}>
                  {n}단계
                </option>
              ))}
            </select>
          </Field>
        </div>
        <Button
          variant="primary"
          disabled={!selected || busy}
          onClick={() => void analyze()}
        >
          현재 버전의 영향 분석
        </Button>
        <p className="muted">
          그래프에서 ‘이 문서 → 의존 대상’ 관계를 등록하세요. 검토 요청은 직접
          의존 문서 소유자가 두 원문을 현재 열람할 수 있을 때만 보냅니다.
        </p>
      </section>
      {impact && (
        <>
          <section className="card">
            <h2>
              {impact.source.title} · v{impact.from} → v{impact.source.version}
            </h2>
            <p className="notice">{impact.notice}</p>
            <p>
              {impact.categories.map((c) => categories[c] || c).join(" · ") ||
                "이 범위에서 감지한 원문 변경 없음"}
            </p>
            {impact.limited && (
              <p role="note">
                표시·검사 한도에 도달했습니다. 누락 없이 전체 의존을 검사한
                결과가 아닙니다.
              </p>
            )}
            <details>
              <summary>원문 변경 구간 확인</summary>
              <p>{impact.diff.notice}</p>
              <pre className="evidence-text">
                {impact.diff.rows
                  .map((row) =>
                    row.kind === "skip"
                      ? `… ${row.count}줄 생략 …\n`
                      : `${row.kind === "add" ? "+" : row.kind === "remove" ? "−" : " "} ${row.text}`,
                  )
                  .join("")}
              </pre>
            </details>
          </section>
          <h2>관련 문서와 Runbook</h2>
          {!impact.documents.length ? (
            <Empty
              title="표시할 영향 후보가 없습니다"
              text="현재 접근 가능한 등록 관계 범위입니다. 알려지지 않은 의존이 없다는 뜻은 아닙니다."
            />
          ) : (
            impact.documents.map((target) => (
              <section className="card" key={target.id}>
                <h3>
                  {target.kind === "runbook" ? "Runbook · " : ""}
                  {target.title}
                </h3>
                <p>
                  {kinds[target.relation_type]} · {target.depth}단계 · v
                  {target.version}
                </p>
                <div className="modal-actions">
                  <Button onClick={() => setPreview(target.id)}>
                    문서 미리보기
                  </Button>
                  <Link to={`/app/documents/${target.id}`}>문서 전체 열기</Link>
                  {publicInfo.approval_enabled &&
                    target.depth === 1 &&
                    impact.source.can_write && (
                      <Button
                        disabled={busy}
                        onClick={() => void requestReview(target)}
                      >
                        소유자에게 영향 검토 요청
                      </Button>
                    )}
                </div>
              </section>
            ))
          )}
          <section className="card">
            <h2>지식 범위가 연결된 Agent</h2>
            <p>
              원문이 Agent의 선택 문서·공간에 포함됩니다. 실제 작업이 영향을
              받는지는 사람이 검토해야 합니다.
            </p>
            {impact.agents.map((a) => (
              <p key={a.id}>
                {a.name} · 설정 v{a.revision} ·{" "}
                {a.reason === "selected_document"
                  ? "문서 직접 선택"
                  : "공간 범위"}
              </p>
            ))}
            {!impact.agents.length && (
              <p className="muted">현재 표시할 Agent가 없습니다.</p>
            )}
          </section>
          <section className="card">
            <h2>내 지식 패키지</h2>
            {impact.packages.map((p) => (
              <p key={p.id}>
                <Link to={`/app/knowledge-packages?id=${p.id}`}>
                  {datetime(p.created_at)} 패키지
                </Link>{" "}
                · {p.stale ? "원문 변경 · 재생성 필요" : "현재 버전 일치"}
              </p>
            ))}
            {!impact.packages.length && (
              <p className="muted">현재 열람할 수 있는 내 패키지가 없습니다.</p>
            )}
            <p className="notice">
              이미 외부에 전달한 원문은 여기서 회수할 수 없습니다. 새 패키지
              전달 여부도 별도로 확인하세요.
            </p>
          </section>
        </>
      )}
      {publicInfo.approval_enabled && (
        <section className="card">
          <h2>영향 검토 처리함</h2>
          <p>
            최신 200건입니다. 영향 검토는 문서 게시 승인·도구 실행 승인과
            별개입니다.
          </p>
          {reviews.map((review) => (
            <div className="evidence-review" key={review.id}>
              <h3>{review.target_title}</h3>
              <p>
                {review.source_title} v{review.source_version} 변경 · 담당{" "}
                {review.owner_name} · {states[review.status]}
                {review.stale ? " · 원문 변경됨, 새 검토 필요" : ""}
              </p>
              <Button disabled={busy} onClick={() => void openReview(review)}>
                검토 내용·이력
              </Button>
              <Button
                disabled={busy}
                onClick={() => setExceptionReview(review.id)}
              >
                예외 승인·이력
              </Button>
            </div>
          ))}
        </section>
      )}
      <Modal
        open={!!editing}
        onOpenChange={(open) => {
          if (!open && !busy) setEditing(null);
        }}
        title="변경 영향 검토"
      >
        {editing && (
          <>
            <p>
              {editing.target_title} · 검토 revision {editing.revision}
            </p>
            <p className="notice">
              현재 대상 v{editing.current_target_version}을 확인하고 의견을
              남기세요. 이 조작으로 문서 본문·게시 상태를 변경하지 않습니다.
              예외 요청은 예외 승인 완료가 아닙니다.
            </p>
            <Button onClick={() => setPreview(editing.target_id)}>
              대상 문서 확인
            </Button>
            <Field label="검토 처리 결과">
              <select
                value={state}
                disabled={!editing.can_write || editing.stale}
                onChange={(e) => setState(e.target.value)}
              >
                {Object.entries(states).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="검토 근거 의견">
              <textarea
                value={note}
                maxLength={1000}
                rows={4}
                onChange={(e) => setNote(e.target.value)}
              />
            </Field>
            <Button
              disabled={
                busy || !editing.can_write || editing.stale || !note.trim()
              }
              onClick={() => void saveReview()}
            >
              현재 버전 검토 결과 저장
            </Button>
            {history.map((h) => (
              <div className="evidence-review" key={h.revision}>
                <strong>{states[h.status]}</strong> · {datetime(h.created_at)}
                <pre className="evidence-text">{h.note}</pre>
              </div>
            ))}
          </>
        )}
      </Modal>
      {exceptionReview && publicInfo.approval_enabled && (
        <Suspense fallback={<Loading />}>
          <ImpactExceptions
            reviewId={exceptionReview}
            onClose={() => setExceptionReview(null)}
          />
        </Suspense>
      )}
      {preview && (
        <Suspense fallback={<Loading />}>
          <DocumentPreview
            documentId={preview}
            onClose={() => setPreview(null)}
          />
        </Suspense>
      )}
    </div>
  );
}
