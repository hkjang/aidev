import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  ArrowLeftRight,
  FileSearch,
  LockKeyhole,
  Plus,
  RefreshCw,
  Trash2,
} from "lucide-react";
import { api, datetime, type DocSummary } from "../api";
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
import DocumentPreview from "../review/DocumentPreview";
import { ChangeReview, RecoveryNotice } from "../review/ChangeReview";
import { useModalReturnFocus } from "../review/useModalReturnFocus";
import "./style.css";

export type ConflictExcerpt = {
  document_id: string;
  title: string;
  version: number;
  document_hash: string;
  start_byte: number;
  end_byte: number;
  line: number;
  text: string;
  excerpt_hash: string;
};
export type ConflictCandidate = {
  id: string;
  run_id: string;
  workspace_id: string;
  revision: number;
  decision: string;
  stale: boolean;
  unavailable?: boolean;
  topic: string;
  rule: string;
  left: ConflictExcerpt;
  right: ConflictExcerpt;
  left_current_version: number;
  right_current_version: number;
  history?: {
    revision: number;
    decision: string;
    note: string;
    created_at: string;
  }[];
};
type Run = {
  id: string;
  created_at: string;
  rule_version: string;
  source_count: number;
  candidate_count: number;
  diagnostics: {
    Documents: number;
    Lines: number;
    Statements: number;
    Truncated: boolean;
  };
  candidates?: ConflictCandidate[];
};
const decisionLabels: Record<string, string> = {
  conflict: "충돌로 판단",
  compatible: "함께 적용 가능",
  needs_review: "추가 확인 필요",
};
const ruleLabels: Record<string, string> = {
  quantity_difference: "수치·단위 차이",
  permission_difference: "허용·금지 차이",
  requirement_difference: "필수·선택 차이",
};
export const conflictLimits =
  "문서 2~32개 · 원문·속성 문서당 1MiB / 합계 8MiB · 문서당 2,000줄 · 후보 최대 64개 · 최대 10초";

