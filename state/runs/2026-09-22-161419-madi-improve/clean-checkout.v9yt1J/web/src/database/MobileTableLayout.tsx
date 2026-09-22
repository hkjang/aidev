import { Fragment, type ReactNode, useEffect, useState, useRef } from "react";
import { useApp } from "../context";
import { usePreferenceWriter } from "../navigation/preferences";
import {
  AdvancedCell,
  effectiveValue,
  plainValue,
  type AdvancedDatabase,
  type AdvancedProperty,
  type AdvancedRow,
} from "../DatabaseAdvanced";
import { Button, Empty, Field } from "../ui";
import "../personalization/mobile.css";
export default function MobileTableLayout({
  database,
  rows,
  columns,
  children,
  onDetail,
  onAction,
  canWrite,
}: {
  database: AdvancedDatabase;
  rows: AdvancedRow[];
  columns: string[];
  children: ReactNode;
  onDetail: (row: AdvancedRow) => void;
  onAction: (property: AdvancedProperty, row: AdvancedRow) => void;
  canWrite: boolean;
}) {
  const { user, notify } = useApp(),
    write = usePreferenceWriter();
  const current = useRef(user.id),
    alive = useRef(true);
  current.current = user.id;
  useEffect(() => {
    alive.current = true;
    return () => {
      alive.current = false;
    };
  }, []);
  const [mobile, setMobile] = useState(
      () => matchMedia("(max-width: 640px)").matches,
    ),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    const media = matchMedia("(max-width: 640px)"),
      update = () => setMobile(media.matches);
    media.addEventListener("change", update);
    return () => media.removeEventListener("change", update);
  }, []);
  const cards = user.preferences.mobile_table_view !== "table";
  const properties = columns.length
    ? database.properties.filter((p) => columns.includes(p.id))
    : database.properties;
  return (
    <>
      <div className="mobile-database-presentation">
        <Field label="모바일 표 표시">
          <select
            value={cards ? "cards" : "table"}
            disabled={busy}
            onChange={(e) => {
              const actor = user.id;
              setBusy(true);
              void write({ mobile_table_view: e.target.value })
                .catch((error) => {
                  if (alive.current && current.current === actor)
                    notify(error.message, "error");
                })
                .finally(() => {
                  if (alive.current && current.current === actor)
                    setBusy(false);
                });
            }}
          >
            <option value="cards">카드 · 행별로 읽기</option>
            <option value="table">표 · 가로 비교와 편집</option>
          </select>
        </Field>
        <span className="muted">개인 설정에 저장됩니다.</span>
      </div>
      {mobile && cards ? (
        <div
          className="mobile-database-cards"
          aria-label="데이터베이스 행 카드"
        >
          {rows.length ? (
            rows.map((row, i) => (
              <article className="panel" key={row.id}>
                <strong>
                  {plainValue(effectiveValue(row, properties[0]?.id)) ||
                    `${i + 1}번째 항목`}
                </strong>
                <dl>
                  {properties.map((p) => (
                    <Fragment key={p.id}>
                      <dt>{p.name}</dt>
                      <dd>
                        <AdvancedCell
                          property={p}
                          row={row}
                          onAction={canWrite ? onAction : undefined}
                        />
                      </dd>
                    </Fragment>
                  ))}
                </dl>
                <Button
                  data-row-detail={row.id}
                  aria-label={`${i + 1}행 항목 상세`}
                  onClick={() => onDetail(row)}
                >
                  항목 상세
                </Button>
              </article>
            ))
          ) : (
            <Empty
              title="표시할 항목이 없습니다"
              text="필터를 변경하거나 새 항목을 추가하세요."
            />
          )}
        </div>
      ) : (
        children
      )}
    </>
  );
}
