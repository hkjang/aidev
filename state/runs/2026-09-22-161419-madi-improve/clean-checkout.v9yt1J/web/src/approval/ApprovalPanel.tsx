import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { CheckCheck, RefreshCw, ShieldCheck } from "lucide-react";
import { api, ApiError, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading, Modal } from "../ui";
import {
  type ApprovalStatus,
  type Request,
  statusNames,
  kindNames,
} from "./types";
import "./style.css";

// Never render synced/plugin blocks inside a frozen review. A dynamic embed
// would display new contents that are not part of this immutable approval.
export function ApprovalReview({
  requestID,
  onClose,
  onChanged,
}: {
  requestID: string;
  onClose: () => void;
  onChanged?: () => void;
}) {
  const { user, notify } = useApp();
  const [request, setRequest] = useState<Request | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [comment, setComment] = useState(""),
    [gate, setGate] = useState(""),
    [reviewed, setReviewed] = useState(false);
  useEffect(() => {
    let active = true;
    setRequest(null);
    setReviewed(false);
    setComment("");
    setError("");
    api<Request>(`/approvals/requests/${requestID}`)
      .then((r) => {
        if (active) {
          setRequest(r);
          setGate(String(r.eligible_gates?.[0] ?? ""));
        }
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, [requestID]);
  // Revalidate silent ACL revocation while displaying an immutable snapshot.
  // Do not replace its review version with a newer request during polling.
  useEffect(() => {
    if (!request) return;
    let active = true;
    const timer = setInterval(() => {
      api<Request>(`/approvals/requests/${requestID}`)
        .then((latest) => {
          if (!active) return;
          if (
            latest.version !== request.version ||
            latest.stale ||
            latest.review_context?.current === false
          ) {
            setReviewed(false);
            setRequest((old) =>
              old
                ? {
                    ...old,
                    stale: true,
                    reason:
                      latest.reason ||
                      latest.review_context?.notice ||
                      "요청 상태가 변경되었습니다. 창을 닫고 다시 확인하세요.",
                  }
                : old,
            );
          }
        })
        .catch((e) => {
          if (!active) return;
          setRequest(null);
          setError(e.message);
        });
    }, 5000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [requestID, request?.version]);
  const decide = async (action: string) => {
    if (!request) return;
    setBusy(true);
    setError("");
    try {
      await api(`/approvals/requests/${requestID}/decisions`, "POST", {
        action,
        request_version: request.version,
        gate_index: gate === "" ? undefined : Number(gate),
        comment,
      });
      notify(
        action === "approve"
          ? "검토 승인을 반영했습니다."
          : action === "reject"
            ? "반려 의견을 전달했습니다."
            : "검토 요청을 취소했습니다.",
      );
      onChanged?.();
      onClose();
    } catch (e) {
      setError((e as Error).message);
      setReviewed(false);
    } finally {
      setBusy(false);
    }
  };
  const canDecide =
    !!request &&
    request.status === "pending" &&
    !request.stale &&
    (request.resource_kind !== "impact_exception" ||
      request.review_context?.current === true) &&
    !!request.eligible_gates?.length;
  return (
    <Modal
      open
      onOpenChange={(v) => {
        if (!v) onClose();
      }}
      title="검토 요청 원본"
      description="이 요청에 고정된 내용과 정책을 확인합니다. 원본이나 정책이 변경되면 기존 요청으로 승인할 수 없습니다."
      wide
    >
      {error && <ErrorBox error={error} />}
      {!request && !error && <Loading />}
      {request && (
        <div className="approval-review">
          <div className="approval-summary">
            <strong>
              {kindNames[request.resource_kind]} · {statusNames[request.status]}
            </strong>
            <span>
              원본 v{request.resource_version} · 요청 v{request.version}
            </span>
          </div>
          {request.stale && (
            <div role="status" className="notice">
              {request.reason}
            </div>
          )}
          <h3>{request.snapshot?.title || request.policy.name}</h3>
          {request.comment && <p>요청 의견: {request.comment}</p>}
          {request.resource_kind === "learning_step" &&
            typeof request.snapshot?.review_url === "string" &&
            /^\/app\/knowledge-paths\?path=[0-9a-f-]{36}&review=[0-9a-f-]{36}$/i.test(
              request.snapshot.review_url,
            ) && (
              <Link to={request.snapshot.review_url}>
                제출한 실습 원문과 근거 확인
              </Link>
            )}
          {request.resource_kind === "knowledge_distribution" &&
            /^[0-9a-f-]{36}$/i.test(request.resource_id) && (
              <Link
                to={`/app/knowledge-distribution?review=${request.resource_id}`}
              >
                수신망·파일·원문 묶음 검토
              </Link>
            )}
          {request.resource_kind === "impact_exception" &&
            request.review_context && (
              <section
                className="notice impact-exception-context"
                aria-label="예외 승인 근거"
              >
                <p>
                  {request.review_context.source_title} v
                  {request.review_context.source_version} →{" "}
                  {request.review_context.target_title} v
                  {request.review_context.target_version}
                </p>
                <p>유효 기한: {datetime(request.review_context.valid_until)}</p>
                <pre className="evidence-text">
                  {request.review_context.reason}
                </pre>
                <p>{request.review_context.notice}</p>
                <p>
                  예외 승인은 문서 게시·실행 승인이 아니며 원문을 변경하지
                  않습니다.
                </p>
                <Link
                  to={`/app/knowledge-impact?document_id=${request.review_context.source_id}`}
                >
                  변경 영향과 연결 문서 확인
                </Link>
              </section>
            )}
          <p className="muted">
            동적 참조·플러그인은 실행하지 않습니다. 문서의 경우 아래 Markdown
            원문이 정확한 검토 대상입니다.
          </p>
          <pre className="approval-snapshot" tabIndex={0}>
            {typeof request.snapshot?.markdown === "string"
              ? request.snapshot.markdown
              : JSON.stringify(request.snapshot, null, 2)}
          </pre>
          {typeof request.snapshot?.markdown === "string" && (
            <details>
              <summary>고정된 문서 속성</summary>
              <pre className="approval-snapshot">
                {JSON.stringify(
                  { ...request.snapshot, markdown: undefined },
                  null,
                  2,
                )}
              </pre>
            </details>
          )}
          <ol className="approval-stages">
            {request.policy.stages.map((stage, si) => (
              <li
                key={si}
                className={si === request.current_stage ? "current" : ""}
              >
                <strong>
                  {si + 1}. {stage.name}
                </strong>
                <span>
                  {stage.mode === "all"
                    ? "모든 검토 대상 승인"
                    : "검토 대상 중 한 곳 승인"}
                </span>
                <ul>
                  {stage.gates.map((g, gi) => (
                    <li key={gi}>
                      {g.name}
                      <small>
                        {request.assignments
                          ?.filter(
                            (a) => a.stage_index === si && a.gate_index === gi,
                          )
                          .map((a) => a.name)
                          .join(", ")}
                      </small>
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
          {!!request.decisions?.length && (
            <section>
              <h3>검토 의견</h3>
              {request.decisions.map((d) => (
                <article className="approval-decision" key={d.id}>
                  <strong>
                    {d.name} · {statusNames[d.decision]}
                  </strong>
                  <span>{datetime(d.created_at)}</span>
                  <p>{d.comment || "의견 없음"}</p>
                </article>
              ))}
            </section>
          )}
          {canDecide && (
            <>
              <Field label="처리할 검토 대상">
                <select
                  value={gate}
                  onChange={(e) => {
                    setGate(e.target.value);
                    setReviewed(false);
                  }}
                >
                  {request.eligible_gates?.map((i) => (
                    <option value={i} key={i}>
                      {
                        request.policy.stages[request.current_stage].gates[i]
                          .name
                      }
                    </option>
                  ))}
                </select>
              </Field>
              <Field
                label="검토 의견"
                hint="반려할 때는 구체적인 사유를 입력하세요."
              >
                <textarea
                  maxLength={10000}
                  rows={3}
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                />
              </Field>
              <label className="approval-confirm">
                <input
                  type="checkbox"
                  checked={reviewed}
                  onChange={(e) => setReviewed(e.target.checked)}
                />
                이 요청의 고정된 원본과 승인 단계를 확인했습니다.
              </label>
              <div className="button-row">
                <Button
                  variant="primary"
                  disabled={busy || !reviewed}
                  onClick={() => void decide("approve")}
                >
                  <CheckCheck size={18} />
                  검토 승인
                </Button>
                <Button
                  disabled={busy || !reviewed || !comment.trim()}
                  onClick={() => void decide("reject")}
                >
                  사유와 함께 반려
                </Button>
              </div>
            </>
          )}
          {request.status === "pending" &&
            [request.requester_id, request.owner_id].includes(user.id) && (
              <Button disabled={busy} onClick={() => void decide("cancel")}>
                검토 요청 취소
              </Button>
            )}
        </div>
      )}
    </Modal>
  );
}

export default function ApprovalPanel({
  kind = "document",
  resourceID,
  canSubmit = false,
  beforeSubmit,
  onChanged,
  revision,
}: {
  kind?: string;
  resourceID: string;
  canSubmit?: boolean;
  beforeSubmit?: () => Promise<boolean>;
  onChanged?: () => void | Promise<void>;
  revision?: number;
}) {
  const { publicInfo, notify } = useApp();
  const [state, setState] = useState<ApprovalStatus | null>(null),
    [error, setError] = useState(""),
    [selected, setSelected] = useState(""),
    [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    try {
      setState(
        await api<ApprovalStatus>(`/approvals/resources/${kind}/${resourceID}`),
      );
      setError("");
    } catch (e) {
      setState(null);
      setError(
        e instanceof ApiError && e.status === 404 ? "" : (e as Error).message,
      );
    }
  }, [kind, resourceID]);
  useEffect(() => {
    if (!publicInfo.approval_enabled) return;
    void load();
    const timer = setInterval(() => void load(), 5000);
    return () => clearInterval(timer);
  }, [load, publicInfo.approval_enabled, revision]);
  if (!publicInfo.approval_enabled) return null;
  const changed = () => {
    void load();
    void onChanged?.();
  };
  const submit = async () => {
    setBusy(true);
    try {
      if (beforeSubmit && !(await beforeSubmit())) return;
      await api(`/documents/${resourceID}/approval`, "POST", {
        action: "submit",
      });
      notify("현재 버전의 검토를 요청했습니다.");
      changed();
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <section className="approval-panel" aria-label="검토 및 승인">
      <div className="approval-summary">
        <h3>
          <ShieldCheck size={19} />
          검토 및 승인
        </h3>
        <Button aria-label="검토 상태 새로고침" onClick={() => void load()}>
          <RefreshCw size={16} />
        </Button>
      </div>
      {error && <ErrorBox error={error} />}
      {state && (
        <>
          <p>
            {state.policy.name} · {state.policy.stages.length}단계
          </p>
          <p className="muted">
            {state.policy.stages.map((s) => s.name).join(" → ")}
          </p>
          {state.request && (
            <div className="approval-summary">
              <strong>
                {statusNames[state.request.status]}
                {state.request.status === "pending"
                  ? ` · ${state.request.current_stage + 1}단계`
                  : ""}
              </strong>
              <Button onClick={() => setSelected(state.request!.id)}>
                검토 원본·결정 보기
              </Button>
            </div>
          )}
          {state.stale && <div className="notice">{state.reason}</div>}
          {canSubmit &&
            kind === "document" &&
            (!state.request ||
              state.request.status !== "pending" ||
              state.stale) && (
              <Button disabled={busy} onClick={() => void submit()}>
                현재 버전 검토 요청
              </Button>
            )}
          {state.history.length > 1 && (
            <details>
              <summary>이전 검토 요청 {state.history.length - 1}건</summary>
              {state.history.slice(1).map((r) => (
                <Button key={r.id} onClick={() => setSelected(r.id)}>
                  원본 v{r.resource_version} · {statusNames[r.status]}
                </Button>
              ))}
            </details>
          )}
          <Link to="/app/approvals">내 검토함 열기</Link>
        </>
      )}
      {selected && (
        <ApprovalReview
          requestID={selected}
          onClose={() => setSelected("")}
          onChanged={changed}
        />
      )}
    </section>
  );
}
