import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, ErrorBox, Field, Loading, Modal } from "../ui";
import { ApprovalReview } from "../approval/ApprovalPanel";
import { statusNames } from "../approval/types";
type Exception = {
  id: string;
  approval_id: string;
  status: string;
  current: boolean;
  effective: boolean;
  reason: string;
  valid_until: string;
  notice: string;
  source_id: string;
  source_title: string;
  source_version: number;
  target_title: string;
  target_version: number;
  approval: { version: number; requester_id: string; owner_id: string } | null;
};
type State = {
  items: Exception[];
  can_request: boolean;
  policy_configured: boolean;
  review_revision: number;
  source_version: number;
  target_version: number;
  notice: string;
};
export default function ImpactExceptions({
  reviewId,
  onClose,
}: {
  reviewId: string;
  onClose: () => void;
}) {
  const { user, workspace, notify } = useApp();
  const scope = `${user.id}:${workspace?.id}:${reviewId}`;
  const scopeRef = useRef(scope);
  scopeRef.current = scope;
  const [state, setState] = useState<State | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false),
    [reason, setReason] = useState(""),
    [days, setDays] = useState(7),
    [confirm, setConfirm] = useState(false),
    [request, setRequest] = useState("");
  const busyRef = useRef(false),
    mounted = useRef(false),
    fingerprint = useRef(""),
    sequence = useRef(0);
  const load = async (quiet = false) => {
    const ticket = ++sequence.current,
      captured = scope;
    try {
      const value = await api<State>(
        `/knowledge/impact-reviews/${reviewId}/exceptions`,
      );
      if (
        !mounted.current ||
        captured !== scopeRef.current ||
        ticket !== sequence.current
      )
        return;
      const fp = JSON.stringify([
        value.review_revision,
        value.source_version,
        value.target_version,
        value.can_request,
        value.policy_configured,
        value.items.map((x) => [x.id, x.status, x.current]),
      ]);
      if (fingerprint.current !== fp) {
        setConfirm(false);
        fingerprint.current = fp;
      }
      setState(value);
      if (!quiet) setError("");
    } catch (e) {
      if (
        mounted.current &&
        captured === scopeRef.current &&
        ticket === sequence.current
      ) {
        setState(null);
        setConfirm(false);
        setRequest("");
        setError((e as Error).message);
      }
    }
  };
  useEffect(() => {
    mounted.current = true;
    setState(null);
    setConfirm(false);
    setReason("");
    setError("");
    setRequest("");
    setBusy(false);
    busyRef.current = false;
    fingerprint.current = "";
    void load();
    const timer = setInterval(() => {
      if (document.visibilityState === "visible" && !busyRef.current)
        void load(true);
    }, 3000);
    return () => {
      mounted.current = false;
      sequence.current++;
      clearInterval(timer);
    };
  }, [scope]);
  const perform = async (action: () => Promise<void>) => {
    if (busyRef.current) return;
    const captured = scope;
    busyRef.current = true;
    setBusy(true);
    setError("");
    try {
      await action();
      if (mounted.current && captured === scopeRef.current) {
        setConfirm(false);
        await load();
      }
    } catch (e) {
      if (mounted.current && captured === scopeRef.current) {
        await load(true);
        setError((e as Error).message);
        setConfirm(false);
      }
    } finally {
      if (mounted.current && captured === scopeRef.current) {
        busyRef.current = false;
        setBusy(false);
      }
    }
  };
  return (
    <>
      <Modal
        open
        onOpenChange={(v) => {
          if (!v && !busy) onClose();
        }}
        title="변경 영향 예외 승인"
      >
        {error && <ErrorBox error={error} />} {!state && !error && <Loading />}
        {state && (
          <>
            <p className="notice">{state.notice}</p>
            <p>
              검토 revision {state.review_revision} · 변경 원문 v
              {state.source_version} · 대상 v{state.target_version}
            </p>
            {!state.policy_configured && (
              <p role="status">
                변경 영향 예외 승인 정책이 없습니다. 관리자가 승인 정책에서
                별도로 설정해야 합니다.
                {user.role === "admin" && (
                  <>
                    {" "}
                    <Link to="/admin/approvals">승인 정책 관리</Link>
                  </>
                )}
              </p>
            )}
            {state.can_request && (
              <section aria-label="새 예외 요청">
                <Field label="예외 적용 근거">
                  <textarea
                    value={reason}
                    rows={4}
                    maxLength={1000}
                    onChange={(e) => {
                      setReason(e.target.value);
                      setConfirm(false);
                    }}
                  />
                </Field>
                <Field label="예외 유효기간">
                  <select
                    value={days}
                    onChange={(e) => {
                      setDays(Number(e.target.value));
                      setConfirm(false);
                    }}
                  >
                    {[1, 7, 14, 30, 90].map((n) => (
                      <option key={n} value={n}>
                        {n}일
                      </option>
                    ))}
                  </select>
                </Field>
                <label className="check-label">
                  <input
                    type="checkbox"
                    checked={confirm}
                    onChange={(e) => setConfirm(e.target.checked)}
                  />
                  현재 원문·대상 버전과 예외 근거·기한을 확인하고 별도 승인을
                  요청합니다.
                </label>
                <Button
                  disabled={
                    busy ||
                    !confirm ||
                    !reason.trim() ||
                    state.items.some((x) => x.status === "pending" && x.current)
                  }
                  onClick={() =>
                    void perform(async () => {
                      const created = await api<{ approval_id: string }>(
                        `/knowledge/impact-reviews/${reviewId}/exceptions`,
                        "POST",
                        {
                          review_revision: state.review_revision,
                          source_version: state.source_version,
                          target_version: state.target_version,
                          reason,
                          valid_until: new Date(
                            Date.now() + days * 86400000,
                          ).toISOString(),
                          confirm: true,
                        },
                      );
                      if (mounted.current && scopeRef.current === scope) {
                        setReason("");
                        notify(
                          "예외 검토를 요청했습니다. 아직 승인 완료가 아닙니다.",
                        );
                        setRequest(created.approval_id);
                      }
                    })
                  }
                >
                  예외 승인 요청
                </Button>
              </section>
            )}
            {!state.items.length && <p>등록된 예외 요청이 없습니다.</p>}
            {state.items.map((item) => (
              <section
                key={item.id}
                className="evidence-review"
                data-exception-state={
                  item.effective ? "effective" : item.status
                }
              >
                <h3>
                  {item.effective
                    ? "현재 유효한 예외 승인"
                    : statusNames[item.status] || item.status}
                  {item.status === "approved" && !item.effective
                    ? " · 현재 무효"
                    : ""}
                </h3>
                <p>
                  {item.source_title} v{item.source_version} →{" "}
                  {item.target_title} v{item.target_version}
                </p>
                <p>유효 기한: {datetime(item.valid_until)}</p>
                <pre className="evidence-text">{item.reason}</pre>
                <p>{item.notice}</p>
                <Button
                  disabled={busy}
                  onClick={() => setRequest(item.approval_id)}
                >
                  예외 검토 내용·결정
                </Button>
                {item.status === "pending" &&
                  item.approval &&
                  (item.approval.requester_id === user.id ||
                    item.approval.owner_id === user.id) && (
                    <Button
                      disabled={busy}
                      onClick={() =>
                        void perform(async () => {
                          await api(
                            `/approvals/requests/${item.approval_id}/decisions`,
                            "POST",
                            {
                              action: "cancel",
                              request_version: item.approval?.version,
                            },
                          );
                          if (mounted.current && scopeRef.current === scope)
                            notify("예외 요청을 취소했습니다.");
                        })
                      }
                    >
                      예외 요청 취소
                    </Button>
                  )}
              </section>
            ))}
          </>
        )}
      </Modal>
      {request && (
        <ApprovalReview
          requestID={request}
          onClose={() => setRequest("")}
          onChanged={() => {
            setRequest("");
            void load();
          }}
        />
      )}
    </>
  );
}
