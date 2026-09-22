import { useEffect, useState, useRef } from "react";
import {
  AlertCircle,
  ArrowRight,
  CalendarDays,
  Check,
  ChevronDown,
  Code2,
  FilePlus2,
  Filter,
  GalleryVerticalEnd,
  Link2,
  LoaderCircle,
  Plus,
  RefreshCw,
  Sparkles,
  X,
} from "lucide-react";
import { api, datetime, type Database, type Property, type Row } from "./api";
import { Badge, Button, CopyButton, Empty, ErrorBox, Field, Modal } from "./ui";
import { useApp } from "./context";

export type AdvancedProperty = Property & {
  expression?: string;
  target_database_id?: string;
  relation_property_id?: string;
  target_property_id?: string;
  aggregation?: string;
  prompt?: string;
  source_property_ids?: string[];
  action?: string;
  template?: string;
};
export type AdvancedDatabase = Omit<Database, "properties"> & {
  properties: AdvancedProperty[];
  can_write?: boolean;
};
export type AdvancedRow = Row & {
  computed_values?: Record<string, any>;
  errors?: Record<string, string>;
  relation_labels?: Record<string, Record<string, string>>;
  updated_at?: string;
};
export type DatabaseFilter = {
  property_id: string;
  operator: string;
  value: any;
  value_type?: "text" | "number" | "boolean";
};
export type DatabaseSort = { property_id: string; direction: "asc" | "desc" };
export const advancedPropertyNames: Record<string, string> = {
  formula: "수식",
  relation: "관계",
  rollup: "롤업",
  ai: "AI 속성",
  created_time: "생성 시간",
  updated_time: "수정 시간",
  created_by: "생성자",
  updated_by: "수정자",
  progress: "진행률",
  status: "상태",
  phone: "전화번호",
  button: "버튼",
};
export const derivedProperty = (p: Property) =>
  [
    "formula",
    "rollup",
    "created_time",
    "updated_time",
    "created_by",
    "updated_by",
    "button",
  ].includes(p.type);
export function effectiveValue(row: AdvancedRow, id: string) {
  return row.computed_values && id in row.computed_values
    ? row.computed_values[id]
    : row.values[id];
}
export function plainValue(value: any): string {
  if (value === null || value === undefined) return "";
  if (Array.isArray(value)) return value.map(plainValue).join(", ");
  if (typeof value === "object") return JSON.stringify(value);
  if (typeof value === "boolean") return value ? "예" : "아니요";
  return String(value);
}

