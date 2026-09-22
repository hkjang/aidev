import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { ChevronLeft, ChevronRight, Plus, Trash2 } from "lucide-react";
import { api } from "./api";
import { useApp } from "./context";
import { Badge, Button, ErrorBox, Field, Modal } from "./ui";
type Row = Record<string, any>;
export function dayKey(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
export function todayKey(timezone: string) {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(new Date());
  return ["year", "month", "day"]
    .map((t) => parts.find((p) => p.type === t)?.value)
    .join("-");
}
const labels: Record<string, string> = {
  task: "할 일",
  database: "데이터베이스",
  daily: "일일 노트",
  meeting: "회의",
  milestone: "마일스톤",
  page: "문서",
  note: "노트",
  decision: "결정 기록",
};
export default function TaskCalendar({
  revision,
  onEditTask,
}: {
  revision: number;
  onEditTask: (task: Row) => void;
}) {
  const { workspace, user, documents, notify } = useApp(),
    [query, setQuery] = useSearchParams();
  const today = todayKey(user.preferences?.timezone || "Asia/Seoul"),
    sourceDocument = query.get("document_id") || "",
    month = /^\d{4}-(0[1-9]|1[0-2])$/.test(query.get("month") || "")
      ? query.get("month")!
      : today.slice(0, 7),
    first = new Date(`${month}-01T12:00:00`);
  const [events, setEvents] = useState<Row[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [truncated, setTruncated] = useState(false),
    [editing, setEditing] = useState<Row | null>(null),
    [deleting, setDeleting] = useState<Row | null>(null),
    [day, setDay] = useState("");
  const writable =
    !!workspace &&
    ["owner", "admin", "editor"].includes(workspace.role) &&
    user.role !== "viewer";
  const loadScope = `${user.id}:${workspace?.id}:${month}:${sourceDocument}`,
    currentScope = useRef(loadScope),
    loadRequest = useRef(0);
  currentScope.current = loadScope;
  const load = useCallback(async () => {
    if (!workspace) return;
    const request = ++loadRequest.current;
    const fresh = () =>
      currentScope.current === loadScope && loadRequest.current === request;
    try {
      if (
        sourceDocument &&
        !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
          sourceDocument,
        )
      )
        throw Error(
          "문서 ID를 확인하세요. 전체 달력으로 대신 조회하지 않습니다.",
        );
      const data = await api(
        `/tasks/calendar?workspace_id=${workspace.id}&month=${month}${sourceDocument ? `&document_id=${encodeURIComponent(sourceDocument)}` : ""}`,
      );
      if (!fresh()) return;
      setEvents(data.events);
      setTruncated(data.truncated);
      setError(null);
    } catch (e) {
      if (!fresh()) return;
      setEvents([]);
      setTruncated(false);
      setError(e);
    }
  }, [workspace?.id, user.id, month, sourceDocument, loadScope]);
  useEffect(() => {
    setEvents([]);
    void load();
    return () => {
      loadRequest.current++;
    };
  }, [load, revision]);
  const changeMonth = (delta: number) => {
    const d = new Date(first.getFullYear(), first.getMonth() + delta, 1, 12);
    const next = new URLSearchParams(query);
    next.set("month", dayKey(d).slice(0, 7));
    setQuery(next);
    setDay("");
  };
  const set = (key: string, value: string) =>
    setEditing((old) => (old ? { ...old, [key]: value } : null));
  const newEvent = (date = today) =>
    setEditing({
      title: "",
      kind: "meeting",
      start_date: date,
      end_date: date,
      visibility: "private",
      document_id: "",
    });
  const active = day
    ? events.filter((e) => e.start <= day && e.end >= day)
    : events;
  return (
    <div className="task-calendar">
      <div className="task-calendar-heading">
        <div className="button-row">
          <Button aria-label="이전 달" onClick={() => changeMonth(-1)}>
            <ChevronLeft size={18} />
          </Button>
          <h2>
            {first.getFullYear()}년 {first.getMonth() + 1}월
          </h2>
          <Button aria-label="다음 달" onClick={() => changeMonth(1)}>
            <ChevronRight size={18} />
          </Button>
        </div>
        {writable && (
          <Button onClick={() => newEvent()}>
            <Plus size={16} />
            회의·마일스톤 추가
          </Button>
        )}
      </div>
      <ErrorBox error={editing ? null : error} />
      {sourceDocument && (
        <p className="notice">
          선택한 문서의 날짜·할 일과 실제로 연결한 일정만 표시합니다. 연결하지
          않은 독립 일정과 데이터베이스 행은 포함하지 않습니다.
        </p>
      )}
      {truncated && (
        <p className="notice">
          표시 한도에 도달했습니다. 날짜와 원본 문서를 좁혀 확인하세요.
        </p>
      )}
      <div className="task-calendar-scroll">
        <div className="task-calendar-grid">
          {["일", "월", "화", "수", "목", "금", "토"].map((v) => (
            <div className="task-calendar-weekday" key={v}>
              {v}
            </div>
          ))}
          {Array.from({ length: 42 }, (_, n) => {
            const d = new Date(
                first.getFullYear(),
                first.getMonth(),
                n - first.getDay() + 1,
                12,
              ),
              key = dayKey(d),
              items = events.filter((e) => e.start <= key && e.end >= key);
            return (
              <div
                className={`task-calendar-day ${d.getMonth() !== first.getMonth() ? "outside" : ""} ${key === today ? "today" : ""} ${day === key ? "selected" : ""}`}
                key={key}
              >
                <button
                  className="task-date-button"
                  onClick={() => setDay(key)}
                  aria-label={`${key} 일정 ${items.length}개 보기`}
                >
                  {d.getDate()}
                  {items.length > 0 && (
                    <small className="task-date-count">{items.length}개</small>
                  )}
                </button>
                {items.slice(0, 3).map((event) => (
                  <button
                    className={`task-calendar-event kind-${event.kind}`}
                    title={event.title}
                    key={event.key}
                    onClick={() => setDay(key)}
                  >
                    <span>{labels[event.kind] || "문서"}</span>
                    {event.title}
                  </button>
                ))}
                {items.length > 3 && (
                  <button
                    className="task-calendar-more"
                    onClick={() => setDay(key)}
                  >
                    +{items.length - 3}개 더 보기
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>
      <div className="task-day-heading">
        <h3>
          {day ? `${day} 일정` : "이번 달 일정"} · {active.length}개
        </h3>
        {day && <Button onClick={() => setDay("")}>이번 달 전체</Button>}
      </div>
      <div className="task-calendar-list">
        {active.length === 0 ? (
          <p className="muted">
            해당 날짜에 등록된 일정이 없습니다. 할 일 마감일·문서의 date
            속성·데이터베이스 날짜가 함께 표시됩니다.
          </p>
        ) : (
          active.map((event) => (
            <article key={event.key}>
              <Badge>{labels[event.kind] || "문서"}</Badge>
              <div>
                <strong>{event.title}</strong>
                <small>
                  {event.start}
                  {event.end !== event.start ? ` ~ ${event.end}` : ""}
                  {event.property ? ` · ${event.property}` : ""}
                </small>
              </div>
              {event.kind === "task" ? (
                <Button
                  disabled={!event.task.can_write}
                  onClick={() => onEditTask(event.task)}
                >
                  할 일 수정
                </Button>
              ) : event.id ? (
                <>
                  {event.document_id && (
                    <Link
                      className="button"
                      to={`/app/documents/${event.document_id}`}
                    >
                      연결 문서
                    </Link>
                  )}
                  {event.can_write && writable && (
                    <Button onClick={() => setEditing(event)}>일정 수정</Button>
                  )}
                </>
              ) : (
                <Link className="button" to={event.url}>
                  원본 보기
                </Link>
              )}
            </article>
          ))
        )}
      </div>
      {editing && (
        <Modal
          open
          title={editing.id ? "일정 수정" : "회의·마일스톤 추가"}
          onOpenChange={(open) => {
            if (!open && !busy) setEditing(null);
          }}
        >
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              try {
                await api(
                  `/tasks/calendar/events${editing.id ? "/" + editing.id : ""}`,
                  editing.id ? "PUT" : "POST",
                  {
                    workspace_id: workspace?.id,
                    title: editing.title,
                    kind: editing.kind,
                    start_date: editing.start_date,
                    end_date: editing.end_date,
                    visibility: editing.visibility,
                    document_id: editing.document_id || "",
                    version: editing.version || 0,
                  },
                );
                setEditing(null);
                await load();
                notify("일정을 저장했습니다");
              } catch (e) {
                setError(e);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={error} />
            <Field label="일정 제목">
              <input
                required
                maxLength={150}
                value={editing.title}
                onChange={(e) => set("title", e.target.value)}
              />
            </Field>
            <Field label="일정 종류">
              <select
                value={editing.kind}
                onChange={(e) => set("kind", e.target.value)}
              >
                <option value="meeting">회의</option>
                <option value="milestone">프로젝트 마일스톤</option>
              </select>
            </Field>
            <div className="two-columns">
              <Field label="시작일">
                <input
                  type="date"
                  required
                  value={editing.start_date}
                  onChange={(e) => set("start_date", e.target.value)}
                />
              </Field>
              <Field label="종료일">
                <input
                  type="date"
                  required
                  min={editing.start_date}
                  value={editing.end_date}
                  onChange={(e) => set("end_date", e.target.value)}
                />
              </Field>
            </div>
            <Field label="일정 공유 범위">
              <select
                value={editing.visibility}
                onChange={(e) => set("visibility", e.target.value)}
              >
                <option value="private">나만 보기</option>
                <option value="workspace">워크스페이스 멤버</option>
              </select>
            </Field>
            <Field
              label="연결할 문서"
              hint="연결 문서가 비공개이면 문서에 접근할 수 있는 사용자에게만 일정이 표시됩니다."
            >
              <select
                value={editing.document_id || ""}
                onChange={(e) => set("document_id", e.target.value)}
              >
                <option value="">연결하지 않음</option>
                {documents
                  .filter((d) => !d.deleted_at)
                  .map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
              </select>
            </Field>
            <div className="modal-actions">
              {editing.id && (
                <Button
                  type="button"
                  disabled={busy}
                  onClick={() => {
                    setDeleting(editing);
                    setEditing(null);
                  }}
                >
                  <Trash2 size={16} />
                  일정 삭제
                </Button>
              )}
              <Button
                type="button"
                disabled={busy}
                onClick={() => setEditing(null)}
              >
                취소
              </Button>
              <Button variant="primary" disabled={busy}>
                일정 저장
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {deleting && (
        <Modal
          open
          title="일정 삭제"
          onOpenChange={(open) => {
            if (!open && !busy) setDeleting(null);
          }}
        >
          <p>
            “{deleting.title}” 일정을 삭제할까요? 연결 문서나 할 일은 삭제되지
            않습니다.
          </p>
          <ErrorBox error={error} />
          <div className="modal-actions">
            <Button onClick={() => setDeleting(null)} disabled={busy}>
              취소
            </Button>
            <Button
              variant="danger"
              disabled={busy}
              onClick={async () => {
                setBusy(true);
                try {
                  await api(`/tasks/calendar/events/${deleting.id}`, "DELETE");
                  setDeleting(null);
                  await load();
                } catch (e) {
                  setError(e);
                } finally {
                  setBusy(false);
                }
              }}
            >
              일정 삭제
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
}
