import { useEffect, useState, useRef } from "react";
import EditableGrid from "./database/EditableGrid";
import DatabaseViews, { type ViewState } from "./database/DatabaseViews";
import RowDetailPanel from "./database/RowDetailPanel";
import MobileTableLayout from "./database/MobileTableLayout";
import NaturalDateInput from "./personalization/NaturalDateInput";
import SaveToWorkset from "./worksets/SaveToWorkset";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import {
  CalendarDays,
  ChevronLeft,
  ChevronRight,
  Columns3,
  Database as DatabaseIcon,
  GripVertical,
  Plus,
  Search,
  Settings2,
  Table2,
  Trash2,
  X,
  GalleryVerticalEnd,
  List,
  ChartNoAxesGantt,
} from "lucide-react";
import { api, date, type Property } from "./api";
import {
  AdvancedCell,
  AdvancedPropertyFields,
  AdvancedViews,
  DatabaseAIAction,
  DatabaseAdvancedStyles,
  DatabaseQueryControls,
  RelationInput,
  advancedPropertyNames,
  derivedProperty,
  effectiveValue,
  plainValue,
  type AdvancedDatabase as Database,
  type AdvancedProperty,
  type AdvancedRow as Row,
  type DatabaseFilter,
  type DatabaseSort,
} from "./DatabaseAdvanced";
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

const propertyNames: Record<string, string> = {
  ...advancedPropertyNames,
  text: "텍스트",
  number: "숫자",
  select: "선택",
  multi_select: "다중 선택",
  date: "날짜",
  checkbox: "체크박스",
  url: "URL",
  email: "이메일",
  user: "사용자",
};
const multiValues = (value: unknown): string[] =>
  Array.isArray(value)
    ? value.filter((v) => typeof v === "string")
    : typeof value === "string" && value
      ? [value]
      : [];
function newProperty(): Property {
  return {
    id: crypto.randomUUID(),
    name: "새 속성",
    type: "text",
    options: [],
  };
}
function CellValue({ property, value }: { property: Property; value: any }) {
  if (value === undefined || value === null || value === "")
    return <span className="muted">—</span>;
  if (property.type === "checkbox")
    return (
      <input
        type="checkbox"
        checked={!!value}
        readOnly
        aria-label={property.name}
      />
    );
  if (["select", "multi_select", "status"].includes(property.type))
    return (
      <div className="tags">
        {(Array.isArray(value) ? value : [value]).map((v) => (
          <Badge key={v} tone="green">
            {v}
          </Badge>
        ))}
      </div>
    );
  return <span>{String(value)}</span>;
}

