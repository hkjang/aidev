import { useCallback, useEffect, useRef, useState } from "react";
import "./jobs.css";
import { Link } from "react-router-dom";
import {
  Activity,
  Clock3,
  Plus,
  RefreshCw,
  Send,
  Trash2,
  Workflow,
} from "lucide-react";
import { api, datetime } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  CopyButton,
  Empty,
  ErrorBox,
  Field,
  Loading,
  Modal,
  PageHeading,
  Toggle,
} from "./ui";

const eventNames: Record<string, string> = {
  "document.created": "문서 생성",
  "document.updated": "문서 수정",
  "document.deleted": "문서 삭제",
  "document.status_changed": "문서 상태 변경",
  "comment.created": "댓글 작성",
  "task.completed": "할 일 완료",
  "database.row.created": "데이터베이스 행 생성",
  "database.row.updated": "데이터베이스 행 수정",
  "date.reached": "예약 시각 도래",
  "database.created": "데이터베이스 생성",
  "database.updated": "데이터베이스 설정 변경",
  "database.deleted": "데이터베이스 삭제",
  "database.row.deleted": "데이터베이스 행 삭제",
};
const actionNames: Record<string, string> = {
  notification: "알림 보내기",
  create_document: "개인 문서 만들기",
  update_document: "문서 변경",
  update_property: "데이터베이스 속성 변경",
  webhook: "Webhook 호출",
  ai: "AI 작성 · 문서에 덧붙이기",
};
const statusNames: Record<string, string> = {
  pending: "대기",
  running: "실행 중",
  succeeded: "완료",
  failed: "실패",
  cancelled: "취소됨",
};
const jobNames: Record<string, string> = {
  "event.dispatch": "이벤트 전달",
  "webhook.deliver": "Webhook 전송",
  "automation.execute": "자동화 실행",
};
type Row = Record<string, any>;
function useRecords(endpoint: string) {
  const [rows, setRows] = useState<Row[]>([]),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true);
  const generation = useRef(0);
  const load = useCallback(async () => {
    const run = ++generation.current;
    setError("");
    try {
      const result = endpoint ? await api<Row[]>(endpoint) : [];
      if (run === generation.current) setRows(result);
    } catch (e) {
      if (run === generation.current) setError((e as Error).message);
    } finally {
      if (run === generation.current) setLoading(false);
    }
  }, [endpoint]);
  useEffect(() => {
    setRows([]);
    setLoading(true);
    void load();
    return () => {
      generation.current++;
    };
  }, [load]);
  return { rows, error, loading, load };
}
function ActionLinks() {
  return (
    <nav className="button-group" aria-label="자동화 관련 메뉴">
      <Link className="button" to="/app/automations">
        자동화
      </Link>
      <Link className="button" to="/app/jobs">
        작업 이력
      </Link>
    </nav>
  );
}
function randomSecret() {
  return Array.from(crypto.getRandomValues(new Uint8Array(32)), (x) =>
    x.toString(16).padStart(2, "0"),
  ).join("");
}

