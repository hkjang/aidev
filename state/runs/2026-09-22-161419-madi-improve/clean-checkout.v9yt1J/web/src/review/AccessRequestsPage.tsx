import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { KeyRound, Send, RefreshCw } from "lucide-react";
import { api, datetime } from "../api";
import { useApp } from "../context";
import { Button, Empty, Field, Loading, Modal, PageHeading } from "../ui";
import { ChangeReview, RecoveryNotice } from "./ChangeReview";
import "./access-requests.css";
type AccessRequest = {
  id: string;
  document_id: string;
  document_title?: string;
  document_version?: number;
  visibility?: string;
  requester_name?: string;
  reason?: string;
  permission: "read" | "write";
  status: string;
  revision: number;
  created_at: string;
  updated_at: string;
};
type RequestLists = {
  mine: AccessRequest[];
  incoming: AccessRequest[];
  notice: string;
  limit: number;
};
const statuses: Record<string, string> = {
  pending: "응답 대기",
  granted: "접근 허용",
  rejected: "요청 거절",
  cancelled: "요청 취소",
};
const visibilityLabel = (value?: string) =>
  value === "private"
    ? "나만 보기"
    : value === "selected"
      ? "선택한 사용자"
      : "워크스페이스";
export function RequestDocumentAccess({ documentID }: { documentID: string }) {
  const { user, workspace } = useApp(),
    [open, setOpen] = useState(false),
    [permission, setPermission] = useState("read"),
    [reason, setReason] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState<unknown>(null),
    [message, setMessage] = useState("");
  const generation = useRef(0);
  useEffect(() => {
    generation.current++;
    setOpen(false);
    setBusy(false);
    setError(null);
    setMessage("");
    setPermission("read");
    setReason("");
    return () => {
      generation.current++;
    };
  }, [documentID, user.id, workspace?.id]);
  const send = async () => {
    if (!workspace || busy) return;
    const expected = generation.current;
    setBusy(true);
    setError(null);
    try {
      const result = await api<{ message: string }>(
        `/documents/${documentID}/access-requests`,
        "POST",
        { workspace_id: workspace.id, permission, reason },
      );
      if (generation.current === expected) setMessage(result.message);
    } catch (e) {
      if (generation.current === expected) setError(e);
    } finally {
      if (generation.current === expected) setBusy(false);
    }
  };
  return (
    <>
      <Button
        disabled={!workspace}
        onClick={() => {
          setOpen(true);
          setMessage("");
          setError(null);
        }}
      >
        <KeyRound size={17} />
        접근 권한 요청
      </Button>
      <Modal
        open={open}
        onOpenChange={(v) => {
          if (!busy) {
            generation.current++;
            setOpen(v);
          }
        }}
        title="문서 접근 요청"
        description="문서의 존재나 제목을 먼저 공개하지 않습니다. 선택한 워크스페이스의 요청 가능한 문서라면 소유자에게 전달합니다."
      >
        <div className="access-request-form">
          <p>
            선택한 워크스페이스: <strong>{workspace?.name}</strong>
          </p>
          {message ? (
            <div className="notice" role="status">
              <p>{message}</p>
              <Link to="/app/access-requests" onClick={() => setOpen(false)}>
                내 접근 요청 확인
              </Link>
            </div>
          ) : (
            <>
              <Field label="요청 권한">
                <select
                  value={permission}
                  onChange={(e) => setPermission(e.target.value)}
                  disabled={busy}
                >
                  <option value="read">읽기</option>
                  <option
                    value="write"
                    disabled={
                      user.role === "viewer" ||
                      !["owner", "admin", "editor"].includes(
                        workspace?.role || "",
                      )
                    }
                  >
                    편집
                  </option>
                </select>
              </Field>
              <Field label="요청 사유">
                <textarea
                  value={reason}
                  maxLength={600}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="업무 목적만 간단히 적어 주세요. 비밀번호나 비밀정보는 넣지 마세요."
                  disabled={busy}
                />
              </Field>
              <p className="muted">
                동일 문서의 열린 요청은 합쳐집니다. 처리 후 10분 동안 다시
                요청할 수 없으며, 열린 요청과 시간당 새 요청은 각각
                20개까지입니다.
              </p>
              {error ? <RecoveryNotice error={error} /> : null}
              <Button
                variant="primary"
                disabled={busy}
                onClick={() => void send()}
              >
                <Send size={17} />
                {busy ? "접수 확인 중…" : "접근 요청 접수"}
              </Button>
            </>
          )}
        </div>
      </Modal>
    </>
  );
}
export default function AccessRequestsPage() {
  const { user, workspace, notify, reload } = useApp(),
    [params, setParams] = useSearchParams();
  const view = params.get("view") === "received" ? "received" : "sent",
    [data, setData] = useState<RequestLists | null>(null),
    [error, setError] = useState<unknown>(null),
    [selected, setSelected] = useState<AccessRequest | null>(null),
    [action, setAction] = useState("grant"),
    [consent, setConsent] = useState(false),
    [busy, setBusy] = useState(false),
    [refresh, setRefresh] = useState(0);
  const scope = `${user.id}:${workspace?.id}`,
    current = useRef(scope),
    selectedScope = useRef(""),
    loaded = useRef(""),
    generation = useRef(0);
  current.current = scope;
  useEffect(() => {
    let active = true;
    generation.current++;
    setData(null);
    setSelected(null);
    setError(null);
    setBusy(false);
    if (!workspace) return;
    const load = () =>
      api<RequestLists>(`/access-requests?workspace_id=${workspace.id}`)
        .then((value) => {
          if (active && current.current === scope) {
            loaded.current = scope;
            setData(value);
          }
        })
        .catch((e) => {
          if (active && current.current === scope) {
            setData(null);
            setSelected(null);
            setError(e);
          }
        });
    void load();
    const timer = setInterval(() => void load(), 5000);
    return () => {
      active = false;
      generation.current++;
      clearInterval(timer);
    };
  }, [scope, refresh]);
  const lists = loaded.current === scope ? data : null,
    rows = view === "received" ? lists?.incoming : lists?.mine;
  const apply = async () => {
    if (!selected || busy) return;
    const revision = generation.current;
    setBusy(true);
    setError(null);
    const fresh = () =>
      revision === generation.current && scope === current.current;
    try {
      await api(`/access-requests/${selected.id}`, "PUT", {
        revision: selected.revision,
        action,
        expected_document_version: selected.document_version,
      });
      if (!fresh()) return;
      setSelected(null);
      setRefresh((v) => v + 1);
      notify(
        action === "grant"
          ? "접근 권한을 허용했습니다. 게시 승인을 처리한 것은 아닙니다."
          : action === "reject"
            ? "요청을 거절했습니다. 문서 권한은 바뀌지 않았습니다."
            : "접근 요청을 취소했습니다.",
      );
      await reload();
    } catch (e) {
      if (fresh()) setError(e);
    } finally {
      if (fresh()) setBusy(false);
    }
  };
  const select = (row: AccessRequest, next: string) => {
    selectedScope.current = scope;
    setSelected(row);
    setAction(next);
    setConsent(false);
    setError(null);
  };
  return (
    <div className="access-requests-page">
      <PageHeading
        eyebrow="ACCESS REQUESTS"
        title="문서 접근 요청"
        description="보안 접근 권한 요청입니다. 문서 작성 검토·게시 승인과는 별도로 처리합니다."
        actions={
          <Button onClick={() => setRefresh((v) => v + 1)} disabled={busy}>
            <RefreshCw size={17} />
            새로고침
          </Button>
        }
      />
      <div className="tabs">
        <button
          className={view === "sent" ? "active" : ""}
          onClick={() => setParams({ view: "sent" })}
        >
          내가 보낸 요청
        </button>
        <button
          className={view === "received" ? "active" : ""}
          onClick={() => setParams({ view: "received" })}
        >
          내 문서에 온 요청
        </button>
      </div>
      {error && !selected ? (
        <RecoveryNotice
          error={error}
          onRetry={() => setRefresh((v) => v + 1)}
        />
      ) : !lists ? (
        <Loading />
      ) : !rows?.length ? (
        <Empty
          title="표시할 접근 요청이 없습니다"
          text={
            view === "received"
              ? "현재 소유하고 접근 가능한 문서의 요청만 표시합니다."
              : "문서를 열 수 없을 때 접근 권한 요청을 보낼 수 있습니다."
          }
        />
      ) : (
        <>
          <p className="notice subtle">
            {lists.notice} 최근 {lists.limit}개까지 표시합니다.
          </p>
          <div className="access-request-list">
            {rows.map((row) => (
              <article className="card" key={row.id}>
                <header>
                  <div>
                    <h3>
                      {view === "received"
                        ? row.document_title
                        : "문서 접근 요청"}
                    </h3>
                    <p>
                      {view === "received"
                        ? `${row.requester_name} · ${row.permission === "write" ? "편집" : "읽기"} 요청`
                        : `요청한 문서 ID: ${row.document_id}`}
                    </p>
                  </div>
                  <span className="badge">
                    {statuses[row.status] || "알 수 없는 상태"}
                  </span>
                </header>
                {view === "received" && row.reason && (
                  <p className="access-request-reason">{row.reason}</p>
                )}
                <footer>
                  <span>{datetime(row.created_at)}</span>
                  <div className="button-row">
                    {view === "received" && (
                      <Link
                        className="button"
                        to={`/app/documents/${row.document_id}?mode=preview`}
                      >
                        문서 확인
                      </Link>
                    )}
                    {row.status === "pending" &&
                      (view === "received" ? (
                        <>
                          <Button onClick={() => select(row, "reject")}>
                            거절
                          </Button>
                          <Button
                            variant="primary"
                            onClick={() => select(row, "grant")}
                          >
                            접근 허용
                          </Button>
                        </>
                      ) : (
                        <Button onClick={() => select(row, "cancel")}>
                          요청 취소
                        </Button>
                      ))}
                  </div>
                </footer>
              </article>
            ))}
          </div>
        </>
      )}
      <Modal
        open={!!selected && selectedScope.current === scope}
        onOpenChange={(v) => {
          if (!v && !busy) {
            setSelected(null);
            setError(null);
          }
        }}
        title={
          action === "grant"
            ? "접근 권한 변경 확인"
            : action === "reject"
              ? "접근 요청 거절"
              : "접근 요청 취소"
        }
        description="확인한 요청과 문서 버전만 처리합니다. 변경 중 권한이나 버전이 달라지면 적용하지 않습니다."
      >
        {selected && selectedScope.current === scope && (
          <ChangeReview
            title={
              action === "grant"
                ? "공유 범위와 권한을 확인하세요"
                : "요청 상태만 변경합니다"
            }
            changes={[
              {
                label:
                  action === "grant"
                    ? selected.document_title || "문서"
                    : "내 요청",
                before:
                  action === "grant"
                    ? visibilityLabel(selected.visibility)
                    : "응답 대기",
                after:
                  action === "grant"
                    ? `${selected.requester_name}: ${selected.permission === "write" ? "편집" : "읽기"} · ${selected.visibility === "private" ? "선택한 사용자" : visibilityLabel(selected.visibility)}`
                    : action === "reject"
                      ? "요청 거절 · 권한 변경 없음"
                      : "요청 취소",
              },
            ]}
            warnings={
              action === "grant"
                ? [
                    "상위 문서·공간·워크스페이스의 역할 상한이 적용됩니다. 요청한 권한을 실제로 제공할 수 없다면 전체 변경을 취소합니다.",
                    ...(selected.visibility === "private"
                      ? [
                          "나만 보기에서 선택한 사용자 공유로 변경합니다. 비활성 개별 공유가 남아 있으면 먼저 공유 설정을 정리해야 합니다.",
                        ]
                      : []),
                  ]
                : []
            }
            disabled={action === "grant" && !consent}
            busy={busy}
            confirmLabel={
              action === "grant"
                ? "확인한 접근 권한 허용"
                : action === "reject"
                  ? "요청 거절"
                  : "요청 취소"
            }
            onConfirm={() => void apply()}
            onCancel={() => {
              setSelected(null);
              setError(null);
            }}
          >
            {action === "grant" && (
              <label className="check-label">
                <input
                  type="checkbox"
                  checked={consent}
                  disabled={busy}
                  onChange={(e) => setConsent(e.target.checked)}
                />
                위 사용자·권한·공유 범위 변경을 확인했습니다.
              </label>
            )}
            {error ? <RecoveryNotice error={error} /> : null}
          </ChangeReview>
        )}
      </Modal>
    </div>
  );
}
