import { lazy, Suspense, useEffect, useRef, useState } from "react";
import { api, ApiError, type Row } from "../api";
import { useApp } from "../context";
import { Button, Loading } from "../ui";
import {
  AdvancedCell,
  derivedProperty,
  effectiveValue,
  plainValue,
  type AdvancedDatabase,
  type AdvancedProperty,
  type AdvancedRow,
} from "../DatabaseAdvanced";
import { parseCellText, parseTabularPaste } from "./cells";
import type { RangeTarget } from "./RangePasteReview";
import "./editing.css";
const RangePasteReview = lazy(() => import("./RangePasteReview"));
type Editing = {
  row: AdvancedRow;
  property: AdvancedProperty;
  rowIndex: number;
  columnIndex: number;
  text: string;
};
export default function EditableGrid({
  database,
  rows,
  columns,
  canWrite,
  onDetail,
  onCreate,
  onAction,
  onChanged,
}: {
  database: AdvancedDatabase;
  rows: AdvancedRow[];
  columns: string[];
  canWrite: boolean;
  onDetail: (row: AdvancedRow) => void;
  onCreate: () => void;
  onAction: (p: AdvancedProperty, row: AdvancedRow) => void;
  onChanged: () => Promise<void>;
}) {
  const { user, workspace } = useApp(),
    scope = `${user.id}:${workspace?.id}:${database.id}`,
    origin = useRef(scope),
    scopeRef = useRef(scope),
    mounted = useRef(true);
  scopeRef.current = scope;
  const [editing, setEditing] = useState<Editing | null>(null),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [metadata, setMetadata] = useState<{
      schema_fingerprint: string;
      properties: AdvancedProperty[];
    } | null>(null),
    [range, setRange] = useState<RangeTarget[] | null>(null),
    [pasteText, setPasteText] = useState("");
  const root = useRef<HTMLDivElement>(null),
    input = useRef<HTMLInputElement | HTMLSelectElement | null>(null),
    busyRef = useRef(false);
  const properties = database.properties.filter(
    (p) => !columns.length || columns.includes(p.id),
  );
  const focus = useRef({ row: 0, column: 0 });
  const alive = () => mounted.current && scopeRef.current === origin.current;
  useEffect(() => {
    mounted.current = true;
    const abort = new AbortController();
    api<{ schema_fingerprint: string; properties: AdvancedProperty[] }>(
      `/databases/${database.id}/editing`,
      "GET",
      undefined,
      { signal: abort.signal },
    )
      .then((v) => {
        if (alive() && !abort.signal.aborted) setMetadata(v);
      })
      .catch((e) => {
        if (alive() && !abort.signal.aborted) setError(e.message);
      });
    return () => {
      mounted.current = false;
      abort.abort();
    };
  }, [JSON.stringify(database.properties)]);
  useEffect(() => {
    if (editing) {
      input.current?.focus();
      if (input.current instanceof HTMLInputElement) input.current.select();
    }
  }, [editing?.row.id, editing?.property.id]);
  const setFocus = (r: number, c: number) => {
    const row = Math.max(0, Math.min(rows.length - 1, r)),
      column = Math.max(0, Math.min(properties.length - 1, c));
    focus.current = { row, column };
    requestAnimationFrame(() => {
      const element = root.current?.querySelector<HTMLElement>(
        `[data-cell="${row}:${column}"]`,
      );
      (element || root.current)?.focus();
    });
  };
  const open = (
    row: AdvancedRow,
    p: AdvancedProperty,
    r: number,
    c: number,
  ) => {
    if (
      !canWrite ||
      derivedProperty(p) ||
      ["ai", "relation"].includes(p.type)
    ) {
      onDetail(row);
      return;
    }
    if (editing) {
      setError("편집 중인 셀을 저장하거나 취소한 뒤 다른 셀로 이동하세요.");
      return;
    }
    setError("");
    setEditing({
      row,
      property: p,
      rowIndex: r,
      columnIndex: c,
      text: plainValue(row.values[p.id]),
    });
  };
  const save = async (next?: { row: number; column: number }) => {
    if (!editing || busyRef.current) return;
    const draft = editing,
      parsed = parseCellText(draft.property, draft.text);
    if (parsed.error) {
      setError(parsed.error);
      return;
    }
    if (!draft.row.version) {
      setError("행 기준 버전이 없습니다. 최신 표를 다시 읽으세요.");
      return;
    }
    busyRef.current = true;
    setBusy(true);
    setError("");
    try {
      await api<Row>(`/databases/${database.id}/rows/${draft.row.id}`, "PUT", {
        expected_version: draft.row.version,
        values: { [draft.property.id]: parsed.value },
      });
      if (!alive()) return;
      setEditing(null);
      await onChanged();
      if (alive())
        setFocus(
          next?.row ?? draft.rowIndex,
          next?.column ?? draft.columnIndex,
        );
    } catch (e) {
      if (alive())
        setError(
          e instanceof ApiError && e.status === 409
            ? "다른 편집자가 변경했습니다. 입력은 보관했습니다. 취소 후 현재 값과 다시 비교하세요."
            : (e as Error).message,
        );
    } finally {
      busyRef.current = false;
      if (alive()) setBusy(false);
    }
  };
  const prepare = (text: string) => {
    if (!canWrite || editing || busy) {
      setError("현재 셀 편집을 마친 뒤 범위를 붙여넣으세요.");
      return;
    }
    if (!metadata) {
      setError("속성 기준을 불러온 뒤 다시 시도하세요.");
      return;
    }
    try {
      const matrix = parseTabularPaste(text),
        { row, column } = focus.current;
      if (
        row + matrix.length > rows.length ||
        column + matrix[0].length > properties.length
      )
        throw Error(
          "현재 행이나 표시 열 범위를 넘습니다. 필요한 행을 먼저 추가하거나 시작 셀을 바꾸세요.",
        );
      const target = matrix.flatMap((line, r) =>
        line.map((value, c) => ({
          row: rows[row + r],
          property: properties[column + c],
          text: value,
        })),
      );
      setError("");
      setRange(target);
      setPasteText(text);
    } catch (e) {
      setError((e as Error).message);
    }
  };
  if (scope !== origin.current) return null;
  return (
    <div className="database-edit-grid" ref={root} tabIndex={-1}>
      <div className="database-grid-help">
        <span>
          방향키 이동 · Enter/F2 편집 · Enter 저장 · Esc 취소 · Tab 저장 후 다음
          셀
        </span>
        <Button
          disabled={!canWrite || !rows.length || !!editing || !metadata}
          onClick={async () => {
            try {
              prepare(await navigator.clipboard.readText());
            } catch {
              setError(
                "클립보드 권한이 없습니다. 시작 셀을 선택하고 Ctrl+V로 붙여넣으세요.",
              );
            }
          }}
        >
          클립보드 범위 붙여넣기
        </Button>
      </div>
      {error && (
        <p className="notice" role="alert">
          {error}
        </p>
      )}
      <div className="panel table-scroll">
        <table
          className="data-table editable-table"
          role="grid"
          aria-label="키보드로 편집하는 데이터베이스 표"
          aria-rowcount={rows.length + 1}
          aria-colcount={properties.length + 1}
        >
          <thead>
            <tr>
              {properties.map((p) => (
                <th key={p.id}>{p.name}</th>
              ))}
              <th>항목 상세</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, r) => (
              <tr key={row.id}>
                {properties.map((p, c) => {
                  const active =
                    editing?.row.id === row.id && editing.property.id === p.id;
                  return (
                    <td
                      key={p.id}
                      role="gridcell"
                      tabIndex={0}
                      data-cell={`${r}:${c}`}
                      aria-label={`${r + 1}행 ${p.name}`}
                      onFocus={() => {
                        focus.current = { row: r, column: c };
                      }}
                      onPaste={(event) => {
                        if (active) return;
                        if (event.clipboardData.types.includes("text/plain")) {
                          event.preventDefault();
                          prepare(event.clipboardData.getData("text/plain"));
                        }
                      }}
                      onKeyDown={(event) => {
                        if (active || event.target !== event.currentTarget)
                          return;
                        const next = {
                          ArrowLeft: [r, c - 1],
                          ArrowRight: [r, c + 1],
                          ArrowUp: [r - 1, c],
                          ArrowDown: [r + 1, c],
                        }[event.key];
                        if (next) {
                          event.preventDefault();
                          setFocus(next[0], next[1]);
                        } else if (["Enter", "F2"].includes(event.key)) {
                          event.preventDefault();
                          open(row, p, r, c);
                        }
                      }}
                    >
                      {active ? (
                        <form
                          className="inline-cell-form"
                          onSubmit={(e) => {
                            e.preventDefault();
                            void save();
                          }}
                          onKeyDown={(e) => {
                            if (e.nativeEvent.isComposing || e.keyCode === 229)
                              return;
                            if (e.key === "Escape") {
                              e.preventDefault();
                              if (!busy) {
                                setEditing(null);
                                setError("");
                                setFocus(r, c);
                              }
                            } else if (e.key === "Tab") {
                              e.preventDefault();
                              const index =
                                  r * properties.length +
                                  c +
                                  (e.shiftKey ? -1 : 1),
                                total = rows.length * properties.length,
                                clamped = Math.min(
                                  total - 1,
                                  Math.max(0, index),
                                );
                              void save({
                                row: Math.floor(clamped / properties.length),
                                column: clamped % properties.length,
                              });
                            }
                          }}
                        >
                          {["select", "status", "checkbox"].includes(p.type) ? (
                            <select
                              aria-label={`${p.name} 셀 입력`}
                              ref={(node) => {
                                input.current = node;
                              }}
                              value={editing.text}
                              disabled={busy}
                              onChange={(e) =>
                                setEditing({ ...editing, text: e.target.value })
                              }
                            >
                              <option value="">비어 있음</option>
                              {(p.type === "checkbox"
                                ? ["예", "아니요"]
                                : p.options || []
                              ).map((v) => (
                                <option key={v} value={v}>
                                  {v}
                                </option>
                              ))}
                            </select>
                          ) : (
                            <input
                              aria-label={`${p.name} 셀 입력`}
                              ref={(node) => {
                                input.current = node;
                              }}
                              type={p.type === "date" ? "date" : "text"}
                              inputMode={
                                ["number", "progress"].includes(p.type)
                                  ? "decimal"
                                  : undefined
                              }
                              value={editing.text}
                              disabled={busy}
                              onChange={(e) =>
                                setEditing({ ...editing, text: e.target.value })
                              }
                              onKeyDown={(e) => {
                                if (
                                  e.key === "Enter" &&
                                  (e.nativeEvent.isComposing ||
                                    e.keyCode === 229)
                                )
                                  e.preventDefault();
                              }}
                            />
                          )}
                          <div>
                            <Button type="submit" disabled={busy}>
                              셀 저장
                            </Button>
                            <Button
                              type="button"
                              disabled={busy}
                              onClick={() => {
                                setEditing(null);
                                setError("");
                                setFocus(r, c);
                              }}
                            >
                              셀 취소
                            </Button>
                          </div>
                        </form>
                      ) : derivedProperty(p) ||
                        ["ai", "relation"].includes(p.type) ? (
                        <AdvancedCell
                          property={p}
                          row={row}
                          onAction={canWrite ? onAction : undefined}
                        />
                      ) : (
                        <button
                          className="cell-button"
                          tabIndex={-1}
                          onClick={() => open(row, p, r, c)}
                        >
                          {p.type === "progress" ? (
                            <span className="progress-cell">
                              <progress
                                aria-label={`${p.name} 진행률`}
                                max={100}
                                value={Number(effectiveValue(row, p.id)) || 0}
                              />
                              <span>
                                {Number(effectiveValue(row, p.id)) || 0}%
                              </span>
                            </span>
                          ) : p.type === "checkbox" ? (
                            effectiveValue(row, p.id) ? (
                              "선택됨"
                            ) : (
                              "선택 안 됨"
                            )
                          ) : (
                            plainValue(effectiveValue(row, p.id)) || "비어 있음"
                          )}
                        </button>
                      )}
                    </td>
                  );
                })}
                <td>
                  <Button
                    onClick={() => onDetail(row)}
                    data-row-detail={row.id}
                    aria-label={`${r + 1}행 항목 상세`}
                  >
                    상세
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {canWrite && <Button onClick={onCreate}>새 항목 추가</Button>}
      </div>
      {range && metadata && (
        <Suspense fallback={<Loading />}>
          <RangePasteReview
            key={pasteText}
            databaseId={database.id}
            schema={metadata.schema_fingerprint}
            targets={range}
            onClose={() => setRange(null)}
            onSaved={onChanged}
          />
        </Suspense>
      )}
    </div>
  );
}