export function WebhooksPage() {
  const { workspace, notify } = useApp();
  const manager = ["owner", "admin"].includes(workspace?.role || "");
  const list = useRecords(
    workspace && manager ? `/webhooks?workspace_id=${workspace.id}` : "",
  );
  const [draft, setDraft] = useState<Row | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const [deliveries, setDeliveries] = useState<{
    name: string;
    rows: Row[];
  } | null>(null);
  const widRef = useRef(workspace?.id);
  widRef.current = workspace?.id;
  useEffect(() => {
    setDraft(null);
    setDeliveries(null);
    setError("");
  }, [workspace?.id]);
  const edit = (row?: Row) => {
    setError("");
    setDraft(
      row
        ? { ...row, secret: "" }
        : {
            name: "",
            url: "",
            secret: randomSecret(),
            events: ["document.created"],
            enabled: true,
            max_attempts: 5,
            timeout_seconds: 15,
          },
    );
  };
  const update = (key: string, value: any) =>
    setDraft((old) => (old ? { ...old, [key]: value } : old));
  async function action(fn: () => Promise<any>, message: string) {
    setBusy(true);
    try {
      await fn();
      notify(message);
      await list.load();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageHeading
        eyebrow="SIGNED DELIVERY"
        title="Webhook"
        description="워크스페이스 이벤트를 서명된 요청으로 내부 시스템에 전달합니다."
        actions={
          <>
            <ActionLinks />
            {manager && (
              <Button variant="primary" onClick={() => edit()}>
                <Plus size={18} />
                Webhook 추가
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={list.error} />
      {!manager ? (
        <Empty
          title="워크스페이스 관리자 전용"
          text="Webhook 주소와 서명 비밀은 공간 소유자 또는 관리자가 설정합니다."
        />
      ) : list.loading ? (
        <Loading />
      ) : (
        <div className="key-grid">
          {list.rows.length ? (
            list.rows.map((row) => (
              <article className="panel padded" key={row.id}>
                <div className="section-heading">
                  <h2>{row.name}</h2>
                  <Badge tone={row.enabled ? "green" : ""}>
                    {row.enabled ? "사용 중" : "중지됨"}
                  </Badge>
                </div>
                <p style={{ overflowWrap: "anywhere" }}>{row.url}</p>
                <p className="muted">
                  {row.events
                    .map((x: string) => eventNames[x] || x)
                    .join(" · ")}
                </p>
                <p className="muted">
                  최대 {row.max_attempts}회 시도 · {row.timeout_seconds}초 제한
                  · 서명 비밀 암호화 저장
                </p>
                <div className="button-group">
                  <Button onClick={() => edit(row)}>설정 변경</Button>
                  <Button
                    disabled={busy || !row.enabled}
                    onClick={() =>
                      void action(
                        () => api(`/webhooks/${row.id}/test`, "POST", {}),
                        "테스트 전송을 작업 큐에 추가했습니다.",
                      )
                    }
                  >
                    <Send size={16} />
                    테스트
                  </Button>
                  <Button
                    disabled={busy}
                    onClick={async () => {
                      const wid = workspace?.id;
                      try {
                        const data = await api<Row[]>(
                          `/webhooks/${row.id}/deliveries`,
                        );
                        if (widRef.current === wid)
                          setDeliveries({ name: row.name, rows: data });
                      } catch (e) {
                        notify((e as Error).message, "error");
                      }
                    }}
                  >
                    전송 이력
                  </Button>
                  <Button
                    disabled={busy}
                    onClick={() => {
                      if (
                        confirm(
                          `'${row.name}' Webhook을 삭제할까요? 대기 작업도 더 이상 전송되지 않습니다.`,
                        )
                      )
                        void action(
                          () => api(`/webhooks/${row.id}`, "DELETE"),
                          "Webhook을 삭제했습니다.",
                        );
                    }}
                  >
                    <Trash2 size={16} />
                    삭제
                  </Button>
                </div>
              </article>
            ))
          ) : (
            <Empty
              title="연결된 Webhook이 없습니다"
              text="내부 자동화 서버나 업무 시스템의 수신 주소를 추가하세요."
            />
          )}
        </div>
      )}
      <section className="panel padded" style={{ marginTop: 24 }}>
        <h2>수신 서버에서 검증할 항목</h2>
        <p>
          서명은{" "}
          <code>
            HMAC-SHA256(비밀, timestamp + "." + deliveryID + "." + 원본 본문)
          </code>
          입니다. <code>X-Madi-Timestamp</code>의 5분 이내 여부와{" "}
          <code>X-Madi-Signature</code>를 검증하세요. 성공한{" "}
          <code>X-Madi-Delivery</code> ID를 저장해 재시도 중복을 막으세요.
        </p>
        <p className="muted">
          2xx 응답은 완료로 처리합니다. 연결 오류·408·429·5xx는 지수 지연 후
          재시도하며, 리다이렉트는 따르지 않습니다. 사내 HTTP/HTTPS를 허용하므로
          수신 주소는 신뢰할 수 있는 관리자만 지정합니다.
        </p>
      </section>
      <Modal
        open={!!draft}
        onOpenChange={(open) => {
          if (!open && !busy) setDraft(null);
        }}
        title={draft?.id ? "Webhook 설정 변경" : "Webhook 추가"}
        description="비밀은 저장 후 다시 표시되지 않습니다. 수신 서버에 먼저 안전하게 전달하세요."
      >
        {draft && (
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                await api(
                  `/webhooks${draft.id ? `/${draft.id}` : ""}`,
                  draft.id ? "PUT" : "POST",
                  { ...draft, workspace_id: workspace?.id },
                );
                setDraft(null);
                await list.load();
                notify("Webhook을 저장했습니다.");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={error} />
            <Field label="이름">
              <input
                value={draft.name}
                onChange={(e) => update("name", e.target.value)}
                required
                maxLength={120}
              />
            </Field>
            <Field
              label="수신 URL"
              hint="내부 HTTP 또는 HTTPS 주소. 리다이렉트·링크 로컬 주소는 허용하지 않습니다."
            >
              <input
                type="url"
                value={draft.url}
                onChange={(e) => update("url", e.target.value)}
                placeholder="https://automation.internal/events"
                required
              />
            </Field>
            <Field
              label={draft.id ? "서명 비밀 교체 (빈 값은 유지)" : "서명 비밀"}
            >
              <div className="input-with-button">
                <input
                  type="password"
                  autoComplete="new-password"
                  value={draft.secret}
                  minLength={16}
                  required={!draft.id}
                  onChange={(e) => update("secret", e.target.value)}
                />
                <CopyButton value={draft.secret} />
              </div>
            </Field>
            <Button
              type="button"
              onClick={() => update("secret", randomSecret())}
            >
              <RefreshCw size={16} />새 비밀 생성
            </Button>
            <Field label="전달할 이벤트">
              <div className="checkbox-options">
                {Object.entries(eventNames).map(([key, label]) => (
                  <label key={key}>
                    <input
                      type="checkbox"
                      checked={draft.events.includes(key)}
                      onChange={(e) =>
                        update(
                          "events",
                          e.target.checked
                            ? [...draft.events, key]
                            : draft.events.filter((x: string) => x !== key),
                        )
                      }
                    />
                    {label}
                  </label>
                ))}
              </div>
            </Field>
            <div className="form-grid">
              <Field label="최대 시도 횟수">
                <input
                  type="number"
                  min={1}
                  max={10}
                  required
                  value={draft.max_attempts}
                  onChange={(e) =>
                    update("max_attempts", Number(e.target.value))
                  }
                />
              </Field>
              <Field label="응답 제한 (초)">
                <input
                  type="number"
                  min={1}
                  max={120}
                  required
                  value={draft.timeout_seconds}
                  onChange={(e) =>
                    update("timeout_seconds", Number(e.target.value))
                  }
                />
              </Field>
            </div>
            <Toggle
              label="Webhook 사용"
              checked={draft.enabled}
              onChange={(v) => update("enabled", v)}
            />
            <Button variant="primary" disabled={busy || !draft.events.length}>
              {busy ? "저장 중…" : "저장"}
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        open={!!deliveries}
        onOpenChange={(v) => {
          if (!v) setDeliveries(null);
        }}
        title={`${deliveries?.name || ""} 전송 이력`}
        description="최신 100개 전송입니다. 자세한 오류와 재시도는 작업 이력에서 확인하세요."
        wide
      >
        {deliveries?.rows.length ? (
          <div className="table-scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th>생성 시각</th>
                  <th>상태</th>
                  <th>시도</th>
                  <th>전송 ID</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.rows.map((row) => (
                  <tr key={row.id}>
                    <td>{datetime(row.created_at)}</td>
                    <td>{statusNames[row.status]}</td>
                    <td>
                      {row.attempts}/{row.max_attempts}
                    </td>
                    <td>
                      <code>{row.id}</code>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <p>아직 전송 이력이 없습니다.</p>
        )}
        <Link className="button" to="/app/jobs">
          작업 이력 열기
        </Link>
      </Modal>
    </>
  );
}

export function AutomationPage() {
  const { workspace, user, documents, notify } = useApp();
  const manager = ["owner", "admin"].includes(workspace?.role || "");
  const canWrite =
    user.role !== "viewer" &&
    ["owner", "admin", "editor"].includes(workspace?.role || "");
  const list = useRecords(
    workspace ? `/automations?workspace_id=${workspace.id}` : "",
  );
  const hooks = useRecords(
    workspace && manager ? `/webhooks?workspace_id=${workspace.id}` : "",
  );
  const [draft, setDraft] = useState<Row | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    setDraft(null);
    setError("");
  }, [workspace?.id]);
  function edit(row?: Row) {
    setError("");
    setDraft(
      row
        ? {
            ...row,
            conditions: { ...row.conditions },
            actions: row.actions.map((a: Row) => ({
              ...a,
              valuesText: JSON.stringify(a.values || {}, null, 2),
            })),
            localTime: row.schedule_at
              ? new Date(
                  new Date(row.schedule_at).getTime() -
                    new Date(row.schedule_at).getTimezoneOffset() * 60000,
                )
                  .toISOString()
                  .slice(0, 16)
              : "",
          }
        : {
            name: "",
            enabled: true,
            trigger: "document.created",
            conditions: {},
            actions: [
              {
                type: "notification",
                title: "{{title}} 문서가 생성되었습니다",
                valuesText: "{}",
              },
            ],
            owner_id: user.id,
            interval_minutes: 0,
            localTime: "",
          },
    );
  }
  const change = (key: string, value: any) =>
    setDraft((old) => (old ? { ...old, [key]: value } : old));
  const changeAction = (index: number, key: string, value: any) =>
    setDraft((old) =>
      old
        ? {
            ...old,
            actions: old.actions.map((a: Row, i: number) =>
              i === index ? { ...a, [key]: value } : a,
            ),
          }
        : old,
    );
  return (
    <>
      <PageHeading
        eyebrow="WHEN · THEN"
        title="자동화"
        description="이벤트와 실행 작업을 연결하고 반복 업무를 줄이세요. 실행 시점의 권한을 그대로 따릅니다."
        actions={
          <>
            <ActionLinks />
            {manager && (
              <Link className="button" to="/app/webhooks">
                Webhook 관리
              </Link>
            )}
            {canWrite && (
              <Button variant="primary" onClick={() => edit()}>
                <Plus size={18} />
                자동화 만들기
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={list.error} />
      <div className="notice subtle">
        <Workflow size={20} />
        <span>
          실행 계정과 이벤트 발생 사용자의 공통 권한 안에서만 동작합니다. 승인
          설정이 켜져 있으면 자동화로 승인 절차를 건너뛸 수 없습니다. 순환은
          최대 5단계로 제한합니다.
        </span>
      </div>
      {list.loading ? (
        <Loading />
      ) : (
        <div className="key-grid">
          {list.rows.length ? (
            list.rows.map((row) => (
              <article className="panel padded" key={row.id}>
                <div className="section-heading">
                  <h2>{row.name}</h2>
                  <Badge tone={row.enabled ? "green" : ""}>
                    {row.enabled ? "사용 중" : "중지됨"}
                  </Badge>
                </div>
                <p>
                  <strong>{eventNames[row.trigger] || row.trigger}</strong> →{" "}
                  {row.actions
                    .map((a: Row) => actionNames[a.type] || a.type)
                    .join(" → ")}
                </p>
                {row.trigger === "date.reached" && (
                  <p className="muted">
                    <Clock3 size={16} />{" "}
                    {row.schedule_at
                      ? datetime(row.schedule_at)
                      : "예약 실행 완료"}
                    {row.interval_minutes
                      ? ` · ${row.interval_minutes}분마다`
                      : " · 한 번 실행"}
                  </p>
                )}
                <div className="button-group">
                  <Button disabled={!canWrite} onClick={() => edit(row)}>
                    설정 변경
                  </Button>
                  <Button
                    disabled={busy || !canWrite}
                    onClick={async () => {
                      if (!confirm(`'${row.name}' 자동화를 삭제할까요?`))
                        return;
                      setBusy(true);
                      try {
                        await api(`/automations/${row.id}`, "DELETE");
                        await list.load();
                        notify("자동화를 삭제했습니다.");
                      } catch (e) {
                        notify((e as Error).message, "error");
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    <Trash2 size={16} />
                    삭제
                  </Button>
                </div>
              </article>
            ))
          ) : (
            <Empty
              title="첫 자동화를 만들어 보세요"
              text="문서가 만들어지면 알림을 보내거나, 지정한 시각에 개인 문서를 만들 수 있습니다."
            />
          )}
        </div>
      )}
      <Modal
        open={!!draft}
        onOpenChange={(v) => {
          if (!v && !busy) setDraft(null);
        }}
        title={draft?.id ? "자동화 설정 변경" : "자동화 만들기"}
        description="설정 변경 시 기존 대기 작업은 이전 설정으로 실행되지 않고 안전하게 중단됩니다."
        wide
      >
        {draft && (
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError("");
              try {
                const actions = draft.actions.map(
                  ({ valuesText, ...a }: Row) => {
                    if (a.type === "update_property") {
                      const values = JSON.parse(valuesText);
                      if (
                        !values ||
                        Array.isArray(values) ||
                        typeof values !== "object"
                      )
                        throw new Error("속성 값은 JSON 객체여야 합니다.");
                      return { ...a, values };
                    }
                    return a;
                  },
                );
                await api(
                  `/automations${draft.id ? `/${draft.id}` : ""}`,
                  draft.id ? "PUT" : "POST",
                  {
                    ...draft,
                    workspace_id: workspace?.id,
                    actions,
                    conditions: draft.conditions.property_id
                      ? {
                          ...draft.conditions,
                          equals: JSON.parse(
                            draft.conditionValueText ??
                              JSON.stringify(draft.conditions.equals ?? ""),
                          ),
                        }
                      : draft.conditions,
                    schedule_at:
                      draft.trigger === "date.reached"
                        ? new Date(draft.localTime).toISOString()
                        : null,
                  },
                );
                setDraft(null);
                await list.load();
                notify("자동화를 저장했습니다.");
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={error} />
            <Field label="자동화 이름">
              <input
                value={draft.name}
                onChange={(e) => change("name", e.target.value)}
                required
                maxLength={120}
              />
            </Field>
            <div className="form-grid">
              <Field label="시작 이벤트">
                <select
                  value={draft.trigger}
                  onChange={(e) => change("trigger", e.target.value)}
                >
                  {Object.entries(eventNames).map(([k, v]) => (
                    <option key={k} value={k}>
                      {v}
                    </option>
                  ))}
                </select>
              </Field>
              {manager && (
                <Field
                  label="실행 계정 ID"
                  hint="기본은 내 계정입니다. 공간에 가입한 서비스 계정의 사용자 ID도 지정할 수 있습니다."
                >
                  <input
                    value={draft.owner_id}
                    onChange={(e) => change("owner_id", e.target.value)}
                    required
                  />
                </Field>
              )}
            </div>
            {draft.trigger === "date.reached" ? (
              <div className="form-grid">
                <Field label="실행 시각 (이 기기의 시간대)">
                  <input
                    type="datetime-local"
                    value={draft.localTime}
                    required
                    onChange={(e) => change("localTime", e.target.value)}
                  />
                </Field>
                <Field label="반복 간격 (분)" hint="0이면 한 번만 실행합니다.">
                  <input
                    type="number"
                    min={0}
                    max={525600}
                    required
                    value={draft.interval_minutes}
                    onChange={(e) =>
                      change("interval_minutes", Number(e.target.value))
                    }
                  />
                </Field>
              </div>
            ) : (
              <div className="form-grid">
                <Field label="특정 문서에서만 실행">
                  <select
                    value={draft.conditions.document_id || ""}
                    onChange={(e) =>
                      change("conditions", {
                        ...draft.conditions,
                        document_id: e.target.value,
                      })
                    }
                  >
                    <option value="">권한이 있는 모든 원본</option>
                    {documents.map((d) => (
                      <option key={d.id} value={d.id}>
                        {d.title}
                      </option>
                    ))}
                  </select>
                </Field>
                <Field label="문서 상태 조건">
                  <select
                    value={draft.conditions.status || ""}
                    onChange={(e) =>
                      change("conditions", {
                        ...draft.conditions,
                        status: e.target.value,
                      })
                    }
                  >
                    <option value="">모든 상태</option>
                    <option value="draft">초안</option>
                    <option value="published">게시됨</option>
                    <option value="review">검토 중</option>
                    <option value="stale">재검토 필요</option>
                    <option value="archived">보관됨</option>
                  </select>
                </Field>
                <Field label="태그 조건">
                  <input
                    value={draft.conditions.tag || ""}
                    onChange={(e) =>
                      change("conditions", {
                        ...draft.conditions,
                        tag: e.target.value,
                      })
                    }
                    placeholder="비우면 모든 태그"
                  />
                </Field>
                {draft.trigger.startsWith("database.row.") && (
                  <>
                    <Field
                      label="데이터베이스 속성 ID 조건"
                      hint="행의 현재 값을 비교합니다. 이벤트 본문에는 속성 값을 담지 않습니다."
                    >
                      <input
                        value={draft.conditions.property_id || ""}
                        onChange={(e) =>
                          change("conditions", {
                            ...draft.conditions,
                            property_id: e.target.value,
                          })
                        }
                      />
                    </Field>
                    <Field
                      label="속성 비교 값 (JSON)"
                      hint={
                        '예: "완료", 100, true. 속성 ID를 비우면 이 조건을 사용하지 않습니다.'
                      }
                    >
                      <input
                        value={
                          draft.conditionValueText ??
                          JSON.stringify(draft.conditions.equals ?? "")
                        }
                        onChange={(e) =>
                          change("conditionValueText", e.target.value)
                        }
                      />
                    </Field>
                  </>
                )}
              </div>
            )}
            <h3>실행할 작업</h3>
            {draft.actions.map((a: Row, index: number) => (
              <section
                className="panel padded"
                key={index}
                style={{ marginBottom: 16 }}
              >
                <div className="section-heading">
                  <strong>작업 {index + 1}</strong>
                  <Button
                    type="button"
                    disabled={draft.actions.length === 1}
                    onClick={() =>
                      change(
                        "actions",
                        draft.actions.filter(
                          (_: Row, i: number) => i !== index,
                        ),
                      )
                    }
                  >
                    <Trash2 size={16} />
                    제거
                  </Button>
                </div>
                <Field label={`작업 ${index + 1} 유형`}>
                  <select
                    value={a.type}
                    onChange={(e) =>
                      change(
                        "actions",
                        draft.actions.map((value: Row, i: number) =>
                          i === index
                            ? { type: e.target.value, valuesText: "{}" }
                            : value,
                        ),
                      )
                    }
                  >
                    {Object.entries(actionNames)
                      .filter(([k]) => manager || k !== "webhook")
                      .map(([k, v]) => (
                        <option key={k} value={k}>
                          {v}
                        </option>
                      ))}
                  </select>
                </Field>
                {["update_document", "ai"].includes(a.type) && (
                  <Field
                    label="대상 문서"
                    hint="비우면 이벤트가 발생한 문서. 두 사용자 모두 수정할 수 있어야 합니다."
                  >
                    <select
                      value={a.document_id || ""}
                      onChange={(e) =>
                        changeAction(index, "document_id", e.target.value)
                      }
                    >
                      <option value="">이벤트 원본 문서</option>
                      {documents
                        .filter((d) => d.can_write !== false)
                        .map((d) => (
                          <option key={d.id} value={d.id}>
                            {d.title}
                          </option>
                        ))}
                    </select>
                  </Field>
                )}
                {[
                  "notification",
                  "create_document",
                  "update_document",
                ].includes(a.type) && (
                  <Field
                    label={
                      a.type === "notification" ? "알림 제목" : "문서 제목"
                    }
                    hint="{{title}}, {{event}}, {{document_id}}를 원본 정보로 바꿉니다."
                  >
                    <input
                      value={a.title || ""}
                      required={a.type === "create_document"}
                      onChange={(e) =>
                        changeAction(index, "title", e.target.value)
                      }
                    />
                  </Field>
                )}
                {a.type === "notification" && (
                  <Field label="수신 사용자 ID (비우면 이벤트 발생 사용자)">
                    <input
                      value={a.user_id || ""}
                      onChange={(e) =>
                        changeAction(index, "user_id", e.target.value)
                      }
                    />
                  </Field>
                )}
                {["create_document", "update_document"].includes(a.type) && (
                  <Field
                    label="Markdown 본문"
                    hint={
                      a.type === "create_document"
                        ? "새 문서는 원본 정보 노출을 막기 위해 개인 문서로 만듭니다."
                        : "입력하면 대상 문서 본문을 교체합니다. 비우면 유지합니다."
                    }
                  >
                    <textarea
                      rows={5}
                      value={a.markdown || ""}
                      onChange={(e) =>
                        changeAction(index, "markdown", e.target.value)
                      }
                    />
                  </Field>
                )}
                {a.type === "update_document" && (
                  <Field label="변경할 문서 상태">
                    <select
                      value={a.status || ""}
                      onChange={(e) =>
                        changeAction(index, "status", e.target.value)
                      }
                    >
                      <option value="">기존 상태 유지</option>
                      <option value="draft">초안</option>
                      <option value="published">게시됨 (승인 정책 적용)</option>
                      <option value="stale">재검토 필요</option>
                      <option value="archived">보관됨</option>
                    </select>
                  </Field>
                )}
                {a.type === "ai" && (
                  <Field
                    label="AI 지시문"
                    hint="대상 문서만 참조해 스트리밍으로 생성하며, 완료된 답변을 본문 끝에 추가합니다."
                  >
                    <textarea
                      rows={4}
                      required
                      value={a.prompt || ""}
                      onChange={(e) =>
                        changeAction(index, "prompt", e.target.value)
                      }
                    />
                  </Field>
                )}
                {a.type === "webhook" && (
                  <Field label="연결할 Webhook">
                    <select
                      required
                      value={a.webhook_id || ""}
                      onChange={(e) =>
                        changeAction(index, "webhook_id", e.target.value)
                      }
                    >
                      <option value="">Webhook 선택</option>
                      {hooks.rows.map((h) => (
                        <option value={h.id} key={h.id}>
                          {h.name}
                        </option>
                      ))}
                    </select>
                    <ErrorBox error={hooks.error} />
                  </Field>
                )}
                {a.type === "update_property" && (
                  <>
                    <div className="form-grid">
                      <Field
                        label="데이터베이스 ID"
                        hint="비우면 데이터베이스 이벤트의 원본을 사용합니다."
                      >
                        <input
                          value={a.database_id || ""}
                          onChange={(e) =>
                            changeAction(index, "database_id", e.target.value)
                          }
                        />
                      </Field>
                      <Field label="행 ID">
                        <input
                          value={a.row_id || ""}
                          onChange={(e) =>
                            changeAction(index, "row_id", e.target.value)
                          }
                        />
                      </Field>
                    </div>
                    <Field
                      label="속성 값 (JSON)"
                      hint={
                        '속성 ID를 키로 입력합니다. 예: {"status": "완료"}. 선택 옵션과 유형을 서버에서 검증합니다.'
                      }
                    >
                      <textarea
                        rows={5}
                        required
                        value={a.valuesText || "{}"}
                        onChange={(e) =>
                          changeAction(index, "valuesText", e.target.value)
                        }
                      />
                    </Field>
                  </>
                )}
              </section>
            ))}
            <Button
              type="button"
              disabled={draft.actions.length >= 10}
              onClick={() =>
                change("actions", [
                  ...draft.actions,
                  { type: "notification", valuesText: "{}" },
                ])
              }
            >
              <Plus size={16} />
              작업 추가 (최대 10개)
            </Button>
            <Toggle
              label="자동화 사용"
              checked={draft.enabled}
              onChange={(v) => change("enabled", v)}
            />
            <Button variant="primary" disabled={busy}>
              {busy ? "저장 중…" : "자동화 저장"}
            </Button>
          </form>
        )}
      </Modal>
    </>
  );
}

export function JobsPage() {
  const { workspace, notify } = useApp();
  const list = useRecords(
    workspace ? `/jobs?workspace_id=${workspace.id}` : "",
  );
  const [detail, setDetail] = useState<Row | null>(null),
    [busy, setBusy] = useState(false),
    [filter, setFilter] = useState("");
  const widRef = useRef(workspace?.id);
  widRef.current = workspace?.id;
  useEffect(() => {
    setDetail(null);
  }, [workspace?.id]);
  useEffect(() => {
    const timer = setInterval(() => void list.load(), 5000);
    return () => clearInterval(timer);
  }, [list.load]);
  async function open(id: string) {
    const wid = workspace?.id;
    try {
      const value = await api<Row>(`/jobs/${id}`);
      if (widRef.current === wid) setDetail(value);
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  return (
    <>
      <PageHeading
        eyebrow="DURABLE QUEUE"
        title="작업 이력"
        description="백그라운드 작업의 상태와 재시도 내역을 확인합니다. 5초마다 새로 고칩니다."
        actions={
          <>
            <ActionLinks />
            <Button onClick={() => void list.load()}>
              <RefreshCw size={18} />
              새로고침
            </Button>
          </>
        }
      />
      <ErrorBox error={list.error} />
      <Field label="작업 상태">
        <select value={filter} onChange={(e) => setFilter(e.target.value)}>
          <option value="">모든 상태</option>
          {Object.entries(statusNames).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </select>
      </Field>
      {list.loading ? (
        <Loading />
      ) : list.rows.length ? (
        <div
          className="panel table-scroll jobs-history-scroll"
          role="region"
          aria-label="작업 이력 표"
          tabIndex={0}
        >
          <p className="jobs-scroll-hint">
            표를 좌우로 스크롤하면 생성 시각과 관리 버튼을 볼 수 있습니다.
          </p>
          <table className="data-table jobs-history-table">
            <thead>
              <tr>
                <th>작업</th>
                <th>상태</th>
                <th>시도</th>
                <th>생성 시각</th>
                <th>관리</th>
              </tr>
            </thead>
            <tbody>
              {list.rows
                .filter((j) => !filter || j.status === filter)
                .map((j) => (
                  <tr key={j.id}>
                    <td>
                      <Activity size={16} /> {jobNames[j.kind] || j.kind}
                    </td>
                    <td>
                      <Badge
                        tone={
                          j.status === "succeeded"
                            ? "green"
                            : j.status === "failed"
                              ? "red"
                              : ""
                        }
                      >
                        {statusNames[j.status]}
                      </Badge>
                    </td>
                    <td>
                      {j.attempts}/{j.max_attempts}
                    </td>
                    <td>{datetime(j.created_at)}</td>
                    <td>
                      <Button onClick={() => void open(j.id)}>상세 보기</Button>
                    </td>
                  </tr>
                ))}
            </tbody>
          </table>
        </div>
      ) : (
        <Empty
          title="아직 작업 이력이 없습니다"
          text="문서 이벤트, 자동화, Webhook 작업이 여기에 표시됩니다."
        />
      )}
      <Modal
        open={!!detail}
        onOpenChange={(v) => {
          if (!v) setDetail(null);
        }}
        title="작업 상세"
        description="이미 완료된 외부 전송은 취소로 되돌릴 수 없습니다. 전송 수신자는 ID로 중복을 방지해야 합니다."
        wide
      >
        {detail && (
          <>
            <p>
              <strong>{jobNames[detail.job.kind] || detail.job.kind}</strong> ·{" "}
              {statusNames[detail.job.status]}
            </p>
            <p className="muted" style={{ overflowWrap: "anywhere" }}>
              작업 ID: {detail.job.id}
            </p>
            <p>
              최대 {detail.job.timeout_seconds}초 · {detail.job.attempts}/
              {detail.job.max_attempts}회 시도
            </p>
            <div className="button-group">
              {["pending", "running", "failed", "cancelled"].includes(
                detail.job.status,
              ) && (
                <Button
                  disabled={busy}
                  onClick={async () => {
                    const action = ["failed", "cancelled"].includes(
                      detail.job.status,
                    )
                      ? "retry"
                      : "cancel";
                    setBusy(true);
                    try {
                      await api(`/jobs/${detail.job.id}/${action}`, "POST", {});
                      await Promise.all([list.load(), open(detail.job.id)]);
                      notify(
                        action === "retry"
                          ? "재시도를 예약했습니다."
                          : "취소를 요청했습니다.",
                      );
                    } catch (e) {
                      notify((e as Error).message, "error");
                    } finally {
                      setBusy(false);
                    }
                  }}
                >
                  {["failed", "cancelled"].includes(detail.job.status)
                    ? "다시 실행"
                    : "취소 요청"}
                </Button>
              )}
              <Button onClick={() => void open(detail.job.id)}>
                상태 새로고침
              </Button>
            </div>
            <h3>시도별 결과</h3>
            {detail.attempts.length ? (
              detail.attempts.map((a: Row) => (
                <section
                  className="panel padded"
                  key={a.id}
                  style={{ marginBottom: 12 }}
                >
                  <p>
                    {a.attempt}차 · {statusNames[a.status] || a.status} ·{" "}
                    {a.duration_ms}ms · {datetime(a.created_at)}
                  </p>
                  <ErrorBox error={a.error} />
                  {a.result && Object.keys(a.result).length > 0 && (
                    <pre className="code-example">
                      {JSON.stringify(a.result, null, 2)}
                    </pre>
                  )}
                </section>
              ))
            ) : (
              <p className="muted">아직 완료된 실행 시도가 없습니다.</p>
            )}
          </>
        )}
      </Modal>
    </>
  );
}

export function JobsSettingsPage() {
  const { notify } = useApp();
  const [settings, setSettings] = useState<Row | null>(null),
    [error, setError] = useState(""),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let active = true;
    api<Row>("/admin/jobs/settings")
      .then((v) => {
        if (active) setSettings(v);
      })
      .catch((e) => {
        if (active) setError(e.message);
      });
    return () => {
      active = false;
    };
  }, []);
  return (
    <>
      <PageHeading
        eyebrow="WORKER OPERATIONS"
        title="작업 처리 설정"
        description="서버 작업 큐의 동시 처리량과 이력 보존기간을 조정합니다."
      />
      <ErrorBox error={error} />
      {settings ? (
        <form
          className="panel padded"
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            setError("");
            try {
              await api("/admin/jobs/settings", "PUT", settings);
              notify("작업 처리 설정을 저장했습니다.");
            } catch (e) {
              setError((e as Error).message);
            } finally {
              setBusy(false);
            }
          }}
        >
          <Toggle
            label="새 작업 실행 일시 중지"
            description="진행 중인 작업은 계속됩니다. 대기 작업은 PostgreSQL에 보존됩니다."
            checked={settings.paused}
            onChange={(v) => setSettings({ ...settings, paused: v })}
          />
          <div className="form-grid">
            <Field label="서버 인스턴스별 동시 작업 수">
              <input
                type="number"
                min={1}
                max={16}
                required
                value={settings.concurrency}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    concurrency: Number(e.target.value),
                  })
                }
              />
            </Field>
            <Field label="완료 작업 이력 보존 (일)">
              <input
                type="number"
                min={1}
                max={3650}
                required
                value={settings.retention_days}
                onChange={(e) =>
                  setSettings({
                    ...settings,
                    retention_days: Number(e.target.value),
                  })
                }
              />
            </Field>
          </div>
          <p className="muted">
            작업은 PostgreSQL 잠금으로 중복 점유를 막고, 서비스 재시작 뒤 만료된
            실행 임대를 복구합니다. 관리자는 워크스페이스 작업 이력에서 개별
            작업을 확인할 수 있습니다.
          </p>
          <Button variant="primary" disabled={busy}>
            {busy ? "저장 중…" : "설정 저장"}
          </Button>
        </form>
      ) : (
        !error && <Loading />
      )}
    </>
  );
}