export function AdvancedPropertyFields({
  property,
  properties,
  databases,
  onChange,
}: {
  property: AdvancedProperty;
  properties: AdvancedProperty[];
  databases: AdvancedDatabase[];
  onChange: (p: AdvancedProperty) => void;
}) {
  const p = property;
  const relation = properties.find((x) => x.id === p.relation_property_id);
  const target = databases.find((d) => d.id === relation?.target_database_id);
  const set = (key: keyof AdvancedProperty, value: any) =>
    onChange({ ...p, [key]: value });
  return (
    <div className="advanced-property-fields">
      {p.type === "formula" && (
        <>
          <Field label="수식">
            <textarea
              rows={3}
              value={p.expression || ""}
              onChange={(e) => set("expression", e.target.value)}
              placeholder={'round(prop("가격") * prop("수량"), 2)'}
            />
          </Field>
          <p className="muted">
            prop("속성 이름")으로 값을 참조하세요. if, concat, dateAdd,
            dateBetween, format, contains, length, round, sum을 사용할 수
            있습니다.
          </p>
        </>
      )}
      {p.type === "relation" && (
        <Field label="연결할 데이터베이스">
          <select
            value={p.target_database_id || ""}
            onChange={(e) => set("target_database_id", e.target.value)}
            required
          >
            <option value="" disabled>
              같은 워크스페이스에서 선택하세요
            </option>
            {databases.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </Field>
      )}
      {p.type === "rollup" && (
        <>
          <Field label="관계 속성">
            <select
              value={p.relation_property_id || ""}
              onChange={(e) =>
                onChange({
                  ...p,
                  relation_property_id: e.target.value,
                  target_property_id: "",
                })
              }
              required
            >
              <option value="" disabled>
                관계 속성을 선택하세요
              </option>
              {properties
                .filter((x) => x.type === "relation")
                .map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
            </select>
          </Field>
          <div className="form-grid">
            <Field label="집계 방식">
              <select
                value={p.aggregation || "count"}
                onChange={(e) => set("aggregation", e.target.value)}
              >
                {[
                  ["count", "개수"],
                  ["sum", "합계"],
                  ["avg", "평균"],
                  ["min", "최솟값"],
                  ["max", "최댓값"],
                  ["unique", "고유한 값"],
                ].map(([v, l]) => (
                  <option key={v} value={v}>
                    {l}
                  </option>
                ))}
              </select>
            </Field>
            <Field label="집계할 속성">
              <select
                disabled={(p.aggregation || "count") === "count"}
                value={p.target_property_id || ""}
                onChange={(e) => set("target_property_id", e.target.value)}
                required={(p.aggregation || "count") !== "count"}
              >
                <option value="">속성을 선택하세요</option>
                {target?.properties.map((x) => (
                  <option key={x.id} value={x.id}>
                    {x.name}
                  </option>
                ))}
              </select>
            </Field>
          </div>
        </>
      )}
      {p.type === "ai" && (
        <>
          <Field label="AI 생성 지시문">
            <textarea
              value={p.prompt || ""}
              onChange={(e) => set("prompt", e.target.value)}
              placeholder="다음 내용을 한 문장으로 요약하세요."
              rows={3}
            />
          </Field>
          <Field label="AI에 전달할 원본 속성">
            <div className="checkbox-options">
              {properties
                .filter(
                  (x) => x.id !== p.id && !["button", "ai"].includes(x.type),
                )
                .map((x) => (
                  <label key={x.id}>
                    <input
                      type="checkbox"
                      checked={(p.source_property_ids || []).includes(x.id)}
                      onChange={(e) =>
                        set(
                          "source_property_ids",
                          e.target.checked
                            ? [...(p.source_property_ids || []), x.id]
                            : (p.source_property_ids || []).filter(
                                (id) => id !== x.id,
                              ),
                        )
                      }
                    />
                    {x.name}
                  </label>
                ))}
            </div>
          </Field>
          <p className="muted">
            관리자가 설정한 AI로 사용자가 생성 버튼을 누를 때만 실행합니다. 변경
            중인 셀의 결과를 자동으로 덮어쓰지 않습니다.
          </p>
        </>
      )}
      {p.type === "button" && (
        <>
          <Field label="버튼 작업">
            <select
              value={p.action || "create_document"}
              onChange={(e) => set("action", e.target.value)}
            >
              <option value="create_document">이 항목으로 문서 만들기</option>
            </select>
          </Field>
          <Field
            label="문서 템플릿"
            hint={"{{속성 이름}} 또는 {{속성 ID}}에 현재 항목 값을 넣습니다."}
          >
            <textarea
              rows={4}
              value={p.template || ""}
              onChange={(e) => set("template", e.target.value)}
              placeholder={"# {{이름}}\n\n항목 정보를 문서로 정리하세요."}
            />
          </Field>
        </>
      )}
    </div>
  );
}

export function RelationInput({
  property,
  value,
  onChange,
}: {
  property: AdvancedProperty;
  value: any;
  onChange: (value: string[]) => void;
}) {
  const [rows, setRows] = useState<AdvancedRow[]>([]),
    [error, setError] = useState(""),
    [search, setSearch] = useState("");
  const selected = Array.isArray(value) ? value : [];
  useEffect(() => {
    let active = true;
    if (property.target_database_id)
      api<AdvancedRow[]>(`/databases/${property.target_database_id}/rows`)
        .then((v) => {
          if (active) setRows(v);
        })
        .catch((e) => {
          if (active) setError(e.message);
        });
    return () => {
      active = false;
    };
  }, [property.target_database_id]);
  const name = (r: AdvancedRow) =>
    Object.values(r.values).find((v) => typeof v === "string" && v) || r.id;
  return (
    <div className="relation-picker">
      <input
        aria-label={`${property.name} 연결 항목 검색`}
        placeholder="연결할 항목 검색…"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
      />
      <ErrorBox error={error} />
      <div className="relation-options">
        {rows
          .filter((r) =>
            String(name(r)).toLowerCase().includes(search.toLowerCase()),
          )
          .slice(0, 100)
          .map((r) => (
            <label key={r.id}>
              <input
                type="checkbox"
                checked={selected.includes(r.id)}
                disabled={!selected.includes(r.id) && selected.length >= 100}
                onChange={(e) =>
                  onChange(
                    e.target.checked
                      ? [...selected, r.id]
                      : selected.filter((id) => id !== r.id),
                  )
                }
              />
              <span>{String(name(r))}</span>
            </label>
          ))}
        {!rows.length && !error && (
          <p className="muted">연결할 항목이 없습니다.</p>
        )}
      </div>
      <small>{selected.length}개 연결 · 최대 100개</small>
    </div>
  );
}

export function AdvancedCell({
  property,
  row,
  onAction,
}: {
  property: AdvancedProperty;
  row: AdvancedRow;
  onAction?: (p: AdvancedProperty, row: AdvancedRow) => void;
}) {
  const value = effectiveValue(row, property.id),
    error = row.errors?.[property.id];
  if (error)
    return (
      <span className="computed-error" title={error}>
        <AlertCircle size={15} /> 오류 <small>{error}</small>
      </span>
    );
  if (property.type === "relation") {
    const ids = Array.isArray(value) ? value : [];
    return (
      <div className="tags">
        {ids.length ? (
          ids.map((id) => (
            <Badge key={id}>
              <Link2 size={12} />
              {row.relation_labels?.[property.id]?.[id] || id.slice(0, 8)}
            </Badge>
          ))
        ) : (
          <span className="muted">—</span>
        )}
      </div>
    );
  }
  if (property.type === "progress")
    return (
      <div className="progress-cell">
        <progress max={100} value={Number(value) || 0} />
        <span>{Number(value) || 0}%</span>
      </div>
    );
  if (property.type === "button")
    return (
      <Button
        type="button"
        disabled={!onAction}
        onClick={(e) => {
          e.stopPropagation();
          onAction?.(property, row);
        }}
      >
        <FilePlus2 size={14} />
        {property.name}
      </Button>
    );
  if (property.type === "ai")
    return (
      <div className="ai-property-cell">
        <span>{plainValue(value) || "아직 생성하지 않음"}</span>
        <button
          type="button"
          className="icon-button"
          aria-label={`${property.name} AI 생성`}
          disabled={!onAction}
          onClick={(e) => {
            e.stopPropagation();
            onAction?.(property, row);
          }}
        >
          <Sparkles size={15} />
        </button>
      </div>
    );
  if (["created_time", "updated_time"].includes(property.type))
    return <span>{value ? datetime(value) : "—"}</span>;
  return <span>{plainValue(value) || "—"}</span>;
}

const operators = [
  ["contains", "포함"],
  ["eq", "같음"],
  ["neq", "다름"],
  ["not_contains", "포함하지 않음"],
  ["gt", "보다 큼"],
  ["gte", "이상"],
  ["lt", "보다 작음"],
  ["lte", "이하"],
  ["is_empty", "비어 있음"],
  ["not_empty", "비어 있지 않음"],
];
export function DatabaseQueryControls({
  properties,
  filters: appliedFilters,
  sorts: appliedSorts,
  onChange: onApply,
}: {
  properties: AdvancedProperty[];
  filters: DatabaseFilter[];
  sorts: DatabaseSort[];
  onChange: (filters: DatabaseFilter[], sorts: DatabaseSort[]) => void;
}) {
  const [open, setOpen] = useState(false);
  const [filters, setFilters] = useState(appliedFilters),
    [sorts, setSorts] = useState(appliedSorts);
  const onChange = (f: DatabaseFilter[], s: DatabaseSort[]) => {
    setFilters(f);
    setSorts(s);
  };
  return (
    <>
      <Button
        onClick={() => {
          setFilters(appliedFilters);
          setSorts(appliedSorts);
          setOpen(true);
        }}
      >
        <Filter size={16} />
        필터 / 정렬
        {appliedFilters.length + appliedSorts.length > 0 && (
          <Badge>{appliedFilters.length + appliedSorts.length}</Badge>
        )}
      </Button>
      <Modal
        open={open}
        onOpenChange={setOpen}
        title="열별 필터와 정렬"
        description="모든 필터 조건을 만족하는 항목을 표시합니다. 수식과 롤업 결과에도 적용됩니다."
        wide
      >
        <h3>필터</h3>
        {filters.map((f, i) => {
          const prop = properties.find((p) => p.id === f.property_id);
          return (
            <div className="advanced-query-row" key={i}>
              <select
                aria-label={`필터 ${i + 1} 속성`}
                value={f.property_id}
                onChange={(e) =>
                  onChange(
                    filters.map((v, j) =>
                      j === i
                        ? {
                            ...v,
                            property_id: e.target.value,
                            value: "",
                            value_type: "text",
                          }
                        : v,
                    ),
                    sorts,
                  )
                }
              >
                <option value="">속성 선택</option>
                {properties
                  .filter((p) => p.type !== "button")
                  .map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
              </select>
              <select
                aria-label={`필터 ${i + 1} 조건`}
                value={f.operator}
                onChange={(e) =>
                  onChange(
                    filters.map((v, j) =>
                      j === i ? { ...v, operator: e.target.value } : v,
                    ),
                    sorts,
                  )
                }
              >
                {operators.map(([v, l]) => (
                  <option key={v} value={v}>
                    {l}
                  </option>
                ))}
              </select>
              {!["is_empty", "not_empty"].includes(f.operator) &&
                ["formula", "rollup"].includes(prop?.type || "") && (
                  <select
                    aria-label={`필터 ${i + 1} 값 유형`}
                    value={f.value_type || "text"}
                    onChange={(e) =>
                      onChange(
                        filters.map((v, j) =>
                          j === i
                            ? {
                                ...v,
                                value_type: e.target
                                  .value as DatabaseFilter["value_type"],
                                value:
                                  e.target.value === "boolean"
                                    ? false
                                    : e.target.value === "number"
                                      ? 0
                                      : "",
                              }
                            : v,
                        ),
                        sorts,
                      )
                    }
                  >
                    <option value="text">텍스트</option>
                    <option value="number">숫자</option>
                    <option value="boolean">참/거짓</option>
                  </select>
                )}
              {!["is_empty", "not_empty"].includes(f.operator) &&
              (prop?.type === "checkbox" || f.value_type === "boolean") ? (
                <select
                  aria-label={`필터 ${i + 1} 값`}
                  value={f.value === true ? "true" : "false"}
                  onChange={(e) =>
                    onChange(
                      filters.map((v, j) =>
                        j === i
                          ? { ...v, value: e.target.value === "true" }
                          : v,
                      ),
                      sorts,
                    )
                  }
                >
                  <option value="true">예 (참)</option>
                  <option value="false">아니요 (거짓)</option>
                </select>
              ) : (
                !["is_empty", "not_empty"].includes(f.operator) && (
                  <input
                    aria-label={`필터 ${i + 1} 값`}
                    type={
                      f.value_type === "number" ||
                      ["number", "progress"].includes(prop?.type || "")
                        ? "number"
                        : prop?.type === "date"
                          ? "date"
                          : "text"
                    }
                    value={f.value ?? ""}
                    step="any"
                    onChange={(e) =>
                      onChange(
                        filters.map((v, j) =>
                          j === i
                            ? {
                                ...v,
                                value:
                                  e.target.type === "number" &&
                                  e.target.value !== ""
                                    ? Number(e.target.value)
                                    : e.target.value,
                              }
                            : v,
                        ),
                        sorts,
                      )
                    }
                  />
                )
              )}
              <button
                className="icon-button"
                aria-label={`필터 ${i + 1} 삭제`}
                onClick={() =>
                  onChange(
                    filters.filter((_, j) => j !== i),
                    sorts,
                  )
                }
              >
                <X size={16} />
              </button>
            </div>
          );
        })}
        <Button
          onClick={() =>
            onChange(
              [
                ...filters,
                {
                  property_id: properties[0]?.id || "",
                  operator: "contains",
                  value: "",
                },
              ],
              sorts,
            )
          }
          disabled={filters.length >= 30}
        >
          <Plus size={15} />
          필터 추가
        </Button>
        <h3 style={{ marginTop: 28 }}>정렬</h3>
        {sorts.map((s, i) => (
          <div className="advanced-query-row" key={i}>
            <select
              aria-label={`정렬 ${i + 1} 속성`}
              value={s.property_id}
              onChange={(e) =>
                onChange(
                  filters,
                  sorts.map((v, j) =>
                    j === i ? { ...v, property_id: e.target.value } : v,
                  ),
                )
              }
            >
              {properties.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
            <select
              aria-label={`정렬 ${i + 1} 방향`}
              value={s.direction}
              onChange={(e) =>
                onChange(
                  filters,
                  sorts.map((v, j) =>
                    j === i
                      ? { ...v, direction: e.target.value as "asc" | "desc" }
                      : v,
                  ),
                )
              }
            >
              <option value="asc">오름차순</option>
              <option value="desc">내림차순</option>
            </select>
            <button
              className="icon-button"
              aria-label={`정렬 ${i + 1} 삭제`}
              onClick={() =>
                onChange(
                  filters,
                  sorts.filter((_, j) => j !== i),
                )
              }
            >
              <X size={16} />
            </button>
          </div>
        ))}
        <Button
          onClick={() =>
            onChange(filters, [
              ...sorts,
              { property_id: properties[0]?.id || "", direction: "asc" },
            ])
          }
          disabled={sorts.length >= 10}
        >
          <Plus size={15} />
          정렬 추가
        </Button>
        <div className="modal-actions">
          <Button onClick={() => onChange([], [])}>모두 초기화</Button>
          <Button
            variant="primary"
            disabled={
              filters.some((f) => !f.property_id) ||
              sorts.some((s) => !s.property_id)
            }
            onClick={() => {
              onApply(filters, sorts);
              setOpen(false);
            }}
          >
            적용
          </Button>
        </div>
      </Modal>
    </>
  );
}

export function AdvancedViews({
  view,
  database,
  rows,
  onEdit,
  onAction,
}: {
  view: string;
  database: AdvancedDatabase;
  rows: AdvancedRow[];
  onEdit: (r: AdvancedRow) => void;
  onAction?: (p: AdvancedProperty, r: AdvancedRow) => void;
}) {
  const title =
    database.properties.find((p) => p.type === "text") ||
    database.properties[0];
  const dates = database.properties.filter((p) => p.type === "date");
  const [startProperty, setStartProperty] = useState(dates[0]?.id || ""),
    [endProperty, setEndProperty] = useState(dates[1]?.id || "");
  useEffect(() => {
    if (!dates.some((p) => p.id === startProperty))
      setStartProperty(dates[0]?.id || "");
    if (endProperty && !dates.some((p) => p.id === endProperty))
      setEndProperty("");
  }, [database.properties]);
  const display = (r: AdvancedRow) =>
    plainValue(effectiveValue(r, title?.id)) || "제목 없는 항목";
  if (!rows.length)
    return (
      <Empty
        title="표시할 항목이 없습니다"
        text="새 항목을 추가하거나 필터를 변경해 보세요."
      />
    );
  if (view === "timeline") {
    if (!dates.length)
      return (
        <Empty
          title="타임라인에 사용할 날짜 속성이 필요해요"
          text="시작 날짜와 종료 날짜 속성을 추가해 일정 기간을 표시하세요."
        />
      );
    const now = Date.now(),
      starts = rows
        .map((r) => Date.parse(r.values[startProperty]))
        .filter(Number.isFinite),
      ends = rows
        .map((r) => Date.parse(r.values[endProperty]))
        .filter(Number.isFinite);
    const min = starts.length ? Math.min(...starts) : now,
      max = Math.max(min + 86400000, ...starts, ...ends),
      span = Math.max(86400000, max - min);
    return (
      <div className="panel advanced-timeline">
        <div className="timeline-controls">
          <Field label="시작 날짜">
            <select
              value={startProperty}
              onChange={(e) => setStartProperty(e.target.value)}
            >
              {dates.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </Field>
          <Field label="종료 날짜">
            <select
              value={endProperty}
              onChange={(e) => setEndProperty(e.target.value)}
            >
              <option value="">시작 날짜만 사용</option>
              {dates.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </select>
          </Field>
        </div>
        <div className="timeline-axis">
          <span>{new Date(min).toLocaleDateString("ko-KR")}</span>
          <span>{new Date(max).toLocaleDateString("ko-KR")}</span>
        </div>
        {rows.map((r) => {
          const start = Date.parse(r.values[startProperty]),
            end = Date.parse(r.values[endProperty]);
          return (
            <div className="timeline-entry" key={r.id}>
              <button onClick={() => onEdit(r)}>{display(r)}</button>
              <div className="timeline-track">
                {Number.isFinite(start) ? (
                  <button
                    title={`${r.values[startProperty]} ~ ${r.values[endProperty] || r.values[startProperty]}`}
                    className="timeline-bar"
                    style={{
                      left: `${Math.max(0, ((start - min) / span) * 100)}%`,
                      width: `${Math.max(2, (((Number.isFinite(end) ? Math.max(end, start) : start) - start) / span) * 100)}%`,
                    }}
                    onClick={() => onEdit(r)}
                  >
                    {display(r)}
                  </button>
                ) : (
                  <span className="muted">날짜 미지정</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  }
  return (
    <div className={view === "gallery" ? "advanced-gallery" : "advanced-list"}>
      {rows.map((r, i) => (
        <article className="panel advanced-view-card" key={r.id}>
          {view === "gallery" && (
            <div className={`gallery-cover tone-${i % 4}`}>
              <GalleryVerticalEnd size={33} />
            </div>
          )}
          <div className="advanced-card-content">
            <button className="advanced-card-title" onClick={() => onEdit(r)}>
              {display(r)}
            </button>
            {database.properties
              .filter((p) => p.id !== title?.id)
              .slice(0, view === "gallery" ? 5 : 8)
              .map((p) => (
                <div className="advanced-card-field" key={p.id}>
                  <small>{p.name}</small>
                  <AdvancedCell property={p} row={r} onAction={onAction} />
                </div>
              ))}
          </div>
        </article>
      ))}
    </div>
  );
}

export function DatabaseAIAction({
  databaseId,
  target,
  onClose,
  onSaved,
}: {
  databaseId: string;
  target: { property: AdvancedProperty; row: AdvancedRow } | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { notify } = useApp();
  const [output, setOutput] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [stored, setStored] = useState(false);
  const controller = useRef<AbortController | null>(null);
  useEffect(() => {
    setOutput("");
    setError("");
    setStored(false);
    return () => {
      controller.current?.abort();
    };
  }, [target?.property.id, target?.row.id]);
  const generate = async () => {
    if (!target) return;
    setBusy(true);
    setOutput("");
    setError("");
    setStored(false);
    const requestController = new AbortController();
    controller.current = requestController;
    try {
      const res = await fetch(
        `/api/v1/databases/${databaseId}/rows/${target.row.id}/ai/${target.property.id}`,
        {
          method: "POST",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            "X-Madi-Request": "1",
          },
          body: "{}",
          signal: requestController.signal,
        },
      );
      if (!res.ok) {
        const value = await res.json();
        throw new Error(value.error);
      }
      if (!res.body) throw new Error("스트리밍 응답이 없습니다");
      const reader = res.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "",
        didStore = false;
      while (true) {
        const chunk = await reader.read();
        if (chunk.done) break;
        buffer += decoder.decode(chunk.value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (!line.startsWith("data:")) continue;
          const raw = line.slice(5).trim();
          if (raw === "[DONE]" || !raw) continue;
          const value = JSON.parse(raw);
          if (value.error) throw new Error(value.error);
          if (value.text) setOutput((v) => v + value.text);
          if (value.stored) {
            didStore = true;
            setStored(true);
          }
        }
      }
      if (!didStore)
        throw new Error(
          "생성한 결과를 셀에 저장하지 못했습니다. 내용을 복사한 후 다시 시도하세요.",
        );
      onSaved();
      notify("AI 속성을 생성하고 저장했습니다.");
    } catch (e) {
      setError(
        requestController.signal.aborted
          ? "생성을 중단했습니다. 완료되지 않은 결과는 셀에 저장하지 않습니다."
          : (e as Error).message,
      );
    } finally {
      requestController.abort();
      setBusy(false);
    }
  };
  return (
    <Modal
      open={!!target}
      onOpenChange={(v) => {
        if (!v && !busy) onClose();
      }}
      title={`${target?.property.name || "AI 속성"} 생성`}
      description="선택한 원본 속성을 관리자 AI 서비스로 전달합니다. 결과는 생성 완료 후 현재 항목에 저장됩니다."
      wide
    >
      <div className="notice subtle">
        <Sparkles size={19} />
        <span>{target?.property.prompt}</span>
      </div>
      <ErrorBox error={error} />
      {output && <div className="ai-property-output">{output}</div>}
      <div className="modal-actions">
        {output && <CopyButton value={output} />}
        {busy && (
          <Button variant="danger" onClick={() => controller.current?.abort()}>
            생성 중단
          </Button>
        )}
        <Button disabled={busy} onClick={onClose}>
          닫기
        </Button>
        <Button variant="primary" disabled={busy} onClick={generate}>
          {busy ? (
            <>
              <LoaderCircle size={17} className="spin" />
              생성 중…
            </>
          ) : stored ? (
            <>
              <RefreshCw size={17} />
              다시 생성
            </>
          ) : (
            <>
              <Sparkles size={17} />
              AI 생성
            </>
          )}
        </Button>
      </div>
    </Modal>
  );
}

export function DatabaseAdvancedStyles() {
  return (
    <>
      <DatabaseAdvancedBaseStyles />
      <style>{`.database-toolbar{flex-wrap:wrap}.database-toolbar .tabs{max-width:100%;overflow-x:auto;flex-wrap:nowrap}.database-tools{flex-wrap:wrap}.advanced-query-row{flex-wrap:wrap}.advanced-query-row input,.advanced-query-row select,.relation-options label,.advanced-card-field,.timeline-entry>button,.ai-property-cell>span{font-size:max(.9rem,15px)}.advanced-property-fields p,.advanced-card-field>small,.advanced-query-result,.advanced-query-pages>span,.timeline-axis,.progress-cell span,.computed-error{font-size:max(.8rem,13px)}.timeline-track{overflow:hidden}.timeline-bar{font-size:13px;height:30px;top:3px}.advanced-card-field .tags{max-width:75%}.advanced-query-row>select{min-width:120px;flex:1}.advanced-query-row>input{min-width:120px;flex:1}.advanced-query-row .icon-button{flex-shrink:0}.relation-options{scrollbar-width:thin;scrollbar-color:#b4cbbf transparent}`}</style>
    </>
  );
}

function DatabaseAdvancedBaseStyles() {
  return (
    <style>{`.advanced-property-fields{margin-top:15px}.advanced-property-fields:empty{display:none}.advanced-property-fields p{font-size:.79rem;color:var(--muted)}.advanced-query-row{display:flex;gap:9px;margin-bottom:12px;align-items:center}.advanced-query-row input,.advanced-query-row select{font-size:.83rem}.relation-picker{border:1px solid var(--border);padding:12px;border-radius:8px}.relation-options{max-height:220px;overflow:auto;display:flex;flex-direction:column;gap:9px;margin:12px 0}.relation-options label{display:flex;align-items:center;gap:9px;font-size:.86rem}.relation-picker>small{color:var(--muted)}.computed-error{display:flex;gap:5px;align-items:center;color:#b14f3e;font-size:.8rem;position:relative}.computed-error small{display:none}.computed-error:hover small{display:block;position:absolute;bottom:100%;left:0;background:var(--surface);box-shadow:var(--shadow);border:1px solid var(--border);padding:10px;border-radius:6px;z-index:10;min-width:200px;white-space:normal}.progress-cell{display:flex;gap:9px;align-items:center;min-width:110px}.progress-cell progress{width:85px;height:8px;accent-color:var(--primary)}.progress-cell span{font-size:.75rem;white-space:nowrap}.ai-property-cell{display:flex;align-items:center;gap:8px}.ai-property-cell>span{display:-webkit-box;-webkit-line-clamp:3;-webkit-box-orient:vertical;overflow:hidden;max-width:290px;font-size:.83rem;white-space:pre-wrap}.ai-property-cell>.icon-button{color:var(--primary)}.advanced-gallery{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:20px}.advanced-view-card{overflow:hidden}.gallery-cover{height:125px;background:var(--soft);display:flex;align-items:center;justify-content:center;color:var(--primary)}.gallery-cover.tone-1{background:#eee9f6;color:#9584af}.gallery-cover.tone-2{background:#f9edda;color:#b49a6f}.gallery-cover.tone-3{background:#e5eef3;color:#83a0b0}.advanced-card-content{padding:22px}.advanced-card-title{border:0;background:none;padding:0;font-size:1.02rem;font-weight:650;text-align:left;margin-bottom:20px;cursor:pointer;color:var(--text)}.advanced-card-field{display:flex;align-items:center;justify-content:space-between;gap:14px;margin-top:12px;font-size:.85rem}.advanced-card-field>small{color:var(--muted);flex-shrink:0;font-size:.75rem}.advanced-card-field>span,.advanced-card-field>div{min-width:0;overflow-wrap:anywhere}.advanced-list{display:flex;flex-direction:column;gap:13px}.advanced-list .advanced-card-content{display:grid;grid-template-columns:1.1fr repeat(3,1fr);gap:15px;align-items:center}.advanced-list .advanced-card-title{margin:0}.advanced-list .advanced-card-field{display:block;margin:0}.advanced-list .advanced-card-field>small{display:block;margin-bottom:6px}.advanced-timeline{padding:24px;overflow:auto}.timeline-controls{display:flex;gap:20px}.timeline-controls .field{flex:1;max-width:270px}.timeline-axis{display:flex;justify-content:space-between;margin:5px 0 22px 210px;font-size:.75rem;color:var(--muted)}.timeline-entry{display:grid;grid-template-columns:190px minmax(350px,1fr);gap:20px;min-height:58px;align-items:center;border-top:1px solid var(--border)}.timeline-entry>button{background:none;border:0;text-align:left;font-size:.85rem;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.timeline-track{position:relative;height:36px;border-left:1px solid var(--border);border-right:1px solid var(--border);background:repeating-linear-gradient(90deg,transparent 0,transparent 19.8%,var(--border) 20%)}.timeline-bar{position:absolute;top:5px;height:26px;background:var(--primary);color:#fff;border:0;border-radius:6px;min-width:8px;max-width:100%;font-size:.69rem;text-align:left;padding:4px 8px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.timeline-track>.muted{font-size:.72rem;padding:6px;display:block}.ai-property-output{white-space:pre-wrap;font-size:.93rem;line-height:1.9;padding:22px;border:1px solid var(--border);border-radius:9px;max-height:430px;overflow:auto;background:var(--bg)}.advanced-query-result{font-size:.76rem;color:var(--muted);display:flex;gap:9px;align-items:center;margin-bottom:14px}.advanced-query-pages{display:flex;justify-content:flex-end;gap:9px;align-items:center;margin-top:18px}.advanced-query-pages>span{font-size:.8rem;color:var(--muted)}@media(max-width:1100px){.advanced-gallery{grid-template-columns:repeat(2,minmax(0,1fr))}.advanced-list .advanced-card-content{grid-template-columns:1fr 1fr}}@media(max-width:640px){.advanced-gallery{grid-template-columns:1fr}.advanced-query-row{flex-wrap:wrap}.advanced-query-row>select{flex:1;min-width:125px}.advanced-query-row>input{flex:1}.advanced-list .advanced-card-content{grid-template-columns:1fr}.timeline-entry{grid-template-columns:130px minmax(280px,1fr)}.timeline-axis{margin-left:150px}.timeline-controls{gap:10px}.advanced-timeline{padding:18px}}`}</style>
  );
}
