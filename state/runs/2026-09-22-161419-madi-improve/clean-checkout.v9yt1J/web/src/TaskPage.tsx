import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useRef,
  useState,
} from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  CalendarDays,
  Eye,
  CheckCircle2,
  Columns3,
  FileText,
  GripVertical,
  List,
  Plus,
  RefreshCw,
  Settings2,
} from "lucide-react";
import { api, type Doc } from "./api";
import { useApp } from "./context";
import {
  Badge,
  Button,
  Empty,
  ErrorBox,
  Field,
  Modal,
  PageHeading,
} from "./ui";
import TaskCalendar, { dayKey, todayKey } from "./TaskCalendar";
import "./tasks.css";
import NaturalDateInput from "./personalization/NaturalDateInput";
import SaveToWorkset from "./worksets/SaveToWorkset";
const DocumentPreview = lazy(() => import("./review/DocumentPreview"));
type Row = Record<string, any>;
const statuses: Record<string, string> = {
    backlog: "나중에",
    todo: "할 일",
    doing: "진행 중",
    review: "검토 대기",
    done: "완료",
  },
  priorities: Record<string, string> = {
    low: "낮음",
    normal: "보통",
    high: "높음",
    urgent: "긴급",
  };
export default function TaskPage() {
  const {
      workspace,
      user,
      documents,
      reload,
      notify,
      publicInfo,
      createDocument,
    } = useApp(),
    [query, setQuery] = useSearchParams();
  const view = ["list", "board", "calendar"].includes(query.get("view") || "")
      ? query.get("view")!
      : "list",
    filter = query.get("filter") || "all",
    team = query.get("team") || "",
    search = query.get("q") || "",
    sourceDocument = query.get("document_id") || "",
    taskFilter = query.get("task") || "",
    today = todayKey(user.preferences?.timezone || "Asia/Seoul");
  const [items, setItems] = useState<Row[]>([]),
    [preview, setPreview] = useState<Row | null>(null),
    [people, setPeople] = useState<Row[]>([]),
    [teams, setTeams] = useState<Row[]>([]),
    [teamMembers, setTeamMembers] = useState<string[]>([]),
    [error, setError] = useState<unknown>(null),
    [busy, setBusy] = useState(false),
    [truncated, setTruncated] = useState(false),
    [editing, setEditing] = useState<Row | null>(null),
    [newTask, setNewTask] = useState(false),
    [taskText, setTaskText] = useState(""),
    [targetDoc, setTargetDoc] = useState(""),
    [revision, setRevision] = useState(0);
  const writable =
      !!workspace &&
      ["owner", "admin", "editor"].includes(workspace.role) &&
      user.role !== "viewer",
    activeStatuses = Object.entries(statuses).filter(
      ([id]) => id !== "review" || publicInfo.approval_enabled,
    );
  const status = (t: Row) =>
    !publicInfo.approval_enabled && t.status === "review" ? "doing" : t.status;
  const set = (key: string, value: string) => {
    const next = new URLSearchParams(query);
    if (value) next.set(key, value);
    else next.delete(key);
    setQuery(next);
  };
  const loadScope = `${user.id}:${workspace?.id}:${sourceDocument}:${taskFilter}`;
  const currentLoadScope = useRef(loadScope),
    loadRequest = useRef(0);
  currentLoadScope.current = loadScope;
  const load = useCallback(async () => {
    if (!workspace) return;
    const request = ++loadRequest.current;
    const fresh = () =>
      currentLoadScope.current === loadScope && loadRequest.current === request;
    try {
      if (
        sourceDocument &&
        !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(
          sourceDocument,
        )
      )
        throw Error(
          "문서 ID를 확인하세요. 잘못된 문서 조건으로 전체 할 일을 조회하지 않습니다.",
        );
      const [data, members, groups] = await Promise.all([
        api(
          `/tasks/board?workspace_id=${workspace.id}${sourceDocument ? `&document_id=${encodeURIComponent(sourceDocument)}` : ""}${taskFilter ? `&task=${encodeURIComponent(taskFilter)}` : ""}`,
        ),
        api<Row[]>(`/workspaces/${workspace.id}/members`),
        api<Row[]>(`/teams?workspace_id=${workspace.id}`),
      ]);
      if (!fresh()) return;
      setItems(data.items);
      setTruncated(data.truncated);
      setPeople(members);
      setTeams(groups);
      setError(null);
    } catch (e) {
      if (!fresh()) return;
      setItems([]);
      setTruncated(false);
      setError(e);
    }
  }, [workspace?.id, user.id, sourceDocument, taskFilter, loadScope]);
  useEffect(() => {
    setPreview(null);
    setItems([]);
    setPeople([]);
    setTeams([]);
    void load();
    return () => {
      loadRequest.current++;
    };
  }, [load]);
  useEffect(() => {
    setTeamMembers([]);
    if (team)
      api<Row[]>(`/teams/${team}/members`)
        .then((rows) => setTeamMembers(rows.map((m) => m.id)))
        .catch(setError);
  }, [team]);
  const documentItems = sourceDocument
    ? items.filter((t) => t.document_id === sourceDocument)
    : items;
  const filtered = documentItems.filter((t) => {
    if (
      search &&
      !`${t.text} ${t.title} ${t.assignee_name || ""}`
        .toLowerCase()
        .includes(search.toLowerCase())
    )
      return false;
    const due = t.due_date || "";
    if (filter === "mine")
      return (
        t.assignee_id === user.id || (!t.assignee_id && t.owner_id === user.id)
      );
    if (filter === "team") return teamMembers.includes(t.assignee_id);
    if (filter === "overdue") return !t.done && due && due < today;
    if (filter === "today") return !t.done && due === today;
    if (filter === "week") {
      const end = new Date(`${today}T12:00:00`);
      end.setDate(end.getDate() + 7 - (end.getDay() || 7));
      return !t.done && due >= today && due <= dayKey(end);
    }
    if (filter === "open") return !t.done;
    if (filter === "done") return t.done;
    return true;
  });
  async function update(t: Row, patch: Row = {}) {
    setBusy(true);
    setError(null);
    try {
      await api("/tasks/details", "PUT", {
        document_id: t.document_id,
        line: t.line,
        source_start: t.source_start,
        version: t.version,
        task_id: t.task_id || "",
        status: status(t),
        priority: t.priority || "normal",
        assignee_id: t.assignee_id || "",
        due_date: t.due_date || "",
        ...patch,
      });
      setEditing(null);
      await load();
      await reload();
      setRevision((n) => n + 1);
      notify("할 일을 저장했습니다");
      return true;
    } catch (e) {
      setError(e);
      return false;
    } finally {
      setBusy(false);
    }
  }
  const edit = (t: Row) => {
    setError(null);
    setEditing({
      ...t,
      status: status(t),
      assignee_id: t.assignee_id || "",
      due_date: t.due_date || "",
      priority: t.priority || "normal",
      separate: false,
    });
  };
  const card = (t: Row, board = false) => (
    <article
      className={`managed-task ${t.done ? "completed" : ""}`}
      key={`${t.document_id}-${t.source_start}`}
      draggable={board && t.can_write && !busy && !t.ambiguous}
      onDragStart={(e) =>
        e.dataTransfer.setData(
          "application/x-madi-task",
          JSON.stringify({
            document_id: t.document_id,
            source_start: t.source_start,
          }),
        )
      }
    >
      {board ? (
        <GripVertical className="task-grip" size={16} />
      ) : (
        <input
          type="checkbox"
          checked={t.done}
          disabled={!t.can_write || busy || t.ambiguous}
          aria-label={`${t.text} 완료 여부`}
          onChange={(e) =>
            void update(t, { status: e.target.checked ? "done" : "todo" })
          }
        />
      )}
      <div className="managed-task-content">
        <strong>{t.text || "내용 없는 할 일"}</strong>
        <Link to={`/app/documents/${t.document_id}`}>
          <FileText size={14} />
          {t.title}
        </Link>
        <button
          type="button"
          className="text-button"
          onClick={() => setPreview(t)}
          aria-label={`${t.title} 문서 미리보기`}
        >
          <Eye size={15} />
          미리보기
        </button>
        <div className="task-details-line">
          {/^[a-f0-9-]{36}$/i.test(t.task_id || "") && (
            <SaveToWorkset
              item={{
                kind: "task",
                resource_id: t.document_id,
                context: { task_id: t.task_id, version: t.version },
              }}
              label="할 일 보관"
            />
          )}
          {t.assignee_name && (
            <span>
              {t.assignee_name}
              {t.assignee_available === false ? " · 권한 없음" : ""}
            </span>
          )}
          {t.due_date && (
            <span className={!t.done && t.due_date < today ? "overdue" : ""}>
              <CalendarDays size={13} />
              {t.due_date}
            </span>
          )}
          <Badge tone={t.priority === "urgent" ? "red" : ""}>
            {priorities[t.priority] || "보통"}
          </Badge>
          {t.ambiguous && <Badge tone="red">중복 참조 · 분리 필요</Badge>}
        </div>
      </div>
      {board ? (
        <label className="task-status-select">
          <span className="sr-only">{t.text} 상태</span>
          <select
            aria-label={`${t.text} 상태`}
            value={status(t)}
            disabled={!t.can_write || busy || t.ambiguous}
            onChange={(e) => void update(t, { status: e.target.value })}
          >
            {activeStatuses.map(([id, label]) => (
              <option key={id} value={id}>
                {label}
              </option>
            ))}
          </select>
        </label>
      ) : (
        <Badge tone={t.done ? "green" : ""}>{statuses[status(t)]}</Badge>
      )}
      <Button
        disabled={!t.can_write || busy}
        aria-label={`${t.text} 속성 수정`}
        onClick={() => edit(t)}
      >
        <Settings2 size={16} />
        {board ? "속성" : ""}
      </Button>
    </article>
  );
  return (
    <div className="page tasks-page">
      {preview && (
        <Suspense fallback={null}>
          <DocumentPreview
            documentId={preview.document_id}
            expectedVersion={preview.version}
            sourceLine={preview.line}
            onClose={() => setPreview(null)}
          />
        </Suspense>
      )}
      <PageHeading
        eyebrow="ONE STEP AT A TIME"
        title="할 일과 일정"
        description="문서 속 체크리스트에 담당자와 기한을 연결하고, 팀의 진행 상황을 함께 확인하세요."
        actions={
          <div className="button-row">
            <Button disabled={busy} onClick={() => void load()}>
              <RefreshCw size={16} />
              새로 불러오기
            </Button>
            {writable && (
              <Button variant="primary" onClick={() => setNewTask(true)}>
                <Plus size={16} />할 일 추가
              </Button>
            )}
          </div>
        }
      />
      <ErrorBox error={editing || newTask ? null : error} />
      {sourceDocument && (
        <div className="notice subtle">
          <FileText size={19} />
          <span>
            선택한 문서의 할 일만 표시합니다. 현재 열람 가능한 항목에
            한정합니다.
          </span>
          <Button onClick={() => set("document_id", "")}>문서 필터 해제</Button>
        </div>
      )}
      <div className="task-summary">
        <div>
          <strong>{documentItems.length}</strong>
          <span>{truncated ? "조회한 할 일" : "전체 할 일"}</span>
        </div>
        <div>
          <strong>{documentItems.filter((t) => !t.done).length}</strong>
          <span>진행 중</span>
        </div>
        <div>
          <strong>
            {
              documentItems.filter(
                (t) => !t.done && t.due_date && t.due_date < today,
              ).length
            }
          </strong>
          <span>기한 초과</span>
        </div>
        <div>
          <strong>{documentItems.filter((t) => t.done).length}</strong>
          <span>완료</span>
        </div>
      </div>
      <div className="task-toolbar">
        <div className="tabs">
          {[
            ["list", "목록", List],
            ["board", "칸반", Columns3],
            ["calendar", "통합 캘린더", CalendarDays],
          ].map(([id, label, Icon]) => (
            <button
              key={String(id)}
              className={view === id ? "active" : ""}
              onClick={() => set("view", String(id))}
            >
              {typeof Icon !== "string" && <Icon size={16} />} {String(label)}
            </button>
          ))}
        </div>
        {query.get("task") && (
          <Button onClick={() => set("task", "")}>전체 할 일 보기</Button>
        )}
      </div>
      {truncated && (
        <p className="notice">
          현재 열람 가능한 최신 문서 최대 2,000개·본문 16MiB·할 일 10,000개
          범위입니다. 전체 문서 총수는 계산하지 않으며, 특정 문서는 문서
          화면에서 확인하세요.
        </p>
      )}
      {view === "calendar" ? (
        <TaskCalendar revision={revision} onEditTask={edit} />
      ) : (
        <>
          <div className="task-filter-row">
            <Field label="할 일 범위">
              <select
                value={filter}
                onChange={(e) => set("filter", e.target.value)}
              >
                {[
                  ["all", "전체"],
                  ["mine", "내 담당·개인 할 일"],
                  ["team", "팀 담당"],
                  ["open", "미완료"],
                  ["done", "완료"],
                  ["overdue", "기한 초과"],
                  ["today", "오늘 마감"],
                  ["week", "이번 주 마감"],
                ].map(([id, label]) => (
                  <option key={id} value={id}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            {filter === "team" && (
              <Field label="담당 팀">
                <select
                  value={team}
                  onChange={(e) => set("team", e.target.value)}
                >
                  <option value="">팀 선택</option>
                  {teams.map((t) => (
                    <option key={t.id} value={t.id}>
                      {t.name}
                    </option>
                  ))}
                </select>
              </Field>
            )}
            <Field label="할 일 검색">
              <input
                value={search}
                onChange={(e) => set("q", e.target.value)}
                placeholder="내용, 문서, 담당자 검색"
              />
            </Field>
          </div>
          {filtered.length === 0 ? (
            <div className="panel padded">
              <Empty
                title="이 범위에 할 일이 없습니다"
                text="문서에 체크리스트를 쓰거나 ‘할 일 추가’로 시작하세요."
              />
            </div>
          ) : view === "board" ? (
            <div className="task-board-scroll">
              <div
                className="task-board"
                style={{
                  gridTemplateColumns: `repeat(${activeStatuses.length},minmax(250px,1fr))`,
                }}
              >
                {activeStatuses.map(([id, label]) => (
                  <section
                    className={`task-board-column status-${id}`}
                    key={id}
                    onDragOver={(e) => {
                      if (!busy) e.preventDefault();
                    }}
                    onDrop={(e) => {
                      e.preventDefault();
                      try {
                        const key = JSON.parse(
                          e.dataTransfer.getData("application/x-madi-task"),
                        );
                        const t = items.find(
                          (t) =>
                            t.document_id === key.document_id &&
                            t.source_start === key.source_start,
                        );
                        if (t && t.can_write && !t.ambiguous && !busy)
                          void update(t, { status: id });
                      } catch {}
                    }}
                  >
                    <h3>
                      {label}
                      <Badge>
                        {filtered.filter((t) => status(t) === id).length}
                      </Badge>
                    </h3>
                    {filtered
                      .filter((t) => status(t) === id)
                      .map((t) => card(t, true))}
                  </section>
                ))}
              </div>
            </div>
          ) : (
            <div className="panel managed-task-list">
              {filtered.map((t) => card(t))}
            </div>
          )}
        </>
      )}
      {editing && (
        <Modal
          open
          title="할 일 속성"
          onOpenChange={(open) => {
            if (!open && !busy) setEditing(null);
          }}
        >
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void update(editing, { separate: !!editing.separate });
            }}
          >
            <p>{editing.text}</p>
            <p className="muted">
              {editing.title} · 문서 v{editing.version}
            </p>
            <ErrorBox error={error} />
            {editing.ambiguous && (
              <label className="task-separate">
                <input
                  type="checkbox"
                  checked={editing.separate}
                  onChange={(e) =>
                    setEditing({ ...editing, separate: e.target.checked })
                  }
                />
                <span>
                  이 항목을 새로운 고유 참조의 별도 작업으로 분리합니다.
                </span>
              </label>
            )}
            <Field label="할 일 담당자">
              <select
                value={editing.assignee_id}
                onChange={(e) =>
                  setEditing({ ...editing, assignee_id: e.target.value })
                }
              >
                <option value="">미지정</option>
                {editing.assignee_id &&
                  !people.some((p) => p.id === editing.assignee_id) && (
                    <option value={editing.assignee_id} disabled>
                      {editing.assignee_name || "기존 담당자"} · 재지정 필요
                    </option>
                  )}
                {people.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} · {p.email}
                  </option>
                ))}
              </select>
            </Field>
            <div className="two-columns">
              <NaturalDateInput
                label="할 일 마감일"
                value={editing.due_date}
                disabled={busy}
                onChange={(value) =>
                  setEditing({ ...editing, due_date: value })
                }
              />
              <Field label="할 일 우선순위">
                <select
                  value={editing.priority}
                  onChange={(e) =>
                    setEditing({ ...editing, priority: e.target.value })
                  }
                >
                  {Object.entries(priorities).map(([id, label]) => (
                    <option key={id} value={id}>
                      {label}
                    </option>
                  ))}
                </select>
              </Field>
            </div>
            <Field label="할 일 상태">
              <select
                value={editing.status}
                onChange={(e) =>
                  setEditing({ ...editing, status: e.target.value })
                }
              >
                {activeStatuses.map(([id, label]) => (
                  <option key={id} value={id}>
                    {label}
                  </option>
                ))}
              </select>
            </Field>
            <p className="muted">
              고유 ‘작업’ 링크를 Markdown 원문에 함께 보존합니다. 담당자
              지정만으로 문서 접근 권한이 추가되지는 않습니다.
            </p>
            <div className="modal-actions">
              {!!error && (
                <Button
                  type="button"
                  onClick={() => {
                    setEditing(null);
                    void load();
                  }}
                >
                  현재 목록 다시 불러오기
                </Button>
              )}
              <Button
                type="button"
                disabled={busy}
                onClick={() => setEditing(null)}
              >
                취소
              </Button>
              <Button
                variant="primary"
                disabled={busy || (editing.ambiguous && !editing.separate)}
              >
                할 일 저장
              </Button>
            </div>
          </form>
        </Modal>
      )}
      {newTask && (
        <Modal
          open
          title="할 일 추가"
          onOpenChange={(open) => {
            if (!open && !busy) setNewTask(false);
          }}
        >
          <form
            onSubmit={async (e) => {
              e.preventDefault();
              setBusy(true);
              setError(null);
              try {
                const markdown = `- [ ] ${taskText.replace(/[\r\n]+/g, " ")}\n`;
                if (targetDoc) {
                  const d = await api<Doc>(`/documents/${targetDoc}`);
                  await api(`/documents/${targetDoc}`, "PUT", {
                    version: d.version,
                    markdown: d.markdown + "\n" + markdown,
                  });
                } else {
                  const d = await createDocument("내 할 일 모음", markdown, {
                    visibility: "private",
                  });
                  if (!d) throw new Error("할 일 문서를 만들지 못했습니다");
                }
                setNewTask(false);
                setTaskText("");
                await load();
                await reload();
                setRevision((n) => n + 1);
                notify("문서에 할 일을 추가했습니다");
              } catch (e) {
                setError(e);
              } finally {
                setBusy(false);
              }
            }}
          >
            <ErrorBox error={error} />
            <Field label="새 할 일 내용">
              <input
                required
                maxLength={1000}
                value={taskText}
                onChange={(e) => setTaskText(e.target.value)}
              />
            </Field>
            <Field label="할 일을 추가할 문서">
              <select
                value={targetDoc}
                onChange={(e) => setTargetDoc(e.target.value)}
              >
                <option value="">새 개인 문서에 저장</option>
                {documents
                  .filter((d) => d.can_write && !d.deleted_at)
                  .map((d) => (
                    <option key={d.id} value={d.id}>
                      {d.title}
                    </option>
                  ))}
              </select>
            </Field>
            <div className="modal-actions">
              <Button
                type="button"
                disabled={busy}
                onClick={() => setNewTask(false)}
              >
                취소
              </Button>
              <Button variant="primary" disabled={busy}>
                할 일 추가
              </Button>
            </div>
          </form>
        </Modal>
      )}
    </div>
  );
}