export default function KnowledgeConflictsPage() {
  const { user, workspace, documents, notify } = useApp();
  const [params, setParams] = useSearchParams();
  const id = params.get("id") || "",
    candidateID = params.get("candidate") || "";
  const [runs, setRuns] = useState<Run[]>([]),
    [run, setRun] = useState<Run | null>(null),
    [detail, setDetail] = useState<ConflictCandidate | null>(null);
  const [loading, setLoading] = useState(true),
    [error, setError] = useState<unknown>(null),
    [mutationError, setMutationError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0);
  const [creating, setCreating] = useState(false),
    [selected, setSelected] = useState<DocSummary[]>([]),
    [query, setQuery] = useState(""),
    [consent, setConsent] = useState(false);
  const [review, setReview] = useState<ConflictCandidate | null>(null),
    [decision, setDecision] = useState("needs_review"),
    [note, setNote] = useState(""),
    [reviewConsent, setReviewConsent] = useState(false),
    [deleting, setDeleting] = useState(false);
  const [preview, setPreview] = useState<ConflictExcerpt | null>(null);
  const scope = `${user.id}:${workspace?.id}:${id}:${candidateID}`,
    current = useRef(scope),
    loaded = useRef(""),
    generation = useRef(0);
  current.current = scope;
  const createFocus = useModalReturnFocus(
      creating,
      `${user.id}:${workspace?.id}`,
    ),
    reviewFocus = useModalReturnFocus(!!review, scope),
    deleteFocus = useModalReturnFocus(deleting, scope);
  const visible = loaded.current === scope;
  useEffect(() => {
    setCreating(false);
    setSelected([]);
    setQuery("");
    setConsent(false);
    setReview(null);
    setNote("");
    setReviewConsent(false);
    setDeleting(false);
    setPreview(null);
  }, [user.id, workspace?.id, id, candidateID]);
  useEffect(() => {
    const epoch = ++generation.current;
    let active = true,
      pending = false;
    setRuns([]);
    setRun(null);
    setDetail(null);
    setLoading(true);
    setError(null);
    setMutationError(null);
    setBusy(false);
    const fresh = () =>
      active && epoch === generation.current && current.current === scope;
    const load = async () => {
      if (pending || !workspace) return;
      pending = true;
      try {
        if (id) {
          const result = await api<Run>(`/knowledge/conflicts/${id}`);
          let next: ConflictCandidate | null = null;
          if (candidateID)
            next = await api<ConflictCandidate>(
              `/knowledge/conflict-candidates/${candidateID}`,
            );
          if (next && next.run_id !== id)
            throw Error("선택한 보고서의 후보가 아닙니다");
          if (fresh()) {
            loaded.current = scope;
            setRun(result);
            setDetail(next);
            setError(null);
            setReview((old) => {
              if (!old) return null;
              const currentCandidate = (result.candidates || []).find(
                (c) => c.id === old.id,
              );
              if (
                !currentCandidate ||
                currentCandidate.unavailable ||
                currentCandidate.stale ||
                currentCandidate.revision !== old.revision
              ) {
                setNote("");
                setReviewConsent(false);
                return null;
              }
              return old;
            });
          }
        } else {
          const result = await api<{ items: Run[] }>(
            `/knowledge/conflicts?workspace_id=${workspace.id}`,
          );
          if (fresh()) {
            loaded.current = scope;
            setRuns(result.items);
            setError(null);
          }
        }
      } catch (e) {
        if (fresh()) {
          setRun(null);
          setDetail(null);
          setRuns([]);
          setReview(null);
          setNote("");
          setPreview(null);
          setError(e);
        }
      } finally {
        pending = false;
        if (fresh()) setLoading(false);
      }
    };
    void load();
    const timer = setInterval(() => void load(), 2000);
    return () => {
      active = false;
      generation.current++;
      clearInterval(timer);
    };
  }, [scope, refresh, workspace]);
  const perform = async (action: () => Promise<void>) => {
    if (busy) return;
    const epoch = generation.current;
    setBusy(true);
    setMutationError(null);
    try {
      await action();
    } catch (e) {
      if (epoch === generation.current) setMutationError(e);
    } finally {
      if (epoch === generation.current) setBusy(false);
    }
  };
  const create = () =>
    perform(async () => {
      if (!workspace) return;
      const epoch = generation.current,
        result = await api<{ id: string }>("/knowledge/conflicts", "POST", {
          workspace_id: workspace.id,
          documents: selected.map((d) => ({ id: d.id, version: d.version })),
          consent,
        });
      if (epoch === generation.current) {
        setCreating(false);
        setParams({ id: result.id });
        notify("규칙 기반 비교를 개인 보고서에 보관했습니다");
      }
    });
  const saveReview = () =>
    perform(async () => {
      if (!review) return;
      const epoch = generation.current;
      await api(`/knowledge/conflict-candidates/${review.id}/reviews`, "POST", {
        revision: review.revision,
        decision,
        note,
        consent: reviewConsent,
      });
      if (epoch === generation.current) {
        setReview(null);
        setNote("");
        setRefresh((v) => v + 1);
        notify("개인 판단을 기록했습니다. 원문은 변경되지 않았습니다");
      }
    });
  const remove = () =>
    perform(async () => {
      const epoch = generation.current;
      await api(`/knowledge/conflicts/${id}`, "DELETE");
      if (epoch === generation.current) {
        setDeleting(false);
        setParams({});
        notify("개인 비교 보고서를 삭제했습니다");
      }
    });
  const startReview = (c: ConflictCandidate) => {
    setMutationError(null);
    setReview(c);
    setDecision(c.decision);
    setNote("");
    setReviewConsent(false);
  };
  const shownDocuments = documents
    .filter(
      (d) =>
        !d.deleted_at &&
        (!query ||
          `${d.title} ${d.tags.join(" ")}`
            .toLocaleLowerCase()
            .includes(query.toLocaleLowerCase())),
    )
    .slice(0, 100);
  const display = (c: ConflictCandidate, full = false) =>
    c.unavailable ? (
      <div className="notice" key={c.id}>
        <LockKeyhole size={20} />
        저장했던 비교 후보에 현재 접근할 수 없습니다.
      </div>
    ) : (
      <article className="panel conflict-candidate" key={c.id}>
        <header>
          <div>
            <span className="eyebrow">
              {ruleLabels[c.rule] || "문자열 차이 후보"}
            </span>
            <h2>{c.topic}</h2>
            <span className="badge">{decisionLabels[c.decision]}</span>
          </div>
          <span className={`badge ${c.stale ? "warning" : ""}`}>
            {c.stale ? "원문·판단 다시 확인" : "분석 당시 버전과 일치"}
          </span>
        </header>
        <div className="conflict-sources">
          {(["left", "right"] as const).map((side) => {
            const source = c[side];
            return (
              <section
                key={side}
                aria-label={`${side === "left" ? "첫째" : "둘째"} 출처`}
              >
                <strong>{source.title}</strong>
                <p className="muted">
                  버전 {source.version} · {source.line}행 · UTF-8 바이트{" "}
                  {source.start_byte}–{source.end_byte}
                </p>
                <pre>{source.text}</pre>
                <details>
                  <summary>원문·구간 해시</summary>
                  <p>
                    문서 SHA-256 <code>{source.document_hash}</code>
                  </p>
                  <p>
                    구간 SHA-256 <code>{source.excerpt_hash}</code>
                  </p>
                  <p className="muted">
                    내용 식별용 해시이며 사실성이나 승인 인증이 아닙니다.
                  </p>
                </details>
                <div className="button-row">
                  <Button onClick={() => setPreview(source)}>
                    현재 원문 미리보기
                  </Button>
                  <Link
                    className="button"
                    to={`/app/documents/${source.document_id}?mode=preview&line=${source.line}`}
                  >
                    원문 열기
                  </Link>
                </div>
              </section>
            );
          })}
        </div>
        {c.stale && (
          <div className="notice warning">
            원문 버전이나 속성이 바뀌었습니다. 이전 구간은 비교 이력이며 새
            판단·변경안에는 다시 분석한 자료를 사용하세요.
          </div>
        )}
        <footer className="button-row">
          <Button disabled={c.stale || busy} onClick={() => startReview(c)}>
            판단 기록
          </Button>
          {!full && (
            <Button onClick={() => setParams({ id, candidate: c.id })}>
              판단 이력 보기
            </Button>
          )}
          {!c.stale && (
            <>
              <Link
                className="button"
                to={`/app/knowledge-proposals?document_id=${c.left.document_id}&conflict_id=${c.id}`}
              >
                첫째 원문 변경안
              </Link>
              <Link
                className="button"
                to={`/app/knowledge-proposals?document_id=${c.right.document_id}&conflict_id=${c.id}`}
              >
                둘째 원문 변경안
              </Link>
            </>
          )}
        </footer>
        {full && (
          <section>
            <h3>개인 판단 이력</h3>
            {!c.history?.length ? (
              <p className="muted">
                아직 기록한 판단이 없습니다. 후보 생성만으로 충돌이 확정되지
                않습니다.
              </p>
            ) : (
              <ol className="conflict-history">
                {c.history.map((event) => (
                  <li key={event.revision}>
                    <strong>
                      {decisionLabels[event.decision]} · 기록 {event.revision}
                    </strong>
                    <span className="muted">{datetime(event.created_at)}</span>
                    <p>{event.note}</p>
                  </li>
                ))}
              </ol>
            )}
          </section>
        )}
      </article>
    );
  return (
    <div className="page conflict-page">
      <PageHeading
        title="문서 차이 검토"
        description="같은 주제의 수치·정책 표현을 나란히 확인하고, 사람의 판단을 기록합니다."
        actions={
          <div className="button-row">
            <Button onClick={() => setRefresh((v) => v + 1)} disabled={busy}>
              <RefreshCw size={18} />
              다시 읽기
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                setCreating(true);
                setSelected([]);
                setConsent(false);
                setMutationError(null);
              }}
            >
              <Plus size={18} />새 비교
            </Button>
          </div>
        }
      />
      <div className="notice conflict-explanation">
        <ArrowLeftRight size={22} />
        <div>
          <strong>후보 발견과 충돌 판정은 다릅니다</strong>
          <p>
            가까운 제목 또는 같은 태그와 문장 구조가 일치할 때
            수치·허용/금지·필수/선택 표현만 비교합니다. 코드·인용 예시·Front
            Matter는 제외하며 의미·사실·절차의 모순을 자동 판정하지 않습니다.
          </p>
          <p>{conflictLimits}. AI·외부 API 전송 없음.</p>
          <p>
            보고서와 판단은 본인에게만 보입니다. 게시 승인 절차와 별개이며
            문서를 자동 수정하지 않습니다.
          </p>
        </div>
      </div>
      <RecoveryNotice
        error={error || mutationError}
        onRetry={() => setRefresh((v) => v + 1)}
      />
      {id && (
        <div className="button-row">
          <Button onClick={() => setParams(candidateID ? { id } : {})}>
            {candidateID ? "보고서로 돌아가기" : "내 보고서 목록"}
          </Button>
          <Button
            onClick={() => {
              setDeleting(true);
              setMutationError(null);
            }}
            disabled={busy}
          >
            <Trash2 size={18} />
            보고서 삭제
          </Button>
        </div>
      )}
      {loading ? (
        <Loading />
      ) : visible && id && run ? (
        <>
          <p className="muted">
            {datetime(run.created_at)} · 규칙 {run.rule_version} · 실제 검사{" "}
            {run.diagnostics.Documents}개 문서 / {run.diagnostics.Lines}줄
            {run.diagnostics.Truncated
              ? " · 한도에 도달해 일부 내용은 제외됨"
              : ""}
          </p>
          {candidateID && detail ? (
            display(detail, true)
          ) : (run.candidates || []).length ? (
            (run.candidates || []).map((c) => display(c))
          ) : (
            <Empty
              title="규칙에 일치하는 차이 후보가 없습니다"
              text="모순이 없다는 뜻은 아닙니다. 의미나 적용 범위가 다른 표현은 이 제한된 규칙으로 찾지 못합니다."
            />
          )}
        </>
      ) : visible && !id ? (
        <div className="conflict-run-list">
          {runs.length ? (
            runs.map((item) => (
              <button
                key={item.id}
                className="panel conflict-run"
                onClick={() => setParams({ id: item.id })}
              >
                <FileSearch size={24} />
                <span>
                  <strong>{datetime(item.created_at)} 비교 보고서</strong>
                  <span>
                    {item.source_count}개 문서 · {item.candidate_count}개 후보 ·
                    본인만 보기
                  </span>
                </span>
              </button>
            ))
          ) : (
            <Empty
              title="아직 비교 보고서가 없습니다"
              text="직접 읽을 수 있는 문서를 선택해 제한된 규칙 검사를 시작하세요."
            />
          )}
        </div>
      ) : null}
      <Modal
        open={creating}
        onOpenChange={(v) => {
          if (!busy) setCreating(v);
        }}
        onCloseAutoFocus={createFocus}
        title="새 문서 비교"
        description="읽을 수 있는 현재 문서의 선택 버전만 비교하고 구간을 개인 보고서에 암호화 보관합니다."
        wide
      >
        <ErrorBox
          error={mutationError instanceof Error ? mutationError.message : ""}
        />
        <Field label="비교 문서 찾기">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="제목 또는 태그"
            disabled={busy}
          />
        </Field>
        <p>
          {selected.length} / 32개 선택 · {conflictLimits}
        </p>
        <div className="conflict-document-picker">
          {shownDocuments.map((d) => (
            <label key={d.id}>
              <input
                type="checkbox"
                checked={selected.some((v) => v.id === d.id)}
                disabled={
                  busy ||
                  (selected.length >= 32 &&
                    !selected.some((v) => v.id === d.id))
                }
                onChange={(e) =>
                  setSelected((items) =>
                    e.target.checked
                      ? [...items, d]
                      : items.filter((v) => v.id !== d.id),
                  )
                }
              />
              <span>
                {d.title}
                <small>
                  버전 {d.version} ·{" "}
                  {d.visibility === "private"
                    ? "나만 보기"
                    : "현재 접근 권한 적용"}
                </small>
              </span>
            </label>
          ))}
        </div>
        <label className="checkbox-line">
          <input
            type="checkbox"
            checked={consent}
            onChange={(e) => setConsent(e.target.checked)}
            disabled={busy}
          />
          선택 원문의 구간을 내 비교 보고서에 보관하고, 규칙 결과를 사람이
          확인하는 데 동의합니다.
        </label>
        <div className="modal-footer">
          <Button onClick={() => setCreating(false)} disabled={busy}>
            취소
          </Button>
          <Button
            variant="primary"
            disabled={busy || selected.length < 2 || !consent}
            onClick={create}
          >
            {busy ? "제한된 규칙으로 비교 중…" : "비교 보고서 만들기"}
          </Button>
        </div>
      </Modal>
      <Modal
        open={!!review}
        onOpenChange={(v) => {
          if (!v && !busy) {
            setReview(null);
            setNote("");
          }
        }}
        onCloseAutoFocus={reviewFocus}
        title="개인 판단 기록"
        wide
      >
        {review && (
          <>
            <ErrorBox
              error={
                mutationError instanceof Error ? mutationError.message : ""
              }
            />
            <Field label="차이 판단">
              <select
                value={decision}
                onChange={(e) => setDecision(e.target.value)}
                disabled={busy}
              >
                {Object.entries(decisionLabels).map(([key, label]) => (
                  <option key={key} value={key}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="판단 근거">
              <textarea
                rows={4}
                value={note}
                maxLength={4000}
                onChange={(e) => setNote(e.target.value)}
                disabled={busy}
                placeholder="적용 대상과 조건을 확인한 근거를 적으세요."
              />
            </Field>
            <ChangeReview
              title="원문을 유지하고 판단만 기록"
              changes={[
                {
                  label: "내 판단 기록",
                  before: decisionLabels[review.decision],
                  after: decisionLabels[decision],
                },
              ]}
              warnings={[
                "게시·검토 승인이나 원문 수정이 아닙니다.",
                "원문 또는 판단 버전이 바뀌면 기록하지 않고 다시 확인합니다.",
              ]}
              confirmLabel="판단 저장"
              busy={busy}
              disabled={!reviewConsent || !note.trim()}
              onConfirm={saveReview}
              onCancel={() => setReview(null)}
            >
              <label className="checkbox-line">
                <input
                  type="checkbox"
                  checked={reviewConsent}
                  onChange={(e) => setReviewConsent(e.target.checked)}
                  disabled={busy}
                />
                두 구간을 확인했으며 내 판단 이력에 보관합니다.
              </label>
            </ChangeReview>
          </>
        )}
      </Modal>
      <Modal
        open={deleting}
        onOpenChange={(v) => {
          if (!busy) setDeleting(v);
        }}
        onCloseAutoFocus={deleteFocus}
        title="개인 보고서 삭제"
      >
        <ErrorBox
          error={mutationError instanceof Error ? mutationError.message : ""}
        />
        <p>
          이 보고서의 비교 구간과 개인 판단 이력을 삭제합니다. 원문은
          유지됩니다. 변경안 출처로 연결된 보고서는 삭제되지 않습니다.
        </p>
        <div className="modal-footer">
          <Button onClick={() => setDeleting(false)} disabled={busy}>
            취소
          </Button>
          <Button variant="danger" onClick={remove} disabled={busy}>
            보고서 삭제 확인
          </Button>
        </div>
      </Modal>
      <DocumentPreview
        documentId={preview?.document_id || null}
        expectedVersion={preview?.version}
        sourceLine={preview?.line}
        onClose={() => setPreview(null)}
      />
    </div>
  );
}