export default function DatabasesPage() {
  const { workspace, user, notify } = useApp();
  const navigate = useNavigate();
  const params = useParams();
  const [viewParams, setViewParams] = useSearchParams();
  const view = [
    "table",
    "board",
    "calendar",
    "list",
    "gallery",
    "timeline",
  ].includes(viewParams.get("view") || "")
    ? viewParams.get("view")!
    : "table";
  const setView = (next: string) =>
    setViewParams((prev) => {
      const nextParams = new URLSearchParams(prev);
      nextParams.set("view", next);
      return nextParams;
    });
  const readArray = <T,>(name: string): T[] => {
    try {
      const value = JSON.parse(viewParams.get(name) || "[]");
      return Array.isArray(value)
        ? value
            .filter(
              (v) =>
                v &&
                typeof v === "object" &&
                typeof v.property_id === "string" &&
                (name === "filters"
                  ? typeof v.operator === "string" &&
                    [
                      "string",
                      "number",
                      "boolean",
                      "object",
                      "undefined",
                    ].includes(typeof v.value)
                  : ["asc", "desc"].includes(v.direction)),
            )
            .slice(0, name === "filters" ? 30 : 10)
        : [];
    } catch {
      return [];
    }
  };
  const filters = readArray<DatabaseFilter>("filters"),
    sorts = readArray<DatabaseSort>("sorts");
  const filtersKey = JSON.stringify(filters),
    sortsKey = JSON.stringify(sorts);
  const offset = Math.min(
    10000,
    Math.max(0, Number(viewParams.get("offset")) || 0),
  );
  const setQueries = (f: DatabaseFilter[], s: DatabaseSort[]) =>
    setViewParams((prev) => {
      const next = new URLSearchParams(prev);
      f.length
        ? next.set("filters", JSON.stringify(f))
        : next.delete("filters");
      s.length ? next.set("sorts", JSON.stringify(s)) : next.delete("sorts");
      next.delete("offset");
      return next;
    });
  const setOffset = (value: number) =>
    setViewParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("offset", String(value));
      return next;
    });
  const selectedId = params["*"]?.split("/")[0] || "";
  const [databases, setDatabases] = useState<Database[]>([]),
    [rows, setRows] = useState<Row[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [query, setQuery] = useState(""),
    [newOpen, setNewOpen] = useState(false),
    [newName, setNewName] = useState(""),
    [propertiesOpen, setPropertiesOpen] = useState(false),
    [properties, setProperties] = useState<AdvancedProperty[]>([]),
    [rowOpen, setRowOpen] = useState(false),
    [editingRow, setEditingRow] = useState<Row | null>(null),
    [values, setValues] = useState<Record<string, any>>({}),
    [busy, setBusy] = useState(false),
    [month, setMonth] = useState(
      new Date(new Date().getFullYear(), new Date().getMonth(), 1),
    );
  const dataScope = `${user.id}:${workspace?.id}`;
  const [loadedScope, setLoadedScope] = useState("");
  const [rowScope, setRowScope] = useState("");
  const [aiTarget, setAITarget] = useState<{
      property: AdvancedProperty;
      row: Row;
    } | null>(null),
    [rowLoading, setRowLoading] = useState(false),
    [total, setTotal] = useState(0),
    [truncated, setTruncated] = useState(false);
  const rowGeneration = useRef(0),
    workspaceGeneration = useRef(0);
  const currentRoute = useRef(selectedId),
    currentWorkspace = useRef(workspace?.id),
    currentUser = useRef(user.id);
  currentUser.current = user.id;
  currentRoute.current = selectedId;
  currentWorkspace.current = workspace?.id;
  const sort =
    sorts.length === 1 && sorts[0].direction === "asc"
      ? sorts[0].property_id
      : "";
  const setSort = (id: string) =>
    setQueries(filters, id ? [{ property_id: id, direction: "asc" }] : []);
  const selected =
    loadedScope === dataScope
      ? databases.find((d) => d.id === selectedId)
      : undefined;
  const visibleColumns = (() => {
    try {
      const value = JSON.parse(viewParams.get("columns") || "[]");
      return Array.isArray(value)
        ? value
            .filter(
              (id): id is string =>
                typeof id === "string" &&
                !!selected?.properties.some((p) => p.id === id),
            )
            .slice(0, 100)
        : [];
    } catch {
      return [];
    }
  })();
  const viewState: ViewState = {
    view,
    filters,
    sorts,
    columns: visibleColumns,
    board_property_id: viewParams.get("board_property") || "",
    date_property_id: viewParams.get("date_property") || "",
  };
  const applyView = (data: ViewState, savedID: string) =>
    setViewParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("view", data.view);
      for (const [key, value] of [
        ["filters", data.filters],
        ["sorts", data.sorts],
        ["columns", data.columns],
      ] as const) {
        if (value?.length) next.set(key, JSON.stringify(value));
        else next.delete(key);
      }
      for (const [key, value] of [
        ["saved_view", savedID],
        ["board_property", data.board_property_id],
        ["date_property", data.date_property_id],
      ]) {
        if (value) next.set(key, value);
        else next.delete(key);
      }
      next.delete("offset");
      return next;
    });
  const workspaceWrite =
    !!workspace &&
    ["owner", "admin", "editor"].includes(workspace.role || "") &&
    user?.role !== "viewer";
  const canWrite = selected?.can_write ?? workspaceWrite;
  const load = async () => {
    if (!workspace || currentWorkspace.current !== workspace.id) return;
    const generation = ++workspaceGeneration.current,
      actor = user.id;
    try {
      const result = await api<Database[]>(
        "/databases?workspace_id=" + workspace.id,
      );
      if (
        generation !== workspaceGeneration.current ||
        currentUser.current !== actor ||
        currentWorkspace.current !== workspace.id
      )
        return;
      setDatabases(result);
      setLoadedScope(dataScope);
      setError("");
    } catch (e) {
      if (generation === workspaceGeneration.current)
        setError((e as Error).message);
    } finally {
      if (generation === workspaceGeneration.current) setLoading(false);
    }
  };
  useEffect(() => {
    setLoading(true);
    setDatabases([]);
    setLoadedScope("");
    setRowOpen(false);
    setPropertiesOpen(false);
    setNewOpen(false);
    setBusy(false);
    setAITarget(null);
    load();
    return () => {
      workspaceGeneration.current++;
    };
  }, [workspace?.id, user.id]);
  const refreshRows = async () => {
    if (currentRoute.current !== selectedId) return;
    const generation = ++rowGeneration.current,
      actor = user.id,
      wid = workspace?.id;
    if (!selectedId) {
      setRows([]);
      setTotal(0);
      return;
    }
    setRowLoading(true);
    try {
      const result = await api<{
        rows: Row[];
        total: number;
        truncated: boolean;
      }>(`/databases/${selectedId}/query`, "POST", {
        filters,
        sorts,
        offset,
        limit: 1000,
      });
      if (
        generation !== rowGeneration.current ||
        currentUser.current !== actor ||
        currentWorkspace.current !== wid
      )
        return;
      setRows(result.rows);
      setRowScope(`${dataScope}:${selectedId}`);
      setTotal(result.total);
      setTruncated(result.truncated);
      setError("");
    } catch (e) {
      if (generation === rowGeneration.current) setError((e as Error).message);
    } finally {
      if (generation === rowGeneration.current) setRowLoading(false);
    }
  };
  useEffect(() => {
    setRows([]);
    refreshRows();
    return () => {
      rowGeneration.current++;
    };
  }, [selectedId, filtersKey, sortsKey, offset, user.id, workspace?.id]);
  useEffect(() => {
    if (!selectedId) return;
    let active = true;
    const abort = new AbortController();
    let checking = false;
    const check = async () => {
      if (checking) return;
      checking = true;
      try {
        const result = await api<Database>(
          `/databases/${selectedId}`,
          "GET",
          undefined,
          { signal: abort.signal },
        );
        if (
          active &&
          currentUser.current === user.id &&
          currentWorkspace.current === workspace?.id &&
          currentRoute.current === selectedId
        )
          setDatabases((values) =>
            values.map((d) =>
              d.id === selectedId ? { ...d, can_write: result.can_write } : d,
            ),
          );
      } catch (e) {
        if (active && !abort.signal.aborted) {
          setDatabases((values) => values.filter((d) => d.id !== selectedId));
          setRows([]);
          setRowOpen(false);
          setPropertiesOpen(false);
          setAITarget(null);
          setError(
            "현재 데이터베이스 권한을 확인하지 못했습니다. 다시 연결한 뒤 목록을 확인하세요.",
          );
        }
      } finally {
        checking = false;
      }
    };
    const timer = setInterval(() => void check(), 2000);
    return () => {
      active = false;
      abort.abort();
      clearInterval(timer);
    };
  }, [selectedId, user.id, workspace?.id]);
  useEffect(() => {
    setRowOpen(false);
    setPropertiesOpen(false);
    setAITarget(null);
    setQuery("");
  }, [selectedId]);
  const filtered = (rowScope === `${dataScope}:${selectedId}` ? rows : [])
    .filter((r) =>
      Object.values({ ...r.values, ...r.computed_values })
        .join(" ")
        .toLowerCase()
        .includes(query.toLowerCase()),
    )
    .sort((a, b) =>
      sort
        ? plainValue(effectiveValue(a, sort)).localeCompare(
            plainValue(effectiveValue(b, sort)),
            "ko",
            { numeric: true },
          )
        : 0,
    );
  const editRow = (r?: Row) => {
    if (!r && !canWrite) return;
    setEditingRow(r || null);
    setValues(
      r
        ? Object.fromEntries(
            Object.entries(r.values).filter(
              ([id]) =>
                !selected?.properties.some(
                  (p) => p.id === id && derivedProperty(p),
                ),
            ),
          )
        : {},
    );
    setRowOpen(true);
  };
  const selectProp = selected?.properties.find(
      (p) =>
        ["select", "status"].includes(p.type) &&
        (!viewState.board_property_id || p.id === viewState.board_property_id),
    ),
    dateProp = selected?.properties.find(
      (p) =>
        p.type === "date" &&
        (!viewState.date_property_id || p.id === viewState.date_property_id),
    );
  const updateRow = async (row: Row, newValues: Record<string, any>) => {
    const actor = user.id,
      current = () =>
        currentRoute.current === selectedId &&
        currentWorkspace.current === workspace?.id &&
        currentUser.current === actor;
    try {
      await api<Row>(`/databases/${selectedId}/rows/${row.id}`, "PUT", {
        values: newValues,
        expected_version: row.version,
      });
      if (!current()) return;
      await refreshRows();
      if (current()) notify("항목을 업데이트했습니다.");
    } catch (e) {
      if (current()) notify((e as Error).message, "error");
    }
  };
  const onAction = async (p: AdvancedProperty, row: Row) => {
    if (!canWrite) return;
    if (p.type === "ai") {
      setAITarget({ property: p, row });
      return;
    }
    if (p.type === "button") {
      if (
        !window.confirm(`'${p.name}' 작업으로 이 항목에서 새 문서를 만들까요?`)
      )
        return;
      try {
        const doc = await api<{ id: string }>(
          `/databases/${selectedId}/rows/${row.id}/button/${p.id}`,
          "POST",
          {},
        );
        notify("항목에서 문서를 만들었습니다.");
        navigate("/app/documents/" + doc.id);
      } catch (e) {
        notify((e as Error).message, "error");
      }
    }
  };
  return (
    <>
      <DatabaseAdvancedStyles />
      <PageHeading
        eyebrow="STRUCTURE YOUR KNOWLEDGE"
        title={selected?.name || "데이터베이스"}
        description={
          selected
            ? "정보를 정리하고, 다양한 보기로 흐름을 살펴보세요."
            : "문서에서 한 걸음 더. 프로젝트와 정보를 구조화하세요."
        }
        actions={
          <>
            {workspaceWrite && (
              <Button onClick={() => setNewOpen(true)}>
                <Plus size={18} /> 데이터베이스 만들기
              </Button>
            )}
            {selected && canWrite && (
              <Button variant="primary" onClick={() => editRow()}>
                <Plus size={18} /> 새 항목
              </Button>
            )}
          </>
        }
      />
      <ErrorBox error={error} />
      {loading || loadedScope !== dataScope ? (
        <Loading />
      ) : selected ? (
        <>
          <div className="database-breadcrumb">
            <button
              className="text-button"
              onClick={() => navigate("/app/databases")}
            >
              데이터베이스
            </button>
            <ChevronRight size={14} />
            <span>{selected.name}</span>
            <button
              className="icon-button danger-text"
              aria-label="데이터베이스 삭제"
              disabled={!canWrite}
              onClick={async () => {
                if (
                  !window.confirm(
                    `'${selected.name}' 데이터베이스와 모든 항목을 삭제할까요? 이 작업은 복원할 수 없습니다.`,
                  )
                )
                  return;
                try {
                  await api("/databases/" + selected.id, "DELETE");
                  navigate("/app/databases");
                  await load();
                  notify("데이터베이스를 삭제했습니다.");
                } catch (e) {
                  notify((e as Error).message, "error");
                }
              }}
            >
              <Trash2 size={17} />
            </button>
          </div>
          <div className="database-toolbar">
            <div className="tabs compact">
              {[
                ["table", "테이블", Table2],
                ["board", "보드", Columns3],
                ["calendar", "캘린더", CalendarDays],
                ["list", "목록", List],
                ["gallery", "갤러리", GalleryVerticalEnd],
                ["timeline", "타임라인", ChartNoAxesGantt],
              ].map(([v, l, Icon]) => (
                <button
                  key={String(v)}
                  className={view === v ? "active" : ""}
                  onClick={() => setView(String(v))}
                >
                  {typeof Icon !== "string" && <Icon size={17} />} {String(l)}
                </button>
              ))}
            </div>
            <div className="database-tools">
              <div className="search-field compact">
                <Search size={17} />
                <input
                  aria-label="데이터베이스 항목 검색"
                  placeholder="표시 중인 항목 검색…"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              </div>
              <select
                aria-label="정렬 속성"
                value={sort}
                onChange={(e) => setSort(e.target.value)}
              >
                <option value="">기본 정렬</option>
                {selected.properties.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} 오름차순
                  </option>
                ))}
              </select>
              <DatabaseQueryControls
                properties={selected.properties}
                filters={filters}
                sorts={sorts}
                onChange={setQueries}
              />
              <Button
                disabled={!canWrite}
                onClick={() => {
                  setProperties(selected.properties.map((p) => ({ ...p })));
                  setPropertiesOpen(true);
                }}
              >
                <Settings2 size={16} /> 속성
              </Button>
            </div>
          </div>
          <div className="advanced-query-result">
            {rowLoading
              ? "항목을 불러오는 중…"
              : `${total.toLocaleString()}개 중 ${rows.length.toLocaleString()}개 표시`}
            {truncated && (
              <span>
                · 계산 보호를 위해 첫 10,000개 항목을 대상으로 조회합니다.
              </span>
            )}
          </div>
          <DatabaseViews
            key={`${user.id}:${workspace?.id}:${selectedId}`}
            databaseId={selectedId}
            properties={selected.properties}
            current={viewState}
            selectedId={viewParams.get("saved_view") || ""}
            canWrite={canWrite}
            onApply={applyView}
            onSelect={(id) =>
              setViewParams((prev) => {
                const next = new URLSearchParams(prev);
                if (id) next.set("saved_view", id);
                else next.delete("saved_view");
                return next;
              })
            }
            onDefault={(data, id) => {
              if (
                !viewParams.has("view") &&
                !viewParams.has("filters") &&
                !viewParams.has("sorts")
              )
                applyView(data, id);
            }}
          />
          <SaveToWorkset
            item={{
              kind: "database",
              resource_id: selectedId,
              context: {
                view: viewState,
                ...(viewParams.get("saved_view")
                  ? { view_id: viewParams.get("saved_view") }
                  : {}),
              },
            }}
            label="현재 보기를 작업 묶음에 보관"
          />
          {rowLoading && !rows.length ? (
            <Loading />
          ) : ["list", "gallery", "timeline"].includes(view) ? (
            <AdvancedViews
              key={selectedId}
              view={view}
              database={selected}
              rows={filtered}
              onEdit={editRow}
              onAction={canWrite ? onAction : undefined}
            />
          ) : view === "table" ? (
            <MobileTableLayout
              database={selected}
              rows={filtered}
              columns={visibleColumns}
              onDetail={editRow}
              onAction={onAction}
              canWrite={canWrite}
            >
              <EditableGrid
                key={`${user.id}:${workspace?.id}:${selectedId}`}
                database={selected}
                rows={filtered}
                columns={visibleColumns}
                canWrite={canWrite}
                onDetail={editRow}
                onCreate={() => editRow()}
                onAction={onAction}
                onChanged={refreshRows}
              />
            </MobileTableLayout>
          ) : view === "board" ? (
            selectProp ? (
              <div className="kanban-board">
                {["", ...(selectProp.options || [])].map((option) => (
                  <section
                    key={option}
                    className="kanban-column"
                    onDragOver={(e) => e.preventDefault()}
                    onDrop={(e) => {
                      e.preventDefault();
                      if (!canWrite) return;
                      const row = rows.find(
                        (r) => r.id === e.dataTransfer.getData("text/madi-row"),
                      );
                      if (row)
                        updateRow(row, {
                          ...row.values,
                          [selectProp.id]: option,
                        });
                    }}
                  >
                    <h3>
                      <span className="status-dot" />
                      {option || "미분류"}
                      <Badge>
                        {
                          filtered.filter(
                            (r) => (r.values[selectProp.id] || "") === option,
                          ).length
                        }
                      </Badge>
                    </h3>
                    {filtered
                      .filter((r) => (r.values[selectProp.id] || "") === option)
                      .map((r) => (
                        <button
                          key={r.id}
                          className="kanban-card"
                          draggable={canWrite}
                          onDragStart={(e) =>
                            e.dataTransfer.setData("text/madi-row", r.id)
                          }
                          onClick={() => editRow(r)}
                        >
                          <strong>
                            {String(
                              effectiveValue(r, selected.properties[0]?.id) ||
                                "제목 없는 항목",
                            )}
                          </strong>
                          {selected.properties
                            .filter((p) => !["button", "ai"].includes(p.type))
                            .slice(1, 4)
                            .map((p) => (
                              <div key={p.id}>
                                <small>{p.name}</small>
                                <CellValue
                                  property={p}
                                  value={effectiveValue(r, p.id)}
                                />
                              </div>
                            ))}
                        </button>
                      ))}
                    <button
                      className="text-button"
                      disabled={!canWrite}
                      onClick={() => {
                        editRow();
                        setValues({ [selectProp.id]: option });
                      }}
                    >
                      <Plus size={16} /> 항목 추가
                    </button>
                  </section>
                ))}
              </div>
            ) : (
              <Empty
                title="보드에 사용할 선택 속성이 필요해요"
                text="속성 설정에서 '선택' 유형을 추가하고 상태 옵션을 입력하세요."
                action={
                  <Button
                    onClick={() => {
                      setProperties([
                        ...selected.properties,
                        {
                          id: crypto.randomUUID(),
                          name: "상태",
                          type: "select",
                          options: ["할 일", "진행 중", "완료"],
                        },
                      ]);
                      setPropertiesOpen(true);
                    }}
                  >
                    상태 속성 추가
                  </Button>
                }
              />
            )
          ) : dateProp ? (
            <div className="panel calendar-panel">
              <div className="calendar-heading">
                <h2>
                  {month.toLocaleDateString("ko-KR", {
                    year: "numeric",
                    month: "long",
                  })}
                </h2>
                <div>
                  <button
                    className="icon-button"
                    aria-label="이전 달"
                    onClick={() =>
                      setMonth(
                        new Date(month.getFullYear(), month.getMonth() - 1, 1),
                      )
                    }
                  >
                    <ChevronLeft size={20} />
                  </button>
                  <button
                    className="icon-button"
                    aria-label="다음 달"
                    onClick={() =>
                      setMonth(
                        new Date(month.getFullYear(), month.getMonth() + 1, 1),
                      )
                    }
                  >
                    <ChevronRight size={20} />
                  </button>
                </div>
              </div>
              <div className="calendar-grid">
                {["일", "월", "화", "수", "목", "금", "토"].map((d) => (
                  <div className="calendar-weekday" key={d}>
                    {d}
                  </div>
                ))}
                {Array.from({ length: 42 }, (_, i) => {
                  const d = new Date(
                      month.getFullYear(),
                      month.getMonth(),
                      i - month.getDay() + 1,
                    ),
                    key = d.toLocaleDateString("sv-SE");
                  return (
                    <div
                      key={key}
                      className={`calendar-day ${d.getMonth() !== month.getMonth() ? "outside" : ""}`}
                    >
                      <span>{d.getDate()}</span>
                      {filtered
                        .filter((r) =>
                          String(r.values[dateProp.id] || "").startsWith(key),
                        )
                        .map((r) => (
                          <button key={r.id} onClick={() => editRow(r)}>
                            {String(
                              r.values[selected.properties[0]?.id] || "항목",
                            )}
                          </button>
                        ))}
                    </div>
                  );
                })}
              </div>
            </div>
          ) : (
            <Empty
              title="캘린더에 사용할 날짜 속성이 필요해요"
              text="속성 설정에서 '날짜' 유형을 추가하세요."
            />
          )}
          {(offset > 0 || offset + rows.length < total) && (
            <div className="advanced-query-pages">
              <Button
                disabled={offset === 0 || rowLoading}
                onClick={() => setOffset(Math.max(0, offset - 1000))}
              >
                <ChevronLeft size={16} />
                이전
              </Button>
              <span>
                {Math.floor(offset / 1000) + 1} /{" "}
                {Math.max(1, Math.ceil(total / 1000))}
              </span>
              <Button
                disabled={offset + rows.length >= total || rowLoading}
                onClick={() => setOffset(offset + 1000)}
              >
                다음
                <ChevronRight size={16} />
              </Button>
            </div>
          )}
        </>
      ) : (
        <div className="database-grid">
          {databases.map((db) => (
            <button
              className="database-card panel"
              key={db.id}
              onClick={() => navigate("/app/databases/" + db.id)}
            >
              <span className="feature-icon lavender">
                <DatabaseIcon size={27} />
              </span>
              <h2>{db.name}</h2>
              <p>
                {db.properties.length}개 속성 · {date(db.created_at)} 생성
              </p>
              <div>
                {db.properties.slice(0, 3).map((p) => (
                  <Badge key={p.id}>{p.name}</Badge>
                ))}
                <ChevronRight size={19} />
              </div>
            </button>
          ))}
          {!databases.length && (
            <Empty
              title="우리 팀의 정보를 구조화하세요"
              text="프로젝트, 자산 목록, 고객 기록 등을 테이블과 보드로 관리하세요."
              action={
                <Button variant="primary" onClick={() => setNewOpen(true)}>
                  <Plus size={18} /> 첫 데이터베이스 만들기
                </Button>
              }
            />
          )}
        </div>
      )}
      <Modal
        open={newOpen}
        onOpenChange={setNewOpen}
        title="데이터베이스 만들기"
      >
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (!workspace) return;
            setBusy(true);
            try {
              const db = await api<Database>("/databases", "POST", {
                workspace_id: workspace.id,
                name: newName,
                properties: [
                  {
                    id: crypto.randomUUID(),
                    name: "이름",
                    type: "text",
                    options: [],
                  },
                  {
                    id: crypto.randomUUID(),
                    name: "상태",
                    type: "select",
                    options: ["할 일", "진행 중", "완료"],
                  },
                  {
                    id: crypto.randomUUID(),
                    name: "날짜",
                    type: "date",
                    options: [],
                  },
                ],
              });
              await load();
              setNewOpen(false);
              setNewName("");
              navigate("/app/databases/" + db.id);
              notify("데이터베이스를 만들었습니다.");
            } catch (e) {
              notify((e as Error).message, "error");
            } finally {
              setBusy(false);
            }
          }}
        >
          <Field label="데이터베이스 이름">
            <input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              required
              placeholder="예: 프로젝트 관리"
            />
          </Field>
          <p className="muted">
            이름, 상태, 날짜 속성으로 시작합니다. 나중에 자유롭게 변경할 수
            있어요.
          </p>
          <div className="modal-actions">
            <Button variant="primary" disabled={busy}>
              만들기
            </Button>
          </div>
        </form>
      </Modal>
      <Modal
        open={propertiesOpen && !!selected}
        onOpenChange={setPropertiesOpen}
        title="데이터베이스 속성"
        description="각 열의 이름과 유형, 선택 옵션을 설정하세요."
        wide
      >
        {!!selected?.properties.some(
          (p) => !properties.some((next) => next.id === p.id),
        ) && (
          <div className="notice error">
            <Trash2 size={20} />
            <span>
              삭제한 속성을 저장하면 해당 열에 저장된 모든 항목의 값도
              삭제됩니다. 복원할 수 없으므로 저장 전 확인하세요.
            </span>
          </div>
        )}
        <div className="property-list">
          {properties.map((p, i) => (
            <div className="property-editor" key={p.id}>
              <div>
                <GripVertical size={17} />
                <input
                  aria-label={`속성 ${i + 1} 이름`}
                  value={p.name}
                  onChange={(e) =>
                    setProperties(
                      properties.map((x) =>
                        x.id === p.id ? { ...x, name: e.target.value } : x,
                      ),
                    )
                  }
                />
                <select
                  aria-label={`속성 ${i + 1} 유형`}
                  value={p.type}
                  onChange={(e) =>
                    setProperties(
                      properties.map((x) =>
                        x.id === p.id
                          ? {
                              id: x.id,
                              name: x.name,
                              type: e.target.value,
                              options: [
                                "select",
                                "multi_select",
                                "status",
                              ].includes(e.target.value)
                                ? x.options
                                : [],
                              ...(e.target.value === "rollup"
                                ? { aggregation: "count" }
                                : {}),
                              ...(e.target.value === "ai"
                                ? {
                                    prompt:
                                      "다음 내용을 한 문장으로 요약하세요.",
                                    source_property_ids: properties
                                      .filter(
                                        (other) =>
                                          other.id !== x.id &&
                                          other.type === "text",
                                      )
                                      .slice(0, 1)
                                      .map((other) => other.id),
                                  }
                                : {}),
                              ...(e.target.value === "button"
                                ? {
                                    action: "create_document",
                                    template: `# {{${properties.find((other) => other.type === "text")?.name || x.name}}}\n\n`,
                                  }
                                : {}),
                            }
                          : x,
                      ),
                    )
                  }
                >
                  {Object.entries(propertyNames).map(([v, l]) => (
                    <option key={v} value={v}>
                      {l}
                    </option>
                  ))}
                </select>
                <button
                  className="icon-button danger-text"
                  aria-label={`속성 ${p.name} 삭제`}
                  onClick={() =>
                    setProperties(properties.filter((x) => x.id !== p.id))
                  }
                >
                  <Trash2 size={17} />
                </button>
              </div>
              {["select", "multi_select", "status"].includes(p.type) && (
                <Field label="선택 옵션 (쉼표로 구분)">
                  <input
                    value={(p.options || []).join(",")}
                    onChange={(e) =>
                      setProperties(
                        properties.map((x) =>
                          x.id === p.id
                            ? { ...x, options: e.target.value.split(",") }
                            : x,
                        ),
                      )
                    }
                    placeholder="할 일,진행 중,완료"
                  />
                </Field>
              )}
              <AdvancedPropertyFields
                property={p}
                properties={properties}
                databases={databases.map((d) =>
                  d.id === selectedId ? { ...d, properties } : d,
                )}
                onChange={(next) =>
                  setProperties(
                    properties.map((x) => (x.id === next.id ? next : x)),
                  )
                }
              />
            </div>
          ))}
        </div>
        <Button onClick={() => setProperties([...properties, newProperty()])}>
          <Plus size={17} /> 속성 추가
        </Button>
        <div className="modal-actions">
          <Button
            variant="primary"
            disabled={
              !properties.length ||
              properties.some((p) => !p.name.trim()) ||
              busy
            }
            onClick={async () => {
              setBusy(true);
              try {
                await api("/databases/" + selectedId, "PUT", {
                  name: selected?.name,
                  properties: properties.map((p) => ({
                    ...p,
                    options: p.options.map((o) => o.trim()).filter(Boolean),
                  })),
                });
                await load();
                await refreshRows();
                setPropertiesOpen(false);
                notify("속성을 저장했습니다.");
              } catch (e) {
                notify((e as Error).message, "error");
              } finally {
                setBusy(false);
              }
            }}
          >
            속성 저장
          </Button>
        </div>
      </Modal>
      <RowDetailPanel
        open={rowOpen && !!selected}
        onOpenChange={setRowOpen}
        title={editingRow ? (canWrite ? "항목 편집" : "항목 보기") : "새 항목"}
      >
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            if (!canWrite) return;
            const actor = user.id,
              current = () =>
                currentRoute.current === selectedId &&
                currentWorkspace.current === workspace?.id &&
                currentUser.current === actor;
            setBusy(true);
            try {
              await api<Row>(
                `/databases/${selectedId}/rows${editingRow ? "/" + editingRow.id : ""}`,
                editingRow ? "PUT" : "POST",
                {
                  values,
                  ...(editingRow
                    ? { expected_version: editingRow.version }
                    : {}),
                },
              );
              if (!current()) return;
              await refreshRows();
              if (!current()) return;
              setRowOpen(false);
              notify("항목을 저장했습니다.");
            } catch (e) {
              if (current()) notify((e as Error).message, "error");
            } finally {
              if (current()) setBusy(false);
            }
          }}
        >
          {!canWrite && (
            <p className="notice subtle">이 데이터베이스는 읽기 전용입니다.</p>
          )}
          <fieldset
            disabled={!canWrite}
            style={{ border: 0, padding: 0, margin: 0, minWidth: 0 }}
          >
            {selected?.properties.map((p) =>
              derivedProperty(p) ? (
                <div className="field" key={p.id}>
                  <span>
                    {p.name} <small className="muted">· 자동 계산</small>
                  </span>
                  {editingRow ? (
                    <AdvancedCell
                      property={p}
                      row={editingRow}
                      onAction={canWrite ? onAction : undefined}
                    />
                  ) : (
                    <span className="muted">저장 후 계산됩니다.</span>
                  )}
                </div>
              ) : p.type === "relation" ? (
                <div className="field" key={p.id}>
                  <span>{p.name}</span>
                  <RelationInput
                    property={p}
                    value={values[p.id]}
                    onChange={(value) =>
                      setValues({ ...values, [p.id]: value })
                    }
                  />
                </div>
              ) : p.type === "date" ? (
                <NaturalDateInput
                  key={p.id}
                  label={p.name}
                  value={values[p.id] || ""}
                  onChange={(value) => setValues({ ...values, [p.id]: value })}
                />
              ) : (
                <Field label={p.name} key={p.id}>
                  {["select", "status"].includes(p.type) ? (
                    <select
                      value={values[p.id] || ""}
                      onChange={(e) =>
                        setValues({ ...values, [p.id]: e.target.value })
                      }
                    >
                      <option value="">선택하지 않음</option>
                      {p.options.map((o) => (
                        <option key={o} value={o}>
                          {o}
                        </option>
                      ))}
                    </select>
                  ) : p.type === "multi_select" ? (
                    <div className="checkbox-options">
                      {p.options.map((o) => (
                        <label key={o}>
                          <input
                            type="checkbox"
                            checked={multiValues(values[p.id]).includes(o)}
                            onChange={(e) =>
                              setValues({
                                ...values,
                                [p.id]: e.target.checked
                                  ? [...multiValues(values[p.id]), o]
                                  : multiValues(values[p.id]).filter(
                                      (v: string) => v !== o,
                                    ),
                              })
                            }
                          />
                          {o}
                        </label>
                      ))}
                    </div>
                  ) : p.type === "checkbox" ? (
                    <input
                      type="checkbox"
                      checked={!!values[p.id]}
                      onChange={(e) =>
                        setValues({ ...values, [p.id]: e.target.checked })
                      }
                    />
                  ) : (
                    <input
                      type={
                        p.type === "progress"
                          ? "number"
                          : ["number", "date", "url", "email"].includes(p.type)
                            ? p.type
                            : "text"
                      }
                      value={values[p.id] ?? ""}
                      min={p.type === "progress" ? 0 : undefined}
                      max={p.type === "progress" ? 100 : undefined}
                      step={
                        ["number", "progress"].includes(p.type)
                          ? "any"
                          : undefined
                      }
                      onChange={(e) =>
                        setValues({
                          ...values,
                          [p.id]:
                            ["number", "progress"].includes(p.type) &&
                            e.target.value !== ""
                              ? Number(e.target.value)
                              : e.target.value,
                        })
                      }
                    />
                  )}
                </Field>
              ),
            )}
          </fieldset>
          <div className="modal-actions">
            {editingRow && canWrite && (
              <Button
                type="button"
                variant="danger"
                onClick={async () => {
                  if (!window.confirm("이 항목을 삭제할까요?")) return;
                  try {
                    await api(
                      `/databases/${selectedId}/rows/${editingRow.id}`,
                      "DELETE",
                      { expected_version: editingRow.version },
                    );
                    await refreshRows();
                    setRowOpen(false);
                    notify("항목을 삭제했습니다.");
                  } catch (e) {
                    notify((e as Error).message, "error");
                  }
                }}
              >
                삭제
              </Button>
            )}
            <Button type="button" onClick={() => setRowOpen(false)}>
              취소
            </Button>
            {canWrite && (
              <Button variant="primary" disabled={busy}>
                저장
              </Button>
            )}
          </div>
        </form>
      </RowDetailPanel>
      <DatabaseAIAction
        databaseId={selectedId}
        target={selected ? aiTarget : null}
        onClose={() => setAITarget(null)}
        onSaved={refreshRows}
      />
    </>
  );
}
